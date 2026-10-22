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
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	info "github.com/google/cadvisor/lib/model"
	"github.com/google/cadvisor/lib/utils/sysfs"
	"github.com/google/cadvisor/lib/utils/sysfs/fakesysfs"
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
	testCases := []struct {
		cache              sysfs.CacheInfo
		nodesPaths         []string
		cpusPaths          map[string][]string
		coresThreads       map[string]string
		memTotal           string
		hugePages          []os.FileInfo
		hugePageNr         map[string]string
		physicalPackageIDs map[string]string
		nodes              int
		cores              int
		distances          []string
		expectedNodes      string
	}{
		{
			sysfs.CacheInfo{
				Id:    0,
				Size:  32 * 1024,
				Type:  "unified",
				Level: 3,
				Cpus:  2,
			},
			[]string{
				"/fakeSysfs/devices/system/node/node0",
				"/fakeSysfs/devices/system/node/node1"},
			map[string][]string{
				"/fakeSysfs/devices/system/node/node0": {
					"/fakeSysfs/devices/system/node/node0/cpu0",
					"/fakeSysfs/devices/system/node/node0/cpu1",
				},
				"/fakeSysfs/devices/system/node/node1": {
					"/fakeSysfs/devices/system/node/node0/cpu2",
					"/fakeSysfs/devices/system/node/node0/cpu3",
				},
			},
			map[string]string{
				"/fakeSysfs/devices/system/node/node0/cpu0": "0",
				"/fakeSysfs/devices/system/node/node0/cpu1": "0",
				"/fakeSysfs/devices/system/node/node0/cpu2": "1",
				"/fakeSysfs/devices/system/node/node0/cpu3": "1",
			},
			"MemTotal:       32817192 kB",
			[]os.FileInfo{
				&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
			},
			map[string]string{
				"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages": "1",
				"/fakeSysfs/devices/system/node/node1/hugepages/hugepages-2048kB/nr_hugepages": "1",
			},
			map[string]string{
				"/fakeSysfs/devices/system/node/node0/cpu0": "0",
				"/fakeSysfs/devices/system/node/node0/cpu1": "0",
				"/fakeSysfs/devices/system/node/node0/cpu2": "1",
				"/fakeSysfs/devices/system/node/node0/cpu3": "1",
			},
			2,
			4,
			[]string{
				"10 11",
				"11 10",
			},
			`
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
        "distances": [
          10,
          11
        ],
        "cores": [
          {
            "core_id": 0,
            "thread_ids": [
              0,
              1
            ],
            "caches": null,
            "uncore_caches": null,
            "socket_id": 0
          }
        ],
        "caches": [
          {
            "id": 0,
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
        "distances": [
          11,
          10
        ],
        "cores": [
          {
            "core_id": 1,
            "thread_ids": [
              2,
              3
            ],
            "caches": null,
            "uncore_caches": null,
            "socket_id": 1
          }
        ],
        "caches": [
          {
            "id": 0,
            "size": 32768,
            "type": "unified",
            "level": 3
          }
        ]
      }
    ]
    `,
		},
		{
			sysfs.CacheInfo{
				Id:    0,
				Size:  32 * 1024,
				Type:  "unified",
				Level: 3,
				Cpus:  6,
			},
			[]string{
				"/fakeSysfs/devices/system/node/node0"},
			map[string][]string{
				"/fakeSysfs/devices/system/node/node0": {
					"/fakeSysfs/devices/system/node/node0/cpu0",
					"/fakeSysfs/devices/system/node/node0/cpu1",
					"/fakeSysfs/devices/system/node/node0/cpu2",
					"/fakeSysfs/devices/system/node/node0/cpu3",
					"/fakeSysfs/devices/system/node/node0/cpu4",
					"/fakeSysfs/devices/system/node/node0/cpu5",
				},
			},
			map[string]string{
				"/fakeSysfs/devices/system/node/node0/cpu0": "0",
				"/fakeSysfs/devices/system/node/node0/cpu1": "0",
				"/fakeSysfs/devices/system/node/node0/cpu2": "1",
				"/fakeSysfs/devices/system/node/node0/cpu3": "1",
				"/fakeSysfs/devices/system/node/node0/cpu4": "2",
				"/fakeSysfs/devices/system/node/node0/cpu5": "2",
			},
			"MemTotal:       32817192 kB",
			[]os.FileInfo{
				&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
			},
			map[string]string{
				"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages": "1",
			},
			map[string]string{
				"/fakeSysfs/devices/system/node/node0/cpu0": "0",
				"/fakeSysfs/devices/system/node/node0/cpu1": "0",
				"/fakeSysfs/devices/system/node/node0/cpu2": "1",
				"/fakeSysfs/devices/system/node/node0/cpu3": "1",
				"/fakeSysfs/devices/system/node/node0/cpu4": "2",
				"/fakeSysfs/devices/system/node/node0/cpu5": "2",
			},
			1,
			6,
			[]string{
				"10",
			},
			`
	[
      {
        "node_id": 0,
        "memory": 33604804608,
        "distances": [
          10
        ],
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
            "caches": null,
            "socket_id": 0,
            "uncore_caches": null
          },
          {
            "core_id": 1,
            "thread_ids": [
              2,
              3
            ],
            "caches": null,
            "socket_id": 1,
            "uncore_caches": null
          },
          {
            "core_id": 2,
            "thread_ids": [
              4,
              5
            ],
            "caches": null,
            "socket_id": 2,
            "uncore_caches": null
          }
        ],
        "caches": [
          {
            "id": 0,
            "size": 32768,
            "type": "unified",
            "level": 3
          }
        ]
      }
    ]
    `,
		},
	}

	for _, test := range testCases {
		fakeSys := &fakesysfs.FakeSysFs{}
		fakeSys.SetCacheInfo(test.cache)
		fakeSys.SetNodesPaths(test.nodesPaths, nil)
		fakeSys.SetCPUsPaths(test.cpusPaths, nil)
		fakeSys.SetCoreThreads(test.coresThreads, nil)
		fakeSys.SetMemory(test.memTotal, nil)
		fakeSys.SetHugePages(test.hugePages, nil)
		fakeSys.SetHugePagesNr(test.hugePageNr, nil)
		fakeSys.SetPhysicalPackageIDs(test.physicalPackageIDs, nil)
		for i, node := range test.nodesPaths {
			fakeSys.SetDistances(node, test.distances[i], nil)
		}

		nodes, cores, err := GetNodesInfo(fakeSys)
		assert.Nil(t, err)
		assert.Equal(t, test.nodes, len(nodes))
		assert.Equal(t, test.cores, cores)

		nodesJSON, err := json.Marshal(nodes)
		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedNodes, string(nodesJSON))
	}
}

func TestGetNodesInfoWithMemoryOnlyNode(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	nodePath := "/fakeSysfs/devices/system/node/node1"
	fakeSys.SetNodesPaths([]string{nodePath}, nil)
	fakeSys.SetCPUsPaths(map[string][]string{nodePath: {}}, nil)
	fakeSys.SetMemory("MemTotal: 1048576 kB", nil)
	fakeSys.SetHugePages(nil, nil)
	fakeSys.SetDistances(nodePath, "10", nil)
	wantMemory := uint64(1024 * 1024 * 1024)

	nodes, cores, err := GetNodesInfo(fakeSys)

	if err != nil {
		t.Fatalf("GetNodesInfo() returned unexpected error: %v", err)
	}
	if cores != 0 {
		t.Errorf("GetNodesInfo() cores = %d, want 0", cores)
	}
	if len(nodes) != 1 {
		t.Fatalf("GetNodesInfo() returned %d nodes, want 1", len(nodes))
	}
	if nodes[0].Id != 1 {
		t.Errorf("GetNodesInfo() node ID = %d, want 1", nodes[0].Id)
	}
	if len(nodes[0].Cores) != 0 {
		t.Errorf("GetNodesInfo() memory-only node cores = %v, want none", nodes[0].Cores)
	}
	if nodes[0].Memory != wantMemory {
		t.Errorf("GetNodesInfo() memory = %d, want %d", nodes[0].Memory, wantMemory)
	}
}

func TestGetNodesInfoReturnsCPUPathError(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	nodePath := "/fakeSysfs/devices/system/node/node0"
	wantErr := errors.New("cannot read CPU paths")
	fakeSys.SetNodesPaths([]string{nodePath}, nil)
	fakeSys.SetCPUsPaths(nil, wantErr)

	nodes, cores, err := GetNodesInfo(fakeSys)

	if !errors.Is(err, wantErr) {
		t.Errorf("GetNodesInfo() error = %v, want wrapped %v", err, wantErr)
	}
	if nodes != nil {
		t.Errorf("GetNodesInfo() nodes on error = %v, want nil", nodes)
	}
	if cores != 0 {
		t.Errorf("GetNodesInfo() cores on error = %d, want 0", cores)
	}
}

func TestGetNodesInfoWithOfflineCPUs(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Id:    0,
		Size:  32 * 1024,
		Type:  "unified",
		Level: 3,
		Cpus:  1,
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
	fakeSys.SetOnlineCPUs(map[string]interface{}{
		"/fakeSysfs/devices/system/node/node0/cpu0": nil,
		"/fakeSysfs/devices/system/node/node0/cpu2": nil,
	})

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

	physicalPackageIDs := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
		"/fakeSysfs/devices/system/node/node0/cpu2": "1",
		"/fakeSysfs/devices/system/node/node0/cpu3": "1",
	}
	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, nil)

	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node0", "10 11", nil)
	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node1", "11 10", nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, 2, cores)

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
        "distances": [
          10,
          11
        ],
        "cores": [
          {
            "core_id": 0,
            "thread_ids": [
              0
            ],
            "caches": null,
            "socket_id": 0,
            "uncore_caches": null
          }
        ],
        "caches": [
          {
            "id": 0,
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
        "distances": [
          11,
          10
        ],
        "cores": [
          {
            "core_id": 1,
            "thread_ids": [
              2
            ],
            "caches": null,
            "socket_id": 1,
            "uncore_caches": null
          }
        ],
        "caches": [
          {
            "id": 0,
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

func TestGetNodesInfoWithoutCacheInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}

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

	physicalPackageIDs := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
		"/fakeSysfs/devices/system/node/node0/cpu2": "1",
		"/fakeSysfs/devices/system/node/node0/cpu3": "1",
	}
	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, nil)

	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node0", "10 11", nil)
	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node1", "11 10", nil)

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
        "distances": [
          10,
          11
        ],
        "cores": [
	  {
            "core_id": 0,
            "thread_ids": [
              0,
              1
            ],
            "caches": null,
            "uncore_caches": null,
            "socket_id": 0
          }
        ],
        "caches": null
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
        "distances": [
          11,
          10
        ],
        "cores": [
          {
            "core_id": 1,
            "thread_ids": [
              2,
              3
            ],
            "caches": null,
            "uncore_caches": null,
            "socket_id": 1
          }
        ],
        "caches": null
      }
    ]`
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetNodesInfoWithoutHugePagesInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Id:    0,
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

	physicalPackageIDs := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
		"/fakeSysfs/devices/system/node/node0/cpu2": "1",
		"/fakeSysfs/devices/system/node/node0/cpu3": "1",
	}
	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, nil)

	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node0", "10 11", nil)
	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node1", "11 10", nil)

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
        "distances": [
          10,
          11
        ],
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
                "id": 0,
                "size": 32768,
                "type": "unified",
                "level": 2
              }
            ],
            "uncore_caches": null,
            "socket_id": 0
          }
        ],
        "caches": null
      },
      {
        "node_id": 1,
        "memory": 33604804608,
        "hugepages": null,
        "distances": [
          11,
          10
        ],
        "cores": [
          {
            "core_id": 1,
            "thread_ids": [
              2,
              3
            ],
            "caches": [
              {
                "id": 0,
                "size": 32768,
                "type": "unified",
                "level": 2
              }
            ],
            "uncore_caches": null,
            "socket_id": 1
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
		Id:    0,
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

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Id < nodes[j].Id
	})

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)

	expectedNodes := `[
		{
			"node_id":0,
			"memory":0,
            "distances": null,
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
						"id": 0,
						"size":32768,
						"type":"unified",
						"level":1
					 }
				  ],
				  "socket_id": 0,
				  "uncore_caches": null
			   }
			],
			"caches":null
		 },
		 {
			"node_id":1,
			"memory":0,
            "distances": null,
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
						"id": 0,
						"size":32768,
						"type":"unified",
						"level":1
					 }
				  ],
				  "uncore_caches": null,
				  "socket_id": 1
			   }
			],
			"caches":null
		 }
	]`
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetNodesInfoWithoutNodesWhenPhysicalPackageIDMissingForOneCPU(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}

	nodesPaths := []string{}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		cpusPath: {
			cpusPath + "/cpu0",
			cpusPath + "/cpu1",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		cpusPath + "/cpu0": "0",
		cpusPath + "/cpu1": "0",
	}

	coreThreadErrors := map[string]error{
		cpusPath + "/cpu0": nil,
		cpusPath + "/cpu1": nil,
	}
	fakeSys.SetCoreThreads(coreThread, coreThreadErrors)

	physicalPackageIDs := map[string]string{
		cpusPath + "/cpu0": "0",
		cpusPath + "/cpu1": "0",
	}

	physicalPackageIDErrors := map[string]error{
		cpusPath + "/cpu0": nil,
		cpusPath + "/cpu1": os.ErrNotExist,
	}
	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, physicalPackageIDErrors)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, 2, cores)

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Id < nodes[j].Id
	})

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)

	fmt.Println(string(nodesJSON))

	expectedNodes := `[
		{
			"node_id":0,
			"memory":0,
            "distances": null,
			"hugepages":null,
			"cores":[
			   {
				  "core_id":0,
				  "thread_ids":[
					 0
				  ],
				  "caches": null,
				  "socket_id": 0,
				  "uncore_caches": null
			   }
			],
			"caches":null
		}
	]`
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetNodesInfoWithoutNodesWhenPhysicalPackageIDMissing(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}

	nodesPaths := []string{}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		cpusPath: {
			cpusPath + "/cpu0",
			cpusPath + "/cpu1",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		cpusPath + "/cpu0": "0",
		cpusPath + "/cpu1": "0",
	}

	coreThreadErrors := map[string]error{
		cpusPath + "/cpu0": nil,
		cpusPath + "/cpu1": nil,
	}
	fakeSys.SetCoreThreads(coreThread, coreThreadErrors)

	physicalPackageIDs := map[string]string{
		cpusPath + "/cpu0": "0",
		cpusPath + "/cpu1": "0",
	}

	physicalPackageIDErrors := map[string]error{
		cpusPath + "/cpu0": os.ErrNotExist,
		cpusPath + "/cpu1": os.ErrNotExist,
	}
	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, physicalPackageIDErrors)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(nodes))
	assert.Equal(t, 2, cores)
}

func TestGetNodesWhenTopologyDirMissingForOneCPU(t *testing.T) {
	/*
		Unit test for case in which:
		- there are two cpus (cpu0 and cpu1) in /sys/devices/system/node/node0/ and /sys/devices/system/cpu
		- topology directory is missing for cpu1 but it exists for cpu0
	*/
	fakeSys := &fakesysfs.FakeSysFs{}

	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
	}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
			"/fakeSysfs/devices/system/node/node0/cpu1",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
	}

	coreThreadErrors := map[string]error{
		"/fakeSysfs/devices/system/node/node0/cpu0": nil,
		"/fakeSysfs/devices/system/node/node0/cpu1": os.ErrNotExist,
	}
	fakeSys.SetCoreThreads(coreThread, coreThreadErrors)

	memTotal := "MemTotal:       32817192 kB"
	fakeSys.SetMemory(memTotal, nil)

	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages": "1",
	}
	fakeSys.SetHugePagesNr(hugePageNr, nil)

	physicalPackageIDs := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
	}

	physicalPackageIDErrors := map[string]error{
		"/fakeSysfs/devices/system/node/node0/cpu0": nil,
		"/fakeSysfs/devices/system/node/node0/cpu1": os.ErrNotExist,
	}

	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, physicalPackageIDErrors)

	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node0", "10", nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, 1, cores)

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Id < nodes[j].Id
	})

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)

	expectedNodes := `[
		{
		   "node_id":0,
		   "memory":33604804608,
           "distances" : [
             10
           ],
		   "hugepages":[
			  {
				 "page_size":2048,
				 "num_pages":1
			  }
		   ],
		   "cores":[
			  {
				 "core_id":0,
				 "thread_ids":[
					0
				 ],
				 "caches":null,
				 "socket_id":0,
				 "uncore_caches":null
			  }
		   ],
		   "caches": null
		}
	 ]`
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetNodesWhenPhysicalPackageIDMissingForOneCPU(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}

	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
	}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
			"/fakeSysfs/devices/system/node/node0/cpu1",
		},
	}
	fakeSys.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
	}

	coreThreadErrors := map[string]error{
		"/fakeSysfs/devices/system/node/node0/cpu0": nil,
		"/fakeSysfs/devices/system/node/node0/cpu1": nil,
	}
	fakeSys.SetCoreThreads(coreThread, coreThreadErrors)

	memTotal := "MemTotal:       32817192 kB"
	fakeSys.SetMemory(memTotal, nil)

	hugePages := []os.FileInfo{
		&fakesysfs.FileInfo{EntryName: "hugepages-2048kB"},
	}
	fakeSys.SetHugePages(hugePages, nil)

	hugePageNr := map[string]string{
		"/fakeSysfs/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages": "1",
	}
	fakeSys.SetHugePagesNr(hugePageNr, nil)

	physicalPackageIDs := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
	}

	physicalPackageIDErrors := map[string]error{
		"/fakeSysfs/devices/system/node/node0/cpu0": nil,
		"/fakeSysfs/devices/system/node/node0/cpu1": os.ErrNotExist,
	}

	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, physicalPackageIDErrors)

	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node0", "10", nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, 1, cores)

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Id < nodes[j].Id
	})

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)

	expectedNodes := `[
		{
		   "node_id":0,
		   "memory":33604804608,
           "distances" : [
             10
           ],
		   "hugepages":[
			  {
				 "page_size":2048,
				 "num_pages":1
			  }
		   ],
		   "cores":[
			  {
				 "core_id":0,
				 "thread_ids":[
					0
				 ],
				 "caches":null,
				 "socket_id":0,
				 "uncore_caches": null
			  }
		   ],
		   "caches": null
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

func TestGetCoresInfoWithOnlineOfflineFile(t *testing.T) {
	sysFs := &fakesysfs.FakeSysFs{}
	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
	}
	sysFs.SetNodesPaths(nodesPaths, nil)

	cpusPaths := map[string][]string{
		"/fakeSysfs/devices/system/node/node0": {
			"/fakeSysfs/devices/system/node/node0/cpu0",
			"/fakeSysfs/devices/system/node/node0/cpu1",
		},
	}
	sysFs.SetCPUsPaths(cpusPaths, nil)

	coreThread := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
	}
	sysFs.SetCoreThreads(coreThread, nil)
	sysFs.SetOnlineCPUs(map[string]interface{}{"/fakeSysfs/devices/system/node/node0/cpu0": nil})
	sysFs.SetPhysicalPackageIDs(map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
	}, nil)

	cores, err := getCoresInfo(
		sysFs,
		[]string{"/fakeSysfs/devices/system/node/node0/cpu0", "/fakeSysfs/devices/system/node/node0/cpu1"},
	)
	assert.NoError(t, err)
	expected := []info.Core{
		{
			Id:       0,
			Threads:  []int{0},
			Caches:   nil,
			SocketID: 0,
		},
	}
	assert.Equal(t, expected, cores)
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
	ignoredDevices := []string{"veth1234", "lo", "docker0", "nerdctl0"}
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
		Id:    0,
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

// perCPUCacheSysFs serves a distinct list of caches for every CPU, in sysfs
// "indexN" order, which the stock FakeSysFs (one cache for all CPUs) cannot.
type perCPUCacheSysFs struct {
	*fakesysfs.FakeSysFs
	caches map[int][]sysfs.CacheInfo
}

func (f *perCPUCacheSysFs) GetCaches(cpu int) ([]os.FileInfo, error) {
	var entries []os.FileInfo
	for i := range f.caches[cpu] {
		entries = append(entries, &fakesysfs.FileInfo{EntryName: fmt.Sprintf("index%d", i)})
	}
	return entries, nil
}

func (f *perCPUCacheSysFs) GetCacheInfo(cpu int, name string) (sysfs.CacheInfo, error) {
	idx, err := strconv.Atoi(strings.TrimPrefix(name, "index"))
	if err != nil {
		return sysfs.CacheInfo{}, err
	}
	return f.caches[cpu][idx], nil
}

// cacheTopology describes a node of identical cores: threadsPerCore threads
// per core, an L2 shared by coresPerL2 cores and an L3 shared by coresPerL3
// cores. It mirrors the sysfs index order: L1d, L1i, L2, L3.
type cacheTopology struct {
	cores          int
	threadsPerCore int
	coresPerL2     int
	coresPerL3     int
}

func (c cacheTopology) build() (*perCPUCacheSysFs, *info.Node) {
	fs := &perCPUCacheSysFs{FakeSysFs: &fakesysfs.FakeSysFs{}, caches: map[int][]sysfs.CacheInfo{}}
	node := &info.Node{}
	for core := 0; core < c.cores; core++ {
		var threads []int
		for t := 0; t < c.threadsPerCore; t++ {
			threads = append(threads, core*c.threadsPerCore+t)
		}
		node.Cores = append(node.Cores, info.Core{Id: core, Threads: threads})
		l2Group, l3Group := core/c.coresPerL2, core/c.coresPerL3
		// ids are deliberately not contiguous or equal to the group number
		perCore := []sysfs.CacheInfo{
			{Id: core * 2, Size: 32 << 10, Type: "Data", Level: 1, Cpus: c.threadsPerCore},
			{Id: core * 2, Size: 64 << 10, Type: "Instruction", Level: 1, Cpus: c.threadsPerCore},
			{Id: 100 + l2Group*3, Size: 4 << 20, Type: "Unified", Level: 2, Cpus: c.coresPerL2 * c.threadsPerCore},
			{Id: 200 + l3Group, Size: 240 << 20, Type: "Unified", Level: 3, Cpus: c.coresPerL3 * c.threadsPerCore},
		}
		for _, thread := range threads {
			fs.caches[thread] = perCore
		}
	}
	return fs, node
}

func cacheOf(id int, size uint64, typ string, level int) info.Cache {
	return info.Cache{Id: id, Size: size, Type: typ, Level: level}
}

func TestAddCacheInfoSharedL2(t *testing.T) {
	l1 := func(core int) []info.Cache {
		return []info.Cache{
			cacheOf(core*2, 32<<10, "Data", 1),
			cacheOf(core*2, 64<<10, "Instruction", 1),
		}
	}
	l2 := func(group int) info.Cache { return cacheOf(100+group*3, 4<<20, "Unified", 2) }
	l3 := func(group int) info.Cache { return cacheOf(200+group, 240<<20, "Unified", 3) }

	testCases := []struct {
		name string
		topo cacheTopology
		// expectations per core index (core -> caches); node caches for the whole node
		wantCaches  func(core int) []info.Cache
		wantUncore  func(core int) []info.Cache
		wantNodeCch []info.Cache
	}{
		{
			// non-SMT cores in pairs sharing an L2, L3 shared by part of the node
			name:       "non-SMT cores sharing a 2-core L2 below a partial L3",
			topo:       cacheTopology{cores: 8, threadsPerCore: 1, coresPerL2: 2, coresPerL3: 4},
			wantCaches: l1,
			wantUncore: func(core int) []info.Cache { return []info.Cache{l2(core / 2), l3(core / 4)} },
		},
		{
			// Xeon 6 with E-cores: 4-core L2 clusters, L3 spans the whole node
			name:        "non-SMT cores sharing a 4-core L2 below a node-wide L3",
			topo:        cacheTopology{cores: 8, threadsPerCore: 1, coresPerL2: 4, coresPerL3: 8},
			wantCaches:  l1,
			wantUncore:  func(core int) []info.Cache { return []info.Cache{l2(core / 4)} },
			wantNodeCch: []info.Cache{l3(0)},
		},
		{
			// L2 shared by every core of the node behaves like a node-wide L3
			name:        "L2 shared by the whole node",
			topo:        cacheTopology{cores: 4, threadsPerCore: 1, coresPerL2: 4, coresPerL3: 4},
			wantCaches:  l1,
			wantUncore:  func(core int) []info.Cache { return nil },
			wantNodeCch: []info.Cache{l2(0), l3(0)},
		},
		{
			// regression: L2 private to an SMT core stays a core cache
			name: "L2 private to an SMT core",
			topo: cacheTopology{cores: 4, threadsPerCore: 2, coresPerL2: 1, coresPerL3: 2},
			wantCaches: func(core int) []info.Cache {
				// threads of one core share the caches, so ids follow the core
				return append(l1(core), l2(core))
			},
			wantUncore: func(core int) []info.Cache { return []info.Cache{l3(core / 2)} },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs, node := tc.topo.build()
			require.NoError(t, addCacheInfo(fs, node))

			for core := range node.Cores {
				assert.Equal(t, tc.wantCaches(core), node.Cores[core].Caches, "Caches of core %d", core)
				assert.Equal(t, tc.wantUncore(core), node.Cores[core].UncoreCaches, "UncoreCaches of core %d", core)
			}
			assert.Equal(t, tc.wantNodeCch, node.Caches, "node caches")
		})
	}
}

// newHybridTopology returns the layout of a Core i7-12700K: 8 SMT cores with a
// private L2, and 4 cores without SMT that share one L2, all behind one L3 that
// covers the 20 threads of the node.
func newHybridTopology() (*perCPUCacheSysFs, *info.Node) {
	fs := &perCPUCacheSysFs{FakeSysFs: &fakesysfs.FakeSysFs{}, caches: map[int][]sysfs.CacheInfo{}}
	node := &info.Node{}
	l3 := sysfs.CacheInfo{Id: 0, Size: 25600 << 10, Type: "Unified", Level: 3, Cpus: 20}

	for core := 0; core < 8; core++ { // threads 2*core and 2*core+1
		threads := []int{2 * core, 2*core + 1}
		node.Cores = append(node.Cores, info.Core{Id: core * 4, Threads: threads})
		perCore := []sysfs.CacheInfo{
			{Id: core * 4, Size: 48 << 10, Type: "Data", Level: 1, Cpus: 2},
			{Id: core * 4, Size: 64 << 10, Type: "Instruction", Level: 1, Cpus: 2},
			{Id: core, Size: 1280 << 10, Type: "Unified", Level: 2, Cpus: 2},
			l3,
		}
		for _, thread := range threads {
			fs.caches[thread] = perCore
		}
	}
	for core := 0; core < 4; core++ { // threads 16 to 19, one per core
		thread := 16 + core
		node.Cores = append(node.Cores, info.Core{Id: 36 + core, Threads: []int{thread}})
		fs.caches[thread] = []sysfs.CacheInfo{
			{Id: 36 + core, Size: 32 << 10, Type: "Data", Level: 1, Cpus: 1},
			{Id: 36 + core, Size: 64 << 10, Type: "Instruction", Level: 1, Cpus: 1},
			{Id: 9, Size: 2048 << 10, Type: "Unified", Level: 2, Cpus: 4},
			l3,
		}
	}
	return fs, node
}

// TestAddCacheInfoHybridCores checks the L2 of a hybrid part. Only the cores
// that share the L2 may list it in UncoreCaches.
func TestAddCacheInfoHybridCores(t *testing.T) {
	fs, node := newHybridTopology()

	require.NoError(t, addCacheInfo(fs, node))

	level2 := func(caches []info.Cache) []info.Cache {
		var out []info.Cache
		for _, c := range caches {
			if c.Level == 2 {
				out = append(out, c)
			}
		}
		return out
	}
	for core := 0; core < 8; core++ {
		assert.Equal(t, []info.Cache{cacheOf(core, 1280<<10, "Unified", 2)}, level2(node.Cores[core].Caches), "private L2 of SMT core %d", core)
		assert.Empty(t, level2(node.Cores[core].UncoreCaches), "SMT core %d must not list an uncore L2", core)
	}
	for core := 8; core < 12; core++ {
		assert.Empty(t, level2(node.Cores[core].Caches), "core %d shares its L2", core)
		assert.Equal(t, []info.Cache{cacheOf(9, 2048<<10, "Unified", 2)}, level2(node.Cores[core].UncoreCaches), "shared L2 of core %d", core)
	}
}
func TestGetNetworkStats(t *testing.T) {
	expectedStats := info.InterfaceStats{
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
	if expectedStats != netStats {
		t.Errorf("expected to get stats %+v, got %+v", expectedStats, netStats)
	}
}

func TestGetSocketFromCPU(t *testing.T) {
	topology := []info.Node{
		{
			Id:        0,
			Memory:    0,
			HugePages: nil,
			Cores: []info.Core{
				{
					Id:       0,
					Threads:  []int{0, 1},
					Caches:   nil,
					SocketID: 0,
				},
				{
					Id:       1,
					Threads:  []int{2, 3},
					Caches:   nil,
					SocketID: 0,
				},
			},
			Caches: nil,
		},
		{
			Id:        1,
			Memory:    0,
			HugePages: nil,
			Cores: []info.Core{
				{
					Id:       0,
					Threads:  []int{4, 5},
					Caches:   nil,
					SocketID: 1,
				},
				{
					Id:       1,
					Threads:  []int{6, 7},
					Caches:   nil,
					SocketID: 1,
				},
			},
			Caches: nil,
		},
	}
	socket := GetSocketFromCPU(topology, 6)
	assert.Equal(t, socket, 1)

	// Check if return "-1" when there is no data about passed CPU.
	socket = GetSocketFromCPU(topology, 8)
	assert.Equal(t, socket, -1)
}

func TestGetOnlineCPUs(t *testing.T) {
	topology := []info.Node{
		{
			Id:        0,
			Memory:    0,
			HugePages: nil,
			Cores: []info.Core{
				{
					Id:       0,
					Threads:  []int{0, 1},
					Caches:   nil,
					SocketID: 0,
				},
				{
					Id:       1,
					Threads:  []int{2, 3},
					Caches:   nil,
					SocketID: 0,
				},
			},
			Caches: nil,
		},
		{
			Id:        1,
			Memory:    0,
			HugePages: nil,
			Cores: []info.Core{
				{
					Id:       0,
					Threads:  []int{4, 5},
					Caches:   nil,
					SocketID: 1,
				},
				{
					Id:       1,
					Threads:  []int{6, 7},
					Caches:   nil,
					SocketID: 1,
				},
			},
			Caches: nil,
		},
	}
	onlineCPUs := GetOnlineCPUs(topology)
	assert.Equal(t, onlineCPUs, []int{0, 1, 2, 3, 4, 5, 6, 7})
}

func TestGetNodesInfoWithUncoreCacheInfo(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	c := sysfs.CacheInfo{
		Id:    0,
		Size:  32 * 1024,
		Type:  "unified",
		Level: 3,
		Cpus:  8,
	}
	fakeSys.SetCacheInfo(c)

	nodesPaths := []string{
		"/fakeSysfs/devices/system/node/node0",
		"/fakeSysfs/devices/system/node/node1",
	}
	fakeSys.SetNodesPaths(nodesPaths, nil)

	memTotal := "MemTotal:       32817192 kB"
	fakeSys.SetMemory(memTotal, nil)

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
	physicalPackageIDs := map[string]string{
		"/fakeSysfs/devices/system/node/node0/cpu0": "0",
		"/fakeSysfs/devices/system/node/node0/cpu1": "0",
		"/fakeSysfs/devices/system/node/node0/cpu2": "1",
		"/fakeSysfs/devices/system/node/node0/cpu3": "1",
	}
	fakeSys.SetPhysicalPackageIDs(physicalPackageIDs, nil)

	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node0", "10 11", nil)
	fakeSys.SetDistances("/fakeSysfs/devices/system/node/node1", "11 10", nil)

	nodes, cores, err := GetNodesInfo(fakeSys)
	assert.Nil(t, err)
	fmt.Println(err)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, 4, cores)

	nodesJSON, err := json.Marshal(nodes)
	assert.Nil(t, err)
	expectedNodes := `
	[
      {
        "node_id": 0,
        "memory": 33604804608,
        "distances" : [
          10,
          11
        ],
        "hugepages": null,
        "cores": [
          {
            "core_id": 0,
            "thread_ids": [
              0,
              1
            ],
            "caches": null,
            "uncore_caches": [
                {
                  "id": 0,
                  "size": 32768,
                  "type": "unified",
                  "level": 3
                }
            ],
            "socket_id": 0
          }
        ],
        "caches": null
      },
      {
        "node_id": 1,
        "memory": 33604804608,
        "distances" : [
          11,
          10
        ],
        "hugepages": null,
        "cores": [
          {
            "core_id": 1,
            "thread_ids": [
              2,
              3
            ],
            "caches": null,
            "uncore_caches": [
                {
                  "id": 0,
                  "size": 32768,
                  "type": "unified",
                  "level": 3
                }
            ],
            "socket_id": 1
          }
        ],
        "caches": null
      }
    ]`
	assert.JSONEq(t, expectedNodes, string(nodesJSON))
}

func TestGetDistances(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	node := "/fakeSysfs/devices/system/node/node0"
	fakeSys.SetDistances(node, "10 11", nil)

	distances, err := getDistances(fakeSys, node)
	assert.Nil(t, err)
	assert.Len(t, distances, 2)
	assert.Equal(t, uint64(10), distances[0])
	assert.Equal(t, uint64(11), distances[1])
}

func TestGetDistancesMissingDistances(t *testing.T) {
	fakeSys := &fakesysfs.FakeSysFs{}
	node := "/fakeSysfs/devices/system/node/node0"
	fakeSys.SetDistances(node, "10 11", fmt.Errorf("no distances file"))

	distances, err := getDistances(fakeSys, node)
	assert.Nil(t, err)
	assert.Len(t, distances, 0)
}

// The L3 of a hybrid part covers every thread of the node, so it is a node level
// cache even though the SMT cores and the other cores have different thread counts.
func TestAddCacheInfoHybridCoresNodeWideL3(t *testing.T) {
	fs, node := newHybridTopology()

	require.NoError(t, addCacheInfo(fs, node))

	assert.Equal(t, []info.Cache{cacheOf(0, 25600<<10, "Unified", 3)}, node.Caches, "node caches")
	for i, core := range node.Cores {
		for _, c := range core.UncoreCaches {
			assert.NotEqual(t, 3, c.Level, "core %d must not list the node wide L3 as an uncore cache", i)
		}
	}
}
