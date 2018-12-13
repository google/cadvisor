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
	"testing"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/utils/sysfs/fakesysfs"
)

var numaPciPath = "./testdata/numa/pci"
var smpPciPath = "./testdata/smp/pci"

func TestTopology(t *testing.T) {
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
	topology, numCores, err := GetTopology(sysFs, string(testcpuinfo), numaPciPath)
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
	pci0 := "0000:00:00.0"
	pci1 := "0000:00:00.1"
	pci2 := "0000:00:01.0"

	for i := 0; i < numNodes; i++ {
		node := info.Node{Id: i}
		// Copy over Memory from result. TODO(rjnagal): Use memory from fake.
		node.Memory = topology[i].Memory
		for j := 0; j < numCoresPerNode; j++ {
			core := info.Core{Id: i*numCoresPerNode + j}
			core.Caches = append(core.Caches, cache)
			for k := 0; k < numThreads; k++ {
				core.Threads = append(core.Threads, k*numCoresPerNode*numNodes+core.Id)
			}
			node.Cores = append(node.Cores, core)
		}
		if i == 0 {
			node.Pcis = append(node.Pcis, pci0)
			node.Pcis = append(node.Pcis, pci1)
		}
		if i == 1 {
			node.Pcis = append(node.Pcis, pci2)
		}
		expected_topology = append(expected_topology, node)
	}

	if !reflect.DeepEqual(topology, expected_topology) {
		t.Errorf("Expected topology %+v, got %+v", expected_topology, topology)
	}
}

func TestTopologyWithSimpleCpuinfo(t *testing.T) {
	sysFs := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Size:  32 * 1024,
		Type:  "unified",
		Level: 1,
		Cpus:  1,
	}
	sysFs.SetCacheInfo(c)
	topology, numCores, err := GetTopology(sysFs, "processor\t: 0\n", smpPciPath)
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
	pci0 := "0000:00:00.0"

	core.Caches = append(core.Caches, cache)
	node.Cores = append(node.Cores, core)
	node.Pcis = append(node.Pcis, pci0)

	// Copy over Memory from result. TODO(rjnagal): Use memory from fake.
	node.Memory = topology[0].Memory
	expected := []info.Node{node}
	if !reflect.DeepEqual(topology, expected) {
		t.Errorf("Expected topology %+v, got %+v", expected, topology)
	}
	if numCores != 1 {
		t.Errorf("Expected 1 core, found %d", numCores)
	}
}

func TestTopologyEmptyCpuinfo(t *testing.T) {
	_, _, err := GetTopology(&fakesysfs.FakeSysFs{}, "", "")
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
