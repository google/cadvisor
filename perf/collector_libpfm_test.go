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
	"testing"

	"github.com/stretchr/testify/assert"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/stats"
)

type buffer struct {
	*bytes.Buffer
}

func (b buffer) Close() error {
	return nil
}

func TestCollector_UpdateStats(t *testing.T) {
	collector := collector{uncore: &stats.NoopCollector{}}
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

	assert.Contains(t, stats.PerfStats, info.PerfStat{
		PerfValue: info.PerfValue{
			ScalingRatio: 0.3333333333333333,
			Value:        999999999,
			Name:         "cycles",
		},
		Cpu: 11,
	})
	assert.Contains(t, stats.PerfStats, info.PerfStat{
		PerfValue: info.PerfValue{
			ScalingRatio: 1,
			Value:        123456789,
			Name:         "instructions",
		},
		Cpu: 0,
	})
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
	perfCollector := newCollector("cgroup", PerfEvents{
		Core: Events{
			Events: [][]Event{{"event_1"}, {"event_2"}},
			CustomEvents: []CustomEvent{{
				Type:   0,
				Config: []uint64{1, 2, 3},
				Name:   "event_2",
			}},
		},
	}, 1, []info.Node{})
	assert.Len(t, perfCollector.eventToCustomEvent, 1)
	assert.Nil(t, perfCollector.eventToCustomEvent[Event("event_1")])
	assert.Same(t, &perfCollector.events.Core.CustomEvents[0], perfCollector.eventToCustomEvent[Event("event_2")])
}

var readPerfStatCases = []struct {
	test     string
	file     ReadFormat
	name     string
	cpu      int
	perfStat info.PerfStat
	err      error
}{
	{
		test: "no scaling",
		file: ReadFormat{
			Value:       5,
			TimeEnabled: 0,
			TimeRunning: 0,
			ID:          0,
		},
		name: "some metric",
		cpu:  1,
		perfStat: info.PerfStat{
			PerfValue: info.PerfValue{
				ScalingRatio: 1,
				Value:        5,
				Name:         "some metric",
			},
			Cpu: 1,
		},
		err: nil,
	},
	{
		test: "no scaling - TimeEnabled = 0",
		file: ReadFormat{
			Value:       5,
			TimeEnabled: 0,
			TimeRunning: 1,
			ID:          0,
		},
		name: "some metric",
		cpu:  1,
		perfStat: info.PerfStat{
			PerfValue: info.PerfValue{
				ScalingRatio: 1,
				Value:        5,
				Name:         "some metric",
			},
			Cpu: 1,
		},
		err: nil,
	},
	{
		test: "scaling - 0.5",
		file: ReadFormat{
			Value:       4,
			TimeEnabled: 4,
			TimeRunning: 2,
			ID:          0,
		},
		name: "some metric",
		cpu:  2,
		perfStat: info.PerfStat{
			PerfValue: info.PerfValue{
				ScalingRatio: 0.5,
				Value:        8,
				Name:         "some metric",
			},
			Cpu: 2,
		},
		err: nil,
	},
	{
		test: "scaling - 0 (TimeEnabled = 1, TimeRunning = 0)",
		file: ReadFormat{
			Value:       4,
			TimeEnabled: 1,
			TimeRunning: 0,
			ID:          0,
		},
		name: "some metric",
		cpu:  3,
		perfStat: info.PerfStat{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        4,
				Name:         "some metric",
			},
			Cpu: 3,
		},
		err: nil,
	},
	{
		test: "scaling - 0 (TimeEnabled = 0, TimeRunning = 1)",
		file: ReadFormat{
			Value:       4,
			TimeEnabled: 0,
			TimeRunning: 1,
			ID:          0,
		},
		name: "some metric",
		cpu:  3,
		perfStat: info.PerfStat{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        4,
				Name:         "some metric",
			},
			Cpu: 3,
		},
		err: nil,
	},
	{
		test: "zeros, zeros everywhere",
		file: ReadFormat{
			Value:       0,
			TimeEnabled: 0,
			TimeRunning: 0,
			ID:          0,
		},
		name: "some metric",
		cpu:  4,
		perfStat: info.PerfStat{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        0,
				Name:         "some metric",
			},
			Cpu: 4,
		},
		err: nil,
	},
	{
		test: "non-zero TimeRunning",
		file: ReadFormat{
			Value:       0,
			TimeEnabled: 0,
			TimeRunning: 3,
			ID:          0,
		},
		name: "some metric",
		cpu:  4,
		perfStat: info.PerfStat{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        0,
				Name:         "some metric",
			},
			Cpu: 4,
		},
		err: nil,
	},
}

func TestReadPerfStat(t *testing.T) {
	for _, test := range readPerfStatCases {
		t.Run(test.test, func(tt *testing.T) {
			buf := &buffer{bytes.NewBuffer([]byte{})}
			err := binary.Write(buf, binary.LittleEndian, test.file)
			assert.NoError(tt, err)
			stat, err := readPerfStat(buf, test.name, test.cpu)
			assert.Equal(tt, test.perfStat, *stat)
			assert.Equal(tt, test.err, err)
		})
	}
}
