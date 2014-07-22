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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/libcontainer"
	"github.com/fsouza/go-dockerclient"
	"github.com/google/cadvisor/container"
	containerLibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/utils"
)

// Basepath to all container specific information that libcontainer stores.
const dockerRootDir = "/var/lib/docker/execdriver/native"

var fileNotFound = errors.New("file not found")

type dockerContainerHandler struct {
	client             *docker.Client
	name               string
	parent             string
	ID                 string
	aliases            []string
	machineInfoFactory info.MachineInfoFactory
	useSystemd         bool
}

func newDockerContainerHandler(
	client *docker.Client,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	useSystemd bool,
) (container.ContainerHandler, error) {
	handler := &dockerContainerHandler{
		client:             client,
		name:               name,
		machineInfoFactory: machineInfoFactory,
		useSystemd:         useSystemd,
	}
	if handler.isDockerRoot() {
		return handler, nil
	}
	parent, id, err := splitName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid docker container %v: %v", name, err)
	}
	handler.parent = parent
	handler.ID = id
	ctnr, err := client.InspectContainer(id)
	// We assume that if Inspect fails then the container is not known to docker.
	if err != nil || !ctnr.State.Running {
		return nil, container.NotActive
	}
	handler.aliases = append(handler.aliases, path.Join("/docker", ctnr.Name))
	return handler, nil
}

func (self *dockerContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return info.ContainerReference{
		Name:    self.name,
		Aliases: self.aliases,
	}, nil
}

func (self *dockerContainerHandler) isDockerRoot() bool {
	// TODO(dengnan): Should we consider other cases?
	return self.name == "/docker"
}

func splitName(containerName string) (string, string, error) {
	parent, id := path.Split(containerName)
	cgroupSelf, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", "", err
	}
	scanner := bufio.NewScanner(cgroupSelf)

	subsys := []string{"memory", "cpu"}
	nestedLevels := 0
	for scanner.Scan() {
		line := scanner.Text()
		elems := strings.Split(line, ":")
		if len(elems) < 3 {
			continue
		}
		for _, s := range subsys {
			if elems[1] == s {
				// count how many nested docker containers are there.
				nestedLevels = strings.Count(elems[2], "/docker")
				break
			}
		}
	}
	if nestedLevels > 0 {
		// we are running inside a docker container
		upperLevel := strings.Repeat("../../", nestedLevels)
		parent = filepath.Join(upperLevel, parent)
	}
	// Strip the last "/"
	if parent[len(parent)-1] == '/' {
		parent = parent[:len(parent)-1]
	}
	return parent, id, nil
}

// TODO(vmarmol): Switch to getting this from libcontainer once we have a solID API.
func (self *dockerContainerHandler) readLibcontainerConfig() (config *libcontainer.Config, err error) {
	configPath := path.Join(dockerRootDir, self.ID, "container.json")
	if !utils.FileExists(configPath) {
		err = fileNotFound
		return
	}
	f, err := os.Open(configPath)
	if err != nil {
		return
	}
	defer f.Close()
	d := json.NewDecoder(f)
	retConfig := new(libcontainer.Config)
	err = d.Decode(retConfig)
	if err != nil {
		return
	}
	config = retConfig

	return
}

func (self *dockerContainerHandler) readLibcontainerState() (state *libcontainer.State, err error) {
	statePath := path.Join(dockerRootDir, self.ID, "state.json")
	if !utils.FileExists(statePath) {
		err = fileNotFound
		return
	}
	f, err := os.Open(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			err = container.NotActive
		}
		return
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

func libcontainerConfigToContainerSpec(config *libcontainer.Config, mi *info.MachineInfo) *info.ContainerSpec {
	spec := new(info.ContainerSpec)
	spec.Memory = new(info.MemorySpec)
	spec.Memory.Limit = math.MaxUint64
	spec.Memory.SwapLimit = math.MaxUint64
	if config.Cgroups.Memory > 0 {
		spec.Memory.Limit = uint64(config.Cgroups.Memory)
	}
	if config.Cgroups.MemorySwap > 0 {
		spec.Memory.SwapLimit = uint64(config.Cgroups.MemorySwap)
	}

	// Get CPU info
	spec.Cpu = new(info.CpuSpec)
	spec.Cpu.Limit = 1024
	if config.Cgroups.CpuShares != 0 {
		spec.Cpu.Limit = uint64(config.Cgroups.CpuShares)
	}
	n := (mi.NumCores + 63) / 64
	spec.Cpu.Mask.Data = make([]uint64, n)
	for i := 0; i < n; i++ {
		spec.Cpu.Mask.Data[i] = math.MaxUint64
	}
	// TODO(vmarmol): Get CPUs from config.Cgroups.CpusetCpus
	return spec
}

func (self *dockerContainerHandler) GetSpec() (spec *info.ContainerSpec, err error) {
	if self.isDockerRoot() {
		return &info.ContainerSpec{}, nil
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
	return
}

func (self *dockerContainerHandler) GetStats() (stats *info.ContainerStats, err error) {
	if self.isDockerRoot() {
		return &info.ContainerStats{}, nil
	}
	config, err := self.readLibcontainerConfig()
	if err != nil {
		if err == fileNotFound {
			return &info.ContainerStats{}, nil
		}
		return
	}
	state, err := self.readLibcontainerState()
	if err != nil {
		if err == fileNotFound {
			return &info.ContainerStats{}, nil
		}
		return
	}

	return containerLibcontainer.GetStats(config, state)
}

func (self *dockerContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	opt := docker.ListContainersOptions{
		All: true,
	}
	containers, err := self.client.ListContainers(opt)
	if err != nil {
		return nil, err
	}
	ret := make([]info.ContainerReference, 0, len(containers)+1)
	for _, c := range containers {
		if c.ID == self.ID {
			// Skip self.
			continue
		}
		if !strings.HasPrefix(c.Status, "Up ") {
			continue
		}
		path := fmt.Sprintf("/docker/%v", c.ID)
		aliases := c.Names
		ref := info.ContainerReference{
			Name:    path,
			Aliases: aliases,
		}
		ret = append(ret, ref)
	}

	return ret, nil
}

func (self *dockerContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	return nil, nil
}

func (self *dockerContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return nil, nil
}
