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

package sysinfo

import (
	"testing"

	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysfs/fakesysfs"
)

func TestGetBlockDeviceInfo(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	disks, err := GetBlockDeviceInfo(&fakeSys)
	if err != nil {
		t.Errorf("expected call to GetBlockDeviceInfo() to succeed. Failed with %s", err)
	}
	if len(disks) != 1 {
		t.Errorf("expected to get one disk entry. Got %d", len(disks))
	}
	key := "8:0"
	disk, ok := disks[key]
	if !ok {
		t.Fatalf("expected key 8:0 to exist in the disk map.")
	}
	if disk.Name != "sda" {
		t.Errorf("expected to get disk named sda. Got %q", disk.Name)
	}
	size := uint64(1234567 * 512)
	if disk.Size != size {
		t.Errorf("expected to get disk size of %d. Got %d", size, disk.Size)
	}
}

func TestGetNetworkDevices(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	devs, err := GetNetworkDevices(&fakeSys)
	if err != nil {
		t.Errorf("expected call to GetNetworkDevices() to succeed. Failed with %s", err)
	}
	if len(devs) != 1 {
		t.Errorf("expected to get one network device. Got %d", len(devs))
	}
	eth := devs[0]
	if eth.Name != "eth0" {
		t.Errorf("expected to find device with name eth0. Found name %q", eth.Name)
	}
	if eth.Mtu != 1024 {
		t.Errorf("expected mtu to be set to 1024. Found %d", eth.Mtu)
	}
	if eth.Speed != 1000 {
		t.Errorf("expected device speed to be set to 1000. Found %d", eth.Speed)
	}
	if eth.MacAddress != "42:01:02:03:04:f4" {
		t.Errorf("expected mac address to be '42:01:02:03:04:f4'. Found %q", eth.MacAddress)
	}
}

func TestGetCacheInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	cacheInfo := sysfs.CacheInfo{
		Size:  1024,
		Type:  "Data",
		Level: 3,
		Cpus:  16,
	}
	fakeSys.SetCacheInfo(cacheInfo)
	caches, err := GetCacheInfo(fakeSys, 0)
	if err != nil {
		t.Errorf("expected call to GetCacheInfo() to succeed. Failed with %s", err)
	}
	if len(caches) != 1 {
		t.Errorf("expected to get one cache. Got %d", len(caches))
	}
	if caches[0] != cacheInfo {
		t.Errorf("expected to find cacheinfo %+v. Got %+v", cacheInfo, caches[0])
	}
}
