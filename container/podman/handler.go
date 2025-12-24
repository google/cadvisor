// Copyright 2021 Google Inc. All Rights Reserved.
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

// Package podman implements a handler for Podman containers.
package podman

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/opencontainers/cgroups"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	"github.com/google/cadvisor/container/docker"
	dockerutil "github.com/google/cadvisor/container/docker/utils"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/devicemapper"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/zfs"
)

type containerHandler struct {
	// machineInfoFactory provides info.MachineInfo
	machineInfoFactory info.MachineInfoFactory

	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	// the docker storage driver
	storageDriver    docker.StorageDriver
	fsInfo           fs.FsInfo
	rootfsStorageDir string

	// Time at which this container was created.
	creationTime time.Time

	// Metadata associated with the container.
	envs   map[string]string
	labels map[string]string

	// Image name used for this container.
	image string

	networkMode dockercontainer.NetworkMode

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
}

func newContainerHandler(
	name string,
	machineInfoFactory info.MachineInfoFactory,
	fsInfo fs.FsInfo,
	storageDriver docker.StorageDriver,
	storageDir string,
	cgroupSubsystems map[string]string,
	inHostNamespace bool,
	metadataEnvAllowList []string,
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

	rootless := path.Base(name) == containerBaseName
	if rootless {
		name, _ = path.Split(name)
	}

	id := dockerutil.ContainerNameToId(name)

	// We assume that if Inspect fails then the container is not known to Podman.
	ctnr, err := InspectContainer(id)
	if err != nil {
		return nil, err
	}

	// Obtain the IP address for the container.
	var ipAddress string
	if ctnr.NetworkSettings != nil && ctnr.HostConfig != nil {
		c := ctnr
		if ctnr.HostConfig.NetworkMode.IsContainer() {
			// If the NetworkMode starts with 'container:' then we need to use the IP address of the container specified.
			// This happens in cases such as kubernetes where the containers doesn't have an IP address itself and we need to use the pod's address
			containerID := ctnr.HostConfig.NetworkMode.ConnectedContainer()
			c, err = InspectContainer(containerID)
			if err != nil {
				return nil, fmt.Errorf("failed to inspect container %q: %v", containerID, err)
			}
		}
		if nw, ok := c.NetworkSettings.Networks[c.HostConfig.NetworkMode.NetworkName()]; ok {
			ipAddress = nw.IPAddress
		}
	}

	layerID, err := rwLayerID(storageDriver, storageDir, id)
	if err != nil {
		return nil, err
	}

	// Determine the rootfs storage dir OR the pool name to determine the device.
	// For devicemapper, we only need the thin pool name, and that is passed in to this call
	rootfsStorageDir, zfsFilesystem, zfsParent, err := determineDeviceStorage(storageDriver, storageDir, layerID)
	if err != nil {
		return nil, err
	}

	otherStorageDir := filepath.Join(storageDir, string(storageDriver)+"-containers", id)

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
		networkMode:        ctnr.HostConfig.NetworkMode,
		fsHandler:          common.NewFsHandler(common.DefaultPeriod, rootfsStorageDir, otherStorageDir, fsInfo),
		metrics:            metrics,
		thinPoolName:       thinPoolName,
		zfsParent:          zfsParent,
		reference: info.ContainerReference{
			// Add the name and bare ID as aliases of the container.
			Id:        id,
			Name:      name,
			Aliases:   []string{strings.TrimPrefix(ctnr.Name, "/"), id},
			Namespace: Namespace,
		},
		libcontainerHandler: containerlibcontainer.NewHandler(cgroupManager, rootFs, ctnr.State.Pid, metrics),
	}

	handler.creationTime, err = time.Parse(time.RFC3339, ctnr.Created)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the create timestamp %q for container %q: %v", ctnr.Created, id, err)
	}

	if ctnr.RestartCount > 0 {
		handler.labels["restartcount"] = strconv.Itoa(ctnr.RestartCount)
	}

	if metrics.Has(container.DiskUsageMetrics) {
		handler.fsHandler = &docker.FsHandler{
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

func determineDeviceStorage(storageDriver docker.StorageDriver, storageDir string, rwLayerID string) (
	rootfsStorageDir string, zfsFilesystem string, zfsParent string, err error) {
	switch storageDriver {
	// Podman aliased the driver names together.
	case docker.OverlayStorageDriver, docker.Overlay2StorageDriver:
		rootfsStorageDir = path.Join(storageDir, "overlay", rwLayerID, "diff")
		return
	default:
		return docker.DetermineDeviceStorage(storageDriver, storageDir, rwLayerID)
	}
}

func (h *containerHandler) ContainerReference() (info.ContainerReference, error) {
	return h.reference, nil
}

func (h *containerHandler) needNet() bool {
	if h.metrics.Has(container.NetworkUsageMetrics) {
		h.networkMode.IsContainer()
		return !h.networkMode.IsContainer()
	}
	return false
}

func (h *containerHandler) GetSpec() (info.ContainerSpec, error) {
	hasFilesystem := h.metrics.Has(container.DiskUsageMetrics)

	spec, err := common.GetSpec(h.cgroupPaths, h.machineInfoFactory, h.needNet(), hasFilesystem)
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
	stats, err := h.libcontainerHandler.GetStats()
	if err != nil {
		return stats, err
	}

	if !h.needNet() {
		stats.Network = info.NetworkStats{}
	}

	// Get filesystem stats.
	err = docker.FsStats(stats, h.machineInfoFactory, h.metrics, h.storageDriver,
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
	return container.ContainerTypePodman
}

func (h *containerHandler) GetExitCode() (int, error) {
	ctnr, err := InspectContainer(h.reference.Id)
	if err != nil {
		return -1, fmt.Errorf("failed to inspect container %s: %w", h.reference.Id, err)
	}

	if ctnr.State == nil {
		return -1, fmt.Errorf("container state not available for %s", h.reference.Id)
	}

	if ctnr.State.Running {
		return -1, fmt.Errorf("container %s is still running", h.reference.Id)
	}

	return ctnr.State.ExitCode, nil
}
