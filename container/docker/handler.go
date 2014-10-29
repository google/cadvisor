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

package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"strings"

	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups"
	cgroup_fs "github.com/docker/libcontainer/cgroups/fs"
	"github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
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

var fileNotFound = errors.New("file not found")

type dockerContainerHandler struct {
	client               *docker.Client
	name                 string
	id                   string
	aliases              []string
	machineInfoFactory   info.MachineInfoFactory
	libcontainerStateDir string
	cgroup               cgroups.Cgroup
	usesAufsDriver       bool
	fsInfo               fs.FsInfo
	storageDirs          []string
}

func newDockerContainerHandler(
	client *docker.Client,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	dockerRootDir string,
	usesAufsDriver bool,
) (container.ContainerHandler, error) {
	fsInfo, err := fs.NewFsInfo()
	if err != nil {
		return nil, err
	}
	handler := &dockerContainerHandler{
		client:               client,
		name:                 name,
		machineInfoFactory:   machineInfoFactory,
		libcontainerStateDir: path.Join(dockerRootDir, pathToLibcontainerState),
		cgroup: cgroups.Cgroup{
			Parent: "/",
			Name:   name,
		},
		usesAufsDriver: usesAufsDriver,
		fsInfo:         fsInfo,
	}
	handler.storageDirs = append(handler.storageDirs, path.Join(dockerRootDir, pathToAufsDir, path.Base(name)))
	if handler.isDockerRoot() {
		return handler, nil
	}
	id := containerNameToDockerId(name)
	handler.id = id
	ctnr, err := client.InspectContainer(id)
	// We assume that if Inspect fails then the container is not known to docker.
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s - %s\n", id, err)
	}
	handler.aliases = append(handler.aliases, path.Join("/docker", ctnr.Name))
	return handler, nil
}

func containerNameToDockerId(name string) string {
	id := path.Base(name)

	// Turn systemd cgroup name into Docker ID.
	if useSystemd {
		const systemdDockerPrefix = "docker-"
		if strings.HasPrefix(id, systemdDockerPrefix) {
			id = id[len(systemdDockerPrefix):]
		}

		const systemdScopeSuffix = ".scope"
		if strings.HasSuffix(id, systemdScopeSuffix) {
			id = id[:len(id)-len(systemdScopeSuffix)]
		}
	}

	return id
}

func (self *dockerContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return info.ContainerReference{
		Name:    self.name,
		Aliases: self.aliases,
	}, nil
}

func (self *dockerContainerHandler) isDockerRoot() bool {
	return self.name == "/docker"
}

// TODO(vmarmol): Switch to getting this from libcontainer once we have a solid API.
func (self *dockerContainerHandler) readLibcontainerConfig() (config *libcontainer.Config, err error) {
	configPath := path.Join(self.libcontainerStateDir, self.id, "container.json")
	if !utils.FileExists(configPath) {
		// TODO(vishh): Return file name as well once we have a better error interface.
		err = fileNotFound
		return
	}
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s - %s\n", configPath, err)
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
	statePath := path.Join(self.libcontainerStateDir, self.id, "state.json")
	if !utils.FileExists(statePath) {
		// TODO(vmarmol): Remove this once we can depend on a newer Docker.
		// Libcontainer changed how its state was stored, try the old way of a "pid" file
		if utils.FileExists(path.Join(self.libcontainerStateDir, self.id, "pid")) {
			// We don't need the old state, return an empty state and we'll gracefully degrade.
			state = new(libcontainer.State)
			return
		}

		// TODO(vishh): Return file name as well once we have a better error interface.
		err = fileNotFound
		return
	}
	f, err := os.Open(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s - %s\n", statePath, err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	retState := new(libcontainer.State)
	err = d.Decode(retState)
	if err != nil {
		return
	}
	state = retState

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
	if self.isDockerRoot() {
		return info.ContainerSpec{}, nil
	}
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
	if self.isDockerRoot() {
		return &info.ContainerStats{}, nil
	}
	state, err := self.readLibcontainerState()
	if err != nil {
		if err == fileNotFound {
			glog.Errorf("Libcontainer state not found for container %q", self.name)
			return &info.ContainerStats{}, nil
		}
		return
	}

	stats, err = containerLibcontainer.GetStats(&self.cgroup, state)
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

	// On non-systemd systems Docker containers are under /docker.
	containerPrefix := "/docker"
	if useSystemd {
		containerPrefix = "/system.slice"
	}

	ret := make([]info.ContainerReference, 0, len(containers)+1)
	for _, c := range containers {
		if !strings.HasPrefix(c.Status, "Up ") {
			continue
		}

		ref := info.ContainerReference{
			Name:    path.Join(containerPrefix, c.ID),
			Aliases: c.Names,
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
