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
	"io/ioutil"
	"os"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"

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
	groupedBuffer := buffer{bytes.NewBuffer([]byte{})}
	err := binary.Write(notScaledBuffer, binary.LittleEndian, GroupReadFormat{
		Nr:          1,
		TimeEnabled: 100,
		TimeRunning: 100,
	})
	assert.NoError(t, err)
	err = binary.Write(notScaledBuffer, binary.LittleEndian, Values{
		Value: 123456789,
		ID:    0,
	})
	assert.NoError(t, err)
	err = binary.Write(scaledBuffer, binary.LittleEndian, GroupReadFormat{
		Nr:          1,
		TimeEnabled: 3,
		TimeRunning: 1,
	})
	assert.NoError(t, err)
	err = binary.Write(scaledBuffer, binary.LittleEndian, Values{
		Value: 333333333,
		ID:    2,
	})
	assert.NoError(t, err)
	err = binary.Write(groupedBuffer, binary.LittleEndian, GroupReadFormat{
		Nr:          2,
		TimeEnabled: 100,
		TimeRunning: 100,
	})
	assert.NoError(t, err)
	err = binary.Write(groupedBuffer, binary.LittleEndian, Values{
		Value: 123456,
		ID:    0,
	})
	assert.NoError(t, err)
	err = binary.Write(groupedBuffer, binary.LittleEndian, Values{
		Value: 654321,
		ID:    1,
	})
	assert.NoError(t, err)

	collector.cpuFiles = map[int]group{
		1: {
			cpuFiles: map[string]map[int]readerCloser{
				"instructions": {0: notScaledBuffer},
			},
			names:      []string{"instructions"},
			leaderName: "instructions",
		},
		2: {
			cpuFiles: map[string]map[int]readerCloser{
				"cycles": {11: scaledBuffer},
			},
			names:      []string{"cycles"},
			leaderName: "cycles",
		},
		3: {
			cpuFiles: map[string]map[int]readerCloser{
				"cache-misses": {
					0: groupedBuffer,
				},
			},
			names:      []string{"cache-misses", "cache-references"},
			leaderName: "cache-misses",
		},
	}

	stats := &info.ContainerStats{}
	err = collector.UpdateStats(stats)

	assert.NoError(t, err)
	assert.Len(t, stats.PerfStats, 4)

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
	assert.Contains(t, stats.PerfStats, info.PerfStat{
		PerfValue: info.PerfValue{
			ScalingRatio: 1.0,
			Value:        123456,
			Name:         "cache-misses",
		},
		Cpu: 0,
	})
	assert.Contains(t, stats.PerfStats, info.PerfStat{
		PerfValue: info.PerfValue{
			ScalingRatio: 1.0,
			Value:        654321,
			Name:         "cache-references",
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
}

func TestSetGroupAttributes(t *testing.T) {
	event := CustomEvent{
		Type:   0x1,
		Config: Config{uint64(0x2), uint64(0x3), uint64(0x4)},
		Name:   "fake_event",
	}

	attributes := createPerfEventAttr(event)
	setAttributes(attributes, true)
	assert.Equal(t, uint64(65536), attributes.Sample_type)
	assert.Equal(t, uint64(0xf), attributes.Read_format)
	assert.Equal(t, uint64(0x3), attributes.Bits)

	attributes = createPerfEventAttr(event)
	setAttributes(attributes, false)
	assert.Equal(t, uint64(65536), attributes.Sample_type)
	assert.Equal(t, uint64(0xf), attributes.Read_format)
	assert.Equal(t, uint64(0x2), attributes.Bits)
}

func TestNewCollector(t *testing.T) {
	perfCollector := newCollector("cgroup", PerfEvents{
		Core: Events{
			Events: []Group{{[]Event{"event_1"}, false}, {[]Event{"event_2"}, false}},
			CustomEvents: []CustomEvent{{
				Type:   0,
				Config: []uint64{1, 2, 3},
				Name:   "event_2",
			}},
		},
	}, []int{0, 1, 2, 3}, map[int]int{})
	assert.Len(t, perfCollector.eventToCustomEvent, 1)
	assert.Nil(t, perfCollector.eventToCustomEvent[Event("event_1")])
	assert.Same(t, &perfCollector.events.Core.CustomEvents[0], perfCollector.eventToCustomEvent[Event("event_2")])
}

func TestCollectorSetup(t *testing.T) {
	path, err := ioutil.TempDir("", "cgroup")
	assert.Nil(t, err)
	defer func() {
		err := os.RemoveAll(path)
		assert.Nil(t, err)
	}()
	events := PerfEvents{
		Core: Events{
			Events: []Group{
				{[]Event{"cache-misses"}, false},
				{[]Event{"non-existing-event"}, false},
			},
		},
	}
	c := newCollector(path, events, []int{0}, map[int]int{0: 0})
	c.perfEventOpen = func(attr *unix.PerfEventAttr, pid int, cpu int, groupFd int, flags int) (fd int, err error) {
		return int(attr.Config), nil
	}
	c.ioctlSetInt = func(fd int, req uint, value int) error {
		return nil
	}
	err = c.setup()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(c.cpuFiles))
	assert.Equal(t, []string{"cache-misses"}, c.cpuFiles[0].names)
}

var readGroupPerfStatCases = []struct {
	test       string
	file       GroupReadFormat
	valuesFile Values
	name       string
	cpu        int
	perfStat   []info.PerfStat
	err        error
}{
	{
		test: "no scaling",
		file: GroupReadFormat{
			TimeEnabled: 0,
			TimeRunning: 0,
			Nr:          1,
		},
		valuesFile: Values{
			Value: 5,
			ID:    0,
		},
		name: "some metric",
		cpu:  1,
		perfStat: []info.PerfStat{{
			PerfValue: info.PerfValue{
				ScalingRatio: 1,
				Value:        5,
				Name:         "some metric",
			},
			Cpu: 1,
		}},
		err: nil,
	},
	{
		test: "no scaling - TimeEnabled = 0",
		file: GroupReadFormat{
			TimeEnabled: 0,
			TimeRunning: 1,
			Nr:          1,
		},
		valuesFile: Values{
			Value: 5,
			ID:    0,
		},
		name: "some metric",
		cpu:  1,
		perfStat: []info.PerfStat{{
			PerfValue: info.PerfValue{
				ScalingRatio: 1,
				Value:        5,
				Name:         "some metric",
			},
			Cpu: 1,
		}},
		err: nil,
	},
	{
		test: "scaling - 0.5",
		file: GroupReadFormat{
			TimeEnabled: 4,
			TimeRunning: 2,
			Nr:          1,
		},
		valuesFile: Values{
			Value: 4,
			ID:    0,
		},
		name: "some metric",
		cpu:  2,
		perfStat: []info.PerfStat{{
			PerfValue: info.PerfValue{
				ScalingRatio: 0.5,
				Value:        8,
				Name:         "some metric",
			},
			Cpu: 2,
		}},
		err: nil,
	},
	{
		test: "scaling - 0 (TimeEnabled = 1, TimeRunning = 0)",
		file: GroupReadFormat{
			TimeEnabled: 1,
			TimeRunning: 0,
			Nr:          1,
		},
		valuesFile: Values{
			Value: 4,
			ID:    0,
		},
		name: "some metric",
		cpu:  3,
		perfStat: []info.PerfStat{{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        4,
				Name:         "some metric",
			},
			Cpu: 3,
		}},
		err: nil,
	},
	{
		test: "scaling - 0 (TimeEnabled = 0, TimeRunning = 1)",
		file: GroupReadFormat{
			TimeEnabled: 0,
			TimeRunning: 1,
			Nr:          1,
		},
		valuesFile: Values{
			Value: 4,
			ID:    0,
		},
		name: "some metric",
		cpu:  3,
		perfStat: []info.PerfStat{{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        4,
				Name:         "some metric",
			},
			Cpu: 3,
		}},
		err: nil,
	},
	{
		test: "zeros, zeros everywhere",
		file: GroupReadFormat{
			TimeEnabled: 0,
			TimeRunning: 0,
			Nr:          1,
		},
		valuesFile: Values{
			Value: 0,
			ID:    0,
		},
		name: "some metric",
		cpu:  4,
		perfStat: []info.PerfStat{{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        0,
				Name:         "some metric",
			},
			Cpu: 4,
		}},
		err: nil,
	},
	{
		test: "non-zero TimeRunning",
		file: GroupReadFormat{
			TimeEnabled: 0,
			TimeRunning: 3,
			Nr:          1,
		},
		valuesFile: Values{
			Value: 0,
			ID:    0,
		},
		name: "some metric",
		cpu:  4,
		perfStat: []info.PerfStat{{
			PerfValue: info.PerfValue{
				ScalingRatio: 1.0,
				Value:        0,
				Name:         "some metric",
			},
			Cpu: 4,
		}},
		err: nil,
	},
}

func TestReadPerfStat(t *testing.T) {
	for _, test := range readGroupPerfStatCases {
		t.Run(test.test, func(tt *testing.T) {
			buf := &buffer{bytes.NewBuffer([]byte{})}
			err := binary.Write(buf, binary.LittleEndian, test.file)
			assert.NoError(tt, err)
			err = binary.Write(buf, binary.LittleEndian, test.valuesFile)
			assert.NoError(tt, err)
			stat, err := readGroupPerfStat(buf, group{
				cpuFiles:   nil,
				names:      []string{test.name},
				leaderName: test.name,
			}, test.cpu, "/")
			assert.Equal(tt, test.perfStat, stat)
			assert.Equal(tt, test.err, err)
		})
	}
}

func TestReadPerfEventAttr(t *testing.T) {
	var testCases = []struct {
		expected      *unix.PerfEventAttr
		pfmMockedFunc func(string, unsafe.Pointer) error
	}{
		{
			&unix.PerfEventAttr{
				Type:               0,
				Size:               0,
				Config:             0,
				Sample:             0,
				Sample_type:        0,
				Read_format:        0,
				Bits:               0,
				Wakeup:             0,
				Bp_type:            0,
				Ext1:               0,
				Ext2:               0,
				Branch_sample_type: 0,
				Sample_regs_user:   0,
				Sample_stack_user:  0,
				Clockid:            0,
				Sample_regs_intr:   0,
				Aux_watermark:      0,
				Sample_max_stack:   0,
			},
			func(s string, pointer unsafe.Pointer) error {
				return nil
			},
		},
	}

	for _, test := range testCases {
		got, err := readPerfEventAttr("event_name", test.pfmMockedFunc)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, got)
	}
}
