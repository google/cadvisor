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

// Handler for Docker containers.
package docker

import (
	"fmt"
	"math"
	"path"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/cadvisor/container"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	cgroupfs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	libcontainerconfigs "github.com/opencontainers/runc/libcontainer/configs"
)

// Path to aufs dir where all the files exist.
// aufs/layers is ignored here since it does not hold a lot of data.
// aufs/mnt contains the mount points used to compose the rootfs. Hence it is also ignored.
const (
	pathToAufsDir       = "aufs/diff"
	pathToContainersDir = "containers"
)

type dockerContainerHandler struct {
	client             *docker.Client
	name               string
	id                 string
	aliases            []string
	machineInfoFactory info.MachineInfoFactory

	// Path to the libcontainer config file.
	libcontainerConfigPath string

	// Path to the libcontainer state file.
	libcontainerStatePath string

	// TODO(vmarmol): Remove when we depend on a newer Docker.
	// Path to the libcontainer pid file.
	libcontainerPidPath string

	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	// Manager of this container's cgroups.
	cgroupManager cgroups.Manager

	storageDriver storageDriver
	fsInfo        fs.FsInfo
	storageDirs   []string

	// Time at which this container was created.
	creationTime time.Time

	// Metadata labels associated with the container.
	labels map[string]string

	// The container PID used to switch namespaces as required
	pid int

	// Image name used for this container.
	image string

	// The host root FS to read
	rootFs string

	// The network mode of the container
	networkMode string

	// Filesystem handler.
	fsHandler fsHandler
}

func newDockerContainerHandler(
	client *docker.Client,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	fsInfo fs.FsInfo,
	storageDriver storageDriver,
	cgroupSubsystems *containerlibcontainer.CgroupSubsystems,
	inHostNamespace bool,
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

	rootFs := "/"
	if !inHostNamespace {
		rootFs = "/rootfs"
	}

	id := ContainerNameToDockerId(name)

	// Add the Containers dir where the log files are stored.
	storageDirs := []string{path.Join(*dockerRootDir, pathToContainersDir, id)}

	switch storageDriver {
	case aufsStorageDriver:
		// Add writable layer for aufs.
		storageDirs = append(storageDirs, path.Join(*dockerRootDir, pathToAufsDir, id))
	}

	handler := &dockerContainerHandler{
		id:                 id,
		client:             client,
		name:               name,
		machineInfoFactory: machineInfoFactory,
		cgroupPaths:        cgroupPaths,
		cgroupManager:      cgroupManager,
		storageDriver:      storageDriver,
		fsInfo:             fsInfo,
		rootFs:             rootFs,
		storageDirs:        storageDirs,
		fsHandler:          newFsHandler(time.Minute, storageDirs, fsInfo),
	}

	// Start the filesystem handler.
	handler.fsHandler.start()

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := client.InspectContainer(id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %q: %v", id, err)
	}
	handler.creationTime = ctnr.Created
	handler.pid = ctnr.State.Pid

	// Add the name and bare ID as aliases of the container.
	handler.aliases = append(handler.aliases, strings.TrimPrefix(ctnr.Name, "/"), id)
	handler.labels = ctnr.Config.Labels
	handler.image = ctnr.Config.Image
	handler.networkMode = ctnr.HostConfig.NetworkMode

	return handler, nil
}

func (self *dockerContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return info.ContainerReference{
		Name:      self.name,
		Aliases:   self.aliases,
		Namespace: DockerNamespace,
	}, nil
}

func (self *dockerContainerHandler) readLibcontainerConfig() (*libcontainerconfigs.Config, error) {
	config, err := containerlibcontainer.ReadConfig(*dockerRootDir, *dockerRunDir, self.id)
	if err != nil {
		return nil, fmt.Errorf("failed to read libcontainer config: %v", err)
	}

	// Replace cgroup parent and name with our own since we may be running in a different context.
	if config.Cgroups == nil {
		config.Cgroups = new(libcontainerconfigs.Cgroup)
	}
	config.Cgroups.Name = self.name
	config.Cgroups.Parent = "/"

	return config, nil
}

func libcontainerConfigToContainerSpec(config *libcontainerconfigs.Config, mi *info.MachineInfo) info.ContainerSpec {
	var spec info.ContainerSpec
	spec.HasMemory = true
	spec.Memory.Limit = math.MaxUint64
	spec.Memory.SwapLimit = math.MaxUint64
	if config.Cgroups.Memory > 0 {
		spec.Memory.Limit = uint64(config.Cgroups.Memory)
	}
	if config.Cgroups.MemorySwap > 0 {
		spec.Memory.SwapLimit = uint64(config.Cgroups.MemorySwap)
	}

	// Get CPU info
	spec.HasCpu = true
	spec.Cpu.Limit = 1024
	if config.Cgroups.CpuShares != 0 {
		spec.Cpu.Limit = uint64(config.Cgroups.CpuShares)
	}
	spec.Cpu.Mask = utils.FixCpuMask(config.Cgroups.CpusetCpus, mi.NumCores)

	spec.HasDiskIo = true

	return spec
}

var (
	hasNetworkModes = map[string]bool{
		"host":    true,
		"bridge":  true,
		"default": true,
	}
)

func hasNet(networkMode string) bool {
	return hasNetworkModes[networkMode]
}

func (self *dockerContainerHandler) GetSpec() (info.ContainerSpec, error) {
	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return info.ContainerSpec{}, err
	}
	libcontainerConfig, err := self.readLibcontainerConfig()
	if err != nil {
		return info.ContainerSpec{}, err
	}

	spec := libcontainerConfigToContainerSpec(libcontainerConfig, mi)
	spec.CreationTime = self.creationTime
	// For now only enable for aufs filesystems
	spec.HasFilesystem = self.storageDriver == aufsStorageDriver
	spec.Labels = self.labels
	spec.Image = self.image
	spec.HasNetwork = hasNet(self.networkMode)

	return spec, err
}

func (self *dockerContainerHandler) getFsStats(stats *info.ContainerStats) error {
	// No support for non-aufs storage drivers.
	if self.storageDriver != aufsStorageDriver {
		return nil
	}

	// As of now we assume that all the storage dirs are on the same device.
	// The first storage dir will be that of the image layers.
	deviceInfo, err := self.fsInfo.GetDirFsDevice(self.storageDirs[0])
	if err != nil {
		return err
	}

	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return err
	}
	var limit uint64 = 0
	// Docker does not impose any filesystem limits for containers. So use capacity as limit.
	for _, fs := range mi.Filesystems {
		if fs.Device == deviceInfo.Device {
			limit = fs.Capacity
			break
		}
	}

	fsStat := info.FsStats{Device: deviceInfo.Device, Limit: limit}

	fsStat.Usage = self.fsHandler.usage()
	stats.Filesystem = append(stats.Filesystem, fsStat)

	return nil
}

// TODO(vmarmol): Get from libcontainer API instead of cgroup manager when we don't have to support older Dockers.
func (self *dockerContainerHandler) GetStats() (*info.ContainerStats, error) {
	stats, err := containerlibcontainer.GetStats(self.cgroupManager, self.rootFs, self.pid)
	if err != nil {
		return stats, err
	}
	// Clean up stats for containers that don't have their own network - this
	// includes containers running in Kubernetes pods that use the network of the
	// infrastructure container. This stops metrics being reported multiple times
	// for each container in a pod.
	if !hasNet(self.networkMode) {
		stats.Network = info.NetworkStats{}
	}

	// Get filesystem stats.
	err = self.getFsStats(stats)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (self *dockerContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	// No-op for Docker driver.
	return []info.ContainerReference{}, nil
}

func (self *dockerContainerHandler) GetCgroupPath(resource string) (string, error) {
	path, ok := self.cgroupPaths[resource]
	if !ok {
		return "", fmt.Errorf("could not find path for resource %q for container %q\n", resource, self.name)
	}
	return path, nil
}

func (self *dockerContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	// TODO(vmarmol): Implement.
	return nil, nil
}

func (self *dockerContainerHandler) GetContainerLabels() map[string]string {
	return self.labels
}

func (self *dockerContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return containerlibcontainer.GetProcesses(self.cgroupManager)
}

func (self *dockerContainerHandler) WatchSubcontainers(events chan container.SubcontainerEvent) error {
	return fmt.Errorf("watch is unimplemented in the Docker container driver")
}

func (self *dockerContainerHandler) StopWatchingSubcontainers() error {
	// No-op for Docker driver.
	return nil
}

func (self *dockerContainerHandler) Exists() bool {
	return containerlibcontainer.Exists(*dockerRootDir, *dockerRunDir, self.id)
}

func DockerInfo() (map[string]string, error) {
	client, err := Client()
	if err != nil {
		return nil, fmt.Errorf("unable to communicate with docker daemon: %v", err)
	}
	info, err := client.Info()
	if err != nil {
		return nil, err
	}
	return info.Map(), nil
}

func DockerImages() ([]docker.APIImages, error) {
	client, err := Client()
	if err != nil {
		return nil, fmt.Errorf("unable to communicate with docker daemon: %v", err)
	}
	images, err := client.ListImages(docker.ListImagesOptions{All: false})
	if err != nil {
		return nil, err
	}
	return images, nil
}
