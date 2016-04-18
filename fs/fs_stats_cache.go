// Copyright 2016 Google Inc. All Rights Reserved.
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

// +build linux

package fs

import (
	"fmt"
	"github.com/golang/glog"
	zfs "github.com/mistifyio/go-zfs"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type cacheEntry struct {
	// Cache management
	createdAt time.Time
	// Actual cache data
	Type       FsType
	Capacity   uint64
	Free       uint64
	Available  uint64
	Inodes     uint64
	InodesFree uint64
}

type realFsStatsCache struct {
	cacheLifetime time.Duration
	cacheLock     sync.RWMutex
	cache         map[string]cacheEntry
	statsHelper   statsHelper
}

func newFsStatsCache(lifetime time.Duration, helper statsHelper) FsStatsCache {
	return &realFsStatsCache{
		cacheLifetime: lifetime,
		cache:         make(map[string]cacheEntry),
		statsHelper:   helper,
	}
}

func NewFsStatsCache() FsStatsCache {
	return newFsStatsCache(time.Minute, &realStatsHelper{})
}

func (self *realFsStatsCache) Clear() {
	self.cacheLock.Lock()
	defer self.cacheLock.Unlock()
	self.cache = make(map[string]cacheEntry)
}

func makeCacheKey(dev string, part partition) string {
	return fmt.Sprintf("%s.%s.%s", part.fsType, part.mountpoint, dev)
}

func unwrapCacheEntry(e cacheEntry) (FsType, uint64, uint64, uint64, uint64, uint64, error) {
	return e.Type, e.Capacity, e.Free, e.Available, e.Inodes, e.InodesFree, nil
}

func (self *realFsStatsCache) FsStats(dev string, part partition) (FsType, uint64, uint64, uint64, uint64, uint64, error) {
	cacheKey := makeCacheKey(dev, part)
	self.cacheLock.RLock()
	e, ok := self.cache[cacheKey]
	self.cacheLock.RUnlock()

	if ok {
		// We have the data in cache. Return it if it's recent enough
		if time.Since(e.createdAt) < self.cacheLifetime {
			glog.V(2).Infof("Consuming stats cache for %q: %+v", cacheKey, e)
			return unwrapCacheEntry(e)
		}
	}

	// Our cache entry is too old, or it doesn't exist. Replace it. Note:
	// this doesn't do anything to prevent a thundering herd. It's up to
	// the consumer to do so.
	glog.V(2).Infof("Refreshing stats cache for %q", cacheKey)
	e = cacheEntry{}

	var err error

	switch part.fsType {
	case DeviceMapper.String():
		e.Capacity, e.Free, e.Available, err = self.statsHelper.GetDmStats(dev, part.blockSize)
		e.Type = DeviceMapper
	case ZFS.String():
		e.Capacity, e.Free, e.Available, err = self.statsHelper.GetZfstats(dev)
		e.Type = ZFS
	default:
		e.Capacity, e.Free, e.Available, e.Inodes, e.InodesFree, err = self.statsHelper.GetVfsStats(part.mountpoint)
		e.Type = VFS
	}

	// We failed. Don't update the cache with dead data.
	if err != nil {
		return "", 0, 0, 0, 0, 0, err
	}

	// We succeeded. Update the cache and return the data.
	e.createdAt = time.Now()

	self.cacheLock.Lock()
	self.cache[cacheKey] = e
	self.cacheLock.Unlock()

	return unwrapCacheEntry(e)
}

type realStatsHelper struct{}

type statsHelper interface {
	GetVfsStats(path string) (total uint64, free uint64, avail uint64, inodes uint64, inodesFree uint64, err error)
	GetDmStats(poolName string, dataBlkSize uint) (uint64, uint64, uint64, error)
	GetZfstats(poolName string) (uint64, uint64, uint64, error)
}

func (*realStatsHelper) GetVfsStats(path string) (total uint64, free uint64, avail uint64, inodes uint64, inodesFree uint64, err error) {
	var s syscall.Statfs_t
	if err = syscall.Statfs(path, &s); err != nil {
		return 0, 0, 0, 0, 0, err
	}
	total = uint64(s.Frsize) * s.Blocks
	free = uint64(s.Frsize) * s.Bfree
	avail = uint64(s.Frsize) * s.Bavail
	inodes = uint64(s.Files)
	inodesFree = uint64(s.Ffree)
	return total, free, avail, inodes, inodesFree, nil
}

func (*realStatsHelper) GetDmStats(poolName string, dataBlkSize uint) (uint64, uint64, uint64, error) {
	out, err := exec.Command("dmsetup", "status", poolName).Output()
	if err != nil {
		return 0, 0, 0, err
	}

	used, total, err := parseDMStatus(string(out))
	if err != nil {
		return 0, 0, 0, err
	}

	used *= 512 * uint64(dataBlkSize)
	total *= 512 * uint64(dataBlkSize)
	free := total - used

	return total, free, free, nil
}

// getZfstats returns ZFS mount stats using zfsutils
func (*realStatsHelper) GetZfstats(poolName string) (uint64, uint64, uint64, error) {
	dataset, err := zfs.GetDataset(poolName)
	if err != nil {
		return 0, 0, 0, err
	}

	total := dataset.Used + dataset.Avail + dataset.Usedbydataset

	return total, dataset.Avail, dataset.Avail, nil
}
