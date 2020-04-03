// +build libpfm,cgo

// Copyright 2020 Google Inc. All Rights Reserved.
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

// Collector of perf events for a container.
package perf

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"testing"

	info "github.com/google/cadvisor/info/v1"
)

type buffer struct {
	*bytes.Buffer
}

func (b buffer) Close() error {
	return nil
}

func TestCollector_UpdateStats(t *testing.T) {
	collector := collector{}
	notScaledBuffer := buffer{bytes.NewBuffer([]byte{})}
	scaledBuffer := buffer{bytes.NewBuffer([]byte{})}
	binary.Write(notScaledBuffer, binary.LittleEndian, ReadFormat{
		Value:       123456789,
		TimeEnabled: 100,
		TimeRunning: 100,
		ID:          1,
	})
	binary.Write(scaledBuffer, binary.LittleEndian, ReadFormat{
		Value:       333333333,
		TimeEnabled: 1,
		TimeRunning: 3,
		ID:          2,
	})
	collector.perfFiles = map[string][]ReaderCloser{
		"instructions": {notScaledBuffer},
		"cycles":       {scaledBuffer},
	}

	stats := &info.ContainerStats{}
	collector.UpdateStats(stats)

	assert.Len(t, stats.PerfStats, 2)
	assert.Equal(t, uint64(123456789), stats.PerfStats[0].Value)
	assert.Equal(t, "instructions", stats.PerfStats[0].Name, "wrong value of instructions perf event; expected: 123456789, actual: %d", stats.PerfStats[0].Value)
	assert.Equal(t, float64(1), stats.PerfStats[0].ScalingRatio)
	assert.NotZero(t, stats.PerfStats[0].Time)
	assert.Equal(t, uint64(111111111), stats.PerfStats[1].Value, "wrong value of cycles perf event; expected: 111111111, actual: %d", stats.PerfStats[1].Value)
	assert.Equal(t, "cycles", stats.PerfStats[1].Name)
	assert.InDelta(t, float64(0.333333333), stats.PerfStats[1].ScalingRatio, 0.0000000004)
	assert.NotZero(t, stats.PerfStats[0].Time)
}
