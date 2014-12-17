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
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"strings"

	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups"
	cgroup_fs "github.com/docker/libcontainer/cgroups/fs"
	"github.com/fsouza/go-dockerclient"
	"github.com/google/cadvisor/container"
	containerLibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/utils"
)

// Relative path from Docker root to the libcontainer per-container state.
const pathToLibcontainerState = "execdriver/native"

// Path to aufs dir where all the files exist.
// aufs/layers is ignored here since it does not hold a lot of data.
// aufs/mnt contains the mount points used to compose the rootfs. Hence it is also ignored.
var pathToAufsDir = "aufs/diff"

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

	cgroup         cgroups.Cgroup
	usesAufsDriver bool
	fsInfo         fs.FsInfo
	storageDirs    []string
}

func newDockerContainerHandler(
	client *docker.Client,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	dockerRootDir string,
	usesAufsDriver bool,
	cgroupSubsystems *containerLibcontainer.CgroupSubsystems,
) (container.ContainerHandler, error) {
	fsInfo, err := fs.NewFsInfo()
	if err != nil {
		return nil, err
	}

	// Create the cgroup paths.
	cgroupPaths := make(map[string]string, len(cgroupSubsystems.MountPoints))
	for key, val := range cgroupSubsystems.MountPoints {
		cgroupPaths[key] = path.Join(val, name)
	}

	id := ContainerNameToDockerId(name)
	handler := &dockerContainerHandler{
		id:                     id,
		client:                 client,
		name:                   name,
		machineInfoFactory:     machineInfoFactory,
		libcontainerConfigPath: path.Join(dockerRootDir, pathToLibcontainerState, id, "container.json"),
		libcontainerStatePath:  path.Join(dockerRootDir, pathToLibcontainerState, id, "state.json"),
		libcontainerPidPath:    path.Join(dockerRootDir, pathToLibcontainerState, id, "pid"),
		cgroupPaths:            cgroupPaths,
		cgroup: cgroups.Cgroup{
			Parent: "/",
			Name:   name,
		},
		usesAufsDriver: usesAufsDriver,
		fsInfo:         fsInfo,
	}
	handler.storageDirs = append(handler.storageDirs, path.Join(dockerRootDir, pathToAufsDir, path.Base(name)))

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := client.InspectContainer(id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %q: %v", id, err)
	}

	// Add the name and bare ID as aliases of the container.
	handler.aliases = append(handler.aliases, strings.TrimPrefix(ctnr.Name, "/"))
	handler.aliases = append(handler.aliases, id)
    handler.aliases = append(handler.aliases, ctnr.Config.Hostname)

	return handler, nil
}

func (self *dockerContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return info.ContainerReference{
		Name:      self.name,
		Aliases:   self.aliases,
		Namespace: DockerNamespace,
	}, nil
}

// TODO(vmarmol): Switch to getting this from libcontainer once we have a solid API.
func (self *dockerContainerHandler) readLibcontainerConfig() (config *libcontainer.Config, err error) {
	f, err := os.Open(self.libcontainerConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s - %s\n", self.libcontainerConfigPath, err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	retConfig := new(libcontainer.Config)
	err = d.Decode(retConfig)
	if err != nil {
		return
	}
	config = retConfig

	// Replace cgroup parent and name with our own since we may be running in a different context.
	config.Cgroups.Name = self.cgroup.Name
	config.Cgroups.Parent = self.cgroup.Parent

	return
}

func (self *dockerContainerHandler) readLibcontainerState() (state *libcontainer.State, err error) {
	// TODO(vmarmol): Remove this once we can depend on a newer Docker.
	// Libcontainer changed how its state was stored, try the old way of a "pid" file
	if !utils.FileExists(self.libcontainerStatePath) {
		if utils.FileExists(self.libcontainerPidPath) {
			// We don't need the old state, return an empty state and we'll gracefully degrade.
			return &libcontainer.State{}, nil
		}
	}
	f, err := os.Open(self.libcontainerStatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s - %s\n", self.libcontainerStatePath, err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	retState := new(libcontainer.State)
	err = d.Decode(retState)
	if err != nil {
		return
	}
	state = retState

	// Create cgroup paths if they don't exist. This is since older Docker clients don't write it.
	if len(state.CgroupPaths) == 0 {
		state.CgroupPaths = self.cgroupPaths
	}

	return
}

func libcontainerConfigToContainerSpec(config *libcontainer.Config, mi *info.MachineInfo) info.ContainerSpec {
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
	if config.Cgroups.CpusetCpus == "" {
		// All cores are active.
		spec.Cpu.Mask = fmt.Sprintf("0-%d", mi.NumCores-1)
	} else {
		spec.Cpu.Mask = config.Cgroups.CpusetCpus
	}

	spec.HasNetwork = true
	return spec
}

func (self *dockerContainerHandler) GetSpec() (spec info.ContainerSpec, err error) {
	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return
	}
	libcontainerConfig, err := self.readLibcontainerConfig()
	if err != nil {
		return
	}

	spec = libcontainerConfigToContainerSpec(libcontainerConfig, mi)

	if self.usesAufsDriver {
		spec.HasFilesystem = true
	}

	return
}

func (self *dockerContainerHandler) getFsStats(stats *info.ContainerStats) error {
	// No support for non-aufs storage drivers.
	if !self.usesAufsDriver {
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

	var usage uint64 = 0
	for _, dir := range self.storageDirs {
		// TODO(Vishh): Add support for external mounts.
		dirUsage, err := self.fsInfo.GetDirUsage(dir)
		if err != nil {
			return err
		}
		usage += dirUsage
	}
	fsStat.Usage = usage
	stats.Filesystem = append(stats.Filesystem, fsStat)

	return nil
}

func (self *dockerContainerHandler) GetStats() (stats *info.ContainerStats, err error) {
	state, err := self.readLibcontainerState()
	if err != nil {
		return
	}

	stats, err = containerLibcontainer.GetStats(state)
	if err != nil {
		return
	}
	err = self.getFsStats(stats)
	if err != nil {
		return
	}

	return stats, nil
}

func (self *dockerContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	if self.name != "/docker" {
		return []info.ContainerReference{}, nil
	}
	opt := docker.ListContainersOptions{
		All: true,
	}
	containers, err := self.client.ListContainers(opt)
	if err != nil {
		return nil, err
	}

	ret := make([]info.ContainerReference, 0, len(containers)+1)
	for _, c := range containers {
		if !strings.HasPrefix(c.Status, "Up ") {
			continue
		}

		ref := info.ContainerReference{
			Name:      FullContainerName(c.ID),
			Aliases:   append(c.Names, c.ID),
			Namespace: DockerNamespace,
		}
		ret = append(ret, ref)
	}

	return ret, nil
}

func (self *dockerContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	return nil, nil
}

func (self *dockerContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return cgroup_fs.GetPids(&self.cgroup)
}

func (self *dockerContainerHandler) WatchSubcontainers(events chan container.SubcontainerEvent) error {
	return fmt.Errorf("watch is unimplemented in the Docker container driver")
}

func (self *dockerContainerHandler) StopWatchingSubcontainers() error {
	// No-op for Docker driver.
	return nil
}

func (self *dockerContainerHandler) Exists() bool {
	// We consider the container existing if both libcontainer config and state files exist.
	return utils.FileExists(self.libcontainerConfigPath) && utils.FileExists(self.libcontainerStatePath)
}
