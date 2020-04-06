// Copyright 2018 Google Inc. All Rights Reserved.
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

// Handler for "mesos" containers.
package mesos

import (
	"fmt"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
)

type mesosContainerHandler struct {
	// Name of the container for this handler.
	name string

	// machineInfoFactory provides info.MachineInfo
	machineInfoFactory info.MachineInfoFactory

	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	// File System Info
	fsInfo fs.FsInfo

	// Metrics to be included.
	includedMetrics container.MetricSet

	labels map[string]string

	// Reference to the container
	reference info.ContainerReference

	libcontainerHandler *containerlibcontainer.Handler
}

func newMesosContainerHandler(
	name string,
	cgroupSubsystems *containerlibcontainer.CgroupSubsystems,
	machineInfoFactory info.MachineInfoFactory,
	fsInfo fs.FsInfo,
	includedMetrics container.MetricSet,
	inHostNamespace bool,
	client mesosAgentClient,
) (container.ContainerHandler, error) {
	cgroupPaths := common.MakeCgroupPaths(cgroupSubsystems.MountPoints, name)

	// Generate the equivalent cgroup manager for this container.
	cgroupManager, err := containerlibcontainer.NewCgroupManager(name, cgroupPaths)
	if err != nil {
		return nil, err
	}

	rootFs := "/"
	if !inHostNamespace {
		rootFs = "/rootfs"
	}

	id := ContainerNameToMesosId(name)

	cinfo, err := client.ContainerInfo(id)

	if err != nil {
		return nil, err
	}

	labels := cinfo.labels
	pid, err := client.ContainerPid(id)
	if err != nil {
		return nil, err
	}

	libcontainerHandler := containerlibcontainer.NewHandler(cgroupManager, rootFs, pid, includedMetrics)

	reference := info.ContainerReference{
		Id:        id,
		Name:      name,
		Namespace: MesosNamespace,
		Aliases:   []string{id, name},
	}

	handler := &mesosContainerHandler{
		name:                name,
		machineInfoFactory:  machineInfoFactory,
		cgroupPaths:         cgroupPaths,
		fsInfo:              fsInfo,
		includedMetrics:     includedMetrics,
		labels:              labels,
		reference:           reference,
		libcontainerHandler: libcontainerHandler,
	}

	return handler, nil
}

func (h *mesosContainerHandler) ContainerReference() (info.ContainerReference, error) {
	// We only know the container by its one name.
	return h.reference, nil
}

// Nothing to start up.
func (h *mesosContainerHandler) Start() {}

// Nothing to clean up.
func (h *mesosContainerHandler) Cleanup() {}

func (h *mesosContainerHandler) GetSpec() (info.ContainerSpec, error) {
	// TODO: Since we dont collect disk usage and network stats for mesos containers, we set
	// hasFilesystem and hasNetwork to false. Revisit when we support disk usage, network
	// stats for mesos containers.
	hasNetwork := false
	hasFilesystem := false

	spec, err := common.GetSpec(h.cgroupPaths, h.machineInfoFactory, hasNetwork, hasFilesystem)
	if err != nil {
		return spec, err
	}

	spec.Labels = h.labels

	return spec, nil
}

func (h *mesosContainerHandler) getFsStats(stats *info.ContainerStats) error {

	mi, err := h.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return err
	}

	if h.includedMetrics.Has(container.DiskIOMetrics) {
		common.AssignDeviceNamesToDiskStats((*common.MachineInfoNamer)(mi), &stats.DiskIo)
	}

	return nil
}

func (h *mesosContainerHandler) GetStats() (*info.ContainerStats, error) {
	stats, err := h.libcontainerHandler.GetStats()
	if err != nil {
		return stats, err
	}

	// Get filesystem stats.
	err = h.getFsStats(stats)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (h *mesosContainerHandler) GetCgroupPath(resource string) (string, error) {
	path, ok := h.cgroupPaths[resource]
	if !ok {
		return "", fmt.Errorf("could not find path for resource %q for container %q\n", resource, h.name)
	}
	return path, nil
}

func (h *mesosContainerHandler) GetContainerLabels() map[string]string {
	return h.labels
}

func (h *mesosContainerHandler) GetContainerIPAddress() string {
	// the IP address for the mesos container corresponds to the system ip address.
	return "127.0.0.1"
}

func (h *mesosContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	return common.ListContainers(h.name, h.cgroupPaths, listType)
}

func (h *mesosContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return h.libcontainerHandler.GetProcesses()
}

func (h *mesosContainerHandler) Exists() bool {
	return common.CgroupExists(h.cgroupPaths)
}

func (h *mesosContainerHandler) Type() container.ContainerType {
	return container.ContainerTypeMesos
}
