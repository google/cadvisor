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
	"testing"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/storage/memory"
	"github.com/google/cadvisor/storage/test"
)

type cacheTestStorageDriver struct {
	base storage.StorageDriver
}

func (self *cacheTestStorageDriver) StatsEq(a, b *info.ContainerStats) bool {
	return test.DefaultStatsEq(a, b)
}

func (self *cacheTestStorageDriver) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	return self.base.AddStats(ref, stats)
}

func (self *cacheTestStorageDriver) RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error) {
	return self.base.RecentStats(containerName, numStats)
}

func (self *cacheTestStorageDriver) Percentiles(containerName string, cpuUsagePercentiles []int, memUsagePercentiles []int) (*info.ContainerStatsPercentiles, error) {
	return self.base.Percentiles(containerName, cpuUsagePercentiles, memUsagePercentiles)
}

func (self *cacheTestStorageDriver) Samples(containerName string, numSamples int) ([]*info.ContainerStatsSample, error) {
	return self.base.Samples(containerName, numSamples)
}

func (self *cacheTestStorageDriver) Close() error {
	return self.base.Close()
}

func runStorageTest(f func(test.TestStorageDriver, *testing.T), t *testing.T) {
	maxSize := 200

	for N := 10; N < maxSize; N += 10 {
		testDriver := &cacheTestStorageDriver{}
		backend := memory.New(N*2, N*2)
		testDriver.base = MemoryCache(N, N, backend)
		f(testDriver, t)
	}

}

func TestMaxMemoryUsage(t *testing.T) {
	runStorageTest(test.StorageDriverTestMaxMemoryUsage, t)
}

func TestSampleCpuUsage(t *testing.T) {
	runStorageTest(test.StorageDriverTestSampleCpuUsage, t)
}

func TestSamplesWithoutSample(t *testing.T) {
	runStorageTest(test.StorageDriverTestSamplesWithoutSample, t)
}

func TestPercentilesWithoutSample(t *testing.T) {
	runStorageTest(test.StorageDriverTestPercentilesWithoutSample, t)
}

func TestPercentiles(t *testing.T) {
	N := 100
	testDriver := &cacheTestStorageDriver{}
	backend := memory.New(N*2, N*2)
	testDriver.base = MemoryCache(N, N, backend)
	test.StorageDriverTestPercentiles(testDriver, t)
}

func TestRetrievePartialRecentStats(t *testing.T) {
	runStorageTest(test.StorageDriverTestRetrievePartialRecentStats, t)
}

func TestRetrieveAllRecentStats(t *testing.T) {
	runStorageTest(test.StorageDriverTestRetrieveAllRecentStats, t)
}

func TestNoRecentStats(t *testing.T) {
	runStorageTest(test.StorageDriverTestNoRecentStats, t)
}

func TestNoSamples(t *testing.T) {
	runStorageTest(test.StorageDriverTestNoSamples, t)
}

func TestPercentilesWithoutStats(t *testing.T) {
	runStorageTest(test.StorageDriverTestPercentilesWithoutStats, t)
}
