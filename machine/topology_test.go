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
	"io/ioutil"
	"reflect"
	"runtime"
	"testing"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysfs/fakesysfs"
	"github.com/stretchr/testify/assert"
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
	cpuBusPath = "./testdata/"           // overwriting global variable to mock sysfs
	testfile := "./testdata/cpuinfo_arm" // mock cpuinfo without core id

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numPhysicalCores := GetPhysicalCores(testcpuinfo)
	assert.Equal(t, 2, numPhysicalCores)
}

func TestPhysicalCoresFromWrongSysFs(t *testing.T) {
	cpuBusPath = "./testdata/wrongsysfs" // overwriting global variable to mock sysfs
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
	cpuBusPath = "./testdata/wrongsysfs" // overwriting global variable to mock sysfs
	testfile := "./testdata/cpuinfo_arm" // mock cpuinfo without physical id

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numSockets := GetSockets(testcpuinfo)
	assert.Equal(t, 0, numSockets)
}

func TestSocketsReadingFromWrongSysFs(t *testing.T) {
	cpuBusPath = "./testdata/"           // overwriting global variable to mock sysfs
	testfile := "./testdata/cpuinfo_arm" // mock cpuinfo without physical id

	testcpuinfo, err := ioutil.ReadFile(testfile)
	assert.Nil(t, err)
	assert.NotNil(t, testcpuinfo)

	numSockets := GetSockets(testcpuinfo)
	assert.Equal(t, 1, numSockets)
}

func TestTopology(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("cpuinfo testdata is for amd64")
	}
	testfile := "./testdata/cpuinfo"
	testcpuinfo, err := ioutil.ReadFile(testfile)
	if err != nil {
		t.Fatalf("unable to read input test file %s", testfile)
	}
	sysFs := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 1,
		Cpus:  2,
	}
	sysFs.SetCacheInfo(c)
	topology, numCores, err := GetTopology(sysFs, string(testcpuinfo))
	if err != nil {
		t.Errorf("failed to get topology for sample cpuinfo %s: %v", string(testcpuinfo), err)
	}

	if numCores != 12 {
		t.Errorf("Expected 12 cores, found %d", numCores)
	}
	expected_topology := []info.Node{}
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
		expected_topology = append(expected_topology, node)
	}

	if !reflect.DeepEqual(topology, expected_topology) {
		t.Errorf("Expected topology %+v, got %+v", expected_topology, topology)
	}
}

func TestTopologyWithSimpleCpuinfo(t *testing.T) {
	if isSystemZ() {
		t.Skip("systemZ has no topology info")
	}
	sysFs := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 1,
		Cpus:  1,
	}
	sysFs.SetCacheInfo(c)
	topology, numCores, err := GetTopology(sysFs, "processor\t: 0\n")
	if err != nil {
		t.Errorf("Expected cpuinfo with no topology data to succeed.")
	}
	node := info.Node{Id: 0}
	core := info.Core{Id: 0}
	core.Threads = append(core.Threads, 0)
	cache := info.Cache{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 1,
	}
	core.Caches = append(core.Caches, cache)
	node.Cores = append(node.Cores, core)
	// Copy over Memory from result. TODO(rjnagal): Use memory from fake.
	node.Memory = topology[0].Memory
	// Copy over HugePagesInfo from result. TODO(ohsewon): Use HugePagesInfo from fake.
	node.HugePages = topology[0].HugePages
	expected := []info.Node{node}
	if !reflect.DeepEqual(topology, expected) {
		t.Errorf("Expected topology %+v, got %+v", expected, topology)
	}
	if numCores != 1 {
		t.Errorf("Expected 1 core, found %d", numCores)
	}
}

func TestTopologyEmptyCpuinfo(t *testing.T) {
	if isSystemZ() {
		t.Skip("systemZ has no topology info")
	}
	_, _, err := GetTopology(&fakesysfs.FakeSysFs{}, "")
	if err == nil {
		t.Errorf("Expected empty cpuinfo to fail.")
	}
}

func TestTopologyCoreId(t *testing.T) {
	val, _ := getCoreIdFromCpuBus("./testdata", 0)
	if val != 0 {
		t.Errorf("Expected core 0, found %d", val)
	}

	val, _ = getCoreIdFromCpuBus("./testdata", 9999)
	if val != 8888 {
		t.Errorf("Expected core 8888, found %d", val)
	}
}

func TestTopologyNodeId(t *testing.T) {
	val, _ := getNodeIdFromCpuBus("./testdata", 0)
	if val != 0 {
		t.Errorf("Expected core 0, found %d", val)
	}

	val, _ = getNodeIdFromCpuBus("./testdata", 9999)
	if val != 1234 {
		t.Errorf("Expected core 1234 , found %d", val)
	}
}

func TestGetHugePagesInfo(t *testing.T) {
	testPath := "./testdata/hugepages/"
	expected := []info.HugePagesInfo{
		{
			NumPages: 1,
			PageSize: 1048576,
		},
		{
			NumPages: 2,
			PageSize: 2048,
		},
	}

	val, err := GetHugePagesInfo(testPath)
	if err != nil {
		t.Errorf("Failed to GetHugePagesInfo() for sample path %s: %v", testPath, err)
	}

	if !reflect.DeepEqual(expected, val) {
		t.Errorf("Expected HugePagesInfo %+v, got %+v", expected, val)
	}
}
