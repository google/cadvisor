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
	err := binary.Write(notScaledBuffer, binary.LittleEndian, ReadFormat{
		Value:       123456789,
		TimeEnabled: 100,
		TimeRunning: 100,
		ID:          1,
	})
	assert.NoError(t, err)
	err = binary.Write(scaledBuffer, binary.LittleEndian, ReadFormat{
		Value:       333333333,
		TimeEnabled: 3,
		TimeRunning: 1,
		ID:          2,
	})
	assert.NoError(t, err)
	collector.cpuFiles = map[string]map[int]readerCloser{
		"instructions": {0: notScaledBuffer},
		"cycles":       {11: scaledBuffer},
	}

	stats := &info.ContainerStats{}
	err = collector.UpdateStats(stats)

	assert.NoError(t, err)
	assert.Len(t, stats.PerfStats, 2)

	assert.Equal(t, uint64(123456789), stats.PerfStats[0].Value)
	assert.Equal(t, "instructions", stats.PerfStats[0].Name, "wrong value of instructions perf event; expected: 123456789, actual: %d", stats.PerfStats[0].Value)
	assert.Equal(t, float64(1), stats.PerfStats[0].ScalingRatio)
	assert.Equal(t, 0, stats.PerfStats[0].Cpu)

	assert.Equal(t, uint64(999999999), stats.PerfStats[1].Value, "wrong value of cycles perf event; expected: 999999999, actual: %d", stats.PerfStats[1].Value)
	assert.Equal(t, "cycles", stats.PerfStats[1].Name)
	assert.InDelta(t, float64(0.333333333), stats.PerfStats[1].ScalingRatio, 0.0000000004)
	assert.Equal(t, 11, stats.PerfStats[1].Cpu)
}

func TestCreatePerfEventAttr(t *testing.T) {
	event := CustomEvent{
		Type:   0x1,
		Config: Config{uint64(0x2), uint64(0x3), uint64(0x4)},
		Name:   "fake_event",
	}

	attributes := createPerfEventAttr(event)

	assert.Equal(t, uint32(1), attributes.Type)
	assert.Equal(t, uint64(2), attributes.Config)
	assert.Equal(t, uint64(3), attributes.Ext1)
	assert.Equal(t, uint64(4), attributes.Ext2)
	assert.Equal(t, uint64(65536), attributes.Sample_type)
	assert.Equal(t, uint64(7), attributes.Read_format)
	assert.Equal(t, uint64(1048578), attributes.Bits)
}

func TestNewCollector(t *testing.T) {
	perfCollector := newCollector("cgroup", Events{
		Events: [][]Event{{"event_1"}, {"event_2"}},
		CustomEvents: []CustomEvent{{
			Type:   0,
			Config: []uint64{1, 2, 3},
			Name:   "event_2",
		}},
	}, 1)
	assert.Len(t, perfCollector.eventToCustomEvent, 1)
	assert.Nil(t, perfCollector.eventToCustomEvent[Event("event_1")])
	assert.Same(t, &perfCollector.events.CustomEvents[0], perfCollector.eventToCustomEvent[Event("event_2")])
}
