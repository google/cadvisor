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

package info

import (
	"testing"
	"time"
)

func TestStatsStartTime(t *testing.T) {
	N := 10
	stats := make([]*ContainerStats, 0, N)
	ct := time.Now()
	for i := 0; i < N; i++ {
		s := &ContainerStats{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
		}
		stats = append(stats, s)
	}
	cinfo := &ContainerInfo{
		Name:  "/some/container",
		Stats: stats,
	}
	ref := ct.Add(time.Duration(N-1) * time.Second)
	end := cinfo.StatsEndTime()

	if !ref.Equal(end) {
		t.Errorf("end time is %v; should be %v", end, ref)
	}
}

func TestStatsEndTime(t *testing.T) {
	N := 10
	stats := make([]*ContainerStats, 0, N)
	ct := time.Now()
	for i := 0; i < N; i++ {
		s := &ContainerStats{
			Timestamp: ct.Add(time.Duration(i) * time.Second),
		}
		stats = append(stats, s)
	}
	cinfo := &ContainerInfo{
		Name:  "/some/container",
		Stats: stats,
	}
	ref := ct
	start := cinfo.StatsStartTime()

	if !ref.Equal(start) {
		t.Errorf("start time is %v; should be %v", start, ref)
	}
}

func TestPercentiles(t *testing.T) {
	N := 100
	data := make([]uint64, N)

	for i := 0; i < N; i++ {
		data[i] = uint64(i)
	}
	ps := []int{
		80,
		90,
		50,
	}
	ss := uint64Slice(data).Percentiles(ps...)
	for i, s := range ss {
		p := ps[i]
		d := uint64(float64(N) * (float64(p) / 100.0))
		if d != s {
			t.Errorf("%v \\%tile data should be %v, but got %v", float64(p)/100.0, d, s)
		}
	}
}
