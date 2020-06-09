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

func (i *FileInfo) Name() string {
	return i.EntryName
}

func (i *FileInfo) Size() int64 {
	return 1234567
}

func (i *FileInfo) Mode() os.FileMode {
	return 0
}

func (i *FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (i *FileInfo) IsDir() bool {
	return true
}

func (i *FileInfo) Sys() interface{} {
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
	coreIDErr  map[string]error

	physicalPackageIDs   map[string]string
	physicalPackageIDErr map[string]error

	memTotal string
	memErr   error

	hugePages    []os.FileInfo
	hugePagesErr error

	hugePagesNr    map[string]string
	hugePagesNrErr error

	onlineCPUs map[string]interface{}
}

func (fs *FakeSysFs) GetNodesPaths() ([]string, error) {
	return fs.nodesPaths, fs.nodePathErr
}

func (fs *FakeSysFs) GetCPUsPaths(cpusPath string) ([]string, error) {
	return fs.cpusPaths[cpusPath], fs.cpuPathErr
}

func (fs *FakeSysFs) GetCoreID(coreIDPath string) (string, error) {
	return fs.coreThread[coreIDPath], fs.coreIDErr[coreIDPath]
}

func (fs *FakeSysFs) GetCPUPhysicalPackageID(cpuPath string) (string, error) {
	return fs.physicalPackageIDs[cpuPath], fs.physicalPackageIDErr[cpuPath]
}

func (fs *FakeSysFs) GetMemInfo(nodePath string) (string, error) {
	return fs.memTotal, fs.memErr
}

func (fs *FakeSysFs) GetHugePagesInfo(hugepagesDirectory string) ([]os.FileInfo, error) {
	return fs.hugePages, fs.hugePagesErr
}

func (fs *FakeSysFs) GetHugePagesNr(hugepagesDirectory string, hugePageName string) (string, error) {
	hugePageFile := fmt.Sprintf("%s%s/%s", hugepagesDirectory, hugePageName, sysfs.HugePagesNrFile)
	return fs.hugePagesNr[hugePageFile], fs.hugePagesNrErr
}

func (fs *FakeSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	fs.info.EntryName = "sda"
	return []os.FileInfo{&fs.info}, nil
}

func (fs *FakeSysFs) GetBlockDeviceSize(name string) (string, error) {
	return "1234567", nil
}

func (fs *FakeSysFs) GetBlockDeviceScheduler(name string) (string, error) {
	return "noop deadline [cfq]", nil
}

func (fs *FakeSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	return "8:0\n", nil
}

func (fs *FakeSysFs) GetNetworkDevices() ([]os.FileInfo, error) {
	return []os.FileInfo{&fs.info}, nil
}

func (fs *FakeSysFs) GetNetworkAddress(name string) (string, error) {
	return "42:01:02:03:04:f4\n", nil
}

func (fs *FakeSysFs) GetNetworkMtu(name string) (string, error) {
	return "1024\n", nil
}

func (fs *FakeSysFs) GetNetworkSpeed(name string) (string, error) {
	return "1000\n", nil
}

func (fs *FakeSysFs) GetNetworkStatValue(name string, stat string) (uint64, error) {
	return 1024, nil
}

func (fs *FakeSysFs) GetCaches(id int) ([]os.FileInfo, error) {
	fs.info.EntryName = "index0"
	return []os.FileInfo{&fs.info}, nil
}

func (fs *FakeSysFs) GetCacheInfo(cpu int, cache string) (sysfs.CacheInfo, error) {
	return fs.cache, nil
}

func (fs *FakeSysFs) SetCacheInfo(cache sysfs.CacheInfo) {
	fs.cache = cache
}

func (fs *FakeSysFs) SetNodesPaths(paths []string, err error) {
	fs.nodesPaths = paths
	fs.nodePathErr = err
}

func (fs *FakeSysFs) SetCPUsPaths(paths map[string][]string, err error) {
	fs.cpusPaths = paths
	fs.cpuPathErr = err
}

func (fs *FakeSysFs) SetCoreThreads(coreThread map[string]string, coreThreadErrors map[string]error) {
	fs.coreThread = coreThread
	fs.coreIDErr = coreThreadErrors
}

func (fs *FakeSysFs) SetPhysicalPackageIDs(physicalPackageIDs map[string]string, physicalPackageIDErrors map[string]error) {
	fs.physicalPackageIDs = physicalPackageIDs
	fs.physicalPackageIDErr = physicalPackageIDErrors
}

func (fs *FakeSysFs) SetMemory(memTotal string, err error) {
	fs.memTotal = memTotal
	fs.memErr = err
}

func (fs *FakeSysFs) SetHugePages(hugePages []os.FileInfo, err error) {
	fs.hugePages = hugePages
	fs.hugePagesErr = err
}

func (fs *FakeSysFs) SetHugePagesNr(hugePagesNr map[string]string, err error) {
	fs.hugePagesNr = hugePagesNr
	fs.hugePagesNrErr = err
}

func (fs *FakeSysFs) SetEntryName(name string) {
	fs.info.EntryName = name
}

func (fs *FakeSysFs) GetSystemUUID() (string, error) {
	return "1F862619-BA9F-4526-8F85-ECEAF0C97430", nil
}

func (fs *FakeSysFs) IsCPUOnline(dir string) bool {
	if fs.onlineCPUs == nil {
		return true
	}
	_, ok := fs.onlineCPUs[dir]
	return ok
}

func (fs *FakeSysFs) SetOnlineCPUs(online map[string]interface{}) {
	fs.onlineCPUs = online
}
