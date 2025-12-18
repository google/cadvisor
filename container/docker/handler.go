// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

// Package docker implements a handler for Docker containers.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	dclient "github.com/docker/docker/client"
	"github.com/opencontainers/cgroups"
	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	"github.com/google/cadvisor/container/containerd"
	"github.com/google/cadvisor/container/containerd/namespaces"
	dockerutil "github.com/google/cadvisor/container/docker/utils"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/devicemapper"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/zfs"
)

const (
	// The read write layers exist here.
	aufsRWLayer     = "diff"
	overlayRWLayer  = "upper"
	overlay2RWLayer = "diff"

	// Path to the directory where docker stores log files if the json logging driver is enabled.
	pathToContainersDir = "containers"
)

type containerHandler struct {
	// machineInfoFactory provides info.MachineInfo
	machineInfoFactory info.MachineInfoFactory

	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	// the docker storage driver
	storageDriver    StorageDriver
	fsInfo           fs.FsInfo
	rootfsStorageDir string

	// Time at which this container was created.
	creationTime time.Time

	// Metadata associated with the container.
	envs   map[string]string
	labels map[string]string

	// Image name used for this container.
	image string

	// Filesystem handler.
	fsHandler common.FsHandler

	// The IP address of the container
	ipAddress string

	metrics container.MetricSet

	// the devicemapper poolname
	thinPoolName string

	// zfsParent is the parent for docker zfs
	zfsParent string

	// Reference to the container
	reference info.ContainerReference

	libcontainerHandler *containerlibcontainer.Handler

	// the docker client is needed to inspect the container and get the health status
	client dclient.APIClient
}

var _ container.ContainerHandler = &containerHandler{}

func getRwLayerID(containerID, storageDir string, sd StorageDriver, dockerVersion []int) (string, error) {
	const (
		// Docker version >=1.10.0 have a randomized ID for the root fs of a container.
		randomizedRWLayerMinorVersion = 10
		rwLayerIDFile                 = "mount-id"
	)
	if (dockerVersion[0] <= 1) && (dockerVersion[1] < randomizedRWLayerMinorVersion) {
		return containerID, nil
	}

	bytes, err := os.ReadFile(path.Join(storageDir, "image", string(sd), "layerdb", "mounts", containerID, rwLayerIDFile))
	if err != nil {
		return "", fmt.Errorf("failed to identify the read-write layer ID for container %q. - %v", containerID, err)
	}
	return string(bytes), err
}

// newContainerHandler returns a new container.ContainerHandler
func newContainerHandler(
	client *dclient.Client,
	containerdClient containerd.ContainerdClient,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	fsInfo fs.FsInfo,
	storageDriver StorageDriver,
	storageDir string,
	cgroupSubsystems map[string]string,
	inHostNamespace bool,
	metadataEnvAllowList []string,
	dockerVersion []int,
	metrics container.MetricSet,
	thinPoolName string,
	thinPoolWatcher *devicemapper.ThinPoolWatcher,
	zfsWatcher *zfs.ZfsWatcher,
) (container.ContainerHandler, error) {
	// Create the cgroup paths.
	cgroupPaths := common.MakeCgroupPaths(cgroupSubsystems, name)

	// Generate the equivalent cgroup manager for this container.
	cgroupManager, err := containerlibcontainer.NewCgroupManager(name, cgroupPaths)
	if err != nil {
		return nil, err
	}

	rootFs := "/"
	if !inHostNamespace {
		rootFs = "/rootfs"
		storageDir = path.Join(rootFs, storageDir)
	}

	id := dockerutil.ContainerNameToId(name)

	// Add the Containers dir where the log files are stored.
	// FIXME: Give `otherStorageDir` a more descriptive name.
	otherStorageDir := path.Join(storageDir, pathToContainersDir, id)

	var rootfsStorageDir, zfsFilesystem, zfsParent string
	if storageDriver == ContainerdSnapshotterStorageDriver {
		ctx := namespaces.WithNamespace(context.Background(), "moby")
		cntr, err := containerdClient.LoadContainer(ctx, id)
		if err != nil {
			return nil, err
		}

		var spec specs.Spec
		if err := json.Unmarshal(cntr.Spec.Value, &spec); err != nil {
			return nil, err
		}
		rootfsStorageDir = spec.Root.Path
	} else {
		rwLayerID, err := getRwLayerID(id, storageDir, storageDriver, dockerVersion)
		if err != nil {
			return nil, err
		}

		// Determine the rootfs storage dir OR the pool name to determine the device.
		// For devicemapper, we only need the thin pool name, and that is passed in to this call
		rootfsStorageDir, zfsFilesystem, zfsParent, err = DetermineDeviceStorage(storageDriver, storageDir, rwLayerID)
		if err != nil {
			return nil, fmt.Errorf("unable to determine device storage: %v", err)
		}
	}

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %q: %v", id, err)
	}

	// Obtain the IP address for the container.
	var ipAddress string
	if ctnr.NetworkSettings != nil && ctnr.HostConfig != nil {
		c := ctnr
		if ctnr.HostConfig.NetworkMode.IsContainer() {
			// If the NetworkMode starts with 'container:' then we need to use the IP address of the container specified.
			// This happens in cases such as kubernetes where the containers doesn't have an IP address itself and we need to use the pod's address
			containerID := ctnr.HostConfig.NetworkMode.ConnectedContainer()
			c, err = client.ContainerInspect(context.Background(), containerID)
			if err != nil {
				return nil, fmt.Errorf("failed to inspect container %q: %v", containerID, err)
			}
		}
		if nw, ok := c.NetworkSettings.Networks[c.HostConfig.NetworkMode.NetworkName()]; ok {
			ipAddress = nw.IPAddress
		}
	}

	// Do not report network metrics for containers that share netns with another container.
	includedMetrics := common.RemoveNetMetrics(metrics, ctnr.HostConfig.NetworkMode.IsContainer())

	handler := &containerHandler{
		machineInfoFactory: machineInfoFactory,
		cgroupPaths:        cgroupPaths,
		storageDriver:      storageDriver,
		fsInfo:             fsInfo,
		rootfsStorageDir:   rootfsStorageDir,
		ipAddress:          ipAddress,
		envs:               make(map[string]string),
		labels:             ctnr.Config.Labels,
		image:              ctnr.Config.Image,
		metrics:            includedMetrics,
		thinPoolName:       thinPoolName,
		zfsParent:          zfsParent,
		client:             client,
		reference: info.ContainerReference{
			// Add the name and bare ID as aliases of the container.
			Id:        id,
			Name:      name,
			Aliases:   []string{strings.TrimPrefix(ctnr.Name, "/"), id},
			Namespace: DockerNamespace,
		},
		libcontainerHandler: containerlibcontainer.NewHandler(cgroupManager, rootFs, ctnr.State.Pid, metrics),
	}

	// Timestamp returned by Docker is in time.RFC3339Nano format.
	handler.creationTime, err = time.Parse(time.RFC3339Nano, ctnr.Created)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the create timestamp %q for container %q: %v", ctnr.Created, id, err)
	}

	if ctnr.RestartCount > 0 {
		handler.labels["restartcount"] = strconv.Itoa(ctnr.RestartCount)
	}

	if includedMetrics.Has(container.DiskUsageMetrics) {
		handler.fsHandler = &FsHandler{
			FsHandler:       common.NewFsHandler(common.DefaultPeriod, rootfsStorageDir, otherStorageDir, fsInfo),
			ThinPoolWatcher: thinPoolWatcher,
			ZfsWatcher:      zfsWatcher,
			DeviceID:        ctnr.GraphDriver.Data["DeviceId"],
			ZfsFilesystem:   zfsFilesystem,
		}
	}

	// Split env vars to get metadata map.
	for _, exposedEnv := range metadataEnvAllowList {
		if exposedEnv == "" {
			// if no dockerEnvWhitelist provided, len(metadataEnvAllowList) == 1, metadataEnvAllowList[0] == ""
			continue
		}

		for _, envVar := range ctnr.Config.Env {
			if envVar != "" {
				splits := strings.SplitN(envVar, "=", 2)
				if len(splits) == 2 && strings.HasPrefix(splits[0], exposedEnv) {
					handler.envs[strings.ToLower(splits[0])] = splits[1]
				}
			}
		}
	}

	return handler, nil
}

func DetermineDeviceStorage(storageDriver StorageDriver, storageDir string, rwLayerID string) (
	rootfsStorageDir string, zfsFilesystem string, zfsParent string, err error) {
	switch storageDriver {
	case AufsStorageDriver:
		rootfsStorageDir = path.Join(storageDir, string(AufsStorageDriver), aufsRWLayer, rwLayerID)
	case OverlayStorageDriver:
		rootfsStorageDir = path.Join(storageDir, string(storageDriver), rwLayerID, overlayRWLayer)
	case Overlay2StorageDriver:
		rootfsStorageDir = path.Join(storageDir, string(storageDriver), rwLayerID, overlay2RWLayer)
	case VfsStorageDriver:
		rootfsStorageDir = path.Join(storageDir)
	case ZfsStorageDriver:
		var status info.DockerStatus
		status, err = Status()
		if err != nil {
			return
		}
		zfsParent = status.DriverStatus[dockerutil.DriverStatusParentDataset]
		zfsFilesystem = path.Join(zfsParent, rwLayerID)
	}
	return
}

func (h *containerHandler) ContainerReference() (info.ContainerReference, error) {
	return h.reference, nil
}

func (h *containerHandler) GetSpec() (info.ContainerSpec, error) {
	hasFilesystem := h.metrics.Has(container.DiskUsageMetrics)
	hasNetwork := h.metrics.Has(container.NetworkUsageMetrics)
	spec, err := common.GetSpec(h.cgroupPaths, h.machineInfoFactory, hasNetwork, hasFilesystem)
	if err != nil {
		return info.ContainerSpec{}, err
	}

	spec.Labels = h.labels
	spec.Envs = h.envs
	spec.Image = h.image
	spec.CreationTime = h.creationTime

	return spec, nil
}

func (h *containerHandler) GetStats() (*info.ContainerStats, error) {
	// TODO(vmarmol): Get from libcontainer API instead of cgroup manager when we don't have to support older Dockers.
	stats, err := h.libcontainerHandler.GetStats()
	if err != nil {
		return stats, err
	}

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := h.client.ContainerInspect(context.Background(), h.reference.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %q: %v", h.reference.Id, err)
	}

	if ctnr.State.Health != nil {
		stats.Health.Status = ctnr.State.Health.Status
	}

	// Get filesystem stats.
	err = FsStats(stats, h.machineInfoFactory, h.metrics, h.storageDriver,
		h.fsHandler, h.fsInfo, h.thinPoolName, h.rootfsStorageDir, h.zfsParent)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (h *containerHandler) ListContainers(container.ListType) ([]info.ContainerReference, error) {
	return []info.ContainerReference{}, nil
}

func (h *containerHandler) ListProcesses(container.ListType) ([]int, error) {
	return h.libcontainerHandler.GetProcesses()
}

func (h *containerHandler) GetCgroupPath(resource string) (string, error) {
	var res string
	if !cgroups.IsCgroup2UnifiedMode() {
		res = resource
	}
	cgroupPath, ok := h.cgroupPaths[res]
	if !ok {
		return "", fmt.Errorf("could not find path for resource %q for container %q", resource, h.reference.Name)
	}
	return cgroupPath, nil
}

func (h *containerHandler) GetContainerLabels() map[string]string {
	return h.labels
}

func (h *containerHandler) GetContainerIPAddress() string {
	return h.ipAddress
}

func (h *containerHandler) Exists() bool {
	return common.CgroupExists(h.cgroupPaths)
}

func (h *containerHandler) Cleanup() {
	if h.fsHandler != nil {
		h.fsHandler.Stop()
	}
}

func (h *containerHandler) Start() {
	if h.fsHandler != nil {
		h.fsHandler.Start()
	}
}

func (h *containerHandler) Type() container.ContainerType {
	return container.ContainerTypeDocker
}

func (h *containerHandler) GetExitCode() (int, error) {
	ctnr, err := h.client.ContainerInspect(context.Background(), h.reference.Id)
	if err != nil {
		return -1, fmt.Errorf("failed to inspect container %s: %w", h.reference.Id, err)
	}

	if ctnr.State.Running {
		return -1, fmt.Errorf("container %s is still running", h.reference.Id)
	}

	return ctnr.State.ExitCode, nil
}
