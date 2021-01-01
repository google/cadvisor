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

// Per-container manager.

package manager

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/cadvisor/cache/memory"
	"github.com/google/cadvisor/collector"
	"github.com/google/cadvisor/container"
	containertest "github.com/google/cadvisor/container/testing"
	info "github.com/google/cadvisor/info/v1"
	itest "github.com/google/cadvisor/info/v1/test"
	v2 "github.com/google/cadvisor/info/v2"

	"github.com/mindprince/gonvml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clock "k8s.io/utils/clock/testing"

	"github.com/google/cadvisor/accelerators"
)

const (
	containerName        = "/container"
	testLongHousekeeping = time.Second
)

// Create a containerData instance for a test.
func setupContainerData(t *testing.T, spec info.ContainerSpec) (*containerData, *containertest.MockContainerHandler, *memory.InMemoryCache, *clock.FakeClock) {
	mockHandler := containertest.NewMockContainerHandler(containerName)
	mockHandler.On("GetSpec").Return(
		spec,
		nil,
	)
	memoryCache := memory.New(60, nil)
	fakeClock := clock.NewFakeClock(time.Now())
	ret, err := newContainerData(containerName, memoryCache, mockHandler, false, &collector.GenericCollectorManager{}, 60*time.Second, true, fakeClock)
	if err != nil {
		t.Fatal(err)
	}
	return ret, mockHandler, memoryCache, fakeClock
}

// Create a containerData instance for a test and add a default GetSpec mock.
func newTestContainerData(t *testing.T) (*containerData, *containertest.MockContainerHandler, *memory.InMemoryCache, *clock.FakeClock) {
	return setupContainerData(t, itest.GenerateRandomContainerSpec(4))
}

func TestUpdateSubcontainers(t *testing.T) {
	subcontainers := []info.ContainerReference{
		{Name: "/container/ee0103"},
		{Name: "/container/abcd"},
		{Name: "/container/something"},
	}
	cd, mockHandler, _, _ := newTestContainerData(t)
	mockHandler.On("ListContainers", container.ListSelf).Return(
		subcontainers,
		nil,
	)

	err := cd.updateSubcontainers()
	if err != nil {
		t.Fatal(err)
	}

	if len(cd.info.Subcontainers) != len(subcontainers) {
		t.Errorf("Received %v subcontainers, should be %v", len(cd.info.Subcontainers), len(subcontainers))
	}

	for _, sub := range cd.info.Subcontainers {
		found := false
		for _, sub2 := range subcontainers {
			if sub.Name == sub2.Name {
				found = true
			}
		}
		if !found {
			t.Errorf("Received unknown sub container %v", sub)
		}
	}

	mockHandler.AssertExpectations(t)
}

func TestUpdateSubcontainersWithError(t *testing.T) {
	cd, mockHandler, _, _ := newTestContainerData(t)
	mockHandler.On("ListContainers", container.ListSelf).Return(
		[]info.ContainerReference{},
		fmt.Errorf("some error"),
	)
	mockHandler.On("Exists").Return(true)

	assert.NotNil(t, cd.updateSubcontainers())
	assert.Empty(t, cd.info.Subcontainers, "subcontainers should not be populated on failure")
	mockHandler.AssertExpectations(t)
}

func TestUpdateSubcontainersWithErrorOnDeadContainer(t *testing.T) {
	cd, mockHandler, _, _ := newTestContainerData(t)
	mockHandler.On("ListContainers", container.ListSelf).Return(
		[]info.ContainerReference{},
		fmt.Errorf("some error"),
	)
	mockHandler.On("Exists").Return(false)

	assert.Nil(t, cd.updateSubcontainers())
	mockHandler.AssertExpectations(t)
}

func checkNumStats(t *testing.T, memoryCache *memory.InMemoryCache, numStats int) {
	var empty time.Time
	stats, err := memoryCache.RecentStats(containerName, empty, empty, -1)
	require.Nil(t, err)
	assert.Len(t, stats, numStats)
}

func TestUpdateStats(t *testing.T) {
	statsList := itest.GenerateRandomStats(1, 4, 1*time.Second)
	stats := statsList[0]

	cd, mockHandler, memoryCache, _ := newTestContainerData(t)
	mockHandler.On("GetStats").Return(
		stats,
		nil,
	)

	err := cd.updateStats()
	if err != nil {
		t.Fatal(err)
	}

	checkNumStats(t, memoryCache, 1)
	mockHandler.AssertExpectations(t)
}

func TestUpdateSpec(t *testing.T) {
	spec := itest.GenerateRandomContainerSpec(4)
	cd, mockHandler, _, _ := newTestContainerData(t)
	mockHandler.On("GetSpec").Return(
		spec,
		nil,
	)

	err := cd.updateSpec()
	if err != nil {
		t.Fatal(err)
	}

	mockHandler.AssertExpectations(t)
}

func TestGetInfo(t *testing.T) {
	spec := itest.GenerateRandomContainerSpec(4)
	subcontainers := []info.ContainerReference{
		{Name: "/container/ee0103"},
		{Name: "/container/abcd"},
		{Name: "/container/something"},
	}
	cd, mockHandler, _, _ := setupContainerData(t, spec)
	mockHandler.On("ListContainers", container.ListSelf).Return(
		subcontainers,
		nil,
	)
	mockHandler.Aliases = []string{"a1", "a2"}

	info, err := cd.GetInfo(true)
	if err != nil {
		t.Fatal(err)
	}

	mockHandler.AssertExpectations(t)

	if len(info.Subcontainers) != len(subcontainers) {
		t.Errorf("Received %v subcontainers, should be %v", len(info.Subcontainers), len(subcontainers))
	}

	for _, sub := range info.Subcontainers {
		found := false
		for _, sub2 := range subcontainers {
			if sub.Name == sub2.Name {
				found = true
			}
		}
		if !found {
			t.Errorf("Received unknown sub container %v", sub)
		}
	}

	if !reflect.DeepEqual(spec, info.Spec) {
		t.Errorf("received wrong container spec")
	}

	if info.Name != mockHandler.Name {
		t.Errorf("received wrong container name: received %v; should be %v", info.Name, mockHandler.Name)
	}
}

func TestUpdateNvidiaStats(t *testing.T) {
	cd, _, _, _ := newTestContainerData(t)
	stats := info.ContainerStats{}

	// When there are no devices, we should not get an error and stats should not change.
	cd.nvidiaCollector = accelerators.NewNvidiaCollector([]gonvml.Device{})
	err := cd.nvidiaCollector.UpdateStats(&stats)
	assert.Nil(t, err)
	assert.Equal(t, info.ContainerStats{}, stats)

	// This is an impossible situation (there are devices but nvml is not initialized).
	// Here I am testing that the CGo gonvml library doesn't panic when passed bad
	// input and instead returns an error.
	cd.nvidiaCollector = accelerators.NewNvidiaCollector([]gonvml.Device{{}, {}})
	err = cd.nvidiaCollector.UpdateStats(&stats)
	assert.NotNil(t, err)
	assert.Equal(t, info.ContainerStats{}, stats)
}

func TestOnDemandHousekeeping(t *testing.T) {
	statsList := itest.GenerateRandomStats(1, 4, 1*time.Second)
	stats := statsList[0]

	cd, mockHandler, memoryCache, fakeClock := newTestContainerData(t)
	mockHandler.On("GetStats").Return(stats, nil)
	defer func() {
		err := cd.Stop()
		assert.NoError(t, err)
	}()

	// 0 seconds should always trigger an update
	go cd.OnDemandHousekeeping(0 * time.Second)
	cd.housekeepingTick(fakeClock.NewTimer(time.Minute).C(), testLongHousekeeping)

	fakeClock.Step(2 * time.Second)

	// This should return without requiring a housekeepingTick because stats have been updated recently enough
	cd.OnDemandHousekeeping(3 * time.Second)

	go cd.OnDemandHousekeeping(1 * time.Second)
	cd.housekeepingTick(fakeClock.NewTimer(time.Minute).C(), testLongHousekeeping)

	checkNumStats(t, memoryCache, 2)
	mockHandler.AssertExpectations(t)
}

func TestConcurrentOnDemandHousekeeping(t *testing.T) {
	statsList := itest.GenerateRandomStats(1, 4, 1*time.Second)
	stats := statsList[0]

	cd, mockHandler, memoryCache, fakeClock := newTestContainerData(t)
	mockHandler.On("GetStats").Return(stats, nil)
	defer func() {
		err := cd.Stop()
		assert.NoError(t, err)
	}()

	numConcurrentCalls := 5
	var waitForHousekeeping sync.WaitGroup
	waitForHousekeeping.Add(numConcurrentCalls)
	onDemandCache := []chan struct{}{}
	for i := 0; i < numConcurrentCalls; i++ {
		go func() {
			cd.OnDemandHousekeeping(0 * time.Second)
			waitForHousekeeping.Done()
		}()
		// Wait for work to be queued
		onDemandCache = append(onDemandCache, <-cd.onDemandChan)
	}
	// Requeue work:
	for _, ch := range onDemandCache {
		cd.onDemandChan <- ch
	}

	go cd.housekeepingTick(fakeClock.NewTimer(time.Minute).C(), testLongHousekeeping)
	// Ensure that all queued calls return with only a single call to housekeepingTick
	waitForHousekeeping.Wait()

	checkNumStats(t, memoryCache, 1)
	mockHandler.AssertExpectations(t)
}

func TestOnDemandHousekeepingReturnsAfterStopped(t *testing.T) {
	statsList := itest.GenerateRandomStats(1, 4, 1*time.Second)
	stats := statsList[0]

	cd, mockHandler, memoryCache, fakeClock := newTestContainerData(t)
	mockHandler.On("GetStats").Return(stats, nil)

	// trigger housekeeping update
	go cd.OnDemandHousekeeping(0 * time.Second)
	cd.housekeepingTick(fakeClock.NewTimer(time.Minute).C(), testLongHousekeeping)

	checkNumStats(t, memoryCache, 1)

	fakeClock.Step(2 * time.Second)

	err := cd.Stop()
	assert.NoError(t, err)
	// housekeeping tick should detect stop and not store any more metrics
	assert.False(t, cd.housekeepingTick(fakeClock.NewTimer(time.Minute).C(), testLongHousekeeping))
	fakeClock.Step(1 * time.Second)
	// on demand housekeeping should not block and return
	cd.OnDemandHousekeeping(-1 * time.Second)

	mockHandler.AssertExpectations(t)
}

func TestOnDemandHousekeepingRace(t *testing.T) {
	statsList := itest.GenerateRandomStats(1, 4, 1*time.Second)
	stats := statsList[0]

	cd, mockHandler, _, _ := newTestContainerData(t)
	mockHandler.On("GetStats").Return(stats, nil)

	wg := sync.WaitGroup{}
	wg.Add(1002)

	go func() {
		time.Sleep(10 * time.Millisecond)
		err := cd.Start()
		assert.NoError(t, err)
		wg.Done()
	}()

	go func() {
		t.Log("starting on demand goroutine")
		for i := 0; i < 1000; i++ {
			go func() {
				time.Sleep(1 * time.Microsecond)
				cd.OnDemandHousekeeping(0 * time.Millisecond)
				wg.Done()
			}()
		}
		wg.Done()
	}()
	wg.Wait()
}

var psOutput = [][]byte{
	[]byte("root       15886       2 23:51  0.1  0.0     0      0 I    00:00:00 kworker/u8:3-ev   3 -\nroot       15887       2 23:51  0.0  0.0     0      0 I<   00:00:00 kworker/1:2H      1 -\nubuntu     15888    1804 23:51  0.0  0.0  2832  10176 R+   00:00:00 ps                1 8:devices:/user.slice,6:pids:/user.slice/user-1000.slice/session-3.scope,5:blkio:/user.slice,2:cpu,cpuacct:/user.slice,1:na"),
	[]byte("root         104       2 21:34  0.0  0.0     0      0 I<   00:00:00 kthrotld          3 -\nroot         105       2 21:34  0.0  0.0     0      0 S    00:00:00 irq/41-aerdrv     0 -\nroot         107       2 21:34  0.0  0.0     0      0 I<   00:00:00 DWC Notificatio   3 -\nroot         109       2 21:34  0.0  0.0     0      0 S<   00:00:00 vchiq-slot/0      1 -\nroot         110       2 21:34  0.0  0.0     0      0 S<   00:00:00 vchiq-recy/0      3 -"),
}

func TestParseProcessList(t *testing.T) {
	for i, ps := range psOutput {
		t.Run(fmt.Sprintf("iteration %d", i), func(tt *testing.T) {
			cd := &containerData{}
			_, err := cd.parseProcessList("/", true, ps)
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
	process           *v2.ProcessInfo
	err               error
	cd                *containerData
}{
	{
		name:              "plain process with cgroup",
		line:              "ubuntu     15888    1804 23:51  0.1  0.0  2832  10176 R+   00:10:00 cadvisor            1 10:cpuset:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,9:devices:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,8:pids:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,7:memory:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,6:freezer:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,5:perf_event:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,4:blkio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,3:cpu,cpuacct:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,2:net_cls,net_prio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,1:name=systemd:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cadvisorContainer: "/docker/cadvisor",
		isHostNamespace:   true,
		process: &v2.ProcessInfo{
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
		cd: &containerData{
			info: containerInfo{ContainerReference: info.ContainerReference{Name: "/"}},
		},
	},
	{
		name:              "process with space in name and no cgroup",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 DWC Notificatio   3 -",
		cadvisorContainer: "/docker/cadvisor",
		process: &v2.ProcessInfo{
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
		cd: &containerData{
			info: containerInfo{ContainerReference: info.ContainerReference{Name: "/"}},
		},
	},
	{
		name:              "process with highly unusual name (one 2 three 4 five 6 eleven), cgroup to be ignored",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 one 2 three 4 five 6 eleven   3 10:cpuset:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,9:devices:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,8:pids:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,7:memory:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,6:freezer:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,5:perf_event:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,4:blkio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,3:cpu,cpuacct:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,2:net_cls,net_prio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,1:name=systemd:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cadvisorContainer: "/docker/cadvisor",
		isHostNamespace:   true,
		process: &v2.ProcessInfo{
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
		cd: &containerData{
			info: containerInfo{ContainerReference: info.ContainerReference{Name: "/"}},
		},
	},
	{
		name:              "wrong field count",
		line:              "ps output it is not",
		cadvisorContainer: "/docker/cadvisor",
		err:               fmt.Errorf("expected at least 13 fields, found 5: output: \"ps output it is not\""),
		cd:                &containerData{},
	},
	{
		name:              "ps running in cadvisor container should be ignored",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 ps   3 10:cpuset:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,9:devices:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,8:pids:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,7:memory:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,6:freezer:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,5:perf_event:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,4:blkio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,3:cpu,cpuacct:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,2:net_cls,net_prio:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831,1:name=systemd:/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cadvisorContainer: "/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cd: &containerData{
			info: containerInfo{ContainerReference: info.ContainerReference{Name: "/"}},
		},
	},
	{
		name: "non-root container but process belongs to the container",
		line: "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 sleep inf   3 10:cpuset:/docker/some-random-container,9:devices:/docker/some-random-container,8:pids:/docker/some-random-container,7:memory:/docker/some-random-container,6:freezer:/docker/some-random-container,5:perf_event:/docker/some-random-container,4:blkio:/docker/some-random-container,3:cpu,cpuacct:/docker/some-random-container,2:net_cls,net_prio:/docker/some-random-container,1:name=systemd:/docker/some-random-container",
		process: &v2.ProcessInfo{
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
		cd: &containerData{
			info: containerInfo{ContainerReference: info.ContainerReference{Name: "/docker/some-random-container"}},
		},
	},
	{
		name:              "non-root container and process belonging to another container",
		line:              "root         107       2 21:34  0.0  0.1     3      4 I<   00:20:00 sleep inf   3 10:cpuset:/docker/some-random-container,9:devices:/docker/some-random-container,8:pids:/docker/some-random-container,7:memory:/docker/some-random-container,6:freezer:/docker/some-random-container,5:perf_event:/docker/some-random-container,4:blkio:/docker/some-random-container,3:cpu,cpuacct:/docker/some-random-container,2:net_cls,net_prio:/docker/some-random-container,1:name=systemd:/docker/some-random-container",
		cadvisorContainer: "/docker/dd479c33249f6c3f0f1189aa88f07dad3eeb3e6fedfc71385c27ddd699994831",
		cd: &containerData{
			info: containerInfo{ContainerReference: info.ContainerReference{Name: "/docker/some-other-container"}},
		},
	},
}

func TestParsePsLine(t *testing.T) {
	for _, ps := range psLine {
		t.Run(ps.name, func(tt *testing.T) {
			process, err := ps.cd.parsePsLine(ps.line, ps.cadvisorContainer, ps.isHostNamespace)
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
			cd := &containerData{}
			path := cd.getCgroupPath(cgroup.cgroups)
			assert.Equal(t, cgroup.path, path)
		})
	}
}
