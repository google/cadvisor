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

package info

import (
	"testing"
	"time"
)

func TestStatsStartTime(t *testing.T) {
	N := 10
	stats := make([]*ContainerStats, 0, N)
	ct := time.Now()
	for i := 0; i < N; i++ {
		s := &ContainerStats{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
		}
		stats = append(stats, s)
	}
	cinfo := &ContainerInfo{
		Name:  "/some/container",
		Stats: stats,
	}
	ref := ct.Add(time.Duration(N-1) * time.Second)
	end := cinfo.StatsEndTime()

	if !ref.Equal(end) {
		t.Errorf("end time is %v; should be %v", end, ref)
	}
}

func TestStatsEndTime(t *testing.T) {
	N := 10
	stats := make([]*ContainerStats, 0, N)
	ct := time.Now()
	for i := 0; i < N; i++ {
		s := &ContainerStats{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
		}
		stats = append(stats, s)
	}
	cinfo := &ContainerInfo{
		Name:  "/some/container",
		Stats: stats,
	}
	ref := ct
	start := cinfo.StatsStartTime()

	if !ref.Equal(start) {
		t.Errorf("start time is %v; should be %v", start, ref)
	}
}

func TestPercentiles(t *testing.T) {
	N := 100
	data := make([]uint64, N)

	for i := 0; i < N; i++ {
		data[i] = uint64(i)
	}
	ps := []int{
		80,
		90,
		50,
	}
	ss := uint64Slice(data).Percentiles(ps...)
	for i, s := range ss {
		p := ps[i]
		d := uint64(float64(N) * (float64(p) / 100.0))
		if d != s {
			t.Errorf("%v \\%tile data should be %v, but got %v", float64(p)/100.0, d, s)
		}
	}
}

func TestAddSampleNilStats(t *testing.T) {
	s := &ContainerStatsSummary{}

	stats := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	stats.Cpu.Usage.PerCpu = []uint64{uint64(10)}
	stats.Cpu.Usage.Total = uint64(10)
	stats.Cpu.Usage.System = uint64(2)
	stats.Cpu.Usage.User = uint64(8)
	stats.Memory.Usage = uint64(200)
	s.AddSample(nil, stats)

	if len(s.Samples) != 0 {
		t.Errorf("added an unexpected sample: %+v", s.Samples)
	}

	s.Samples = nil
	s.AddSample(stats, nil)

	if len(s.Samples) != 0 {
		t.Errorf("added an unexpected sample: %+v", s.Samples)
	}
}

func TestAddSample(t *testing.T) {
	s := &ContainerStatsSummary{}

	cpuPrevUsage := uint64(10)
	cpuCurrentUsage := uint64(15)
	memCurrentUsage := uint64(200)

	prev := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	prev.Cpu.Usage.PerCpu = []uint64{cpuPrevUsage}
	prev.Cpu.Usage.Total = cpuPrevUsage
	prev.Cpu.Usage.System = 0
	prev.Cpu.Usage.User = cpuPrevUsage
	prev.Timestamp = time.Now()

	current := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	current.Cpu.Usage.PerCpu = []uint64{cpuCurrentUsage}
	current.Cpu.Usage.Total = cpuCurrentUsage
	current.Cpu.Usage.System = 0
	current.Cpu.Usage.User = cpuCurrentUsage
	current.Memory.Usage = memCurrentUsage
	current.Timestamp = prev.Timestamp.Add(1 * time.Second)

	s.AddSample(prev, current)

	if len(s.Samples) != 1 {
		t.Fatalf("unexpected samples: %+v", s.Samples)
	}

	if s.Samples[0].Memory.Usage != memCurrentUsage {
		t.Errorf("wrong memory usage: %v. should be %v", s.Samples[0].Memory.Usage, memCurrentUsage)
	}

	if s.Samples[0].Cpu.Usage != cpuCurrentUsage-cpuPrevUsage {
		t.Errorf("wrong CPU usage: %v. should be %v", s.Samples[0].Cpu.Usage, cpuCurrentUsage-cpuPrevUsage)
	}
}

func TestAddSampleIncompleteStats(t *testing.T) {
	s := &ContainerStatsSummary{}

	cpuPrevUsage := uint64(10)
	cpuCurrentUsage := uint64(15)
	memCurrentUsage := uint64(200)

	prev := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	prev.Cpu.Usage.PerCpu = []uint64{cpuPrevUsage}
	prev.Cpu.Usage.Total = cpuPrevUsage
	prev.Cpu.Usage.System = 0
	prev.Cpu.Usage.User = cpuPrevUsage
	prev.Timestamp = time.Now()

	current := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	current.Cpu.Usage.PerCpu = []uint64{cpuCurrentUsage}
	current.Cpu.Usage.Total = cpuCurrentUsage
	current.Cpu.Usage.System = 0
	current.Cpu.Usage.User = cpuCurrentUsage
	current.Memory.Usage = memCurrentUsage
	current.Timestamp = prev.Timestamp.Add(1 * time.Second)

	stats := &ContainerStats{
		Cpu:    prev.Cpu,
		Memory: nil,
	}
	s.AddSample(stats, current)
	if len(s.Samples) != 0 {
		t.Errorf("added an unexpected sample: %+v", s.Samples)
	}
	s.Samples = nil

	s.AddSample(prev, stats)
	if len(s.Samples) != 0 {
		t.Errorf("added an unexpected sample: %+v", s.Samples)
	}
	s.Samples = nil

	stats = &ContainerStats{
		Cpu:    nil,
		Memory: prev.Memory,
	}
	s.AddSample(stats, current)
	if len(s.Samples) != 0 {
		t.Errorf("added an unexpected sample: %+v", s.Samples)
	}
	s.Samples = nil

	s.AddSample(prev, stats)
	if len(s.Samples) != 0 {
		t.Errorf("added an unexpected sample: %+v", s.Samples)
	}
	s.Samples = nil
}

func TestAddSampleWrongOrder(t *testing.T) {
	s := &ContainerStatsSummary{}

	cpuPrevUsage := uint64(10)
	cpuCurrentUsage := uint64(15)
	memCurrentUsage := uint64(200)

	prev := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	prev.Cpu.Usage.PerCpu = []uint64{cpuPrevUsage}
	prev.Cpu.Usage.Total = cpuPrevUsage
	prev.Cpu.Usage.System = 0
	prev.Cpu.Usage.User = cpuPrevUsage
	prev.Timestamp = time.Now()

	current := &ContainerStats{
		Cpu:    &CpuStats{},
		Memory: &MemoryStats{},
	}
	current.Cpu.Usage.PerCpu = []uint64{cpuCurrentUsage}
	current.Cpu.Usage.Total = cpuCurrentUsage
	current.Cpu.Usage.System = 0
	current.Cpu.Usage.User = cpuCurrentUsage
	current.Memory.Usage = memCurrentUsage
	current.Timestamp = prev.Timestamp.Add(1 * time.Second)

	s.AddSample(current, prev)
	if len(s.Samples) != 0 {
		t.Errorf("added an unexpected sample: %+v", s.Samples)
	}
}
