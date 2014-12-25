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

const BlockDir = "/sys/block"

// Abstracts the lowest level calls to sysfs.
type SysFs interface {
	// Get directory information for available block devices.
	GetBlockDevices() ([]os.FileInfo, error)
	// Get Size of a given block device.
	GetBlockDeviceSize(string) (string, error)
	// Get device major:minor number string.
	GetBlockDeviceNumbers(string) (string, error)
}

type realSysFs struct{}

func NewRealSysFs() (SysFs, error) {
	return &realSysFs{}, nil
}

func (self *realSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	return ioutil.ReadDir(BlockDir)
}

func (self *realSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	dev, err := ioutil.ReadFile(path.Join(BlockDir, name, "/dev"))
	if err != nil {
		return "", err
	}
	return string(dev), nil
}

func (self *realSysFs) GetBlockDeviceSize(name string) (string, error) {
	size, err := ioutil.ReadFile(path.Join(BlockDir, name, "/size"))
	if err != nil {
		return "", err
	}
	return string(size), nil
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
