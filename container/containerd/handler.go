// Copyright 2017 Google Inc. All Rights Reserved.
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

// Handler for containerd containers.
package containerd

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/containerd/containerd/errdefs"
	cgroupfs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	libcontainerconfigs "github.com/opencontainers/runc/libcontainer/configs"
	"golang.org/x/net/context"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type containerdContainerHandler struct {
	machineInfoFactory info.MachineInfoFactory
	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string
	fsInfo      fs.FsInfo
	// Metadata associated with the container.
	reference info.ContainerReference
	envs      map[string]string
	labels    map[string]string
	// Image name used for this container.
	image string
	// Filesystem handler.
	includedMetrics container.MetricSet

	libcontainerHandler *containerlibcontainer.Handler
}

var _ container.ContainerHandler = &containerdContainerHandler{}

// newContainerdContainerHandler returns a new container.ContainerHandler
func newContainerdContainerHandler(
	client containerdClient,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	fsInfo fs.FsInfo,
	cgroupSubsystems *containerlibcontainer.CgroupSubsystems,
	inHostNamespace bool,
	metadataEnvs []string,
	includedMetrics container.MetricSet,
) (container.ContainerHandler, error) {
	// Create the cgroup paths.
	cgroupPaths := make(map[string]string, len(cgroupSubsystems.MountPoints))
	for key, val := range cgroupSubsystems.MountPoints {
		cgroupPaths[key] = path.Join(val, name)
	}

	// Generate the equivalent cgroup manager for this container.
	cgroupManager := &cgroupfs.Manager{
		Cgroups: &libcontainerconfigs.Cgroup{
			Name: name,
		},
		Paths: cgroupPaths,
	}

	id := ContainerNameToContainerdID(name)
	// We assume that if load fails then the container is not known to containerd.
	ctx := context.Background()
	cntr, err := client.LoadContainer(ctx, id)
	if err != nil {
		return nil, err
	}

	var spec specs.Spec
	if err := json.Unmarshal(cntr.Spec.Value, &spec); err != nil {
		return nil, err
	}

	// Cgroup is created during task creation. When cadvisor sees the cgroup,
	// task may not be fully created yet. Use a retry+backoff to tolerant the
	// race condition.
	// TODO(random-liu): Use cri-containerd client to talk with cri-containerd
	// instead. cri-containerd has some internal synchronization to make sure
	// `ContainerStatus` only returns result after `StartContainer` finishes.
	var taskPid uint32
	backoff := 100 * time.Millisecond
	retry := 5
	for {
		taskPid, err = client.TaskPid(ctx, id)
		if err == nil {
			break
		}
		retry--
		if !errdefs.IsNotFound(err) || retry == 0 {
			return nil, err
		}
		time.Sleep(backoff)
		backoff *= 2
	}

	rootfs := "/"
	if !inHostNamespace {
		rootfs = "/rootfs"
	}

	containerReference := info.ContainerReference{
		Id:        id,
		Name:      name,
		Namespace: k8sContainerdNamespace,
		Aliases:   []string{id, name},
	}

	libcontainerHandler := containerlibcontainer.NewHandler(cgroupManager, rootfs, int(taskPid), includedMetrics)

	handler := &containerdContainerHandler{
		machineInfoFactory:  machineInfoFactory,
		cgroupPaths:         cgroupPaths,
		fsInfo:              fsInfo,
		envs:                make(map[string]string),
		labels:              cntr.Labels,
		includedMetrics:     includedMetrics,
		reference:           containerReference,
		libcontainerHandler: libcontainerHandler,
	}
	// Add the name and bare ID as aliases of the container.
	handler.image = cntr.Image
	for _, envVar := range spec.Process.Env {
		if envVar != "" {
			splits := strings.SplitN(envVar, "=", 2)
			if len(splits) == 2 {
				handler.envs[splits[0]] = splits[1]
			}
		}
	}

	return handler, nil
}

func (self *containerdContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return self.reference, nil
}

func (self *containerdContainerHandler) needNet() bool {
	// Since containerd does not handle networking ideally we need to return based
	// on includedMetrics list. Here the assumption is the presence of cri-containerd
	// label
	if self.includedMetrics.Has(container.NetworkUsageMetrics) {
		//TODO change it to exported cri-containerd constants
		return self.labels["io.cri-containerd.kind"] == "sandbox"
	}
	return false
}

func (self *containerdContainerHandler) GetSpec() (info.ContainerSpec, error) {
	// TODO: Since we dont collect disk usage stats for containerd, we set hasFilesystem
	// to false. Revisit when we support disk usage stats for containerd
	hasFilesystem := false
	spec, err := common.GetSpec(self.cgroupPaths, self.machineInfoFactory, self.needNet(), hasFilesystem)
	spec.Labels = self.labels
	spec.Envs = self.envs
	spec.Image = self.image

	return spec, err
}

func (self *containerdContainerHandler) getFsStats(stats *info.ContainerStats) error {
	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return err
	}

	if self.includedMetrics.Has(container.DiskIOMetrics) {
		common.AssignDeviceNamesToDiskStats((*common.MachineInfoNamer)(mi), &stats.DiskIo)
	}
	return nil
}

func (self *containerdContainerHandler) GetStats() (*info.ContainerStats, error) {
	stats, err := self.libcontainerHandler.GetStats()
	if err != nil {
		return stats, err
	}
	// Clean up stats for containers that don't have their own network - this
	// includes containers running in Kubernetes pods that use the network of the
	// infrastructure container. This stops metrics being reported multiple times
	// for each container in a pod.
	if !self.needNet() {
		stats.Network = info.NetworkStats{}
	}

	// Get filesystem stats.
	err = self.getFsStats(stats)
	return stats, err
}

func (self *containerdContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	return []info.ContainerReference{}, nil
}

func (self *containerdContainerHandler) GetCgroupPath(resource string) (string, error) {
	path, ok := self.cgroupPaths[resource]
	if !ok {
		return "", fmt.Errorf("could not find path for resource %q for container %q\n", resource, self.reference.Name)
	}
	return path, nil
}

func (self *containerdContainerHandler) GetContainerLabels() map[string]string {
	return self.labels
}

func (self *containerdContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return self.libcontainerHandler.GetProcesses()
}

func (self *containerdContainerHandler) Exists() bool {
	return common.CgroupExists(self.cgroupPaths)
}

func (self *containerdContainerHandler) Type() container.ContainerType {
	return container.ContainerTypeContainerd
}

func (self *containerdContainerHandler) Start() {
}

func (self *containerdContainerHandler) Cleanup() {
}

func (self *containerdContainerHandler) GetContainerIPAddress() string {
	// containerd doesnt take care of networking.So it doesnt maintain networking states
	return ""
}
