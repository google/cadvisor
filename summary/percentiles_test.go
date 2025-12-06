// Copyright 2015 Google Inc. All Rights Reserved.
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

package summary

import (
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v2"
)

const Nanosecond = 1000000000

func assertPercentile(t *testing.T, s Uint64Slice, f float64, want uint64) {
	if got := s.GetPercentile(f); got != want {
		t.Errorf("GetPercentile(%f) is %d, should be %d.", f, got, want)
	}
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name string
		data Uint64Slice
		p    float64
		want uint64
	}{

		{"20p of 100 elements", generateSlice(100, true), 0.2, 20},
		{"70p of 100 elements", generateSlice(100, true), 0.7, 70},
		{"90p of 100 elements", generateSlice(100, true), 0.9, 90},

		// 90p should be between 94 and 95. Promoted to 95.
		{"20p of 105 elements", generateSlice(105, true), 0.2, 21},
		{"70p of 105 elements", generateSlice(105, true), 0.7, 74},
		{"90p of 105 elements", generateSlice(105, true), 0.9, 95},

		// boundary value
		{"90p of 5 elements", Uint64Slice{1, 2, 3, 4, 5}, 0.9, 5},
		{"Out of range p > 1", Uint64Slice{1, 2, 3, 4, 5}, 1.1, 0},
		{"Empty slice", Uint64Slice{}, 0, 0},
		{"Zero percentile", generateSlice(100, true), 0, 1},
		{"100p of 11 elements", Uint64Slice{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, 1.0, 11},
		{"100p of 105 elements", generateSlice(105, true), 1.0, 105},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPercentile(t, tt.data, tt.p, tt.want)
		})
	}
}

func generateSlice(n int, descending bool) Uint64Slice {
	s := make(Uint64Slice, 0, n)
	for i := 1; i <= n; i++ {
		if descending {
			s = append(s, uint64(n-i+1))
		} else {
			s = append(s, uint64(i))
		}
	}
	return s
}

func TestMean(t *testing.T) {
	var i, N uint64
	N = 100
	mean := mean{count: 0, Mean: 0}
	for i = 1; i < N; i++ {
		mean.Add(i)
	}
	if mean.Mean != 50.0 {
		t.Errorf("Mean is %f, should be 50.0", mean.Mean)
	}
}

func TestAggregates(t *testing.T) {
	N := uint64(100)
	var i uint64
	ct := time.Now()
	stats := make([]*secondSample, 0, N)
	for i = 1; i < N; i++ {
		s := &secondSample{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
			// cpu rate is 1 s/s
			Cpu: i * Nanosecond,
			// Memory grows by a KB every second.
			Memory: i * 1024,
		}
		stats = append(stats, s)
	}
	usage := GetMinutePercentiles(stats)
	// Cpu mean, max, and 90p should all be 1000 ms/s.
	cpuExpected := info.Percentiles{
		Present:    true,
		Mean:       1000,
		Max:        1000,
		Fifty:      1000,
		Ninety:     1000,
		NinetyFive: 1000,
	}
	if usage.Cpu != cpuExpected {
		t.Errorf("cpu stats are %+v. Expected %+v", usage.Cpu, cpuExpected)
	}
	memExpected := info.Percentiles{
		Present:    true,
		Mean:       50 * 1024,
		Max:        99 * 1024,
		Fifty:      50 * 1024,
		Ninety:     90 * 1024,
		NinetyFive: 95 * 1024,
	}
	if usage.Memory != memExpected {
		t.Errorf("memory stats are mean %+v. Expected %+v", usage.Memory, memExpected)
	}
}
func TestSamplesCloseInTimeIgnored(t *testing.T) {
	N := uint64(100)
	var i uint64
	ct := time.Now()
	stats := make([]*secondSample, 0, N*2)
	for i = 1; i < N; i++ {
		s1 := &secondSample{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
			// cpu rate is 1 s/s
			Cpu: i * Nanosecond,
			// Memory grows by a KB every second.
			Memory: i * 1024,
		}
		stats = append(stats, s1)

		// Add another dummy sample too close in time to the last one.
		s2 := &secondSample{
			// Add extra millisecond.
			Timestamp: ct.Add(time.Duration(i) * time.Second).Add(time.Duration(1) * time.Millisecond),
			Cpu:       i * 100 * Nanosecond,
			Memory:    i * 1024 * 1024,
		}
		stats = append(stats, s2)
	}
	usage := GetMinutePercentiles(stats)
	// Cpu mean, max, and 90p should all be 1000 ms/s. All high-value samples are discarded.
	cpuExpected := info.Percentiles{
		Present:    true,
		Mean:       1000,
		Max:        1000,
		Fifty:      1000,
		Ninety:     1000,
		NinetyFive: 1000,
	}
	if usage.Cpu != cpuExpected {
		t.Errorf("cpu stats are %+v. Expected %+v", usage.Cpu, cpuExpected)
	}
	memExpected := info.Percentiles{
		Present:    true,
		Mean:       50 * 1024,
		Max:        99 * 1024,
		Fifty:      50 * 1024,
		Ninety:     90 * 1024,
		NinetyFive: 95 * 1024,
	}
	if usage.Memory != memExpected {
		t.Errorf("memory stats are mean %+v. Expected %+v", usage.Memory, memExpected)
	}
}

func TestDerivedStats(t *testing.T) {
	N := uint64(100)
	var i uint64
	stats := make([]*info.Usage, 0, N)
	for i = 1; i < N; i++ {
		s := &info.Usage{
			PercentComplete: 100,
			Cpu: info.Percentiles{
				Present:    true,
				Mean:       i * Nanosecond,
				Max:        i * Nanosecond,
				Fifty:      i * Nanosecond,
				Ninety:     i * Nanosecond,
				NinetyFive: i * Nanosecond,
			},
			Memory: info.Percentiles{
				Present:    true,
				Mean:       i * 1024,
				Max:        i * 1024,
				Fifty:      i * 1024,
				Ninety:     i * 1024,
				NinetyFive: i * 1024,
			},
		}
		stats = append(stats, s)
	}
	usage := GetDerivedPercentiles(stats)
	cpuExpected := info.Percentiles{
		Present:    true,
		Mean:       50 * Nanosecond,
		Max:        99 * Nanosecond,
		Fifty:      50 * Nanosecond,
		Ninety:     90 * Nanosecond,
		NinetyFive: 95 * Nanosecond,
	}
	if usage.Cpu != cpuExpected {
		t.Errorf("cpu stats are %+v. Expected %+v", usage.Cpu, cpuExpected)
	}
	memExpected := info.Percentiles{
		Present:    true,
		Mean:       50 * 1024,
		Max:        99 * 1024,
		Fifty:      50 * 1024,
		Ninety:     90 * 1024,
		NinetyFive: 95 * 1024,
	}
	if usage.Memory != memExpected {
		t.Errorf("memory stats are mean %+v. Expected %+v", usage.Memory, memExpected)
	}
}
