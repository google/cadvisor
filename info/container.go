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
	"sort"
	"time"
)

type CpuSpecMask struct {
	Data []uint64 `json:"data,omitempty"`
}

type CpuSpec struct {
	Limit    uint64      `json:"limit"`
	MaxLimit uint64      `json:"max_limit"`
	Mask     CpuSpecMask `json:"mask,omitempty"`
}

type MemorySpec struct {
	// The amount of memory requested. Default is unlimited (-1).
	// Units: bytes.
	Limit uint64 `json:"limit,omitempty"`

	// The amount of guaranteed memory.  Default is 0.
	// Units: bytes.
	Reservation uint64 `json:"reservation,omitempty"`

	// The amount of swap space requested. Default is unlimited (-1).
	// Units: bytes.
	SwapLimit uint64 `json:"swap_limit,omitempty"`
}

type ContainerSpec struct {
	Cpu    *CpuSpec    `json:"cpu,omitempty"`
	Memory *MemorySpec `json:"memory,omitempty"`
}

type ContainerInfo struct {
	// The absolute name of the container.
	Name string `json:"name"`

	// The direct subcontainers of the current container.
	Subcontainers []string `json:"subcontainers,omitempty"`

	// The isolation used in the container.
	Spec *ContainerSpec `json:"spec,omitempty"`

	// Historical statistics gathered from the container.
	Stats []*ContainerStats `json:"stats,omitempty"`

	StatsSummary *ContainerStatsSummary `json:"stats_summary,omitempty"`
}

func (self *ContainerInfo) StatsAfter(ref time.Time) []*ContainerStats {
	n := len(self.Stats) + 1
	for i, s := range self.Stats {
		if s.Timestamp.After(ref) {
			n = i
			break
		}
	}
	if n > len(self.Stats) {
		return nil
	}
	return self.Stats[n:]
}

func (self *ContainerInfo) StatsStartTime() time.Time {
	var ret time.Time
	for _, s := range self.Stats {
		if s.Timestamp.Before(ret) || ret.IsZero() {
			ret = s.Timestamp
		}
	}
	return ret
}

func (self *ContainerInfo) StatsEndTime() time.Time {
	var ret time.Time
	for i := len(self.Stats) - 1; i >= 0; i-- {
		s := self.Stats[i]
		if s.Timestamp.After(ret) {
			ret = s.Timestamp
		}
	}
	return ret
}

type CpuStats struct {
	Usage struct {
		// Number of nanoseconds of CPU time used by the container
		// since the beginning. This is a ccumulative
		// value, not an instantaneous value.
		Total uint64 `json:"total"`

		// Per CPU/core usage of the container.
		// Unit: nanoseconds.
		PerCpu []uint64 `json:"per_cpu,omitempty"`

		// How much time was spent in user space since beginning.
		// Unit: nanoseconds
		User uint64 `json:"user"`

		// How much time was spent in kernel space since beginning.
		// Unit: nanoseconds
		System uint64 `json:"system"`
	} `json:"usage"`
	Load int32 `json:"load"`
}

type MemoryStats struct {
	// Memory limit, equivalent to "limit" in MemorySpec.
	// Units: Bytes.
	Limit uint64 `json:"limit,omitempty"`

	// Usage statistics.

	// Current memory usage, this includes all memory regardless of when it was
	// accessed.
	// Units: Bytes.
	Usage uint64 `json:"usage,omitempty"`

	// The amount of working set memory, this includes recently accessed memory,
	// dirty memory, and kernel memmory. Working set is <= "usage".
	// Units: Bytes.
	WorkingSet uint64 `json:"working_set,omitempty"`

	ContainerData    MemoryStatsMemoryData `json:"container_data,omitempty"`
	HierarchicalData MemoryStatsMemoryData `json:"hierarchical_data,omitempty"`
}

type MemoryStatsMemoryData struct {
	Pgfault    uint64 `json:"pgfault,omitempty"`
	Pgmajfault uint64 `json:"pgmajfault,omitempty"`
}

type ContainerStats struct {
	// The time of this stat point.
	Timestamp time.Time    `json:"timestamp"`
	Cpu       *CpuStats    `json:"cpu,omitempty"`
	Memory    *MemoryStats `json:"memory,omitempty"`
}

type ContainerStatsSample struct {
	Cpu struct {
		// number of nanoseconds of CPU time used by the container
		// within one second.
		Usage uint64 `json:"usage"`
	} `json:"cpu"`
	Memory struct {
		// Units: Bytes.
		Usage uint64 `json:"usage"`
	} `json:"memory"`
}

// This is not exported.
// Use FillPercentile to calculate percentiles
type percentile struct {
	Percentage int    `json:"percentage"`
	Value      uint64 `json:"value"`
}

type ContainerStatsSummary struct {
	// TODO(dengnan): More things?
	MaxMemoryUsage         uint64                  `json:"max_memory_usage,omitempty"`
	AvgMemoryUsage         uint64                  `json:"avg_memory_usage,omitempty"`
	Samples                []*ContainerStatsSample `json:"samples,omitempty"`
	MemoryUsagePercentiles []percentile            `json:"memory_usage_percentiles,omitempty"`
	CpuUsagePercentiles    []percentile            `json:"cpu_usage_percentiles,omitempty"`
}

// Each sample needs two stats because the cpu usage in ContainerStats is
// cumulative.
// prev should be an earlier observation than current.
// This method is not thread/goroutine safe.
func (self *ContainerStatsSummary) AddSample(prev, current *ContainerStats) {
	if prev == nil || current == nil {
		return
	}
	// Ignore this sample if it is incomplete
	if prev.Cpu == nil || prev.Memory == nil || current.Cpu == nil || current.Memory == nil {
		return
	}
	// prev must be an early observation
	if !current.Timestamp.After(prev.Timestamp) {
		return
	}
	sample := new(ContainerStatsSample)
	// Caculate the diff to get the CPU usage within the time interval.
	sample.Cpu.Usage = current.Cpu.Usage.Total - prev.Cpu.Usage.Total
	// Memory usage is current memory usage
	sample.Memory.Usage = current.Memory.Usage

	self.Samples = append(self.Samples, sample)
	return
}

type uint64Slice []uint64

func (self uint64Slice) Len() int {
	return len(self)
}

func (self uint64Slice) Less(i, j int) bool {
	return self[i] < self[j]
}

func (self uint64Slice) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self uint64Slice) Percentiles(ps ...int) []uint64 {
	if len(self) == 0 {
		return nil
	}
	ret := make([]uint64, 0, len(ps))
	sort.Sort(self)
	for _, p := range ps {
		idx := (float64(p) / 100.0) * float64(len(self)+1)
		if idx > float64(len(self)-1) {
			ret = append(ret, self[len(self)-1])
		} else {
			ret = append(ret, self[int(idx)])
		}
	}
	return ret
}

// len(bs) <= len(as)
func intZipuint64(as []int, bs []uint64) []percentile {
	if len(bs) == 0 {
		return nil
	}
	ret := make([]percentile, len(bs))
	for i, b := range bs {
		a := as[i]
		ret[i] = percentile{
			Percentage: a,
			Value:      b,
		}
	}
	return ret
}

func (self *ContainerStatsSummary) FillPercentiles(cpuPercentages, memoryPercentages []int) {
	if len(self.Samples) == 0 {
		return
	}
	cpuUsages := make([]uint64, 0, len(self.Samples))
	memUsages := make([]uint64, 0, len(self.Samples))

	for _, sample := range self.Samples {
		if sample == nil {
			continue
		}
		cpuUsages = append(cpuUsages, sample.Cpu.Usage)
		memUsages = append(memUsages, sample.Memory.Usage)
	}

	cpuPercentiles := uint64Slice(cpuUsages).Percentiles(cpuPercentages...)
	memPercentiles := uint64Slice(memUsages).Percentiles(memoryPercentages...)
	self.CpuUsagePercentiles = intZipuint64(cpuPercentages, cpuPercentiles)
	self.MemoryUsagePercentiles = intZipuint64(memoryPercentages, memPercentiles)
}
