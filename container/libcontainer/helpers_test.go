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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/opencontainers/runc/libcontainer/cgroups"
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
	ourSubsystems := []string{"cpu,cpuacct", "devices", "memory", "hugetlb", "cpuset", "blkio", "pids"}

	testCases := []struct {
		mounts   []cgroups.Mount
		expected CgroupSubsystems
		err      bool
	}{
		{
			mounts: []cgroups.Mount{},
			err:    true,
		},
		{
			// normal case
			mounts: cgroupMountsAt("/sys/fs/cgroup", defaultCgroupSubsystems),
			expected: CgroupSubsystems{
				MountPoints: map[string]string{
					"blkio":   "/sys/fs/cgroup/blkio",
					"cpu":     "/sys/fs/cgroup/cpu,cpuacct",
					"cpuacct": "/sys/fs/cgroup/cpu,cpuacct",
					"cpuset":  "/sys/fs/cgroup/cpuset",
					"devices": "/sys/fs/cgroup/devices",
					"memory":  "/sys/fs/cgroup/memory",
					"hugetlb": "/sys/fs/cgroup/hugetlb",
					"pids":    "/sys/fs/cgroup/pids",
				},
				Mounts: cgroupMountsAt("/sys/fs/cgroup", ourSubsystems),
			},
		},
		{
			// multiple croup subsystems, should ignore second one
			mounts: append(cgroupMountsAt("/sys/fs/cgroup", defaultCgroupSubsystems),
				cgroupMountsAt("/var/lib/rkt/pods/run/ccdd4e36-2d4c-49fd-8b94-4fb06133913d/stage1/rootfs/opt/stage2/flannel/rootfs/sys/fs/cgroup", defaultCgroupSubsystems)...),
			expected: CgroupSubsystems{
				MountPoints: map[string]string{
					"blkio":   "/sys/fs/cgroup/blkio",
					"cpu":     "/sys/fs/cgroup/cpu,cpuacct",
					"cpuacct": "/sys/fs/cgroup/cpu,cpuacct",
					"cpuset":  "/sys/fs/cgroup/cpuset",
					"devices": "/sys/fs/cgroup/devices",
					"memory":  "/sys/fs/cgroup/memory",
					"hugetlb": "/sys/fs/cgroup/hugetlb",
					"pids":    "/sys/fs/cgroup/pids",
				},
				Mounts: cgroupMountsAt("/sys/fs/cgroup", ourSubsystems),
			},
		},
		{
			// most subsystems not mounted
			mounts: cgroupMountsAt("/sys/fs/cgroup", []string{"cpu"}),
			expected: CgroupSubsystems{
				MountPoints: map[string]string{
					"cpu": "/sys/fs/cgroup/cpu",
				},
				Mounts: cgroupMountsAt("/sys/fs/cgroup", []string{"cpu"}),
			},
		},
	}

	for i, testCase := range testCases {
		subSystems, err := getCgroupSubsystemsHelper(testCase.mounts, map[string]struct{}{})
		if testCase.err {
			if err == nil {
				t.Fatalf("[case %d] Expected error but didn't get one", i)
			}
			continue
		}
		if err != nil {
			t.Fatalf("[case %d] Expected no error, but got %v", i, err)
		}
		assertCgroupSubsystemsEqual(t, testCase.expected, subSystems, fmt.Sprintf("[case %d]", i))
	}
}

func assertCgroupSubsystemsEqual(t *testing.T, expected, actual CgroupSubsystems, message string) {
	if !reflect.DeepEqual(expected.MountPoints, actual.MountPoints) {
		t.Fatalf("%s Expected %v == %v", message, expected.MountPoints, actual.MountPoints)
	}

	sort.Slice(expected.Mounts, func(i, j int) bool {
		return expected.Mounts[i].Mountpoint < expected.Mounts[j].Mountpoint
	})
	sort.Slice(actual.Mounts, func(i, j int) bool {
		return actual.Mounts[i].Mountpoint < actual.Mounts[j].Mountpoint
	})
	if !reflect.DeepEqual(expected.Mounts, actual.Mounts) {
		t.Fatalf("%s Expected %v == %v", message, expected.Mounts, actual.Mounts)
	}
}

func getFileContent(t *testing.T, filePath string) string {
	fileContent, err := ioutil.ReadFile(filePath)
	assert.Nil(t, err)
	return string(fileContent)
}

func clearTestData(t *testing.T, clearRefsPaths []string) {
	for _, clearRefsPath := range clearRefsPaths {
		err := ioutil.WriteFile(clearRefsPath, []byte("0\n"), 0644)
		assert.Nil(t, err)
	}
}
