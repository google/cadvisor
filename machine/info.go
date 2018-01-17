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
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stddef.h>
#include "nvml.h"

void *nvml = NULL;

const char* (*nvmlErrorStringFunc)(nvmlReturn_t result);
const char* nvmlErrorString(nvmlReturn_t result)
{
	if (nvml == NULL) {
		return "library nvml not found";
	}
	if (nvmlErrorStringFunc == NULL) {
		return "function nvmlErrorString not found";
	}
	return nvmlErrorStringFunc(result);
}

nvmlReturn_t (*nvmlDeviceGetMemoryInfoFunc)(nvmlDevice_t device, nvmlMemory_t *memory);
nvmlReturn_t nvmlDeviceGetMemoryInfo(nvmlDevice_t device, nvmlMemory_t *memory)
{
	if (nvml == NULL) {
		return NVML_ERROR_LIBRARY_NOT_FOUND;
	}
	if (nvmlDeviceGetMemoryInfoFunc == NULL) {
		return NVML_ERROR_FUNCTION_NOT_FOUND;
	}
	return nvmlDeviceGetMemoryInfoFunc(device, memory);
}

nvmlReturn_t (*nvmlDeviceGetMinorNumberFunc)(nvmlDevice_t device, unsigned int *minorNumber);
nvmlReturn_t nvmlDeviceGetMinorNumber(nvmlDevice_t device, unsigned int *minorNumber)
{
	if (nvml == NULL) {
		return NVML_ERROR_LIBRARY_NOT_FOUND;
	}
	if (nvmlDeviceGetMinorNumberFunc == NULL) {
		return NVML_ERROR_FUNCTION_NOT_FOUND;
	}
	return nvmlDeviceGetMinorNumberFunc(device, minorNumber);
}

nvmlReturn_t (*nvmlDeviceGetNameFunc)(nvmlDevice_t device, char *name, unsigned int length);
nvmlReturn_t nvmlDeviceGetName(nvmlDevice_t device, char *name, unsigned int length)
{
	if (nvml == NULL) {
		return NVML_ERROR_LIBRARY_NOT_FOUND;
	}
	if (nvmlDeviceGetNameFunc == NULL) {
		return NVML_ERROR_FUNCTION_NOT_FOUND;
	}
	return nvmlDeviceGetNameFunc(device, name, length);
}

nvmlReturn_t (*nvmlDeviceGetHandleByIndexFunc)(unsigned int index, nvmlDevice_t *device);
nvmlReturn_t nvmlDeviceGetHandleByIndex(unsigned int index, nvmlDevice_t *device)
{
	if (nvml == NULL) {
		return NVML_ERROR_LIBRARY_NOT_FOUND;
	}
	if (nvmlDeviceGetHandleByIndexFunc == NULL) {
		return NVML_ERROR_FUNCTION_NOT_FOUND;
	}
	return nvmlDeviceGetHandleByIndexFunc(index, device);
}

nvmlReturn_t (*nvmlDeviceGetCountFunc)(unsigned int *deviceCount);
nvmlReturn_t nvmlDeviceGetCount(unsigned int *deviceCount)
{
	if (nvml == NULL) {
		return NVML_ERROR_LIBRARY_NOT_FOUND;
	}
	if (nvmlDeviceGetCountFunc == NULL) {
		return NVML_ERROR_FUNCTION_NOT_FOUND;
	}
	return nvmlDeviceGetCountFunc(deviceCount);
}

nvmlReturn_t (*nvmlShutdownFunc)(void);
nvmlReturn_t nvmlShutdown(void)
{
	if (nvml == NULL) {
		return NVML_ERROR_LIBRARY_NOT_FOUND;
	}
	if (nvmlShutdownFunc == NULL) {
		return NVML_ERROR_FUNCTION_NOT_FOUND;
	}
	nvmlReturn_t result = nvmlShutdownFunc();
	dlclose(nvml);
	nvmlShutdownFunc = NULL;
	nvml = NULL;
	return result;
}


nvmlReturn_t (*nvmlInitFunc)(void);
nvmlReturn_t nvmlInit(void)
{
        nvml = dlopen("libnvidia-ml.so.1", RTLD_LAZY);
	if (nvml == NULL) {
		return NVML_ERROR_LIBRARY_NOT_FOUND;
	}
        nvmlInitFunc = dlsym(nvml, "nvmlInit_v2");
	if (nvmlInitFunc == NULL) {
		return NVML_ERROR_FUNCTION_NOT_FOUND;
	}
	nvmlReturn_t result = nvmlInitFunc();
	if (result != NVML_SUCCESS) {
		dlclose(nvml);
		nvmlInitFunc = NULL;
		nvml = NULL;
		return result;
	}
        nvmlShutdownFunc = dlsym(nvml, "nvmlShutdown_v2");
        nvmlDeviceGetCountFunc = dlsym(nvml, "nvmlDeviceGetCount_v2");
        nvmlDeviceGetHandleByIndexFunc = dlsym(nvml, "nvmlDeviceGetHandleByIndex_v2");
        nvmlDeviceGetNameFunc = dlsym(nvml, "nvmlDeviceGetName");
        nvmlDeviceGetMinorNumberFunc = dlsym(nvml, "nvmlDeviceGetMinorNumber");
        nvmlDeviceGetMemoryInfoFunc = dlsym(nvml, "nvmlDeviceGetMemoryInfo");
        nvmlErrorStringFunc = dlsym(nvml, "nvmlErrorString");
	return NVML_SUCCESS;
}
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

	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils/cloudinfo"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysinfo"

	"github.com/golang/glog"

	"golang.org/x/sys/unix"
)

func GPUInfoInit() {
	if r := C.nvmlInit(); r != C.NVML_SUCCESS {
		glog.Infof("Couldn't initialize nvml library: %v", r)
	}
}

func GPUInfoShutdown() {
	C.nvmlShutdown()
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
	glog.Warningf("Couldn't collect info from any of the files in %q", filePaths)
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
	uname := &unix.Utsname{}

	if err := unix.Uname(uname); err != nil {
		return "Unknown"
	}

	return string(uname.Release[:bytes.IndexByte(uname.Release[:], 0)])
}
