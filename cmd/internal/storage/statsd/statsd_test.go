// Copyright 2026 Google Inc. All Rights Reserved.
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

package statsd

import (
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
)

func TestContainerStatsToValuesNilOptionalFields(t *testing.T) {
	s := &statsdStorage{}
	// Cpu, Network and Memory all omitted - used to panic at stats.Cpu.Usage.Total.
	stats := &info.ContainerStats{Timestamp: time.Now()}
	values := s.containerStatsToValues(stats)
	s.memoryStatsToValues(&values, stats)

	if _, ok := values[serCPUUsageTotal]; ok {
		t.Errorf("did not expect cpu_usage_total when Cpu is nil")
	}
	if _, ok := values[serRxBytes]; ok {
		t.Errorf("did not expect rx_bytes when Network is nil")
	}
	if _, ok := values[serMemoryUsage]; ok {
		t.Errorf("did not expect memory_usage when Memory is nil")
	}
}

func TestContainerStatsToValuesPresent(t *testing.T) {
	s := &statsdStorage{}
	stats := &info.ContainerStats{
		Timestamp: time.Now(),
		Cpu: &info.CpuStats{
			Usage: info.CpuUsage{
				Total:  1000,
				User:   600,
				System: 400,
			},
		},
		Memory: &info.MemoryStats{Usage: 2048},
		Network: &info.NetworkStats{
			InterfaceStats: info.InterfaceStats{RxBytes: 1, TxBytes: 2},
		},
	}
	values := s.containerStatsToValues(stats)
	s.memoryStatsToValues(&values, stats)

	if values[serCPUUsageTotal] != 1000 {
		t.Errorf("expected cpu_usage_total=1000, got %d", values[serCPUUsageTotal])
	}
	if values[serMemoryUsage] != 2048 {
		t.Errorf("expected memory_usage=2048, got %d", values[serMemoryUsage])
	}
	if values[serRxBytes] != 1 {
		t.Errorf("expected rx_bytes=1, got %d", values[serRxBytes])
	}
}
