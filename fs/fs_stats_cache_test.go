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
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testStatsHelper struct {
	CacheMissCount uint32
}

func (self *testStatsHelper) GetVfsStats(path string) (uint64, uint64, uint64, uint64, uint64, error) {
	atomic.AddUint32(&self.CacheMissCount, 1)

	if path == "/" {
		return 1000, 400, 500, 100, 20, nil
	}

	return 0, 0, 0, 0, 0, fmt.Errorf("Unexpected VFS mountpoint: %q", path)
}

func (self *testStatsHelper) GetZfstats(poolName string) (uint64, uint64, uint64, error) {
	atomic.AddUint32(&self.CacheMissCount, 1)

	if poolName == "some-pool" {
		return 2000, 1000, 1500, nil
	}

	return 0, 0, 0, fmt.Errorf("Unexpected ZFS pool: %q", poolName)
}

func (self *testStatsHelper) GetDmStats(poolName string, dataBlkSize uint) (uint64, uint64, uint64, error) {
	atomic.AddUint32(&self.CacheMissCount, 1)

	if poolName == "some-dm" && dataBlkSize == 2048 {
		return 1500, 100, 200, nil
	}

	return 0, 0, 0, fmt.Errorf("Unexpected DM pool or block size: %q %d", poolName, dataBlkSize)
}

var testCases = []struct {
	device           string
	fsType           string
	mountpoint       string
	blockSize        uint
	expectFsType     FsType
	expectCapacity   uint64
	expectFree       uint64
	expectAvailable  uint64
	expectInodes     uint64
	expectInodesFree uint64
	expectErr        bool
}{
	{
		device:           "/dev/sda1",
		fsType:           "ext4",
		mountpoint:       "/",
		expectFsType:     VFS,
		expectCapacity:   1000,
		expectFree:       400,
		expectAvailable:  500,
		expectInodes:     100,
		expectInodesFree: 20,
		expectErr:        false,
	},
	{
		device:           "/dev/sda1",
		fsType:           "ext4",
		mountpoint:       "/does-not-exist",
		expectFsType:     "",
		expectCapacity:   0,
		expectFree:       0,
		expectAvailable:  0,
		expectInodes:     0,
		expectInodesFree: 0,
		expectErr:        true,
	},
	{
		device:           "some-pool",
		fsType:           "zfs",
		expectFsType:     ZFS,
		expectCapacity:   2000,
		expectFree:       1000,
		expectAvailable:  1500,
		expectInodes:     0,
		expectInodesFree: 0,
		expectErr:        false,
	},
	{
		device:           "some-dm",
		blockSize:        2048,
		expectFsType:     DeviceMapper,
		fsType:           "devicemapper",
		expectCapacity:   1500,
		expectFree:       100,
		expectAvailable:  200,
		expectInodes:     0,
		expectInodesFree: 0,
		expectErr:        false,
	},
}

func TestCaching(t *testing.T) {
	as := assert.New(t)

	for _, testCase := range testCases {
		helper := testStatsHelper{}
		fsCache := newFsStatsCache(time.Minute, &helper)
		as.Equal(uint32(0), helper.CacheMissCount)

		part := partition{
			fsType:     testCase.fsType,
			mountpoint: testCase.mountpoint,
			blockSize:  testCase.blockSize,
		}

		fsType, capacity, free, available, inodes, inodesFree, err := fsCache.FsStats(testCase.device, part)
		as.Equal(uint32(1), helper.CacheMissCount)

		if testCase.expectErr {
			as.Error(err)
			_, _, _, _, _, _, err = fsCache.FsStats(testCase.device, part)
			as.Error(err)
			as.Equal(uint32(2), helper.CacheMissCount)
			continue
		}

		as.NoError(err)

		as.Equal(testCase.expectFsType, fsType)
		as.Equal(testCase.expectCapacity, capacity)
		as.Equal(testCase.expectFree, free)
		as.Equal(testCase.expectAvailable, available)
		as.Equal(testCase.expectInodes, inodes)
		as.Equal(testCase.expectInodesFree, inodesFree)

		_, _, _, _, _, _, err = fsCache.FsStats(testCase.device, part)
		as.Equal(uint32(1), helper.CacheMissCount)
	}
}

func TestCacheExpiry(t *testing.T) {
	as := assert.New(t)

	for _, testCase := range testCases {
		helper := testStatsHelper{}
		fsCache := newFsStatsCache(0, &helper)
		as.Equal(uint32(0), helper.CacheMissCount)

		part := partition{
			fsType:     testCase.fsType,
			mountpoint: testCase.mountpoint,
			blockSize:  testCase.blockSize,
		}

		fsCache.FsStats(testCase.device, part)
		as.Equal(uint32(1), helper.CacheMissCount)

		fsCache.FsStats(testCase.device, part)
		as.Equal(uint32(2), helper.CacheMissCount)
	}
}

func TestRace(t *testing.T) {
	as := assert.New(t)

	helper := testStatsHelper{}
	fsCache := newFsStatsCache(0, &helper)

	goroutinesPerTestCase := 20

	wg := sync.WaitGroup{}
	wg.Add(len(testCases) * goroutinesPerTestCase)

	for _, testCase := range testCases {
		// Copy into local variables to avoid tripping the race detector
		device := testCase.device
		part := partition{
			fsType:     testCase.fsType,
			mountpoint: testCase.mountpoint,
			blockSize:  testCase.blockSize,
		}

		for i := 0; i < goroutinesPerTestCase; i++ {
			go func() {
				defer wg.Done()

				fsCache.FsStats(device, part)
			}()
		}
	}

	wg.Wait()
	as.Equal(uint(len(testCases)*goroutinesPerTestCase), helper.CacheMissCount)
}
