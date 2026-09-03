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

package influxdb

import (
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
)

func TestContainerStatsToPointsNilOptionalFields(t *testing.T) {
	s := &influxdbStorage{machineName: "testMachine"}
	cInfo := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{Name: "/test"},
	}
	// Cpu, Network and Memory all omitted - used to panic at stats.Cpu.Usage.Total.
	stats := &info.ContainerStats{Timestamp: time.Now()}
	points := s.containerStatsToPoints(cInfo, stats)
	points = append(points, s.memoryStatsToPoints(cInfo, stats)...)

	// Only the ReferencedMemory point survives.
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Measurement != serReferencedMemory {
		t.Errorf("expected measurement %q, got %q", serReferencedMemory, points[0].Measurement)
	}
}

func TestContainerStatsToPointsCpuOnly(t *testing.T) {
	s := &influxdbStorage{machineName: "testMachine"}
	cInfo := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{Name: "/test"},
	}
	// Cpu present, Network omitted - used to panic at stats.Network.RxBytes.
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
	points := s.containerStatsToPoints(cInfo, stats)
	points = append(points, s.memoryStatsToPoints(cInfo, stats)...)

	// CPU total/system/user/load average + referenced memory. No network points.
	if len(points) != 5 {
		t.Fatalf("expected 5 points, got %d", len(points))
	}
	for _, p := range points {
		if p.Measurement == serRxBytes {
			t.Errorf("did not expect rx_bytes point when Network is nil")
		}
	}
}
