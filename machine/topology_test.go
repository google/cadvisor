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

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysfs/fakesysfs"
)

func TestPhysicalCores(t *testing.T) {
	testfile := "./testdata/cpuinfo"

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numPhysicalCores := GetPhysicalCores(testcpuinfo)
	assert.Equal(t, 6, numPhysicalCores)
}

func TestPhysicalCoresReadingFromCpuBus(t *testing.T) {
	origCPUBusPath := cpuBusPath
	defer func() {
		cpuBusPath = origCPUBusPath
	}()
	cpuBusPath = "./testdata/sysfs_cpus/" // overwriting package variable to mock sysfs
	testfile := "./testdata/cpuinfo_arm"  // mock cpuinfo without core id

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numPhysicalCores := GetPhysicalCores(testcpuinfo)
	assert.Equal(t, 1, numPhysicalCores)
}

func TestPhysicalCoresFromWrongSysFs(t *testing.T) {
	origCPUBusPath := cpuBusPath
	defer func() {
		cpuBusPath = origCPUBusPath
	}()
	cpuBusPath = "./testdata/wrongsysfs" // overwriting package variable to mock sysfs
	testfile := "./testdata/cpuinfo_arm" // mock cpuinfo without core id

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numPhysicalCores := GetPhysicalCores(testcpuinfo)
	assert.Equal(t, 0, numPhysicalCores)
}

func TestSockets(t *testing.T) {
	testfile := "./testdata/cpuinfo"

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numSockets := GetSockets(testcpuinfo)
	assert.Equal(t, 2, numSockets)
}

func TestSocketsReadingFromCpuBus(t *testing.T) {
	origCPUBusPath := cpuBusPath
	defer func() {
		cpuBusPath = origCPUBusPath
	}()
	cpuBusPath = "./testdata/wrongsysfs" // overwriting package variable to mock sysfs
	testfile := "./testdata/cpuinfo_arm" // mock cpuinfo without physical id

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numSockets := GetSockets(testcpuinfo)
	assert.Equal(t, 0, numSockets)
}

func TestSocketsReadingFromWrongSysFs(t *testing.T) {
	path, err := filepath.Abs("./testdata/sysfs_cpus/")
	assert.NoError(t, err)

	origCPUBusPath := cpuBusPath
	defer func() {
		cpuBusPath = origCPUBusPath
	}()
	cpuBusPath = path                    // overwriting package variable to mock sysfs
	testfile := "./testdata/cpuinfo_arm" // mock cpuinfo without physical id

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numSockets := GetSockets(testcpuinfo)
	assert.Equal(t, 1, numSockets)
}

func TestTopology(t *testing.T) {
	machineArch = "" // overwrite package variable
	sysFs := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 1,
		Cpus:  2,
	}
	sysFs.SetCacheInfo(c)

	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
		"/fakeSysfs/devices/system/node/node1",
	}
	sysFs.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
			"/fakeSysfs/devices/system/node/node0/cpu1",
			"/fakeSysfs/devices/system/node/node0/cpu2",
			"/fakeSysfs/devices/system/node/node0/cpu6",
			"/fakeSysfs/devices/system/node/node0/cpu7",
			"/fakeSysfs/devices/system/node/node0/cpu8",
		},
		"/fakeSysfs/devices/system/node/node1": {
			"/fakeSysfs/devices/system/node/node0/cpu3",
			"/fakeSysfs/devices/system/node/node0/cpu4",
			"/fakeSysfs/devices/system/node/node0/cpu5",
			"/fakeSysfs/devices/system/node/node0/cpu9",
			"/fakeSysfs/devices/system/node/node0/cpu10",
			"/fakeSysfs/devices/system/node/node0/cpu11",
		},
	}
	sysFs.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu1":  "1",
		"/fakeSysfs/devices/system/node/node0/cpu2":  "2",
		"/fakeSysfs/devices/system/node/node0/cpu3":  "3",
		"/fakeSysfs/devices/system/node/node0/cpu4":  "4",
		"/fakeSysfs/devices/system/node/node0/cpu5":  "5",
		"/fakeSysfs/devices/system/node/node0/cpu6":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu7":  "1",
		"/fakeSysfs/devices/system/node/node0/cpu8":  "2",
		"/fakeSysfs/devices/system/node/node0/cpu9":  "3",
		"/fakeSysfs/devices/system/node/node0/cpu10": "4",
		"/fakeSysfs/devices/system/node/node0/cpu11": "5",
	}
	sysFs.SetCoreThreads(coreThread, nil)

	memTotal := "MemTotal:       32817192 kB"
	sysFs.SetMemory(memTotal, nil)

	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
		&fakesysfs.FileInfo{EntryName: "hugepages-1048576kB"},
	}
	sysFs.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages":    "1",
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages": "1",
		"/fakeSysfs/devices/system/node/node1/hugepages/hugepages-2048kB/nr_hugepages":    "1",
		"/fakeSysfs/devices/system/node/node1/hugepages/hugepages-1048576kB/nr_hugepages": "1",
	}
	sysFs.SetHugePagesNr(hugePageNr, nil)

	physicalPackageIDs := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu1":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu2":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu3":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu4":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu5":  "0",
		"/fakeSysfs/devices/system/node/node0/cpu6":  "1",
		"/fakeSysfs/devices/system/node/node0/cpu7":  "1",
		"/fakeSysfs/devices/system/node/node0/cpu8":  "1",
		"/fakeSysfs/devices/system/node/node0/cpu9":  "1",
		"/fakeSysfs/devices/system/node/node0/cpu10": "1",
		"/fakeSysfs/devices/system/node/node0/cpu11": "1",
	}
	sysFs.SetPhysicalPackageIDs(physicalPackageIDs, nil)
	topology, numCores, err := GetTopology(sysFs)
	assert.Nil(t, err)
	assert.Equal(t, 12, numCores)

	expectedTopology := []info.Node{}
	numNodes := 2
	numCoresPerNode := 3
	numThreads := 2
	cache := info.Cache{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 1,
	}
	for i := 0; i < numNodes; i++ {
		node := info.Node{Id: i}
		// Copy over Memory from result. TODO(rjnagal): Use memory from fake.
		node.Memory = topology[i].Memory
		// Copy over HugePagesInfo from result. TODO(ohsewon): Use HugePagesInfo from fake.
		node.HugePages = topology[i].HugePages
		for j := 0; j < numCoresPerNode; j++ {
			core := info.Core{Id: i*numCoresPerNode + j}
			core.Caches = append(core.Caches, cache)
			for k := 0; k < numThreads; k++ {
				core.Threads = append(core.Threads, k*numCoresPerNode*numNodes+core.Id)
			}
			node.Cores = append(node.Cores, core)
		}
		expectedTopology = append(expectedTopology, node)
	}

	assert.NotNil(t, reflect.DeepEqual(topology, expectedTopology))
}

func TestTopologyEmptySysFs(t *testing.T) {
	machineArch = "" // overwrite package variable
	_, _, err := GetTopology(&fakesysfs.FakeSysFs{})
	assert.NotNil(t, err)
}

func TestTopologyWithoutNodes(t *testing.T) {
	machineArch = "" // overwrite package variable
	sysFs := &fakesysfs.FakeSysFs{}

	c := sysfs.CacheInfo{
		Id:    0,
		Size:  32 * 1024,
		Type:  "unified",
		Level: 0,
		Cpus:  2,
	}
	sysFs.SetCacheInfo(c)

	nodesPaths := []string{}
	sysFs.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/sys/devices/system/cpu": {
			"/sys/devices/system/cpu/cpu0",
			"/sys/devices/system/cpu/cpu1",
			"/sys/devices/system/cpu/cpu2",
			"/sys/devices/system/cpu/cpu3",
		},
	}
	sysFs.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/sys/devices/system/cpu/cpu0": "0",
		"/sys/devices/system/cpu/cpu1": "1",
		"/sys/devices/system/cpu/cpu2": "0",
		"/sys/devices/system/cpu/cpu3": "1",
	}
	sysFs.SetCoreThreads(coreThread, nil)

	physicalPackageIDs := map[string]string{
		"/sys/devices/system/cpu/cpu0": "0",
		"/sys/devices/system/cpu/cpu1": "1",
		"/sys/devices/system/cpu/cpu2": "0",
		"/sys/devices/system/cpu/cpu3": "1",
	}
	sysFs.SetPhysicalPackageIDs(physicalPackageIDs, nil)

	topology, numCores, err := GetTopology(sysFs)
	sort.SliceStable(topology, func(i, j int) bool {
		return topology[i].Id < topology[j].Id
	})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(topology))
	assert.Equal(t, 4, numCores)

	topologyJSON1, err := json.Marshal(topology[0])
	assert.Nil(t, err)
	topologyJSON2, err := json.Marshal(topology[1])
	assert.Nil(t, err)

	expectedTopology1 := `{"node_id":0,"memory":0,"hugepages":null,"cores":[{"core_id":0,"thread_ids":[0,2],"caches":[{"id":0, "size":32768,"type":"unified","level":0}], "socket_id": 0, "uncore_caches":null}],"caches":null}`
	expectedTopology2 := `
		{
			"node_id":1,
			"memory":0,
			"hugepages":null,
			"cores":[
				{
					"core_id":1,
					"thread_ids":[
					1,
					3
					],
					"caches":[
					{
						"id": 0,
						"size":32768,
						"type":"unified",
						"level":0
					}
					],
					"socket_id": 1,
					"uncore_caches": null
				}
			],
			"caches":null
		}`

	json1 := string(topologyJSON1)
	json2 := string(topologyJSON2)

	assert.JSONEq(t, expectedTopology1, json1)
	assert.JSONEq(t, expectedTopology2, json2)
}

func TestTopologyWithNodesWithoutCPU(t *testing.T) {
	machineArch = "" // overwrite package variable
	sysFs := &fakesysfs.FakeSysFs{}
	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
		"/fakeSysfs/devices/system/node/node1",
	}
	sysFs.SetNodesPaths(nodesPaths, nil)

	memTotal := "MemTotal:       32817192 kB"
	sysFs.SetMemory(memTotal, nil)

	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
		&fakesysfs.FileInfo{EntryName: "hugepages-1048576kB"},
	}
	sysFs.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages":    "1",
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages": "1",
		"/fakeSysfs/devices/system/node/node1/hugepages/hugepages-2048kB/nr_hugepages":    "1",
		"/fakeSysfs/devices/system/node/node1/hugepages/hugepages-1048576kB/nr_hugepages": "1",
	}
	sysFs.SetHugePagesNr(hugePageNr, nil)

	topology, numCores, err := GetTopology(sysFs)

	assert.Nil(t, err)
	assert.Equal(t, 0, numCores)

	topologyJSON, err := json.Marshal(topology)
	assert.Nil(t, err)

	expectedTopology := `[
     {
      "caches": null,
      "cores": null,
      "hugepages": [
       {
        "num_pages": 1,
        "page_size": 2048
       },
       {
        "num_pages": 1,
        "page_size": 1048576
       }
      ],
      "memory": 33604804608,
      "node_id": 0
     },
     {
      "caches": null,
      "cores": null,
      "hugepages": [
       {
        "num_pages": 1,
        "page_size": 2048
       },
       {
        "num_pages": 1,
        "page_size": 1048576
       }
      ],
      "memory": 33604804608,
      "node_id": 1
     }
    ]
    `
	assert.JSONEq(t, expectedTopology, string(topologyJSON))
}

func TestTopologyOnSystemZ(t *testing.T) {
	machineArch = "s390" // overwrite package variable
	nodes, cores, err := GetTopology(&fakesysfs.FakeSysFs{})
	assert.Nil(t, err)
	assert.Nil(t, nodes)
	assert.NotNil(t, cores)
}

func TestMemoryInfo(t *testing.T) {
	testPath := "./testdata/edac/mc"
	memory, err := GetMachineMemoryByType(testPath)

	assert.Nil(t, err)
	assert.Len(t, memory, 2)
	assert.Equal(t, uint64(789*1024*1024), memory["Unbuffered-DDR4"].Capacity)
	assert.Equal(t, uint64(579*1024*1024), memory["Non-volatile-RAM"].Capacity)
	assert.Equal(t, uint(1), memory["Unbuffered-DDR4"].DimmCount)
	assert.Equal(t, uint(2), memory["Non-volatile-RAM"].DimmCount)
}

func TestMemoryInfoOnArchThatDoNotExposeMemoryController(t *testing.T) {
	testPath := "./there/is/no/spoon"
	memory, err := GetMachineMemoryByType(testPath)

	assert.Nil(t, err)
	assert.Len(t, memory, 0)
}

func TestClockSpeedOnCpuUpperCase(t *testing.T) {
	maxFreqFile = ""                            // do not read the system max frequency
	machineArch = ""                            // overwrite package variable
	testfile := "./testdata/cpuinfo_upper_case" // mock cpuinfo with CPU MHz

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	clockSpeed, err := GetClockSpeed(testcpuinfo)
	assert.Nil(t, err)
	assert.NotNil(t, clockSpeed)
	assert.Equal(t, uint64(1800*1000), clockSpeed)
}

func TestClockSpeedOnCpuLowerCase(t *testing.T) {
	maxFreqFile = ""                            // do not read the system max frequency
	machineArch = ""                            // overwrite package variable
	testfile := "./testdata/cpuinfo_lower_case" // mock cpuinfo with cpu MHz

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	clockSpeed, err := GetClockSpeed(testcpuinfo)
	assert.Nil(t, err)
	assert.NotNil(t, clockSpeed)
	assert.Equal(t, uint64(1450*1000), clockSpeed)
}
