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
)

func createStats(id int32) *info.ContainerStats {
	return &info.ContainerStats{
		Cpu: info.CpuStats{
			Load: id,
		},
	}
}

func expectSize(t *testing.T, sb *StatsBuffer, expectedSize int) {
	if sb.Size() != expectedSize {
		t.Errorf("Expected size %v, got %v", expectedSize, sb.Size())
	}
}

func expectElements(t *testing.T, sb *StatsBuffer, expected []int32) {
	res := sb.FirstN(sb.Size())
	if len(res) != len(expected) {
		t.Errorf("Expected elements %v, got %v", expected, res)
		return
	}
	for i, el := range res {
		if el.Cpu.Load != expected[i] {
			t.Errorf("Expected elements %v, got %v", expected, res)
		}
	}
}

func TestAddAndFirstN(t *testing.T) {
	sb := NewStatsBuffer(5)

	// Add 1.
	sb.Add(createStats(1))
	expectSize(t, sb, 1)
	expectElements(t, sb, []int32{1})

	// Fill the buffer.
	for i := 1; i <= 5; i++ {
		expectSize(t, sb, i)
		sb.Add(createStats(int32(i)))
	}
	expectSize(t, sb, 5)
	expectElements(t, sb, []int32{1, 2, 3, 4, 5})

	// Add more than is available in the buffer
	sb.Add(createStats(6))
	expectSize(t, sb, 5)
	expectElements(t, sb, []int32{2, 3, 4, 5, 6})

	// Replace all elements.
	for i := 7; i <= 10; i++ {
		sb.Add(createStats(int32(i)))
	}
	expectSize(t, sb, 5)
	expectElements(t, sb, []int32{6, 7, 8, 9, 10})
}
