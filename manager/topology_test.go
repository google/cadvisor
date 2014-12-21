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

package manager

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/google/cadvisor/info"
)

func TestTopology(t *testing.T) {
	testfile := "./testdata/cpuinfo"
	testcpuinfo, err := ioutil.ReadFile(testfile)
	if err != nil {
		t.Fatalf("unable to read input test file %s", testfile)
	}
	topology, numCores, err := getTopology(string(testcpuinfo))
	if err != nil {
		t.Errorf("failed to get topology for sample cpuinfo %s", string(testcpuinfo))
	}

	if numCores != 12 {
		t.Errorf("Expected 12 cores, found %d", numCores)
	}
	expected_topology := []info.Node{}
	numNodes := 2
	numCoresPerNode := 3
	numThreads := 2
	for i := 0; i < numNodes; i++ {
		node := info.Node{Id: i}
		// Copy over Memory from result. TODO(rjnagal): Use memory from fake.
		node.Memory = topology[i].Memory
		for j := 0; j < numCoresPerNode; j++ {
			core := info.Core{Id: i*numCoresPerNode + j}
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
	topology, numCores, err := getTopology("processor\t: 0\n")
	if err != nil {
		t.Errorf("Expected cpuinfo with no topology data to succeed.")
	}
	node := info.Node{Id: 0}
	core := info.Core{Id: 0}
	core.Threads = append(core.Threads, 0)
	node.Cores = append(node.Cores, core)
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
	_, _, err := getTopology("")
	if err == nil {
		t.Errorf("Expected empty cpuinfo to fail.")
	}
}
