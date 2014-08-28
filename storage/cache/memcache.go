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

package cache

import (
	"fmt"
	"sync"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/storage/memory"
)

type containerStatsRefPair struct {
	ref   info.ContainerReference
	stats *info.ContainerStats
}

type cachedStorageDriver struct {
	maxNumStatsInCache   int
	maxNumSamplesInCache int
	maxNumDirtyStats     int
	dirtyStats           []containerStatsRefPair
	cache                storage.StorageDriver
	backend              storage.StorageDriver
	lock                 sync.RWMutex
}

func (self *cachedStorageDriver) Flush() error {
	self.lock.Lock()
	defer self.lock.Unlock()
	var err error
	for _, pair := range self.dirtyStats {
		err = self.backend.AddStats(pair.ref, pair.stats)
		if err != nil {
			return fmt.Errorf("error when writing stats for container %v: %v", pair.ref.Name, err)
		}
	}
	self.dirtyStats = self.dirtyStats[:0]
	return nil
}

func (self *cachedStorageDriver) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	err := self.cache.AddStats(ref, stats)
	if err != nil {
		return err
	}
	self.lock.Lock()
	self.dirtyStats = append(self.dirtyStats, containerStatsRefPair{ref, stats.Copy(nil)})
	if len(self.dirtyStats) >= self.maxNumDirtyStats {
		self.lock.Unlock()
		return self.Flush()
	} else {
		self.lock.Unlock()
	}
	return nil
}

func (self *cachedStorageDriver) RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error) {
	if numStats <= self.maxNumStatsInCache {
		return self.cache.RecentStats(containerName, numStats)
	}
	return self.backend.RecentStats(containerName, numStats)
}

// TODO(vishh): Calculate percentiles from cached stats instead of reaching the DB. This will make the UI truly independent of the backend storage.
func (self *cachedStorageDriver) Percentiles(containerName string, cpuUsagePercentiles []int, memUsagePercentiles []int) (*info.ContainerStatsPercentiles, error) {
	err := self.Flush()
	if err != nil {
		return nil, fmt.Errorf("unable to query percentiles: %v", err)
	}
	return self.backend.Percentiles(containerName, cpuUsagePercentiles, memUsagePercentiles)
}

func (self *cachedStorageDriver) Samples(containerName string, numSamples int) ([]*info.ContainerStatsSample, error) {
	if numSamples <= self.maxNumSamplesInCache {
		return self.cache.Samples(containerName, numSamples)
	}
	return self.backend.Samples(containerName, numSamples)
}

func (self *cachedStorageDriver) Close() error {
	self.cache.Close()
	return self.backend.Close()
}

// TODO(vishh): Cache all samples for a given duration and do not cap the maximum number of samples. This is useful if we happen to change the housekeeping duration.
func MemoryCache(maxNumSamplesInCache, maxNumStatsInCache int, driver storage.StorageDriver) storage.StorageDriver {
	return &cachedStorageDriver{
		maxNumStatsInCache:   maxNumStatsInCache,
		maxNumSamplesInCache: maxNumSamplesInCache,
		maxNumDirtyStats:     maxNumStatsInCache,
		cache:                memory.New(maxNumSamplesInCache, maxNumStatsInCache),
		backend:              driver,
	}
}
