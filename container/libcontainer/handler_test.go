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
	"testing"

	info "github.com/google/cadvisor/info/v1"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/system"
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

	stats, err := scanUdpStats(r)
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

var clockTicks = uint64(system.GetClockTicks())

func TestMorePossibleCPUs(t *testing.T) {
	realNumCPUs := uint32(8)
	numCpusFunc = func() (uint32, error) {
		return realNumCPUs, nil
	}
	possibleCPUs := uint32(31)

	perCpuUsage := make([]uint64, possibleCPUs)
	for i := uint32(0); i < realNumCPUs; i++ {
		perCpuUsage[i] = 8562955455524
	}

	s := &cgroups.Stats{
		CpuStats: cgroups.CpuStats{
			CpuUsage: cgroups.CpuUsage{
				PercpuUsage:       perCpuUsage,
				TotalUsage:        33802947350272,
				UsageInKernelmode: 734746 * nanosecondsInSeconds / clockTicks,
				UsageInUsermode:   2767637 * nanosecondsInSeconds / clockTicks,
			},
		},
	}
	var ret info.ContainerStats
	setCpuStats(s, &ret, true)

	expected := info.ContainerStats{
		Cpu: info.CpuStats{
			Usage: info.CpuUsage{
				PerCpu: perCpuUsage[0:realNumCPUs],
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
