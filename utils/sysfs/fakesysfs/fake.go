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

func (fi *FileInfo) Name() string {
	return fi.EntryName
}

func (fi *FileInfo) Size() int64 {
	return 1234567
}

func (fi *FileInfo) Mode() os.FileMode {
	return 0
}

func (fi *FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (fi *FileInfo) IsDir() bool {
	return true
}

func (fi *FileInfo) Sys() interface{} {
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

func (f *FakeSysFs) GetNodesPaths() ([]string, error) {
	return f.nodesPaths, f.nodePathErr
}

func (f *FakeSysFs) GetCPUsPaths(cpusPath string) ([]string, error) {
	return f.cpusPaths[cpusPath], f.cpuPathErr
}

func (f *FakeSysFs) GetCoreID(coreIDPath string) (string, error) {
	return f.coreThread[coreIDPath], f.coreIDErr
}

func (f *FakeSysFs) GetCPUPhysicalPackageID(cpuPath string) (string, error) {
	return f.physicalPackageIDs[cpuPath], f.physicalPackageIDErr
}

func (f *FakeSysFs) GetMemInfo(nodePath string) (string, error) {
	return f.memTotal, f.memErr
}

func (f *FakeSysFs) GetHugePagesInfo(hugepagesDirectory string) ([]os.FileInfo, error) {
	return f.hugePages, f.hugePagesErr
}

func (f *FakeSysFs) GetHugePagesNr(hugepagesDirectory string, hugePageName string) (string, error) {
	hugePageFile := fmt.Sprintf("%s%s/%s", hugepagesDirectory, hugePageName, sysfs.HugePagesNrFile)
	return f.hugePagesNr[hugePageFile], f.hugePagesNrErr
}

func (f *FakeSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	f.info.EntryName = "sda"
	return []os.FileInfo{&f.info}, nil
}

func (f *FakeSysFs) GetBlockDeviceSize(name string) (string, error) {
	return "1234567", nil
}

func (f *FakeSysFs) GetBlockDeviceScheduler(name string) (string, error) {
	return "noop deadline [cfq]", nil
}

func (f *FakeSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	return "8:0\n", nil
}

func (f *FakeSysFs) GetNetworkDevices() ([]os.FileInfo, error) {
	return []os.FileInfo{&f.info}, nil
}

func (f *FakeSysFs) GetNetworkAddress(name string) (string, error) {
	return "42:01:02:03:04:f4\n", nil
}

func (f *FakeSysFs) GetNetworkMtu(name string) (string, error) {
	return "1024\n", nil
}

func (f *FakeSysFs) GetNetworkSpeed(name string) (string, error) {
	return "1000\n", nil
}

func (f *FakeSysFs) GetNetworkStatValue(name string, stat string) (uint64, error) {
	return 1024, nil
}

func (f *FakeSysFs) GetCaches(id int) ([]os.FileInfo, error) {
	f.info.EntryName = "index0"
	return []os.FileInfo{&f.info}, nil
}

func (f *FakeSysFs) GetCacheInfo(cpu int, cache string) (sysfs.CacheInfo, error) {
	return f.cache, nil
}

func (f *FakeSysFs) SetCacheInfo(cache sysfs.CacheInfo) {
	f.cache = cache
}

func (f *FakeSysFs) SetNodesPaths(paths []string, err error) {
	f.nodesPaths = paths
	f.nodePathErr = err
}

func (f *FakeSysFs) SetCPUsPaths(paths map[string][]string, err error) {
	f.cpusPaths = paths
	f.cpuPathErr = err
}

func (f *FakeSysFs) SetCoreThreads(coreThread map[string]string, err error) {
	f.coreThread = coreThread
	f.coreIDErr = err
}

func (f *FakeSysFs) SetPhysicalPackageIDs(physicalPackageIDs map[string]string, err error) {
	f.physicalPackageIDs = physicalPackageIDs
	f.physicalPackageIDErr = err
}

func (f *FakeSysFs) SetMemory(memTotal string, err error) {
	f.memTotal = memTotal
	f.memErr = err
}

func (f *FakeSysFs) SetHugePages(hugePages []os.FileInfo, err error) {
	f.hugePages = hugePages
	f.hugePagesErr = err
}

func (f *FakeSysFs) SetHugePagesNr(hugePagesNr map[string]string, err error) {
	f.hugePagesNr = hugePagesNr
	f.hugePagesNrErr = err
}

func (f *FakeSysFs) SetEntryName(name string) {
	f.info.EntryName = name
}

func (f *FakeSysFs) GetSystemUUID() (string, error) {
	return "1F862619-BA9F-4526-8F85-ECEAF0C97430", nil
}
