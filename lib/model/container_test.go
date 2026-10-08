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

package model_test

import (
	"testing"

	"github.com/google/cadvisor/lib/model"
)

// TestStatsEqMonotonicCounters demonstrates that StatsEq returns false
// when only monotonic memory pressure counters change, preventing
// cadvisor from backing off housekeeping frequency.
func TestStatsEqMonotonicCounters(t *testing.T) {
	base := &model.ContainerStats{
		Memory: &model.MemoryStats{
			Usage:      100 << 20,
			Cache:      10 << 20,
			RSS:        90 << 20,
			WorkingSet: 85 << 20,
			// Monotonic counters set by kernel reclaim when memory.high is active
			Pgscan:                1000,
			Pgsteal:               800,
			WorkingsetRefaultFile: 50,
			WorkingsetRefaultAnon: 30,
			Events:                model.MemoryEvents{High: 5},
		},
	}

	// Simulate the next housekeeping tick: only counters changed
	next := &model.ContainerStats{
		Memory: &model.MemoryStats{
			Usage:      100 << 20,
			Cache:      10 << 20,
			RSS:        90 << 20,
			WorkingSet: 85 << 20,
			// Counters incremented by kernel — usage is identical
			Pgscan:                1042,
			Pgsteal:               833,
			WorkingsetRefaultFile: 52,
			WorkingsetRefaultAnon: 31,
			Events:                model.MemoryEvents{High: 6},
		},
	}

	if base.StatsEq(next) {
		t.Log("StatsEq correctly treats counter-only changes as equal")
	} else {
		t.Error("StatsEq returns false when only monotonic counters changed; " +
			"cadvisor will never back off housekeeping beyond 1s")
	}
}
