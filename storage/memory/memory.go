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
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
)

// containerStorage is used to store per-container information
type containerStorage struct {
	ref         info.ContainerReference
	recentStats *StatsBuffer
	maxNumStats int
	lock        sync.RWMutex
}

func (self *containerStorage) AddStats(stats *info.ContainerStats) error {
	self.lock.Lock()
	defer self.lock.Unlock()

	// Add the stat to storage.
	self.recentStats.Add(stats)
	return nil
}

func (self *containerStorage) RecentStats(numStats int) ([]*info.ContainerStats, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.recentStats.Size() < numStats || numStats < 0 {
		numStats = self.recentStats.Size()
	}

	// Stats in the recentStats list are stored in reverse chronological
	// order, i.e. most recent stats is in the front.
	// numStats will always <= recentStats.Size() so that there will be
	// always at least numStats available stats to retrieve. We traverse
	// the recentStats list from its head so that the returned slice will be in chronological
	// order. The order of the returned slice is not specified by the
	// StorageDriver interface, so it is not necessary for other storage
	// drivers to return the slice in the same order.
	return self.recentStats.FirstN(numStats), nil
}

func newContainerStore(ref info.ContainerReference, maxNumStats int) *containerStorage {
	return &containerStorage{
		ref:         ref,
		recentStats: NewStatsBuffer(maxNumStats),
		maxNumStats: maxNumStats,
	}
}

type InMemoryStorage struct {
	lock                sync.RWMutex
	containerStorageMap map[string]*containerStorage
	maxNumStats         int
	backend             storage.StorageDriver
}

func (self *InMemoryStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	var cstore *containerStorage
	var ok bool

	func() {
		self.lock.Lock()
		defer self.lock.Unlock()
		if cstore, ok = self.containerStorageMap[ref.Name]; !ok {
			cstore = newContainerStore(ref, self.maxNumStats)
			self.containerStorageMap[ref.Name] = cstore
		}
	}()

	if self.backend != nil {
		// TODO(monnand): To deal with long delay write operations, we
		// may want to start a pool of goroutines to do write
		// operations.
		if err := self.backend.AddStats(ref, stats); err != nil {
			glog.Error(err)
		}
	}
	return cstore.AddStats(stats)
}

func (self *InMemoryStorage) RecentStats(name string, numStats int) ([]*info.ContainerStats, error) {
	var cstore *containerStorage
	var ok bool
	err := func() error {
		self.lock.RLock()
		defer self.lock.RUnlock()
		if cstore, ok = self.containerStorageMap[name]; !ok {
			return fmt.Errorf("unable to find data for container %v", name)
		}
		return nil
	}()
	if err != nil {
		return nil, err
	}

	return cstore.RecentStats(numStats)
}

func (self *InMemoryStorage) Close() error {
	self.lock.Lock()
	self.containerStorageMap = make(map[string]*containerStorage, 32)
	self.lock.Unlock()
	return nil
}

func New(
	maxNumStats int,
	backend storage.StorageDriver,
) *InMemoryStorage {
	ret := &InMemoryStorage{
		containerStorageMap: make(map[string]*containerStorage, 32),
		maxNumStats:         maxNumStats,
		backend:             backend,
	}
	return ret
}
