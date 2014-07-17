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

package raw

import (
	"io/ioutil"
	"path/filepath"

	"github.com/docker/libcontainer/cgroups"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/info"
)

type rawContainerHandler struct {
	name             string
	cgroupSubsystems *cgroupSubsystems
}

func newRawContainerHandler(name string, cgroupSubsystems *cgroupSubsystems) (container.ContainerHandler, error) {
	return &rawContainerHandler{
		name:             name,
		cgroupSubsystems: cgroupSubsystems,
	}, nil
}

func (self *rawContainerHandler) ContainerReference() (info.ContainerReference, error) {
	// We only know the container by its one name.
	return info.ContainerReference{
		Name: self.name,
	}, nil
}

func (self *rawContainerHandler) GetSpec() (*info.ContainerSpec, error) {
	ret := new(info.ContainerSpec)
	ret.Cpu = new(info.CpuSpec)
	ret.Memory = new(info.MemorySpec)

	// TODO(vmarmol): Implement
	return ret, nil
}

func (self *rawContainerHandler) GetStats() (stats *info.ContainerStats, err error) {
	cgroup := &cgroups.Cgroup{
		Parent: "/",
		Name:   self.name,
	}

	return libcontainer.GetStats(cgroup, false)
}

// Lists all directories under "path" and outputs the results as children of "parent".
func listDirectories(path string, parent string, recursive bool, output map[string]struct{}) error {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		// We only grab directories.
		if entry.IsDir() {
			name := filepath.Join(parent, entry.Name())
			output[name] = struct{}{}

			// List subcontainers if asked to.
			if recursive {
				err := listDirectories(filepath.Join(path, entry.Name()), name, true, output)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (self *rawContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	containers := make(map[string]struct{}, 16)
	for _, subsystem := range self.cgroupSubsystems.mounts {
		err := listDirectories(filepath.Join(subsystem.Mountpoint, self.name), self.name, listType == container.LIST_RECURSIVE, containers)
		if err != nil {
			return nil, err
		}
	}

	// Make into container references.
	ret := make([]info.ContainerReference, 0, len(containers))
	for cont, _ := range containers {
		ret = append(ret, info.ContainerReference{
			Name: cont,
		})
	}

	return ret, nil
}

func (self *rawContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	// TODO(vmarmol): Implement
	return nil, nil
}

func (self *rawContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	// TODO(vmarmol): Implement
	return nil, nil
}
