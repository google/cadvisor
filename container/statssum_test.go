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
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/google/cadvisor/info"
)

func init() {
	// NOTE(dengnan): Even if we picked a good random seed,
	// the random number from math/rand is still not cryptographically secure!
	var seed int64
	binary.Read(crand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)
}

type notARealContainer struct {
}

func (self *notARealContainer) GetSpec() (*info.ContainerSpec, error) {
	return nil, nil
}
func (self *notARealContainer) ListContainers(listType ListType) ([]string, error) {
	return nil, nil
}

func (self *notARealContainer) ListThreads(listType ListType) ([]int, error) {
	return nil, nil
}

func (self *notARealContainer) ListProcesses(listType ListType) ([]int, error) {
	return nil, nil
}

type randomMemoryUsageContainer struct {
	NoStatsSummary
	notARealContainer
}

func (self *randomMemoryUsageContainer) GetStats() (*info.ContainerStats, error) {
	stats := new(info.ContainerStats)
	stats.Cpu = new(info.CpuStats)
	stats.Memory = new(info.MemoryStats)
	stats.Memory.Usage = uint64(rand.Intn(2048))
	return stats, nil
}

func TestMaxMemoryUsage(t *testing.T) {
	handler, err := AddStatsSummary(
		&randomMemoryUsageContainer{},
		&StatsParameter{
			Sampler:    "uniform",
			NumSamples: 10,
		},
	)
	if err != nil {
		t.Error(err)
	}
	var maxUsage uint64
	N := 100
	for i := 0; i < N; i++ {
		stats, err := handler.GetStats()
		if err != nil {
			t.Errorf("Error when get stats: %v", err)
			continue
		}
		if stats.Memory.Usage > maxUsage {
			maxUsage = stats.Memory.Usage
		}
	}
	summary, err := handler.StatsSummary()
	if err != nil {
		t.Fatalf("Error when get summary: %v", err)
	}
	if summary.MaxMemoryUsage != maxUsage {
		t.Fatalf("Max memory usage should be %v; received %v", maxUsage, summary.MaxMemoryUsage)
	}
}

type replayCpuTrace struct {
	NoStatsSummary
	notARealContainer
	cpuTrace    []uint64
	totalUsage  uint64
	currenttime time.Time
	duration    time.Duration
	lock        sync.Mutex
}

func containerWithCpuTrace(duration time.Duration, cpuUsages ...uint64) ContainerHandler {
	return &replayCpuTrace{
		duration:    duration,
		cpuTrace:    cpuUsages,
		currenttime: time.Now(),
	}
}

func (self *replayCpuTrace) GetStats() (*info.ContainerStats, error) {
	stats := new(info.ContainerStats)
	stats.Cpu = new(info.CpuStats)
	stats.Memory = new(info.MemoryStats)
	stats.Memory.Usage = uint64(rand.Intn(2048))

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

	// We need N+1 observations to get N samples
	for i := 0; i < N+1; i++ {
		usage := uint64(rand.Intn(1000))
		cpuTrace = append(cpuTrace, usage)
	}

	samplePeriod := 1 * time.Second

	handler, err := AddStatsSummary(
		containerWithCpuTrace(samplePeriod, cpuTrace...),
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

	// we want to set our own time.
	if w, ok := handler.(*statsSummaryContainerHandlerWrapper); ok {
		w.dontSetTimestamp = true
	} else {
		t.Fatal("handler is not an instance of statsSummaryContainerHandlerWrapper")
	}

	// request stats/obervation N+1 times, so that there will be N samples
	for i := 0; i < N+1; i++ {
		_, err = handler.GetStats()
		if err != nil {
			t.Fatal(err)
		}
	}

	s, err := handler.StatsSummary()
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
