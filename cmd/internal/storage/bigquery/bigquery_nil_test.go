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

package bigquery

import (
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
)

func TestContainerStatsToRowsNilOptionalFields(t *testing.T) {
	s := &bigqueryStorage{machineName: "testMachine"}
	cInfo := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{Name: "/test"},
	}
	// Cpu, Network and Memory all omitted - used to panic at stats.Cpu.Usage.Total.
	stats := &info.ContainerStats{Timestamp: time.Now()}
	row := s.containerStatsToRows(cInfo, stats)

	if _, ok := row[colCPUCumulativeUsage]; ok {
		t.Errorf("did not expect cpu_cumulative_usage when Cpu is nil")
	}
	if _, ok := row[colMemoryUsage]; ok {
		t.Errorf("did not expect memory_usage when Memory is nil")
	}
	if _, ok := row[colRxBytes]; ok {
		t.Errorf("did not expect rx_bytes when Network is nil")
	}
	if row[colContainerName] != "/test" {
		t.Errorf("expected container name %q, got %v", "/test", row[colContainerName])
	}
}

func TestContainerStatsToRowsCpuOnly(t *testing.T) {
	s := &bigqueryStorage{machineName: "testMachine"}
	cInfo := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{Name: "/test"},
	}
	// Cpu present, Network/Memory omitted - used to panic at stats.Network.RxBytes.
	stats := &info.ContainerStats{
		Timestamp: time.Now(),
		Cpu: &info.CpuStats{
			Usage: info.CpuUsage{
				Total:  1000,
				User:   600,
				System: 400,
			},
		},
	}
	row := s.containerStatsToRows(cInfo, stats)

	if row[colCPUCumulativeUsage] != uint64(1000) {
		t.Errorf("expected cpu_cumulative_usage=1000, got %v", row[colCPUCumulativeUsage])
	}
	if _, ok := row[colRxBytes]; ok {
		t.Errorf("did not expect rx_bytes when Network is nil")
	}
}
