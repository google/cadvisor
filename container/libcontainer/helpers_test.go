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

package libcontainer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/opencontainers/cgroups"
	"github.com/stretchr/testify/assert"
)

var defaultCgroupSubsystems = []string{
	"systemd", "freezer", "memory", "blkio", "hugetlb", "net_cls,net_prio", "pids", "cpu,cpuacct", "devices", "cpuset", "perf_events",
}

func cgroupMountsAt(path string, subsystems []string) []cgroups.Mount {
	res := []cgroups.Mount{}
	for _, subsystem := range subsystems {
		res = append(res, cgroups.Mount{
			Root:       "/",
			Subsystems: strings.Split(subsystem, ","),
			Mountpoint: filepath.Join(path, subsystem),
		})
	}
	return res
}

func TestGetCgroupSubsystems(t *testing.T) {
	testCases := []struct {
		mounts   []cgroups.Mount
		expected map[string]string
		err      bool
	}{
		{
			mounts: []cgroups.Mount{},
			err:    true,
		},
		{
			// normal case
			mounts: cgroupMountsAt("/sys/fs/cgroup", defaultCgroupSubsystems),
			expected: map[string]string{
				"blkio":   "/sys/fs/cgroup/blkio",
				"cpu":     "/sys/fs/cgroup/cpu,cpuacct",
				"cpuacct": "/sys/fs/cgroup/cpu,cpuacct",
				"cpuset":  "/sys/fs/cgroup/cpuset",
				"devices": "/sys/fs/cgroup/devices",
				"memory":  "/sys/fs/cgroup/memory",
				"hugetlb": "/sys/fs/cgroup/hugetlb",
				"pids":    "/sys/fs/cgroup/pids",
			},
		},
		{
			// multiple croup subsystems, should ignore second one
			mounts: append(cgroupMountsAt("/sys/fs/cgroup", defaultCgroupSubsystems),
				cgroupMountsAt("/var/lib/rkt/pods/run/ccdd4e36-2d4c-49fd-8b94-4fb06133913d/stage1/rootfs/opt/stage2/flannel/rootfs/sys/fs/cgroup", defaultCgroupSubsystems)...),
			expected: map[string]string{
				"blkio":   "/sys/fs/cgroup/blkio",
				"cpu":     "/sys/fs/cgroup/cpu,cpuacct",
				"cpuacct": "/sys/fs/cgroup/cpu,cpuacct",
				"cpuset":  "/sys/fs/cgroup/cpuset",
				"devices": "/sys/fs/cgroup/devices",
				"memory":  "/sys/fs/cgroup/memory",
				"hugetlb": "/sys/fs/cgroup/hugetlb",
				"pids":    "/sys/fs/cgroup/pids",
			},
		},
		{
			// most subsystems not mounted
			mounts: cgroupMountsAt("/sys/fs/cgroup", []string{"cpu"}),
			expected: map[string]string{
				"cpu": "/sys/fs/cgroup/cpu",
			},
		},
	}

	for i, testCase := range testCases {
		subSystems, err := getCgroupSubsystemsHelper(testCase.mounts, nil)
		if testCase.err {
			if err == nil {
				t.Fatalf("[case %d] Expected error but didn't get one", i)
			}
			continue
		}
		if err != nil {
			t.Fatalf("[case %d] Expected no error, but got %v", i, err)
		}
		if !reflect.DeepEqual(testCase.expected, subSystems) {
			t.Fatalf("[case %d] Expected %v == %v", i, testCase.expected, subSystems)
		}
	}
}

func getFileContent(t *testing.T, filePath string) string {
	fileContent, err := os.ReadFile(filePath)
	assert.Nil(t, err)
	return string(fileContent)
}

func clearTestData(t *testing.T, clearRefsPaths []string) {
	for _, clearRefsPath := range clearRefsPaths {
		err := os.WriteFile(clearRefsPath, []byte("0\n"), 0o644)
		assert.Nil(t, err)
	}
}
