// Copyright 2018 Google Inc. All Rights Reserved.
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
	"reflect"
	"testing"

	info "github.com/google/cadvisor/info/v1"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/stretchr/testify/assert"
)

func TestScanInterfaceStats(t *testing.T) {
	stats, err := scanInterfaceStats("testdata/procnetdev")
	if err != nil {
		t.Error(err)
	}

	var netdevstats = []info.InterfaceStats{
		{
			Name:      "wlp4s0",
			RxBytes:   1,
			RxPackets: 2,
			RxErrors:  3,
			RxDropped: 4,
			TxBytes:   9,
			TxPackets: 10,
			TxErrors:  11,
			TxDropped: 12,
		},
		{
			Name:      "em1",
			RxBytes:   315849,
			RxPackets: 1172,
			RxErrors:  0,
			RxDropped: 0,
			TxBytes:   315850,
			TxPackets: 1173,
			TxErrors:  0,
			TxDropped: 0,
		},
	}

	if len(stats) != len(netdevstats) {
		t.Errorf("Expected 2 net stats, got %d", len(stats))
	}

	for i, v := range netdevstats {
		if v != stats[i] {
			t.Errorf("Expected %#v, got %#v", v, stats[i])
		}
	}
}

func TestScanUDPStats(t *testing.T) {
	udpStatsFile := "testdata/procnetudp"
	r, err := os.Open(udpStatsFile)
	if err != nil {
		t.Errorf("failure opening %s: %v", udpStatsFile, err)
	}

	stats, err := scanUDPStats(r)
	if err != nil {
		t.Error(err)
	}

	var udpstats = info.UdpStat{
		Listen:   2,
		Dropped:  4,
		RxQueued: 10,
		TxQueued: 11,
	}

	if stats != udpstats {
		t.Errorf("Expected %#v, got %#v", udpstats, stats)
	}
}

// https://github.com/docker/libcontainer/blob/v2.2.1/cgroups/fs/cpuacct.go#L19
const nanosecondsInSeconds = 1000000000

// https://github.com/containerd/cgroups/pull/12
const clockTicks = 100

func TestSetCPUStats(t *testing.T) {
	perCPUUsage := make([]uint64, 31)
	for i := uint32(0); i < 31; i++ {
		perCPUUsage[i] = 8562955455524
	}
	s := &cgroups.Stats{
		CpuStats: cgroups.CpuStats{
			CpuUsage: cgroups.CpuUsage{
				PercpuUsage:       perCPUUsage,
				TotalUsage:        33802947350272,
				UsageInKernelmode: 734746 * nanosecondsInSeconds / clockTicks,
				UsageInUsermode:   2767637 * nanosecondsInSeconds / clockTicks,
			},
		},
	}
	var ret info.ContainerStats
	setCPUStats(s, &ret, true)

	expected := info.ContainerStats{
		Cpu: info.CpuStats{
			Usage: info.CpuUsage{
				PerCpu: perCPUUsage,
				User:   s.CpuStats.CpuUsage.UsageInUsermode,
				System: s.CpuStats.CpuUsage.UsageInKernelmode,
				Total:  33802947350272,
			},
		},
	}

	if !ret.Eq(&expected) {
		t.Fatalf("expected %+v == %+v", ret, expected)
	}
}

func TestSetProcessesStats(t *testing.T) {
	ret := info.ContainerStats{
		Processes: info.ProcessStats{
			ProcessCount: 1,
			FdCount:      2,
		},
	}
	s := &cgroups.Stats{
		PidsStats: cgroups.PidsStats{
			Current: 5,
			Limit:   100,
		},
	}
	setThreadsStats(s, &ret)

	expected := info.ContainerStats{

		Processes: info.ProcessStats{
			ProcessCount:   1,
			FdCount:        2,
			ThreadsCurrent: s.PidsStats.Current,
			ThreadsMax:     s.PidsStats.Limit,
		},
	}

	if expected.Processes.ProcessCount != ret.Processes.ProcessCount {
		t.Fatalf("expected ProcessCount: %d == %d", ret.Processes.ProcessCount, expected.Processes.ProcessCount)
	}
	if expected.Processes.FdCount != ret.Processes.FdCount {
		t.Fatalf("expected FdCount: %d == %d", ret.Processes.FdCount, expected.Processes.FdCount)
	}

	if expected.Processes.ThreadsCurrent != ret.Processes.ThreadsCurrent {
		t.Fatalf("expected current threads: %d == %d", ret.Processes.ThreadsCurrent, expected.Processes.ThreadsCurrent)
	}
	if expected.Processes.ThreadsMax != ret.Processes.ThreadsMax {
		t.Fatalf("expected max threads: %d == %d", ret.Processes.ThreadsMax, expected.Processes.ThreadsMax)
	}

}

func TestParseLimitsFile(t *testing.T) {
	var testData = []struct {
		limitLine string
		expected  []info.UlimitSpec
	}{
		{
			"Limit                     Soft Limit           Hard Limit           Units   \n",
			[]info.UlimitSpec{},
		},
		{
			"Max open files            8192                 8192                 files   \n",
			[]info.UlimitSpec{{Name: "max_open_files", SoftLimit: 8192, HardLimit: 8192}},
		},
		{
			"Max open files            85899345920          85899345920          files   \n",
			[]info.UlimitSpec{{Name: "max_open_files", SoftLimit: 85899345920, HardLimit: 85899345920}},
		},
		{
			"Max open files            gibberish1           8192                 files   \n",
			[]info.UlimitSpec{},
		},
		{
			"Max open files            8192                 0xbaddata            files   \n",
			[]info.UlimitSpec{},
		},
		{
			"Max stack size            8192                 8192                 files   \n",
			[]info.UlimitSpec{},
		},
	}

	for _, testItem := range testData {
		actual := processLimitsFile(testItem.limitLine)
		if reflect.DeepEqual(actual, testItem.expected) == false {
			t.Fatalf("Parsed ulimit doesn't match expected values for line: %s", testItem.limitLine)
		}
	}
}

func TestReferencedBytesStat(t *testing.T) {
	//overwrite package variables
	smapsFilePathPattern = "testdata/smaps%d"
	clearRefsFilePathPattern = "testdata/clear_refs%d"

	pids := []int{4, 6, 8}
	stat, err := referencedBytesStat(pids, 1, 3)
	assert.Nil(t, err)
	assert.Equal(t, uint64(416*1024), stat)

	clearRefsFiles := []string{
		"testdata/clear_refs4",
		"testdata/clear_refs6",
		"testdata/clear_refs8"}

	//check if clear_refs files have proper values
	assert.Equal(t, "0\n", getFileContent(t, clearRefsFiles[0]))
	assert.Equal(t, "0\n", getFileContent(t, clearRefsFiles[1]))
	assert.Equal(t, "0\n", getFileContent(t, clearRefsFiles[2]))
}

func TestReferencedBytesStatWhenNeverCleared(t *testing.T) {
	//overwrite package variables
	smapsFilePathPattern = "testdata/smaps%d"
	clearRefsFilePathPattern = "testdata/clear_refs%d"

	pids := []int{4, 6, 8}
	stat, err := referencedBytesStat(pids, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, uint64(416*1024), stat)

	clearRefsFiles := []string{
		"testdata/clear_refs4",
		"testdata/clear_refs6",
		"testdata/clear_refs8"}

	//check if clear_refs files have proper values
	assert.Equal(t, "0\n", getFileContent(t, clearRefsFiles[0]))
	assert.Equal(t, "0\n", getFileContent(t, clearRefsFiles[1]))
	assert.Equal(t, "0\n", getFileContent(t, clearRefsFiles[2]))
}

func TestReferencedBytesStatWhenResetIsNeeded(t *testing.T) {
	//overwrite package variables
	smapsFilePathPattern = "testdata/smaps%d"
	clearRefsFilePathPattern = "testdata/clear_refs%d"

	pids := []int{4, 6, 8}
	stat, err := referencedBytesStat(pids, 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, uint64(416*1024), stat)

	clearRefsFiles := []string{
		"testdata/clear_refs4",
		"testdata/clear_refs6",
		"testdata/clear_refs8"}

	//check if clear_refs files have proper values
	assert.Equal(t, "1\n", getFileContent(t, clearRefsFiles[0]))
	assert.Equal(t, "1\n", getFileContent(t, clearRefsFiles[1]))
	assert.Equal(t, "1\n", getFileContent(t, clearRefsFiles[2]))

	clearTestData(t, clearRefsFiles)
}

func TestGetReferencedKBytesWhenSmapsMissing(t *testing.T) {
	//overwrite package variable
	smapsFilePathPattern = "testdata/smaps%d"

	pids := []int{10}
	referenced, err := getReferencedKBytes(pids)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), referenced)
}

func TestClearReferencedBytesWhenClearRefsMissing(t *testing.T) {
	//overwrite package variable
	clearRefsFilePathPattern = "testdata/clear_refs%d"

	pids := []int{10}
	err := clearReferencedBytes(pids, 0, 1)
	assert.Nil(t, err)
}
