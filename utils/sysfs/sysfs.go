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

package sysfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/google/cadvisor/info"
)

const (
	blockDir = "/sys/block"
	cacheDir = "/sys/devices/system/cpu/cpu"
	netDir   = "/sys/class/net"
)

// Abstracts the lowest level calls to sysfs.
type SysFs interface {
	// Get directory information for available block devices.
	GetBlockDevices() ([]os.FileInfo, error)
	// Get Size of a given block device.
	GetBlockDeviceSize(string) (string, error)
	// Get device major:minor number string.
	GetBlockDeviceNumbers(string) (string, error)

	GetNetworkDevices() ([]os.FileInfo, error)
	GetNetworkAddress(string) (string, error)
	GetNetworkMtu(string) (string, error)
	GetNetworkSpeed(string) (string, error)

	// Get directory information for available caches accessible to given cpu.
	GetCaches(id int) ([]os.FileInfo, error)
	// Get information for a cache accessible from the given cpu.
	GetCacheInfo(cpu int, cache string) (CacheInfo, error)
}

type realSysFs struct{}

func NewRealSysFs() (SysFs, error) {
	return &realSysFs{}, nil
}

func (self *realSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	return ioutil.ReadDir(blockDir)
}

func (self *realSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	dev, err := ioutil.ReadFile(path.Join(blockDir, name, "/dev"))
	if err != nil {
		return "", err
	}
	return string(dev), nil
}

func (self *realSysFs) GetBlockDeviceSize(name string) (string, error) {
	size, err := ioutil.ReadFile(path.Join(blockDir, name, "/size"))
	if err != nil {
		return "", err
	}
	return string(size), nil
}

func (self *realSysFs) GetNetworkDevices() ([]os.FileInfo, error) {
	return ioutil.ReadDir(netDir)
}

func (self *realSysFs) GetNetworkAddress(name string) (string, error) {
	address, err := ioutil.ReadFile(path.Join(netDir, name, "/address"))
	if err != nil {
		return "", err
	}
	return string(address), nil
}

func (self *realSysFs) GetNetworkMtu(name string) (string, error) {
	mtu, err := ioutil.ReadFile(path.Join(netDir, name, "/mtu"))
	if err != nil {
		return "", err
	}
	return string(mtu), nil
}

func (self *realSysFs) GetNetworkSpeed(name string) (string, error) {
	speed, err := ioutil.ReadFile(path.Join(netDir, name, "/speed"))
	if err != nil {
		return "", err
	}
	return string(speed), nil
}

func (self *realSysFs) GetCaches(id int) ([]os.FileInfo, error) {
	cpuPath := fmt.Sprintf("%s%d/cache", cacheDir, id)
	return ioutil.ReadDir(cpuPath)
}

func (self *realSysFs) GetCacheInfo(id int, name string) (CacheInfo, error) {
	cachePath := fmt.Sprintf("%s%d/cache/%s", cacheDir, id, name)
	out, err := ioutil.ReadFile(path.Join(cachePath, "/size"))
	if err != nil {
		return CacheInfo{}, err
	}
	var size uint64
	n, err := fmt.Sscanf(string(out), "%dK", &size)
	if err != nil || n != 1 {
		return CacheInfo{}, err
	}
	// convert to bytes
	size = size * 1024
	out, err = ioutil.ReadFile(path.Join(cachePath, "/level"))
	if err != nil {
		return CacheInfo{}, err
	}
	var level int
	n, err = fmt.Sscanf(string(out), "%d", &level)
	if err != nil || n != 1 {
		return CacheInfo{}, err
	}

	out, err = ioutil.ReadFile(path.Join(cachePath, "/type"))
	if err != nil {
		return CacheInfo{}, err
	}
	cacheType := strings.TrimSpace(string(out))
	out, err = ioutil.ReadFile(path.Join(cachePath, "/shared_cpu_list"))
	if err != nil {
		return CacheInfo{}, err
	}
	cpus := strings.Split(string(out), ",")
	return CacheInfo{
		Size:  size,
		Level: level,
		Type:  cacheType,
		Cpus:  len(cpus),
	}, nil
}

// Get information about block devices present on the system.
// Uses the passed in system interface to retrieve the low level OS information.
func GetBlockDeviceInfo(sysfs SysFs) (map[string]info.DiskInfo, error) {
	disks, err := sysfs.GetBlockDevices()
	if err != nil {
		return nil, err
	}

	diskMap := make(map[string]info.DiskInfo)
	for _, disk := range disks {
		name := disk.Name()
		// Ignore non-disk devices.
		// TODO(rjnagal): Maybe just match hd, sd, and dm prefixes.
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "sr") {
			continue
		}
		disk_info := info.DiskInfo{
			Name: name,
		}
		dev, err := sysfs.GetBlockDeviceNumbers(name)
		if err != nil {
			return nil, err
		}
		n, err := fmt.Sscanf(dev, "%d:%d", &disk_info.Major, &disk_info.Minor)
		if err != nil || n != 2 {
			return nil, fmt.Errorf("could not parse device numbers from %s for device %s", dev, name)
		}
		out, err := sysfs.GetBlockDeviceSize(name)
		if err != nil {
			return nil, err
		}
		// Remove trailing newline before conversion.
		size, err := strconv.ParseUint(strings.TrimSpace(out), 10, 64)
		if err != nil {
			return nil, err
		}
		// size is in 512 bytes blocks.
		disk_info.Size = size * 512

		device := fmt.Sprintf("%d:%d", disk_info.Major, disk_info.Minor)
		diskMap[device] = disk_info
	}
	return diskMap, nil
}

// Get information about network devices present on the system.
func GetNetworkDevices(sysfs SysFs) ([]info.NetInfo, error) {
	devs, err := sysfs.GetNetworkDevices()
	if err != nil {
		return nil, err
	}
	netDevices := []info.NetInfo{}
	for _, dev := range devs {
		name := dev.Name()
		// Only consider ethernet devices for now.
		if !strings.HasPrefix(name, "eth") {
			continue
		}
		address, err := sysfs.GetNetworkAddress(name)
		if err != nil {
			return nil, err
		}
		mtuStr, err := sysfs.GetNetworkMtu(name)
		if err != nil {
			return nil, err
		}
		var mtu uint64
		n, err := fmt.Sscanf(mtuStr, "%d", &mtu)
		if err != nil || n != 1 {
			return nil, fmt.Errorf("could not parse mtu from %s for device %s", mtuStr, name)
		}
		netInfo := info.NetInfo{
			Name:       name,
			MacAddress: strings.TrimSpace(address),
			Mtu:        mtu,
		}
		speed, err := sysfs.GetNetworkSpeed(name)
		// Some devices don't set speed.
		if err == nil {
			var s uint64
			n, err := fmt.Sscanf(speed, "%d", &s)
			if err != nil || n != 1 {
				return nil, fmt.Errorf("could not parse speed from %s for device %s", speed, name)
			}
			netInfo.Speed = s
		}
		netDevices = append(netDevices, netInfo)
	}
	return netDevices, nil
}

type CacheInfo struct {
	// size in bytes
	Size uint64
	// cache type - instruction, data, unified
	Type string
	// distance from cpus in a multi-level hierarchy
	Level int
	// number of cpus that can access this cache.
	Cpus int
}

func GetCacheInfo(sysfs SysFs, id int) ([]CacheInfo, error) {
	caches, err := sysfs.GetCaches(id)
	if err != nil {
		return nil, err
	}

	info := []CacheInfo{}
	for _, cache := range caches {
		if !strings.HasPrefix(cache.Name(), "index") {
			continue
		}
		cacheInfo, err := sysfs.GetCacheInfo(id, cache.Name())
		if err != nil {
			return nil, err
		}
		info = append(info, cacheInfo)
	}
	return info, nil
}
