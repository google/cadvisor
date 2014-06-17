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
	"container/list"
	"fmt"
	"sync"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/sampling"
	"github.com/google/cadvisor/storage"
)

// containerStorage is used to store per-container information
type containerStorage struct {
	ref            info.ContainerReference
	prevStats      *info.ContainerStats
	sampler        sampling.Sampler
	recentStats    *list.List
	numRecentStats int
	maxMemUsage    uint64
	lock           sync.RWMutex
}

func (self *containerStorage) updatePrevStats(stats *info.ContainerStats) {
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

func (self *containerStorage) AddStats(stats *info.ContainerStats) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.prevStats != nil {
		sample, err := info.NewSample(self.prevStats, stats)
		if err != nil {
			return fmt.Errorf("wrong stats: %v", err)
		}
		if sample != nil {
			self.sampler.Update(sample)
		}
	}
	if stats.Memory != nil {
		if self.maxMemUsage < stats.Memory.Usage {
			self.maxMemUsage = stats.Memory.Usage
		}
	}
	if self.recentStats.Len() >= self.numRecentStats {
		self.recentStats.Remove(self.recentStats.Front())
	}
	self.recentStats.PushBack(stats)
	self.updatePrevStats(stats)
	return nil
}

func (self *containerStorage) RecentStats(numStats int) ([]*info.ContainerStats, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.recentStats.Len() < numStats || numStats < 0 {
		numStats = self.recentStats.Len()
	}
	ret := make([]*info.ContainerStats, 0, numStats)
	e := self.recentStats.Front()
	for i := 0; i < numStats; i++ {
		data := e.Value.(*info.ContainerStats)
		ret = append(ret, data)
		e = e.Next()
		if e == nil {
			break
		}
	}
	return ret, nil
}

func (self *containerStorage) Samples(numSamples int) ([]*info.ContainerStatsSample, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.sampler.Len() < numSamples || numSamples < 0 {
		numSamples = self.sampler.Len()
	}
	ret := make([]*info.ContainerStatsSample, 0, numSamples)

	self.sampler.Map(func(d interface{}) {
		if len(ret) >= numSamples {
			return
		}
		sample := d.(*info.ContainerStatsSample)
		ret = append(ret, sample)
	})
	return ret, nil
}

func (self *containerStorage) Percentiles(cpuPercentiles, memPercentiles []int) (*info.ContainerStatsPercentiles, error) {
	samples, err := self.Samples(-1)
	if err != nil {
		return nil, err
	}
	ret := new(info.ContainerStatsPercentiles)
	ret.FillPercentiles(samples, cpuPercentiles, memPercentiles)
	ret.MaxMemoryUsage = self.maxMemUsage
	return ret, nil
}

func newContainerStore(ref info.ContainerReference, maxNumSamples, numRecentStats int) *containerStorage {
	s := sampling.NewReservoirSampler(maxNumSamples)
	return &containerStorage{
		ref:            ref,
		recentStats:    list.New(),
		sampler:        s,
		numRecentStats: numRecentStats,
	}
}

type InMemoryStorage struct {
	lock                sync.RWMutex
	containerStorageMap map[string]*containerStorage
	maxNumSamples       int
	numRecentStats      int
}

func (self *InMemoryStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	var cstore *containerStorage
	var ok bool
	self.lock.Lock()
	if cstore, ok = self.containerStorageMap[ref.Name]; !ok {
		cstore = newContainerStore(ref, self.maxNumSamples, self.numRecentStats)
		self.containerStorageMap[ref.Name] = cstore
	}
	self.lock.Unlock()
	return cstore.AddStats(stats)
}

func (self *InMemoryStorage) Samples(name string, numSamples int) ([]*info.ContainerStatsSample, error) {
	var cstore *containerStorage
	var ok bool
	self.lock.RLock()
	if cstore, ok = self.containerStorageMap[name]; !ok {
		return nil, fmt.Errorf("unable to find data for container %v", name)
	}
	self.lock.RUnlock()

	return cstore.Samples(numSamples)
}

func (self *InMemoryStorage) RecentStats(name string, numStats int) ([]*info.ContainerStats, error) {
	var cstore *containerStorage
	var ok bool
	self.lock.RLock()
	if cstore, ok = self.containerStorageMap[name]; !ok {
		return nil, fmt.Errorf("unable to find data for container %v", name)
	}
	self.lock.RUnlock()

	return cstore.RecentStats(numStats)
}

func (self *InMemoryStorage) Percentiles(name string, cpuPercentiles, memPercentiles []int) (*info.ContainerStatsPercentiles, error) {
	var cstore *containerStorage
	var ok bool
	self.lock.RLock()
	if cstore, ok = self.containerStorageMap[name]; !ok {
		return nil, fmt.Errorf("unable to find data for container %v", name)
	}
	self.lock.RUnlock()

	return cstore.Percentiles(cpuPercentiles, memPercentiles)
}

func New(maxNumSamples, numRecentStats int) storage.StorageDriver {
	ret := &InMemoryStorage{
		containerStorageMap: make(map[string]*containerStorage, 32),
		maxNumSamples:       maxNumSamples,
		numRecentStats:      numRecentStats,
	}
	return ret
}
