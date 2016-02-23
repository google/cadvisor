// Copyright 2016 Google Inc. All Rights Reserved.
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

package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils"
	"github.com/google/cadvisor/utils/machine"

	"github.com/golang/glog"
)

func DebugInfo(watches map[string][]string) map[string][]string {
	out := make(map[string][]string)

	lines := make([]string, 0, len(watches))
	for containerName, cgroupWatches := range watches {
		lines = append(lines, fmt.Sprintf("%s:", containerName))
		for _, cg := range cgroupWatches {
			lines = append(lines, fmt.Sprintf("\t%s", cg))
		}
	}
	out["Inotify watches"] = lines

	return out

}

type AbstractContainerHandler interface {
	GetCgroupPaths() map[string]string
	GetMachineInfoFactory() info.MachineInfoFactory
	GetName() string
	GetRootNetworkDevices() ([]info.NetInfo, error)
	GetExternalMounts() []Mount
	HasNetwork() bool
}

func GetSpec(self AbstractContainerHandler) (info.ContainerSpec, error) {
	cgroupPaths := self.GetCgroupPaths()
	machineInfoFactory := self.GetMachineInfoFactory()
	name := self.GetName()
	externalMounts := self.GetExternalMounts()

	var spec info.ContainerSpec

	// The raw driver assumes unified hierarchy containers.

	// Get the lowest creation time from all hierarchies as the container creation time.
	now := time.Now()
	lowestTime := now
	for _, cgroupPath := range cgroupPaths {
		// The modified time of the cgroup directory changes whenever a subcontainer is created.
		// eg. /docker will have creation time matching the creation of latest docker container.
		// Use clone_children as a workaround as it isn't usually modified. It is only likely changed
		// immediately after creating a container.
		cgroupPath = path.Join(cgroupPath, "cgroup.clone_children")
		fi, err := os.Stat(cgroupPath)
		if err == nil && fi.ModTime().Before(lowestTime) {
			lowestTime = fi.ModTime()
		}
	}
	if lowestTime != now {
		spec.CreationTime = lowestTime
	}

	// Get machine info.
	mi, err := machineInfoFactory.GetMachineInfo()
	if err != nil {
		return spec, err
	}

	// CPU.
	cpuRoot, ok := cgroupPaths["cpu"]
	if ok {
		if utils.FileExists(cpuRoot) {
			spec.HasCpu = true
			spec.Cpu.Limit = readUInt64(cpuRoot, "cpu.shares")
		}
	}

	// Cpu Mask.
	// This will fail for non-unified hierarchies. We'll return the whole machine mask in that case.
	cpusetRoot, ok := cgroupPaths["cpuset"]
	if ok {
		if utils.FileExists(cpusetRoot) {
			spec.HasCpu = true
			mask := readString(cpusetRoot, "cpuset.cpus")
			spec.Cpu.Mask = utils.FixCpuMask(mask, mi.NumCores)
		}
	}

	// Memory
	if name == "/" {
		// Get memory and swap limits of the running machine
		memLimit, err := machine.GetMachineMemoryCapacity()
		if err != nil {
			glog.Warningf("failed to obtain memory limit for machine container")
			spec.HasMemory = false
		} else {
			spec.Memory.Limit = uint64(memLimit)
			// Spec is marked to have memory only if the memory limit is set
			spec.HasMemory = true
		}

		swapLimit, err := machine.GetMachineSwapCapacity()
		if err != nil {
			glog.Warningf("failed to obtain swap limit for machine container")
		} else {
			spec.Memory.SwapLimit = uint64(swapLimit)
		}
	} else {
		memoryRoot, ok := cgroupPaths["memory"]
		if ok {
			if utils.FileExists(memoryRoot) {
				spec.HasMemory = true
				spec.Memory.Limit = readUInt64(memoryRoot, "memory.limit_in_bytes")
				spec.Memory.SwapLimit = readUInt64(memoryRoot, "memory.memsw.limit_in_bytes")
			}
		}
	}

	// Fs.
	if name == "/" || externalMounts != nil {
		spec.HasFilesystem = true
	}

	//Network
	spec.HasNetwork = self.HasNetwork()

	// DiskIo.
	if blkioRoot, ok := cgroupPaths["blkio"]; ok && utils.FileExists(blkioRoot) {
		spec.HasDiskIo = true
	}

	// Check physical network devices for root container.
	nd, err := self.GetRootNetworkDevices()
	if err != nil {
		return spec, err
	}
	if len(nd) != 0 {
		spec.HasNetwork = true
	}
	return spec, nil
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
		glog.Errorf("rkt driver: Failed to read %q: %s", cgroupFile, err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readUInt64(dirpath string, file string) uint64 {
	out := readString(dirpath, file)
	if out == "" {
		return 0
	}

	val, err := strconv.ParseUint(out, 10, 64)
	if err != nil {
		glog.Errorf("rkt driver: Failed to parse int %q from file %q: %s", out, path.Join(dirpath, file), err)
		return 0
	}

	return val
}

// Lists all directories under "path" and outputs the results as children of "parent".
func ListDirectories(dirpath string, parent string, recursive bool, output map[string]struct{}) error {
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
				err := ListDirectories(path.Join(dirpath, entry.Name()), name, true, output)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
