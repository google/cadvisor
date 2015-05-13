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
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const containerName = "/container"

var (
	containerRef = info.ContainerReference{Name: containerName}
	zero         time.Time
)

// Make stats with the specified identifier.
func makeStat(i int) *info.ContainerStats {
	return &info.ContainerStats{
		Timestamp: zero.Add(time.Duration(i) * time.Second),
		Cpu: info.CpuStats{
			LoadAverage: int32(i),
		},
	}
}

func getRecentStats(t *testing.T, memoryStorage *InMemoryStorage, numStats int) []*info.ContainerStats {
	stats, err := memoryStorage.RecentStats(containerName, zero, zero, numStats)
	require.Nil(t, err)
	return stats
}

func TestAddStats(t *testing.T) {
	memoryStorage := New(60*time.Second, nil)

	assert := assert.New(t)
	assert.Nil(memoryStorage.AddStats(containerRef, makeStat(0)))
	assert.Nil(memoryStorage.AddStats(containerRef, makeStat(1)))
	assert.Nil(memoryStorage.AddStats(containerRef, makeStat(2)))
	assert.Nil(memoryStorage.AddStats(containerRef, makeStat(0)))
	containerRef2 := info.ContainerReference{
		Name: "/container2",
	}
	assert.Nil(memoryStorage.AddStats(containerRef2, makeStat(0)))
	assert.Nil(memoryStorage.AddStats(containerRef2, makeStat(1)))
}

func TestRecentStatsNoRecentStats(t *testing.T) {
	memoryStorage := makeWithStats(0)

	_, err := memoryStorage.RecentStats(containerName, zero, zero, 60)
	assert.NotNil(t, err)
}

// Make an instance of InMemoryStorage with n stats.
func makeWithStats(n int) *InMemoryStorage {
	memoryStorage := New(60*time.Second, nil)

	for i := 0; i < n; i++ {
		memoryStorage.AddStats(containerRef, makeStat(i))
	}
	return memoryStorage
}

func TestRecentStatsGetZeroStats(t *testing.T) {
	memoryStorage := makeWithStats(10)

	assert.Len(t, getRecentStats(t, memoryStorage, 0), 0)
}

func TestRecentStatsGetSomeStats(t *testing.T) {
	memoryStorage := makeWithStats(10)

	assert.Len(t, getRecentStats(t, memoryStorage, 5), 5)
}

func TestRecentStatsGetAllStats(t *testing.T) {
	memoryStorage := makeWithStats(10)

	assert.Len(t, getRecentStats(t, memoryStorage, -1), 10)
}
