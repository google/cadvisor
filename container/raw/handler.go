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
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"strings"

	"github.com/docker/libcontainer/cgroups"
	"github.com/docker/libcontainer/cgroups/fs"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/utils"
)

type rawContainerHandler struct {
	name               string
	cgroup             *cgroups.Cgroup
	cgroupSubsystems   *cgroupSubsystems
	machineInfoFactory info.MachineInfoFactory
}

func newRawContainerHandler(name string, cgroupSubsystems *cgroupSubsystems, machineInfoFactory info.MachineInfoFactory) (container.ContainerHandler, error) {
	return &rawContainerHandler{
		name: name,
		cgroup: &cgroups.Cgroup{
			Parent: "/",
			Name:   name,
		},
		cgroupSubsystems:   cgroupSubsystems,
		machineInfoFactory: machineInfoFactory,
	}, nil
}

func (self *rawContainerHandler) ContainerReference() (info.ContainerReference, error) {
	// We only know the container by its one name.
	return info.ContainerReference{
		Name: self.name,
	}, nil
}

func readString(dirpath string, file string) string {
	cgroupFile := path.Join(dirpath, file)

	// Ignore non-existent files
	if !utils.FileExists(cgroupFile) {
		return ""
	}

	// Read
	out, err := ioutil.ReadFile(cgroupFile)
	if err != nil {
		log.Printf("raw driver: Failed to read %q: %s", cgroupFile, err)
		return ""
	}
	return string(out)
}

func readInt64(dirpath string, file string) uint64 {
	out := readString(dirpath, file)
	if out == "" {
		return 0
	}

	val, err := strconv.ParseUint(strings.TrimSpace(out), 10, 64)
	if err != nil {
		log.Printf("raw driver: Failed to parse in %q from file %q: %s", out, path.Join(dirpath, file), err)
		return 0
	}

	return val
}

func (self *rawContainerHandler) GetSpec() (*info.ContainerSpec, error) {
	spec := new(info.ContainerSpec)

	// The raw driver assumes unified hierarchy containers.

	// Get machine info.
	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return nil, err
	}

	// CPU.
	cpuRoot, ok := self.cgroupSubsystems.mountPoints["cpu"]
	if ok {
		cpuRoot = path.Join(cpuRoot, self.name)
		if utils.FileExists(cpuRoot) {
			spec.Cpu = new(info.CpuSpec)
			spec.Cpu.Limit = readInt64(cpuRoot, "cpu.shares")
		}
	}

	// Cpu Mask.
	// This will fail for non-unified hierarchies. We'll return the whole machine mask in that case.
	cpusetRoot, ok := self.cgroupSubsystems.mountPoints["cpuset"]
	if ok {
		if spec.Cpu == nil {
			spec.Cpu = new(info.CpuSpec)
		}
		cpusetRoot = path.Join(cpusetRoot, self.name)
		if utils.FileExists(cpusetRoot) {
			spec.Cpu.Mask = readString(cpusetRoot, "cpuset.cpus")
			if spec.Cpu.Mask == "" {
				spec.Cpu.Mask = fmt.Sprintf("0-%d", mi.NumCores-1)
			}
		}
	}

	// Memory.
	memoryRoot, ok := self.cgroupSubsystems.mountPoints["memory"]
	if ok {
		memoryRoot = path.Join(memoryRoot, self.name)
		if utils.FileExists(memoryRoot) {
			spec.Memory = new(info.MemorySpec)
			spec.Memory.Limit = readInt64(memoryRoot, "memory.limit_in_bytes")
			spec.Memory.SwapLimit = readInt64(memoryRoot, "memory.memsw.limit_in_bytes")
		}
	}

	return spec, nil
}

func (self *rawContainerHandler) GetStats() (stats *info.ContainerStats, err error) {
	return libcontainer.GetStatsCgroupOnly(self.cgroup)
}

// Lists all directories under "path" and outputs the results as children of "parent".
func listDirectories(dirpath string, parent string, recursive bool, output map[string]struct{}) error {
	// Ignore if this hierarchy does not exist.
	if !utils.FileExists(dirpath) {
		return nil
	}

	entries, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		// We only grab directories.
		if entry.IsDir() {
			name := path.Join(parent, entry.Name())
			output[name] = struct{}{}

			// List subcontainers if asked to.
			if recursive {
				err := listDirectories(path.Join(dirpath, entry.Name()), name, true, output)
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
		err := listDirectories(path.Join(subsystem.Mountpoint, self.name), self.name, listType == container.LIST_RECURSIVE, containers)
		if err != nil {
			return nil, err
		}
	}

	// Make into container references.
	ret := make([]info.ContainerReference, 0, len(containers))
	for cont := range containers {
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
	return fs.GetPids(self.cgroup)
}
