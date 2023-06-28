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

package podman

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/opencontainers/runc/libcontainer/cgroups"

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

type podmanContainerHandler struct {
	// machineInfoFactory provides info.MachineInfo
	machineInfoFactory info.MachineInfoFactory

	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	storageDriver    docker.StorageDriver
	fsInfo           fs.FsInfo
	rootfsStorageDir string

	creationTime time.Time

	// Metadata associated with the container.
	envs   map[string]string
	labels map[string]string

	image string

	networkMode dockercontainer.NetworkMode

	fsHandler common.FsHandler

	ipAddress string

	metrics container.MetricSet

	thinPoolName string

	zfsParent string

	reference info.ContainerReference

	libcontainerHandler *containerlibcontainer.Handler
}

func newPodmanContainerHandler(
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

	rwLayerID, err := rwLayerID(storageDriver, storageDir, id)
	if err != nil {
		return nil, err
	}

	rootfsStorageDir, zfsParent, zfsFilesystem, err := determineDeviceStorage(storageDriver, storageDir, rwLayerID)
	if err != nil {
		return nil, err
	}

	otherStorageDir := filepath.Join(storageDir, string(storageDriver)+"-containers", id)

	handler := &podmanContainerHandler{
		machineInfoFactory: machineInfoFactory,
		cgroupPaths:        cgroupPaths,
		storageDriver:      storageDriver,
		fsInfo:             fsInfo,
		rootfsStorageDir:   rootfsStorageDir,
		ipAddress:          ctnr.NetworkSettings.IPAddress,
		envs:               make(map[string]string),
		labels:             ctnr.Config.Labels,
		image:              ctnr.Config.Image,
		networkMode:        ctnr.HostConfig.NetworkMode,
		fsHandler:          common.NewFsHandler(common.DefaultPeriod, rootfsStorageDir, otherStorageDir, fsInfo),
		metrics:            metrics,
		thinPoolName:       thinPoolName,
		zfsParent:          zfsParent,
		reference: info.ContainerReference{
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
		handler.labels["restartcount"] = fmt.Sprint(ctnr.RestartCount)
	}

	// Obtain the IP address for the container.
	// If the NetworkMode starts with 'container:' then we need to use the IP address of the container specified.
	// This happens in cases such as kubernetes where the containers doesn't have an IP address itself and we need to use the pod's address
	networkMode := string(handler.networkMode)
	if handler.ipAddress == "" && strings.HasPrefix(networkMode, "container:") {
		id := strings.TrimPrefix(networkMode, "container:")
		ctnr, err := InspectContainer(id)
		if err != nil {
			return nil, err
		}
		handler.ipAddress = ctnr.NetworkSettings.IPAddress
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

func (p podmanContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return p.reference, nil
}

func (p podmanContainerHandler) needNet() bool {
	if p.metrics.Has(container.NetworkUsageMetrics) {
		p.networkMode.IsContainer()
		return !p.networkMode.IsContainer()
	}
	return false
}

func (p podmanContainerHandler) GetSpec() (info.ContainerSpec, error) {
	hasFilesystem := p.metrics.Has(container.DiskUsageMetrics)

	spec, err := common.GetSpec(p.cgroupPaths, p.machineInfoFactory, p.needNet(), hasFilesystem)
	if err != nil {
		return info.ContainerSpec{}, err
	}

	spec.Labels = p.labels
	spec.Envs = p.envs
	spec.Image = p.image
	spec.CreationTime = p.creationTime

	return spec, nil
}

func (p podmanContainerHandler) GetStats() (*info.ContainerStats, error) {
	stats, err := p.libcontainerHandler.GetStats()
	if err != nil {
		return stats, err
	}

	if !p.needNet() {
		stats.Network = info.NetworkStats{}
	}

	err = docker.FsStats(stats, p.machineInfoFactory, p.metrics, p.storageDriver,
		p.fsHandler, p.fsInfo, p.thinPoolName, p.rootfsStorageDir, p.zfsParent)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (p podmanContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	return []info.ContainerReference{}, nil
}

func (p podmanContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return p.libcontainerHandler.GetProcesses()
}

func (p podmanContainerHandler) GetCgroupPath(resource string) (string, error) {
	var res string
	if !cgroups.IsCgroup2UnifiedMode() {
		res = resource
	}
	path, ok := p.cgroupPaths[res]
	if !ok {
		return "", fmt.Errorf("couldn't find path for resource %q for container %q", resource, p.reference.Name)
	}

	return path, nil
}

func (p podmanContainerHandler) GetContainerLabels() map[string]string {
	return p.labels
}

func (p podmanContainerHandler) GetContainerIPAddress() string {
	return p.ipAddress
}

func (p podmanContainerHandler) Exists() bool {
	return common.CgroupExists(p.cgroupPaths)
}

func (p podmanContainerHandler) Cleanup() {
	if p.fsHandler != nil {
		p.fsHandler.Stop()
	}
}

func (p podmanContainerHandler) Start() {
	if p.fsHandler != nil {
		p.fsHandler.Start()
	}
}

func (p podmanContainerHandler) Type() container.ContainerType {
	return container.ContainerTypePodman
}
