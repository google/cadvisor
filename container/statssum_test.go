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
	"testing"

	"github.com/google/cadvisor/info"
)

func init() {
	// NOTE(dengnan): Even if we picked a good random seed,
	// the random number from math/rand is still not cryptographically secure!
	var seed int64
	binary.Read(crand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)
}

type randomMemoryUsageContainer struct {
	NoStatsSummary
}

func (self *randomMemoryUsageContainer) GetSpec() (*info.ContainerSpec, error) {
	return nil, nil
}

func (self *randomMemoryUsageContainer) GetStats() (*info.ContainerStats, error) {
	stats := new(info.ContainerStats)
	stats.Cpu = new(info.CpuStats)
	stats.Memory = new(info.MemoryStats)
	stats.Memory.Usage = uint64(rand.Intn(2048))
	return stats, nil
}

func (self *randomMemoryUsageContainer) ListContainers(listType ListType) ([]string, error) {
	return nil, nil
}

func (self *randomMemoryUsageContainer) ListThreads(listType ListType) ([]int, error) {
	return nil, nil
}

func (self *randomMemoryUsageContainer) ListProcesses(listType ListType) ([]int, error) {
	return nil, nil
}

func TestAvgMaxMemoryUsage(t *testing.T) {
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
	var totalUsage uint64
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
		totalUsage += stats.Memory.Usage
	}
	summary, err := handler.StatsSummary()
	if err != nil {
		t.Fatalf("Error when get summary: %v", err)
	}
	if summary.MaxMemoryUsage != maxUsage {
		t.Fatalf("Max memory usage should be %v; received %v", maxUsage, summary.MaxMemoryUsage)
	}
	avg := totalUsage / uint64(N)
	if summary.AvgMemoryUsage != avg {
		t.Fatalf("Avg memory usage should be %v; received %v", avg, summary.AvgMemoryUsage)
	}
}
