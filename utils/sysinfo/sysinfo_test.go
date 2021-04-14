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
	"sort"
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
		nodes, cores, err := GetNodesInfo(fakeSys)
		assert.Nil(t, err)
		assert.Equal(t, test.nodes, len(nodes))
		assert.Equal(t, test.cores, cores)

		nodesJSON, err := json.Marshal(nodes)
		assert.Nil(t, err)
		assert.JSONEq(t, test.expectedNodes, string(nodesJSON))
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
