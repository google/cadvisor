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

package manager

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	dclient "github.com/fsouza/go-dockerclient"
	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/fs"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/utils"
	"github.com/google/cadvisor/utils/sysfs"
)

var numCpuRegexp = regexp.MustCompile("processor\\t*: +[0-9]+")
var CpuClockSpeedMHz = regexp.MustCompile("cpu MHz\\t*: +([0-9]+.[0-9]+)")
var memoryCapacityRegexp = regexp.MustCompile("MemTotal: *([0-9]+) kB")

func getClockSpeed(procInfo []byte) (uint64, error) {
	// First look through sys to find a max supported cpu frequency.
	const maxFreqFile = "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"
	if utils.FileExists(maxFreqFile) {
		val, err := ioutil.ReadFile(maxFreqFile)
		if err != nil {
			return 0, err
		}
		var maxFreq uint64
		n, err := fmt.Sscanf(string(val), "%d", &maxFreq)
		if err != nil || n != 1 {
			return 0, fmt.Errorf("could not parse frequency %q", val)
		}
		return maxFreq, nil
	}
	// Fall back to /proc/cpuinfo
	matches := CpuClockSpeedMHz.FindSubmatch(procInfo)
	if len(matches) != 2 {
		return 0, fmt.Errorf("could not detect clock speed from output: %q", string(procInfo))
	}
	speed, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		return 0, err
	}
	// Convert to kHz
	return uint64(speed * 1000), nil
}

func getMachineInfo(sysFs sysfs.SysFs) (*info.MachineInfo, error) {
	// Get the number of CPUs from /proc/cpuinfo.
	out, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	numCores := len(numCpuRegexp.FindAll(out, -1))
	if numCores == 0 {
		return nil, fmt.Errorf("failed to count cores in output: %s", string(out))
	}
	clockSpeed, err := getClockSpeed(out)
	if err != nil {
		return nil, err
	}

	// Get the amount of usable memory from /proc/meminfo.
	out, err = ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	matches := memoryCapacityRegexp.FindSubmatch(out)
	if len(matches) != 2 {
		return nil, fmt.Errorf("failed to find memory capacity in output: %s", string(out))
	}
	memoryCapacity, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		return nil, err
	}

	// Capacity is in KB, convert it to bytes.
	memoryCapacity = memoryCapacity * 1024

	fsInfo, err := fs.NewFsInfo()
	if err != nil {
		return nil, err
	}
	filesystems, err := fsInfo.GetGlobalFsInfo()
	if err != nil {
		return nil, err
	}

	diskMap, err := sysfs.GetBlockDeviceInfo(sysFs)
	if err != nil {
		return nil, err
	}

	machineInfo := &info.MachineInfo{
		NumCores:       numCores,
		CpuFrequency:   clockSpeed,
		MemoryCapacity: memoryCapacity,
		DiskMap:        diskMap,
	}

	for _, fs := range filesystems {
		machineInfo.Filesystems = append(machineInfo.Filesystems, info.FsInfo{fs.Device, fs.Capacity})
	}

	return machineInfo, nil
}

func getVersionInfo() (*info.VersionInfo, error) {

	kernel_version := getKernelVersion()
	container_os := getContainerOsVersion()
	docker_version := getDockerVersion()

	return &info.VersionInfo{
		KernelVersion:      kernel_version,
		ContainerOsVersion: container_os,
		DockerVersion:      docker_version,
		CadvisorVersion:    info.VERSION,
	}, nil
}

func getContainerOsVersion() string {
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

func getDockerVersion() string {
	docker_version := "Unknown"
	client, err := dclient.NewClient(*docker.ArgDockerEndpoint)
	if err == nil {
		version, err := client.Version()
		if err == nil {
			docker_version = version.Get("Version")
		}
	}
	return docker_version
}

func getKernelVersion() string {
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
