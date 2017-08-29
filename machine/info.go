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

package machine

/*
#cgo LDFLAGS: -ldl -Wl,--unresolved-symbols=ignore-in-object-files
#include <dlfcn.h>
#include <stddef.h>
#include "nvml.h"

void *nvml = NULL;
*/
import "C"

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils/cloudinfo"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysinfo"

	"github.com/golang/glog"
)

func GPUInfoInit() {
	C.nvml = C.dlopen(C.CString("libnvidia-ml.so.1"), C.RTLD_LAZY|C.RTLD_GLOBAL)
	if C.nvml == nil {
		glog.Infof("Couldn't load nvml library")
		return
	}
	if r := C.nvmlInit(); r != C.NVML_SUCCESS {
		glog.Infof("Couldn't initialize nvml: %v", r)
		C.nvml = nil
		return
	}
}

func GPUInfoShutdown() {
	if C.nvml == nil {
		return
	}
	C.nvmlShutdown()
	C.dlclose(C.nvml)
}

const hugepagesDirectory = "/sys/kernel/mm/hugepages/"

var machineIdFilePath = flag.String("machine_id_file", "/etc/machine-id,/var/lib/dbus/machine-id", "Comma-separated list of files to check for machine-id. Use the first one that exists.")
var bootIdFilePath = flag.String("boot_id_file", "/proc/sys/kernel/random/boot_id", "Comma-separated list of files to check for boot-id. Use the first one that exists.")

func getInfoFromFiles(filePaths string) string {
	if len(filePaths) == 0 {
		return ""
	}
	for _, file := range strings.Split(filePaths, ",") {
		id, err := ioutil.ReadFile(file)
		if err == nil {
			return strings.TrimSpace(string(id))
		}
	}
	glog.Infof("Couldn't collect info from any of the files in %q", filePaths)
	return ""
}

// GetHugePagesInfo returns information about pre-allocated huge pages
func GetHugePagesInfo() ([]info.HugePagesInfo, error) {
	var hugePagesInfo []info.HugePagesInfo
	files, err := ioutil.ReadDir(hugepagesDirectory)
	if err != nil {
		// treat as non-fatal since kernels and machine can be
		// configured to disable hugepage support
		return hugePagesInfo, nil
	}
	for _, st := range files {
		nameArray := strings.Split(st.Name(), "-")
		pageSizeArray := strings.Split(nameArray[1], "kB")
		pageSize, err := strconv.ParseUint(string(pageSizeArray[0]), 10, 64)
		if err != nil {
			return hugePagesInfo, err
		}

		numFile := hugepagesDirectory + st.Name() + "/nr_hugepages"
		val, err := ioutil.ReadFile(numFile)
		if err != nil {
			return hugePagesInfo, err
		}
		var numPages uint64
		// we use sscanf as the file as a new-line that trips up ParseUint
		// it returns the number of tokens successfully parsed, so if
		// n != 1, it means we were unable to parse a number from the file
		n, err := fmt.Sscanf(string(val), "%d", &numPages)
		if err != nil || n != 1 {
			return hugePagesInfo, fmt.Errorf("could not parse file %v contents %q", numFile, string(val))
		}

		hugePagesInfo = append(hugePagesInfo, info.HugePagesInfo{
			NumPages: numPages,
			PageSize: pageSize,
		})
	}
	return hugePagesInfo, nil
}

func getGPUs() []info.GPUInfo {
	var gpus []info.GPUInfo
	if C.nvml == nil {
		glog.Errorf("Failed to initialize nvml library")
		return gpus
	}
	var n C.uint = 0
	if r := C.nvmlDeviceGetCount(&n); r != C.NVML_SUCCESS {
		glog.Errorf("Failed to get GPU devices: %v", C.GoString(C.nvmlErrorString(r)))
		return gpus
	}

	for i := 0; i < int(n); i++ {
		var device C.nvmlDevice_t
		if r := C.nvmlDeviceGetHandleByIndex(C.uint(i), &device); r != C.NVML_SUCCESS {
			glog.Errorf("Failed to get GPU device %d: %v", i, C.GoString(C.nvmlErrorString(r)))
			return gpus
		}

		var name [C.NVML_DEVICE_NAME_BUFFER_SIZE]C.char
		if r := C.nvmlDeviceGetName(device, &name[0], C.NVML_DEVICE_NAME_BUFFER_SIZE); r != C.NVML_SUCCESS {
			glog.Errorf("Failed to get GPU device name for GPU %d: %v", i, C.GoString(C.nvmlErrorString(r)))
			return gpus
		}

		var minor C.uint
		if r := C.nvmlDeviceGetMinorNumber(device, &minor); r != C.NVML_SUCCESS {
			glog.Errorf("Failed to get GPU device minor number for GPU %d: %v", i, C.GoString(C.nvmlErrorString(r)))
			return gpus
		}

		var mem C.nvmlMemory_t
		if r := C.nvmlDeviceGetMemoryInfo(device, &mem); r != C.NVML_SUCCESS {
			glog.Errorf("Failed to get GPU device minor number for GPU %d: %v", i, C.GoString(C.nvmlErrorString(r)))
			return gpus
		}

		gpus = append(gpus, info.GPUInfo{
			Model:    C.GoString(&name[0]),
			Path:     fmt.Sprintf("/dev/nvidia%d", minor),
			MemTotal: uint64(mem.total),
			MemUsed:  uint64(mem.used),
		})

	}
	return gpus
}

func Info(sysFs sysfs.SysFs, fsInfo fs.FsInfo, inHostNamespace bool) (*info.MachineInfo, error) {
	rootFs := "/"
	if !inHostNamespace {
		rootFs = "/rootfs"
	}

	cpuinfo, err := ioutil.ReadFile(filepath.Join(rootFs, "/proc/cpuinfo"))
	clockSpeed, err := GetClockSpeed(cpuinfo)
	if err != nil {
		return nil, err
	}

	memoryCapacity, err := GetMachineMemoryCapacity()
	if err != nil {
		return nil, err
	}

	hugePagesInfo, err := GetHugePagesInfo()
	if err != nil {
		return nil, err
	}

	filesystems, err := fsInfo.GetGlobalFsInfo()
	if err != nil {
		glog.Errorf("Failed to get global filesystem information: %v", err)
	}

	diskMap, err := sysinfo.GetBlockDeviceInfo(sysFs)
	if err != nil {
		glog.Errorf("Failed to get disk map: %v", err)
	}

	netDevices, err := sysinfo.GetNetworkDevices(sysFs)
	if err != nil {
		glog.Errorf("Failed to get network devices: %v", err)
	}

	topology, numCores, err := GetTopology(sysFs, string(cpuinfo))
	if err != nil {
		glog.Errorf("Failed to get topology information: %v", err)
	}

	systemUUID, err := sysinfo.GetSystemUUID(sysFs)
	if err != nil {
		glog.Errorf("Failed to get system UUID: %v", err)
	}

	realCloudInfo := cloudinfo.NewRealCloudInfo()
	cloudProvider := realCloudInfo.GetCloudProvider()
	instanceType := realCloudInfo.GetInstanceType()
	instanceID := realCloudInfo.GetInstanceID()

	gpus := getGPUs()

	machineInfo := &info.MachineInfo{
		NumCores:       numCores,
		CpuFrequency:   clockSpeed,
		MemoryCapacity: memoryCapacity,
		HugePages:      hugePagesInfo,
		DiskMap:        diskMap,
		NetworkDevices: netDevices,
		Topology:       topology,
		MachineID:      getInfoFromFiles(filepath.Join(rootFs, *machineIdFilePath)),
		SystemUUID:     systemUUID,
		BootID:         getInfoFromFiles(filepath.Join(rootFs, *bootIdFilePath)),
		CloudProvider:  cloudProvider,
		InstanceType:   instanceType,
		InstanceID:     instanceID,
		GPUs:           gpus,
	}

	for i := range filesystems {
		fs := filesystems[i]
		inodes := uint64(0)
		if fs.Inodes != nil {
			inodes = *fs.Inodes
		}
		machineInfo.Filesystems = append(machineInfo.Filesystems, info.FsInfo{Device: fs.Device, DeviceMajor: uint64(fs.Major), DeviceMinor: uint64(fs.Minor), Type: fs.Type.String(), Capacity: fs.Capacity, Inodes: inodes, HasInodes: fs.Inodes != nil})
	}

	return machineInfo, nil
}

func ContainerOsVersion() string {
	container_os := "Unknown"
	os_release, err := ioutil.ReadFile("/etc/os-release")
	if err == nil {
		// We might be running in a busybox or some hand-crafted image.
		// It's useful to know why cadvisor didn't come up.
		for _, line := range strings.Split(string(os_release), "\n") {
			parsed := strings.Split(line, "\"")
			if len(parsed) == 3 && parsed[0] == "PRETTY_NAME=" {
				container_os = parsed[1]
				break
			}
		}
	}
	return container_os
}

func KernelVersion() string {
	uname := &syscall.Utsname{}

	if err := syscall.Uname(uname); err != nil {
		return "Unknown"
	}

	release := make([]byte, len(uname.Release))
	i := 0
	for _, c := range uname.Release {
		release[i] = byte(c)
		i++
	}
	release = release[:bytes.IndexByte(release, 0)]

	return string(release)
}
