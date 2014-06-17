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

package memory

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/cadvisor/info"
)

func buildTrace(cpu, mem []uint64, duration time.Duration) []*info.ContainerStats {
	if len(cpu) != len(mem) {
		panic("len(cpu) != len(mem)")
	}

	ret := make([]*info.ContainerStats, len(cpu))
	currentTime := time.Now()

	var cpuTotalUsage uint64 = 0
	for i, cpuUsage := range cpu {
		cpuTotalUsage += cpuUsage
		stats := new(info.ContainerStats)
		stats.Cpu = new(info.CpuStats)
		stats.Memory = new(info.MemoryStats)
		stats.Timestamp = currentTime
		currentTime = currentTime.Add(duration)

		stats.Cpu.Usage.Total = cpuTotalUsage
		stats.Cpu.Usage.User = stats.Cpu.Usage.Total
		stats.Cpu.Usage.System = 0

		stats.Memory.Usage = mem[i]

		ret[i] = stats
	}
	return ret
}

func TestSampleCpuUsage(t *testing.T) {
	// Number of samples
	N := 10
	cpuTrace := make([]uint64, 0, N)
	memTrace := make([]uint64, 0, N)

	// We need N+1 observations to get N samples
	for i := 0; i < N+1; i++ {
		cpuusage := uint64(rand.Intn(1000))
		memusage := uint64(rand.Intn(1000))
		cpuTrace = append(cpuTrace, cpuusage)
		memTrace = append(memTrace, memusage)
	}

	samplePeriod := 1 * time.Second

	storage := New(N, N)

	ref := info.ContainerReference{
		Name: "container",
	}

	trace := buildTrace(cpuTrace, memTrace, samplePeriod)

	for _, stats := range trace {
		storage.AddStats(ref, stats)
	}

	samples, err := storage.Samples(ref.Name, N)
	if err != nil {
		t.Errorf("unable to sample stats: %v", err)
	}
	for _, sample := range samples {
		if sample.Duration != samplePeriod {
			t.Errorf("sample duration is %v, not %v", sample.Duration, samplePeriod)
		}
		cpuUsage := sample.Cpu.Usage
		found := false
		for _, u := range cpuTrace {
			if u == cpuUsage {
				found = true
			}
		}
		if !found {
			t.Errorf("unable to find cpu usage %v", cpuUsage)
		}
	}
}

func TestMaxMemoryUsage(t *testing.T) {
	N := 100
	memTrace := make([]uint64, N)
	cpuTrace := make([]uint64, N)
	for i := 0; i < N; i++ {
		memTrace[i] = uint64(i + 1)
		cpuTrace[i] = uint64(1)
	}

	storage := New(N-10, N-10)

	ref := info.ContainerReference{
		Name: "container",
	}

	trace := buildTrace(cpuTrace, memTrace, 1*time.Second)

	for _, stats := range trace {
		storage.AddStats(ref, stats)
	}

	percentiles, err := storage.Percentiles(ref.Name, []int{50}, []int{50})
	if err != nil {
		t.Errorf("unable to call Percentiles(): %v", err)
	}
	maxUsage := uint64(N)
	if percentiles.MaxMemoryUsage != maxUsage {
		t.Fatalf("Max memory usage should be %v; received %v", maxUsage, percentiles.MaxMemoryUsage)
	}
}

func TestSamplesWithoutSample(t *testing.T) {
	storage := New(10, 10)
	trace := buildTrace(
		[]uint64{10},
		[]uint64{10},
		1*time.Second)
	ref := info.ContainerReference{
		Name: "container",
	}
	storage.AddStats(ref, trace[0])
	samples, err := storage.Samples(ref.Name, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 0 {
		t.Errorf("There should be no sample")
	}
}

func TestPercentilesWithoutSample(t *testing.T) {
	storage := New(10, 10)
	trace := buildTrace(
		[]uint64{10},
		[]uint64{10},
		1*time.Second)
	ref := info.ContainerReference{
		Name: "container",
	}
	storage.AddStats(ref, trace[0])
	percentiles, err := storage.Percentiles(
		ref.Name,
		[]int{50},
		[]int{50},
	)
	if err != nil {
		t.Fatal(err)
	}
	if percentiles != nil {
		t.Errorf("There should be no percentiles")
	}
}
