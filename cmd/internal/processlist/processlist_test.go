// Copyright 2024 Google Inc. All Rights Reserved.
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

package processlist

import (
	"fmt"
	"testing"

	model "github.com/google/cadvisor/lib/model"

	"github.com/stretchr/testify/assert"
)

var psOutput = [][]byte{
	[]byte("root       15886       2 23:51  0.1  0.0     0      0 I    00:00:00 kworker/u8:3-ev   3 -\nroot       15887       2 23:51  0.0  0.0     0      0 I<   00:00:00 kworker/1:2H      1 -\nubuntu     15888    1804 23:51  0.0  0.0  2832  10176 R+   00:00:00 ps                1 8:devices:/user.slice,6:pids:/user.slice/user-1000.slice/session-3.scope,5:blkio:/user.slice,2:cpu,cpuacct:/user.slice,1:na"),
	[]byte("root         104       2 21:34  0.0  0.0     0      0 I<   00:00:00 kthrotld          3 -\nroot         105       2 21:34  0.0  0.0     0      0 S    00:00:00 irq/41-aerdrv     0 -\nroot         107       2 21:34  0.0  0.0     0      0 I<   00:00:00 DWC Notificatio   3 -\nroot         109       2 21:34  0.0  0.0     0      0 S<   00:00:00 vchiq-slot/0      1 -\nroot         110       2 21:34  0.0  0.0     0      0 S<   00:00:00 vchiq-recy/0      3 -"),
}

func TestParseProcessList(t *testing.T) {
	for i, ps := range psOutput {
		t.Run(fmt.Sprintf("iteration %d", i), func(tt *testing.T) {
			_, err := parseProcessList(ps, "", false, "/", true)
			// checking *only* parsing errors - otherwise /proc would have to be emulated.
			assert.NoError(tt, err)
		})
	}
}

var psLine = []struct {
	name              string
	line              string
	cadvisorContainer string
	isHostNamespace   bool
	process           *model.ProcessInfo
	err               error
	containerName     string
}{
	{
		name:              "plain process with cgroup",
		line:              "ubuntu     15888    1804 23:51  0.1  0.0  2832  10176 R+   00:10:00 cadvisor            1 10:cpuset:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,9:devices:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,8:pids:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,7:memory:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,6:freezer:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,5:perf_event:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,4:blkio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,3:cpu,cpuacct:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,2:net_cls,net_prio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,1:name=systemd:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cadvisorContainer: "/docker/cadvisor",
		isHostNamespace:   true,
		process: &model.ProcessInfo{
			User:          "ubuntu",
			Pid:           15888,
			Ppid:          1804,
			StartTime:     "23:51",
			PercentCpu:    0.1,
			PercentMemory: 0.0,
			RSS:           2899968,
			VirtualSize:   10420224,
			Status:        "R+",
			RunningTime:   "00:10:00",
			CgroupPath:    "/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
			Cmd:           "cadvisor",
			Psr:           1,
		},
		containerName: "/",
	},
	{
		name:              "process with space in name and no cgroup",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 DWC Notificatio   3 -",
		cadvisorContainer: "/docker/cadvisor",
		process: &model.ProcessInfo{
			User:          "root",
			Pid:           107,
			Ppid:          2,
			StartTime:     "21:34",
			PercentCpu:    0.0,
			PercentMemory: 0.1,
			RSS:           3072,
			VirtualSize:   4096,
			Status:        "I<",
			RunningTime:   "00:20:00",
			CgroupPath:    "/",
			Cmd:           "DWC Notificatio",
			Psr:           3,
		},
		containerName: "/",
	},
	{
		name:              "process with highly unusual name (one 2 three 4 five 6 eleven), cgroup to be ignored",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 one 2 three 4 five 6 eleven   3 10:cpuset:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,9:devices:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,8:pids:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,7:memory:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,6:freezer:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,5:perf_event:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,4:blkio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,3:cpu,cpuacct:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,2:net_cls,net_prio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,1:name=systemd:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cadvisorContainer: "/docker/cadvisor",
		isHostNamespace:   true,
		process: &model.ProcessInfo{
			User:          "root",
			Pid:           107,
			Ppid:          2,
			StartTime:     "21:34",
			PercentCpu:    0.0,
			PercentMemory: 0.1,
			RSS:           3072,
			VirtualSize:   4096,
			Status:        "I<",
			RunningTime:   "00:20:00",
			Cmd:           "one 2 three 4 five 6 eleven",
			Psr:           3,
			CgroupPath:    "/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		},
		containerName: "/",
	},
	{
		name:              "wrong field count",
		line:              "ps output it is not",
		cadvisorContainer: "/docker/cadvisor",
		err:               fmt.Errorf("expected at least 13 fields, found 5: output: \"ps output it is not\""),
		containerName:     "",
	},
	{
		name:              "ps running in cadvisor container should be ignored",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 ps   3 10:cpuset:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,9:devices:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,8:pids:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,7:memory:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,6:freezer:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,5:perf_event:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,4:blkio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,3:cpu,cpuacct:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,2:net_cls,net_prio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,1:name=systemd:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cadvisorContainer: "/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		containerName:     "/",
	},
	{
		name: "non-root container but process belongs to the container",
		line: "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 sleep inf   3 10:cpuset:/docker/some-random-container,9:devices:/docker/some-random-container,8:pids:/docker/some-random-container,7:memory:/docker/some-random-container,6:freezer:/docker/some-random-container,5:perf_event:/docker/some-random-container,4:blkio:/docker/some-random-container,3:cpu,cpuacct:/docker/some-random-container,2:net_cls,net_prio:/docker/some-random-container,1:name=systemd:/docker/some-random-container",
		process: &model.ProcessInfo{
			User:          "root",
			Pid:           107,
			Ppid:          2,
			StartTime:     "21:34",
			PercentCpu:    0.0,
			PercentMemory: 0.1,
			RSS:           3072,
			VirtualSize:   4096,
			Status:        "I<",
			RunningTime:   "00:20:00",
			Cmd:           "sleep inf",
			Psr:           3,
		},
		cadvisorContainer: "/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		containerName:     "/docker/some-random-container",
	},
	{
		name:              "non-root container and process belonging to another container",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 sleep inf   3 10:cpuset:/docker/some-random-container,9:devices:/docker/some-random-container,8:pids:/docker/some-random-container,7:memory:/docker/some-random-container,6:freezer:/docker/some-random-container,5:perf_event:/docker/some-random-container,4:blkio:/docker/some-random-container,3:cpu,cpuacct:/docker/some-random-container,2:net_cls,net_prio:/docker/some-random-container,1:name=systemd:/docker/some-random-container",
		cadvisorContainer: "/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		containerName:     "/docker/some-other-container",
	},
}

func TestParsePsLine(t *testing.T) {
	for _, ps := range psLine {
		t.Run(ps.name, func(tt *testing.T) {
			process, err := parsePsLine(ps.line, ps.containerName, ps.containerName == "/", ps.cadvisorContainer, ps.isHostNamespace)
			assert.Equal(tt, ps.err, err)
			assert.EqualValues(tt, ps.process, process)
		})
	}
}

var cgroupCases = []struct {
	name    string
	cgroups string
	path    string
}{
	{
		name:    "no cgroup",
		cgroups: "-",
		path:    "/",
	},
	{
		name:    "random and meaningless string",
		cgroups: "/this/is/a/path/to/some.file",
		path:    "/",
	},
	{
		name:    "0::-type cgroup",
		cgroups: "0::/docker/some-cgroup",
		path:    "/docker/some-cgroup",
	},
	{
		name:    "memory cgroup",
		cgroups: "4:memory:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6,2:net_cls:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
		path:    "/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
	},
	{
		name:    "cpu,cpuacct cgroup",
		cgroups: "4:cpu,cpuacct:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6,2:net_cls:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
		path:    "/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
	},
	{
		name:    "cpu cgroup",
		cgroups: "4:cpu:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6,2:net_cls:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
		path:    "/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
	},
	{
		name:    "cpuacct cgroup",
		cgroups: "4:cpuacct:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6,2:net_cls:/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
		path:    "/docker/09c89cd48b3597db904ab8e6920fef2cbf93588d037d9613ce362e25188f8ec6",
	},
}

func TestGetCgroupPath(t *testing.T) {
	for _, cgroup := range cgroupCases {
		t.Run(cgroup.name, func(tt *testing.T) {
			path := getCgroupPath(cgroup.cgroups)
			assert.Equal(t, cgroup.path, path)
		})
	}
}
