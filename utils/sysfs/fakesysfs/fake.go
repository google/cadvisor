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

package fakesysfs

import (
	"fmt"
	"os"
	"time"

	"github.com/google/cadvisor/utils/sysfs"
)

// If we extend sysfs to support more interfaces, it might be worth making this a mock instead of a fake.
type FileInfo struct {
	EntryName string
}

func (self *FileInfo) Name() string {
	return self.EntryName
}

func (self *FileInfo) Size() int64 {
	return 1234567
}

func (self *FileInfo) Mode() os.FileMode {
	return 0
}

func (self *FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (self *FileInfo) IsDir() bool {
	return true
}

func (self *FileInfo) Sys() interface{} {
	return nil
}

type FakeSysFs struct {
	info  FileInfo
	cache sysfs.CacheInfo

	nodesPaths  []string
	nodePathErr error

	cpusPaths  map[string][]string
	cpuPathErr error

	coreThread map[string]string
	coreIDErr  error

	physicalPackageIDs   map[string]string
	physicalPackageIDErr error

	memTotal string
	memErr   error

	hugePages    []os.FileInfo
	hugePagesErr error

	hugePagesNr    map[string]string
	hugePagesNrErr error
}

func (self *FakeSysFs) GetNodesPaths() ([]string, error) {
	return self.nodesPaths, self.nodePathErr
}

func (self *FakeSysFs) GetCPUsPaths(cpusPath string) ([]string, error) {
	return self.cpusPaths[cpusPath], self.cpuPathErr
}

func (self *FakeSysFs) GetCoreID(coreIDPath string) (string, error) {
	return self.coreThread[coreIDPath], self.coreIDErr
}

func (self *FakeSysFs) GetCPUPhysicalPackageID(cpuPath string) (string, error) {
	return self.physicalPackageIDs[cpuPath], self.physicalPackageIDErr
}

func (self *FakeSysFs) GetMemInfo(nodePath string) (string, error) {
	return self.memTotal, self.memErr
}

func (self *FakeSysFs) GetHugePagesInfo(hugepagesDirectory string) ([]os.FileInfo, error) {
	return self.hugePages, self.hugePagesErr
}

func (self *FakeSysFs) GetHugePagesNr(hugepagesDirectory string, hugePageName string) (string, error) {
	hugePageFile := fmt.Sprintf("%s%s/%s", hugepagesDirectory, hugePageName, sysfs.HugePagesNrFile)
	return self.hugePagesNr[hugePageFile], self.hugePagesNrErr
}

func (self *FakeSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	self.info.EntryName = "sda"
	return []os.FileInfo{&self.info}, nil
}

func (self *FakeSysFs) GetBlockDeviceSize(name string) (string, error) {
	return "1234567", nil
}

func (self *FakeSysFs) GetBlockDeviceScheduler(name string) (string, error) {
	return "noop deadline [cfq]", nil
}

func (self *FakeSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	return "8:0\n", nil
}

func (self *FakeSysFs) GetNetworkDevices() ([]os.FileInfo, error) {
	return []os.FileInfo{&self.info}, nil
}

func (self *FakeSysFs) GetNetworkAddress(name string) (string, error) {
	return "42:01:02:03:04:f4\n", nil
}

func (self *FakeSysFs) GetNetworkMtu(name string) (string, error) {
	return "1024\n", nil
}

func (self *FakeSysFs) GetNetworkSpeed(name string) (string, error) {
	return "1000\n", nil
}

func (self *FakeSysFs) GetNetworkStatValue(name string, stat string) (uint64, error) {
	return 1024, nil
}

func (self *FakeSysFs) GetCaches(id int) ([]os.FileInfo, error) {
	self.info.EntryName = "index0"
	return []os.FileInfo{&self.info}, nil
}

func (self *FakeSysFs) GetCacheInfo(cpu int, cache string) (sysfs.CacheInfo, error) {
	return self.cache, nil
}

func (self *FakeSysFs) SetCacheInfo(cache sysfs.CacheInfo) {
	self.cache = cache
}

func (self *FakeSysFs) SetNodesPaths(paths []string, err error) {
	self.nodesPaths = paths
	self.nodePathErr = err
}

func (self *FakeSysFs) SetCPUsPaths(paths map[string][]string, err error) {
	self.cpusPaths = paths
	self.cpuPathErr = err
}

func (self *FakeSysFs) SetCoreThreads(coreThread map[string]string, err error) {
	self.coreThread = coreThread
	self.coreIDErr = err
}

func (self *FakeSysFs) SetPhysicalPackageIDs(physicalPackageIDs map[string]string, err error) {
	self.physicalPackageIDs = physicalPackageIDs
	self.physicalPackageIDErr = err
}

func (self *FakeSysFs) SetMemory(memTotal string, err error) {
	self.memTotal = memTotal
	self.memErr = err
}

func (self *FakeSysFs) SetHugePages(hugePages []os.FileInfo, err error) {
	self.hugePages = hugePages
	self.hugePagesErr = err
}

func (self *FakeSysFs) SetHugePagesNr(hugePagesNr map[string]string, err error) {
	self.hugePagesNr = hugePagesNr
	self.hugePagesNrErr = err
}

func (self *FakeSysFs) SetEntryName(name string) {
	self.info.EntryName = name
}

func (self *FakeSysFs) GetSystemUUID() (string, error) {
	return "1F862619-BA9F-4526-8F85-ECEAF0C97430", nil
}
