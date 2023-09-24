// Copyright 2023 Google Inc. All Rights Reserved.
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

// Handler for isulad containers.
package isulad

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"golang.org/x/net/context"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	"github.com/google/cadvisor/container/docker"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/devicemapper"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
)

type isuladContainerHandler struct {
	machineInfoFactory info.MachineInfoFactory
	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	storageDriver    docker.StorageDriver
	fsInfo           fs.FsInfo
	rootfsStorageDir string

	creationTime time.Time

	// Metadata associated with the container.
	reference info.ContainerReference
	envs      map[string]string
	labels    map[string]string

	// Image name used for this container.
	image string

	// Filesystem handler.
	fsHandler common.FsHandler

	// The IP address of the container
	ipAddress string

	includedMetrics container.MetricSet

	thinPoolName string

	libcontainerHandler *containerlibcontainer.Handler
}

var _ container.ContainerHandler = &isuladContainerHandler{}

// newIsuladContainerHandler returns a new container.ContainerHandler
func newIsuladContainerHandler(
	client IsuladClient,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	fsInfo fs.FsInfo,
	storageDriver docker.StorageDriver,
	storageDir string,
	cgroupSubsystems map[string]string,
	inHostNamespace bool,
	metadataEnvAllowList []string,
	includedMetrics container.MetricSet,
	thinPoolName string,
	thinPoolWatcher *devicemapper.ThinPoolWatcher,
) (container.ContainerHandler, error) {
	// Create the cgroup paths.
	cgroupPaths := common.MakeCgroupPaths(cgroupSubsystems, name)

	// Generate the equivalent cgroup manager for this container.
	cgroupManager, err := containerlibcontainer.NewCgroupManager(name, cgroupPaths)
	if err != nil {
		return nil, err
	}

	rootfs := "/"
	if !inHostNamespace {
		rootfs = "/rootfs"
		storageDir = path.Join(rootfs, storageDir)
	}

	id := ContainerNameToIsuladID(name)

	// We assume that if load fails then the container is not known to isulad.
	cntr, err := client.InspectContainer(context.Background(), id)
	if err != nil {
		return nil, err
	}

	// Do not report network metrics for containers that share netns with another container.
	metrics := common.RemoveNetMetrics(includedMetrics, cntr.HostConfig.NetworkMode.IsContainer())

	rwLayerID, err := rwLayerID(storageDriver, storageDir, id)
	if err != nil {
		return nil, err
	}

	// rootfs storagedir of overlay and overlay2 both under .../overlay/<rwLayerID>/diff
	var rootfsStorageDir string
	if storageDriver == docker.OverlayStorageDriver || storageDriver == docker.Overlay2StorageDriver {
		rootfsStorageDir = path.Join(storageDir, "storage", "overlay", rwLayerID, "diff")
	}

	otherStorageDir := filepath.Join(storageDir, "storage", string(storageDriver)+"-containers", id)

	handler := &isuladContainerHandler{
		machineInfoFactory: machineInfoFactory,
		cgroupPaths:        cgroupPaths,
		storageDriver:      storageDriver,
		fsInfo:             fsInfo,
		rootfsStorageDir:   rootfsStorageDir,
		envs:               make(map[string]string),
		labels:             cntr.Config.Labels,
		image:              cntr.Config.Image,
		includedMetrics:    metrics,
		thinPoolName:       thinPoolName,
		fsHandler:          common.NewFsHandler(common.DefaultPeriod, rootfsStorageDir, otherStorageDir, fsInfo),
		reference: info.ContainerReference{
			Id:        id,
			Name:      name,
			Namespace: isuladNamespace,
			Aliases:   []string{id, name},
		},
		libcontainerHandler: containerlibcontainer.NewHandler(cgroupManager, rootfs, int(cntr.State.Pid), includedMetrics),
	}

	handler.creationTime, err = time.Parse(time.RFC3339, cntr.Created)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the create timestamp %q for container %q: %v", cntr.Created, id, err)
	}

	if cntr.RestartCount > 0 {
		handler.labels["restartcount"] = fmt.Sprintf("%d", cntr.RestartCount)
	}

	// Obtain the IP address for the container.
	// If the NetworkMode starts with 'container:' then we need to use the IP address of the container specified.
	// This happens in cases such as kubernetes where the containers doesn't have an IP address itself and we need to use the pod's address
	ipAddress := cntr.NetworkSettings.IPAddress
	networkMode := string(cntr.HostConfig.NetworkMode)
	if ipAddress == "" && strings.HasPrefix(networkMode, "container:") {
		containerID := strings.TrimPrefix(networkMode, "container:")
		c, err := client.InspectContainer(context.Background(), containerID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container %q: %v", id, err)
		}
		ipAddress = c.NetworkSettings.IPAddress
	}

	handler.ipAddress = ipAddress

	if metrics.Has(container.DiskUsageMetrics) {
		handler.fsHandler = &docker.FsHandler{
			FsHandler:       common.NewFsHandler(common.DefaultPeriod, rootfsStorageDir, otherStorageDir, fsInfo),
			ThinPoolWatcher: thinPoolWatcher,
			DeviceID:        cntr.GraphDriver.Data["DeviceId"],
		}
	}

	for _, exposedEnv := range metadataEnvAllowList {
		if exposedEnv == "" {
			// if no containerdEnvWhitelist provided, len(metadataEnvAllowList) == 1, metadataEnvAllowList[0] == ""
			continue
		}

		for _, envVar := range cntr.Config.Env {
			if envVar != "" {
				splits := strings.SplitN(envVar, "=", 2)
				if len(splits) == 2 && strings.HasPrefix(splits[0], exposedEnv) {
					handler.envs[splits[0]] = splits[1]
				}
			}
		}
	}

	return handler, nil
}

func (h *isuladContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return h.reference, nil
}

func (h *isuladContainerHandler) GetSpec() (info.ContainerSpec, error) {
	hasFilesystem := h.includedMetrics.Has(container.DiskUsageMetrics)
	hasNet := h.includedMetrics.Has(container.NetworkUsageMetrics)
	spec, err := common.GetSpec(h.cgroupPaths, h.machineInfoFactory, hasNet, hasFilesystem)
	spec.Labels = h.labels
	spec.Envs = h.envs
	spec.Image = h.image
	spec.CreationTime = h.creationTime

	return spec, err
}

func (h *isuladContainerHandler) GetStats() (*info.ContainerStats, error) {
	stats, err := h.libcontainerHandler.GetStats()
	if err != nil {
		return stats, err
	}

	if !h.includedMetrics.Has(container.NetworkUsageMetrics) {
		stats.Network = info.NetworkStats{}
	}

	err = docker.FsStats(stats, h.machineInfoFactory, h.includedMetrics, h.storageDriver,
		h.fsHandler, h.fsInfo, h.thinPoolName, h.rootfsStorageDir, "")
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (h *isuladContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	return []info.ContainerReference{}, nil
}

func (h *isuladContainerHandler) GetCgroupPath(resource string) (string, error) {
	var res string
	if !cgroups.IsCgroup2UnifiedMode() {
		res = resource
	}
	path, ok := h.cgroupPaths[res]
	if !ok {
		return "", fmt.Errorf("could not find path for resource %q for container %q", resource, h.reference.Name)
	}
	return path, nil
}

func (h *isuladContainerHandler) GetContainerLabels() map[string]string {
	return h.labels
}

func (h *isuladContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return h.libcontainerHandler.GetProcesses()
}

func (h *isuladContainerHandler) Exists() bool {
	return common.CgroupExists(h.cgroupPaths)
}

func (h *isuladContainerHandler) Type() container.ContainerType {
	return container.ContainerTypeContainerd
}

func (h *isuladContainerHandler) Start() {
	if h.fsHandler != nil {
		h.fsHandler.Start()
	}
}

func (h *isuladContainerHandler) Cleanup() {
	if h.fsHandler != nil {
		h.fsHandler.Stop()
	}
}

func (h *isuladContainerHandler) GetContainerIPAddress() string {
	return h.ipAddress
}
