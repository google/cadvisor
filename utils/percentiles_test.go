// Copyright 2015 Google Inc. All Rights Reserved.
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

package utils

import (
	"testing"
	"time"

	"github.com/google/cadvisor/info"
)

const Nanosecond = 1000000000

func Test90Percentile(t *testing.T) {
	N := 100
	stats := make(uint64Slice, 0, N)
	for i := N; i > 0; i-- {
		stats = append(stats, uint64(i))
	}
	p := stats.Get90Percentile()
	if p != 90 {
		t.Errorf("90th percentile is %d, should be 90.", p)
	}
	// 90p should be between 94 and 95. Promoted to 95.
	N = 105
	for i := 101; i <= N; i++ {
		stats = append(stats, uint64(i))
	}
	p = stats.Get90Percentile()
	if p != 95 {
		t.Errorf("90th percentile is %d, should be 95.", p)
	}
}

func TestMean(t *testing.T) {
	var i, N uint64
	N = 100
	mean := Mean{count: 0, Mean: 0}
	for i = 1; i < N; i++ {
		mean.Add(i)
	}
	if mean.Mean != 50.0 {
		t.Errorf("Mean is %f, should be 50.0", mean.Mean)
	}
}

func TestAggregates(t *testing.T) {
	N := uint64(100)
	var i uint64
	ct := time.Now()
	stats := make([]*info.ContainerStats, 0, N)
	for i = 1; i < N; i++ {
		s := &info.ContainerStats{
			Cpu:       info.CpuStats{},
			Timestamp: ct.Add(time.Duration(i) * time.Second),
			Memory: info.MemoryStats{
				// Memory grows by a KB every second.
				WorkingSet: i * 1024,
			},
		}
		// cpu rate is 1 s/s
		s.Cpu.Usage.Total = i * Nanosecond
		stats = append(stats, s)
	}
	cpu, mem := GetPercentiles(stats)
	// Cpu mean, max, and 90p should all be 1000 ms/s.
	cpuExpected := Percentiles{
		Mean:   1000,
		Max:    1000,
		Ninety: 1000,
	}
	if cpu != cpuExpected {
		t.Errorf("cpu stats are %+v. Expected %+v", cpu, cpuExpected)
	}
	memExpected := Percentiles{
		Mean:   50 * 1024,
		Max:    99 * 1024,
		Ninety: 90 * 1024,
	}
	if mem != memExpected {
		t.Errorf("memory stats are mean %+v. Expected %+v", mem, memExpected)
	}
}
func TestSamplesCloseInTimeIgnored(t *testing.T) {
	N := uint64(100)
	var i uint64
	ct := time.Now()
	stats := make([]*info.ContainerStats, 0, N*2)
	for i = 1; i < N; i++ {
		s1 := &info.ContainerStats{
			Cpu:       info.CpuStats{},
			Timestamp: ct.Add(time.Duration(i) * time.Second),
			Memory: info.MemoryStats{
				// Memory grows by a KB every second.
				WorkingSet: i * 1024,
			},
		}
		// cpu rate is 1 s/s
		s1.Cpu.Usage.Total = i * Nanosecond
		stats = append(stats, s1)

		// Add another dummy sample too close in time to the last one.
		s2 := &info.ContainerStats{
			Cpu: info.CpuStats{},
			// Add extra millisecond.
			Timestamp: ct.Add(time.Duration(i) * time.Second).Add(time.Duration(1) * time.Millisecond),
			Memory: info.MemoryStats{
				WorkingSet: i * 1024 * 1024,
			},
		}
		s2.Cpu.Usage.Total = i * 100 * Nanosecond
		stats = append(stats, s2)
	}
	cpu, mem := GetPercentiles(stats)
	// Cpu mean, max, and 90p should all be 1000 ms/s. All high-value samples are discarded.
	cpuExpected := Percentiles{
		Mean:   1000,
		Max:    1000,
		Ninety: 1000,
	}
	if cpu != cpuExpected {
		t.Errorf("cpu stats are %+v. Expected %+v", cpu, cpuExpected)
	}
	memExpected := Percentiles{
		Mean:   50 * 1024,
		Max:    99 * 1024,
		Ninety: 90 * 1024,
	}
	if mem != memExpected {
		t.Errorf("memory stats are mean %+v. Expected %+v", mem, memExpected)
	}
}
