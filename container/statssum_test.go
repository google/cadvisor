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

package container

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/cadvisor/info"
)

type mockContainer struct {
}

func (self *mockContainer) GetSpec() (*info.ContainerSpec, error) {
	return nil, nil
}
func (self *mockContainer) ListContainers(listType ListType) ([]info.ContainerReference, error) {
	return nil, nil
}

func (self *mockContainer) ListThreads(listType ListType) ([]int, error) {
	return nil, nil
}

func (self *mockContainer) ListProcesses(listType ListType) ([]int, error) {
	return nil, nil
}

func TestMaxMemoryUsage(t *testing.T) {
	N := 100
	memTrace := make([]uint64, N)
	for i := 0; i < N; i++ {
		memTrace[i] = uint64(i + 1)
	}
	handler, err := AddStatsSummary(
		containerWithTrace(1*time.Second, nil, memTrace),
		&StatsParameter{
			Sampler:    "uniform",
			NumSamples: 10,
		},
	)
	if err != nil {
		t.Error(err)
	}
	maxUsage := uint64(N)
	for i := 0; i < N; i++ {
		_, err := handler.GetStats()
		if err != nil {
			t.Errorf("Error when get stats: %v", err)
			continue
		}
	}
	summary, err := handler.StatsPercentiles()
	if err != nil {
		t.Fatalf("Error when get summary: %v", err)
	}
	if summary.MaxMemoryUsage != maxUsage {
		t.Fatalf("Max memory usage should be %v; received %v", maxUsage, summary.MaxMemoryUsage)
	}
}

type replayTrace struct {
	NoStatsSummary
	mockContainer
	cpuTrace    []uint64
	memTrace    []uint64
	totalUsage  uint64
	currenttime time.Time
	duration    time.Duration
	lock        sync.Mutex
}

func containerWithTrace(duration time.Duration, cpuUsages []uint64, memUsages []uint64) ContainerHandler {
	return &replayTrace{
		duration:    duration,
		cpuTrace:    cpuUsages,
		memTrace:    memUsages,
		currenttime: time.Now(),
	}
}

func (self *replayTrace) ContainerReference() (info.ContainerReference, error) {
	return info.ContainerReference{
		Name: "replay",
	}, nil
}

func (self *replayTrace) GetStats() (*info.ContainerStats, error) {
	stats := new(info.ContainerStats)
	stats.Cpu = new(info.CpuStats)
	stats.Memory = new(info.MemoryStats)
	if len(self.memTrace) > 0 {
		stats.Memory.Usage = self.memTrace[0]
		self.memTrace = self.memTrace[1:]
	}

	self.lock.Lock()
	defer self.lock.Unlock()

	cpuTrace := self.totalUsage
	if len(self.cpuTrace) > 0 {
		cpuTrace += self.cpuTrace[0]
		self.cpuTrace = self.cpuTrace[1:]
	}
	self.totalUsage = cpuTrace
	stats.Timestamp = self.currenttime
	self.currenttime = self.currenttime.Add(self.duration)
	stats.Cpu.Usage.Total = cpuTrace
	stats.Cpu.Usage.PerCpu = []uint64{cpuTrace}
	stats.Cpu.Usage.User = cpuTrace
	stats.Cpu.Usage.System = 0
	return stats, nil
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

	handler, err := AddStatsSummary(
		containerWithTrace(samplePeriod, cpuTrace, memTrace),
		&StatsParameter{
			// Use uniform sampler with sample size of N, so that
			// we will be guaranteed to store the first N samples.
			Sampler:    "uniform",
			NumSamples: N,
		},
	)
	if err != nil {
		t.Error(err)
	}

	// request stats/observation N+1 times, so that there will be N samples
	for i := 0; i < N+1; i++ {
		_, err = handler.GetStats()
		if err != nil {
			t.Fatal(err)
		}
	}

	s, err := handler.StatsPercentiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range s.Samples {
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
