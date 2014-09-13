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
	"testing"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/storage/test"
)

type memoryTestStorageDriver struct {
	storage.StorageDriver
}

func (self *memoryTestStorageDriver) StatsEq(a, b *info.ContainerStats) bool {
	return test.DefaultStatsEq(a, b)
}

func runStorageTest(f func(test.TestStorageDriver, *testing.T), t *testing.T) {
	maxSize := 200

	for N := 10; N < maxSize; N += 10 {
		testDriver := &memoryTestStorageDriver{}
		testDriver.StorageDriver = New(N, nil)
		f(testDriver, t)
	}
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

func TestRetrieveZeroStats(t *testing.T) {
	runStorageTest(test.StorageDriverTestRetrieveZeroRecentStats, t)
}
