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
	"encoding/json"
	"fmt"
	"os"
	"testing"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysfs/fakesysfs"
	"github.com/stretchr/testify/assert"
)

func TestGetHugePagesInfo(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
		&fakesysfs.FileInfo{EntryName: "hugepages-1048576kB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages":    "1",
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages": "1",
	}
	fakeSys.SetHugePagesNr(hugePageNr, nil)

	hugePagesInfo, err := GetHugePagesInfo(&fakeSys, "/fakeSysfs/devices/system/node/node0/hugepages/")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(hugePagesInfo))
}

func TestGetHugePagesInfoWithHugePagesDirectory(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	hugePagesInfo, err := GetHugePagesInfo(&fakeSys, "/fakeSysfs/devices/system/node/node0/hugepages/")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(hugePagesInfo))
}

func TestGetHugePagesInfoWithWrongDirName(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-abckB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePagesInfo, err := GetHugePagesInfo(&fakeSys, "/fakeSysfs/devices/system/node/node0/hugepages/")
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(hugePagesInfo))
}

func TestGetHugePagesInfoWithReadingNrHugePagesError(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
		&fakesysfs.FileInfo{EntryName: "hugepages-1048576kB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages":    "1",
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages": "1",
	}
	fakeSys.SetHugePagesNr(hugePageNr, fmt.Errorf("Error in reading nr_hugepages"))

	hugePagesInfo, err := GetHugePagesInfo(&fakeSys, "/fakeSysfs/devices/system/node/node0/hugepages/")
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(hugePagesInfo))
}

func TestGetHugePagesInfoWithWrongNrHugePageValue(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
		&fakesysfs.FileInfo{EntryName: "hugepages-1048576kB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages":    "*****",
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages": "1",
	}
	fakeSys.SetHugePagesNr(hugePageNr, nil)

	hugePagesInfo, err := GetHugePagesInfo(&fakeSys, "/fakeSysfs/devices/system/node/node0/hugepages/")
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(hugePagesInfo))
}

func TestGetNodesInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 3,
		Cpus:  2,
	}
	fakeSys.SetCacheInfo(c)

	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
		"/fakeSysfs/devices/system/node/node1",
	}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
			"/fakeSysfs/devices/system/node/node0/cpu1",
		},
		"/fakeSysfs/devices/system/node/node1": {
			"/fakeSysfs/devices/system/node/node0/cpu2",
			"/fakeSysfs/devices/system/node/node0/cpu3",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
		"/fakeSysfs/devices/system/node/node0/cpu2": "1",
		"/fakeSysfs/devices/system/node/node0/cpu3": "1",
	}
	fakeSys.SetCoreThreads(coreThread, nil)

	memTotal := "MemTotal:       32817192 kB"
	fakeSys.SetMemory(memTotal, nil)

	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages": "1",
		"/fakeSysfs/devices/system/node/node1/hugepages/hugepages-2048kB/nr_hugepages": "1",
	}
	fakeSys.SetHugePagesNr(hugePageNr, nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, 4, cores)

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)
	expectedNodes := `
	[
      {
        "node_id": 0,
        "memory": 33604804608,
        "hugepages": [
          {
            "page_size": 2048,
            "num_pages": 1
          }
        ],
        "cores": [
          {
            "core_id": 0,
            "thread_ids": [
              0,
              1
            ],
            "caches": null
          }
        ],
        "caches": [
          {
            "size": 32768,
            "type": "unified",
            "level": 3
          }
        ]
      },
      {
        "node_id": 1,
        "memory": 33604804608,
        "hugepages": [
          {
            "page_size": 2048,
            "num_pages": 1
          }
        ],
        "cores": [
          {
            "core_id": 1,
            "thread_ids": [
              2,
              3
            ],
            "caches": null
          }
        ],
        "caches": [
          {
            "size": 32768,
            "type": "unified",
            "level": 3
          }
        ]
      }
    ]
    `
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetNodesWithoutMemoryInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 3,
		Cpus:  2,
	}
	fakeSys.SetCacheInfo(c)

	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
		"/fakeSysfs/devices/system/node/node1",
	}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
			"/fakeSysfs/devices/system/node/node0/cpu1",
		},
		"/fakeSysfs/devices/system/node/node1": {
			"/fakeSysfs/devices/system/node/node0/cpu2",
			"/fakeSysfs/devices/system/node/node0/cpu3",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
		"/fakeSysfs/devices/system/node/node0/cpu2": "1",
		"/fakeSysfs/devices/system/node/node0/cpu3": "1",
	}
	fakeSys.SetCoreThreads(coreThread, nil)

	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages": "1",
		"/fakeSysfs/devices/system/node/node1/hugepages/hugepages-2048kB/nr_hugepages": "1",
	}
	fakeSys.SetHugePagesNr(hugePageNr, nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.NotNil(t, err)
	assert.Equal(t, []info.Node([]info.Node(nil)), nodes)
	assert.Equal(t, 0, cores)
}

func TestGetNodesInfoWithoutHugePagesInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 2,
		Cpus:  2,
	}
	fakeSys.SetCacheInfo(c)

	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
		"/fakeSysfs/devices/system/node/node1",
	}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
			"/fakeSysfs/devices/system/node/node0/cpu1",
		},
		"/fakeSysfs/devices/system/node/node1": {
			"/fakeSysfs/devices/system/node/node0/cpu2",
			"/fakeSysfs/devices/system/node/node0/cpu3",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
		"/fakeSysfs/devices/system/node/node0/cpu2": "1",
		"/fakeSysfs/devices/system/node/node0/cpu3": "1",
	}
	fakeSys.SetCoreThreads(coreThread, nil)

	memTotal := "MemTotal:       32817192 kB"
	fakeSys.SetMemory(memTotal, nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, 4, cores)

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)
	expectedNodes := `
	[
      {
        "node_id": 0,
        "memory": 33604804608,
        "hugepages": null,
        "cores": [
          {
            "core_id": 0,
            "thread_ids": [
              0,
              1
            ],
            "caches": [
              {
                "size": 32768,
                "type": "unified",
                "level": 2
              }
            ]
          }
        ],
        "caches": null
      },
      {
        "node_id": 1,
        "memory": 33604804608,
        "hugepages": null,
        "cores": [
          {
            "core_id": 1,
            "thread_ids": [
              2,
              3
            ],
            "caches": [
              {
                "size": 32768,
                "type": "unified",
                "level": 2
              }
            ]
          }
        ],
        "caches": null
      }
    ]`
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetNodesInfoWithoutNodes(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}

	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 1,
		Cpus:  2,
	}
	fakeSys.SetCacheInfo(c)

	nodesPaths := []string{}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		cpusPath: {
			cpusPath + "/cpu0",
			cpusPath + "/cpu1",
			cpusPath + "/cpu2",
			cpusPath + "/cpu3",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		cpusPath + "/cpu0": "0",
		cpusPath + "/cpu1": "0",
		cpusPath + "/cpu2": "1",
		cpusPath + "/cpu3": "1",
	}
	fakeSys.SetCoreThreads(coreThread, nil)

	physicalPackageIDs := map[string]string{
		"/sys/devices/system/cpu/cpu0": "0",
		"/sys/devices/system/cpu/cpu1": "0",
		"/sys/devices/system/cpu/cpu2": "1",
		"/sys/devices/system/cpu/cpu3": "1",
	}
	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, 4, cores)

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)
	fmt.Println(string(nodesJSON))

	expectedNodes := `[
		{
			"node_id":0,
			"memory":0,
			"hugepages":null,
			"cores":[
			   {
				  "core_id":0,
				  "thread_ids":[
					 0,
					 1
				  ],
				  "caches":[
					 {
						"size":32768,
						"type":"unified",
						"level":1
					 }
				  ]
			   }
			],
			"caches":null
		 },
		 {
			"node_id":1,
			"memory":0,
			"hugepages":null,
			"cores":[
			   {
				  "core_id":1,
				  "thread_ids":[
					 2,
					 3
				  ],
				  "caches":[
					 {
						"size":32768,
						"type":"unified",
						"level":1
					 }
				  ]
			   }
			],
			"caches":null
		 }
	]`
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetNodeMemInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	memTotal := "MemTotal:       32817192 kB"
	fakeSys.SetMemory(memTotal, nil)

	mem, err := getNodeMemInfo(fakeSys, "/fakeSysfs/devices/system/node/node0")
	assert.Nil(t, err)
	assert.Equal(t, uint64(32817192*1024), mem)
}

func TestGetNodeMemInfoWithMissingMemTotaInMemInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	memTotal := "MemXXX:       32817192 kB"
	fakeSys.SetMemory(memTotal, nil)

	mem, err := getNodeMemInfo(fakeSys, "/fakeSysfs/devices/system/node/node0")
	assert.NotNil(t, err)
	assert.Equal(t, uint64(0), mem)
}

func TestGetNodeMemInfoWhenMemInfoMissing(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	memTotal := ""
	fakeSys.SetMemory(memTotal, fmt.Errorf("Cannot read meminfo file"))

	mem, err := getNodeMemInfo(fakeSys, "/fakeSysfs/devices/system/node/node0")
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), mem)
}

func TestGetCoresInfoWhenCoreIDIsNotDigit(t *testing.T) {
	sysFs := &fakesysfs.FakeSysFs{}
	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
	}
	sysFs.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
		},
	}
	sysFs.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "abc",
	}
	sysFs.SetCoreThreads(coreThread, nil)

	cores, err := getCoresInfo(sysFs, []string{"/fakeSysfs/devices/system/node/node0/cpu0"})
	assert.NotNil(t, err)
	assert.Equal(t, []info.Core(nil), cores)
}

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
	if disk.Scheduler != "cfq" {
		t.Errorf("expected to get scheduler type of cfq. Got %q", disk.Scheduler)
	}
}

func TestGetNetworkDevices(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	fakeSys.SetEntryName("eth0")
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

func TestIgnoredNetworkDevices(t *testing.T) {
	fakeSys := fakesysfs.FakeSysFs{}
	ignoredDevices := []string{"veth1234", "lo", "docker0"}
	for _, name := range ignoredDevices {
		fakeSys.SetEntryName(name)
		devs, err := GetNetworkDevices(&fakeSys)
		if err != nil {
			t.Errorf("expected call to GetNetworkDevices() to succeed. Failed with %s", err)
		}
		if len(devs) != 0 {
			t.Errorf("expected dev %s to be ignored, but got info %+v", name, devs)
		}
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

func TestGetNetworkStats(t *testing.T) {
	expected_stats := info.InterfaceStats{
		Name:      "eth0",
		RxBytes:   1024,
		RxPackets: 1024,
		RxErrors:  1024,
		RxDropped: 1024,
		TxBytes:   1024,
		TxPackets: 1024,
		TxErrors:  1024,
		TxDropped: 1024,
	}
	fakeSys := &fakesysfs.FakeSysFs{}
	netStats, err := getNetworkStats("eth0", fakeSys)
	if err != nil {
		t.Errorf("call to getNetworkStats() failed with %s", err)
	}
	if expected_stats != netStats {
		t.Errorf("expected to get stats %+v, got %+v", expected_stats, netStats)
	}
}
