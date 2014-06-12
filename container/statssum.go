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
	"fmt"
	"sync"
	"time"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/sampling"
)

type percentilesContainerHandlerWrapper struct {
	handler              ContainerHandler
	containerPercentiles *info.ContainerStatsPercentiles
	prevStats            *info.ContainerStats
	numStats             uint64
	sampler              sampling.Sampler
	lock                 sync.Mutex
}

func (self *percentilesContainerHandlerWrapper) GetSpec() (*info.ContainerSpec, error) {
	return self.handler.GetSpec()
}

func (self *percentilesContainerHandlerWrapper) updatePrevStats(stats *info.ContainerStats) {
	if stats == nil || stats.Cpu == nil || stats.Memory == nil {
		// discard incomplete stats
		self.prevStats = nil
		return
	}
	if self.prevStats == nil {
		self.prevStats = &info.ContainerStats{
			Cpu:    &info.CpuStats{},
			Memory: &info.MemoryStats{},
		}
	}
	// make a deep copy.
	self.prevStats.Timestamp = stats.Timestamp
	*self.prevStats.Cpu = *stats.Cpu
	self.prevStats.Cpu.Usage.PerCpu = make([]uint64, len(stats.Cpu.Usage.PerCpu))
	for i, perCpu := range stats.Cpu.Usage.PerCpu {
		self.prevStats.Cpu.Usage.PerCpu[i] = perCpu
	}
	*self.prevStats.Memory = *stats.Memory
}

func (self *percentilesContainerHandlerWrapper) GetStats() (*info.ContainerStats, error) {
	stats, err := self.handler.GetStats()
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, fmt.Errorf("container handler returns a nil error and a nil stats")
	}
	if stats.Timestamp.IsZero() {
		return nil, fmt.Errorf("container handler did not set timestamp")
	}
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.prevStats != nil {
		sample, err := info.NewSample(self.prevStats, stats)
		if err != nil {
			return nil, fmt.Errorf("wrong stats: %v", err)
		}
		if sample != nil {
			self.sampler.Update(sample)
		}
	}
	self.updatePrevStats(stats)
	if self.containerPercentiles == nil {
		self.containerPercentiles = new(info.ContainerStatsPercentiles)
	}
	self.numStats++
	if stats.Memory != nil {
		if stats.Memory.Usage > self.containerPercentiles.MaxMemoryUsage {
			self.containerPercentiles.MaxMemoryUsage = stats.Memory.Usage
		}
	}
	return stats, nil
}

func (self *percentilesContainerHandlerWrapper) ListContainers(listType ListType) ([]string, error) {
	return self.handler.ListContainers(listType)
}

func (self *percentilesContainerHandlerWrapper) ListThreads(listType ListType) ([]int, error) {
	return self.handler.ListThreads(listType)
}

func (self *percentilesContainerHandlerWrapper) ListProcesses(listType ListType) ([]int, error) {
	return self.handler.ListProcesses(listType)
}

func (self *percentilesContainerHandlerWrapper) StatsSummary() (*info.ContainerStatsPercentiles, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	samples := make([]*info.ContainerStatsSample, 0, self.sampler.Len())
	self.sampler.Map(func(d interface{}) {
		stats := d.(*info.ContainerStatsSample)
		samples = append(samples, stats)
	})
	self.containerPercentiles.Samples = samples
	// XXX(dengnan): probably add to StatsParameter?
	self.containerPercentiles.FillPercentiles(
		[]int{50, 80, 90, 95, 99},
		[]int{50, 80, 90, 95, 99},
	)
	return self.containerPercentiles, nil
}

type StatsParameter struct {
	Sampler     string
	NumSamples  int
	WindowSize  int
	ResetPeriod time.Duration
}

func AddStatsSummary(handler ContainerHandler, parameter *StatsParameter) (ContainerHandler, error) {
	sampler, err := NewSampler(parameter)
	if err != nil {
		return nil, err
	}
	return &percentilesContainerHandlerWrapper{
		handler:              handler,
		containerPercentiles: &info.ContainerStatsPercentiles{},
		sampler:              sampler,
	}, nil
}
