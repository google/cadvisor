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
	"log"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/libcontainer/cgroups"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/utils"
)

type rawContainerHandler struct {
	name               string
	cgroupSubsystems   *cgroupSubsystems
	machineInfoFactory info.MachineInfoFactory
}

func newRawContainerHandler(name string, cgroupSubsystems *cgroupSubsystems, machineInfoFactory info.MachineInfoFactory) (container.ContainerHandler, error) {
	return &rawContainerHandler{
		name:               name,
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

func readInt64(path string, file string) uint64 {
	cgroupFile := filepath.Join(path, file)

	// Ignore non-existent files
	if !utils.FileExists(cgroupFile) {
		return 0
	}

	// Read
	out, err := ioutil.ReadFile(cgroupFile)
	if err != nil {
		log.Printf("raw driver: Failed to read %q: %s", cgroupFile, err)
		return 0
	}
	val, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		log.Printf("raw driver: Failed to parse in %q from file %q: %s", string(out), cgroupFile, err)
		return 0
	}

	return val
}

func (self *rawContainerHandler) GetSpec() (*info.ContainerSpec, error) {
	spec := new(info.ContainerSpec)

	// The raw driver assumes unified hierarchy containers.

	// CPU.
	cpuRoot, ok := self.cgroupSubsystems.mountPoints["cpu"]
	if ok {
		cpuRoot = filepath.Join(cpuRoot, self.name)
		if utils.FileExists(cpuRoot) {
			// Get machine info.
			mi, err := self.machineInfoFactory.GetMachineInfo()
			if err != nil {
				return nil, err
			}

			spec.Cpu = new(info.CpuSpec)
			spec.Cpu.Limit = readInt64(cpuRoot, "cpu.shares")

			// TODO(vmarmol): Get CPUs from config.Cgroups.CpusetCpus
			n := (mi.NumCores + 63) / 64
			spec.Cpu.Mask.Data = make([]uint64, n)
			for i := 0; i < n; i++ {
				spec.Cpu.Mask.Data[i] = math.MaxUint64
			}
		}
	}

	// Memory.
	memoryRoot, ok := self.cgroupSubsystems.mountPoints["memory"]
	if ok {
		memoryRoot = filepath.Join(memoryRoot, self.name)
		if utils.FileExists(memoryRoot) {
			spec.Memory = new(info.MemorySpec)
			spec.Memory.Limit = readInt64(memoryRoot, "memory.limit_in_bytes")
			spec.Memory.SwapLimit = readInt64(memoryRoot, "memory.limit_in_bytes")
		}
	}

	return spec, nil
}

func (self *rawContainerHandler) GetStats() (stats *info.ContainerStats, err error) {
	cgroup := &cgroups.Cgroup{
		Parent: "/",
		Name:   self.name,
	}

	return libcontainer.GetStatsCgroupOnly(cgroup)
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
	// TODO(vmarmol): Implement
	return nil, nil
}
