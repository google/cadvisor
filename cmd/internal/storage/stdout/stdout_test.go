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

package stdout

import (
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
)

func TestAddStatsNilOptionalFields(t *testing.T) {
	driver, err := newStorage("testhost")
	if err != nil {
		t.Fatal(err)
	}

	cInfo := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{
			Name: "/test",
		},
	}
	// Cpu present, Network/Memory omitted — used to panic at stats.Network.RxBytes.
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

	if err := driver.AddStats(cInfo, stats); err != nil {
		t.Fatalf("AddStats: %v", err)
	}
}

func TestAddStatsNilStats(t *testing.T) {
	driver, err := newStorage("testhost")
	if err != nil {
		t.Fatal(err)
	}
	cInfo := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{Name: "/test"},
	}
	if err := driver.AddStats(cInfo, nil); err != nil {
		t.Fatalf("AddStats(nil): %v", err)
	}
}
