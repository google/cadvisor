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

// Uncore perf events logic tests.
package perf

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/stretchr/testify/assert"

	v1 "github.com/google/cadvisor/info/v1"
)

func mockSystemDevices() (string, error) {
	testDir, err := ioutil.TempDir("", "uncore_imc_test")
	if err != nil {
		return "", err
	}

	// First Uncore IMC PMU.
	firstPMUPath := filepath.Join(testDir, "uncore_imc_0")
	err = os.MkdirAll(firstPMUPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(filepath.Join(firstPMUPath, "cpumask"), []byte("0-1"), 0777)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(filepath.Join(firstPMUPath, "type"), []byte("18"), 0777)
	if err != nil {
		return "", err
	}

	// Second Uncore IMC PMU.
	secondPMUPath := filepath.Join(testDir, "uncore_imc_1")
	err = os.MkdirAll(secondPMUPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(filepath.Join(secondPMUPath, "cpumask"), []byte("0,1"), 0777)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(filepath.Join(secondPMUPath, "type"), []byte("19"), 0777)
	if err != nil {
		return "", err
	}

	return testDir, nil
}

func TestUncore(t *testing.T) {
	path, err := mockSystemDevices()
	assert.Nil(t, err)
	defer func() {
		err := os.RemoveAll(path)
		assert.Nil(t, err)
	}()

	actual, err := getUncorePMUs(path)
	assert.Nil(t, err)
	expected := uncorePMUs{
		"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
		"uncore_imc_1": {name: "uncore_imc_1", typeOf: 19, cpus: []uint32{0, 1}},
	}
	assert.Equal(t, expected, actual)

	pmuSet := uncorePMUs{
		"uncore_imc_0": actual["uncore_imc_0"],
		"uncore_imc_1": actual["uncore_imc_1"],
	}

	actualPMU, err := getPMU(pmuSet, expected["uncore_imc_0"].typeOf)
	assert.Nil(t, err)
	assert.Equal(t, expected["uncore_imc_0"], *actualPMU)
}

func TestUncoreCollectorSetup(t *testing.T) {
	path, err := mockSystemDevices()
	assert.Nil(t, err)
	defer func() {
		err := os.RemoveAll(path)
		assert.Nil(t, err)
	}()

	events := PerfEvents{
		Core: Events{
			Events: []Group{
				{[]Event{"cache-misses"}, false},
			},
		},
		Uncore: Events{
			Events: []Group{
				{[]Event{"uncore_imc_1/cas_count_read"}, false},
				{[]Event{"uncore_imc_1/non_existing_event"}, false},
				{[]Event{"uncore_imc_0/cas_count_write", "uncore_imc_0/cas_count_read"}, true},
			},
			CustomEvents: []CustomEvent{
				{19, Config{0x01, 0x02}, "uncore_imc_1/cas_count_read"},
				{0, Config{0x02, 0x03}, "uncore_imc_0/cas_count_write"},
				{18, Config{0x01, 0x02}, "uncore_imc_0/cas_count_read"},
			},
		},
	}

	collector := &uncoreCollector{}
	collector.perfEventOpen = func(attr *unix.PerfEventAttr, pid int, cpu int, groupFd int, flags int) (fd int, err error) {
		return int(attr.Config), nil
	}
	collector.ioctlSetInt = func(fd int, req uint, value int) error {
		return nil
	}

	err = collector.setup(events, path)
	assert.Equal(t, []string{"uncore_imc_1/cas_count_read"},
		getMapKeys(collector.cpuFiles[0]["uncore_imc_1"].cpuFiles))
	assert.ElementsMatch(t, []string{"uncore_imc_0/cas_count_write", "uncore_imc_0/cas_count_read"},
		getMapKeys(collector.cpuFiles[2]["uncore_imc_0"].cpuFiles))

	// There are no errors.
	assert.Nil(t, err)
}

func TestParseUncoreEvents(t *testing.T) {
	events := PerfEvents{
		Uncore: Events{
			Events: []Group{
				{[]Event{"cas_count_read"}, false},
				{[]Event{"cas_count_write"}, false},
			},
			CustomEvents: []CustomEvent{
				{
					Type:   17,
					Config: Config{0x50, 0x60},
					Name:   "cas_count_read",
				},
			},
		},
	}
	eventToCustomEvent := parseUncoreEvents(events.Uncore)
	assert.Len(t, eventToCustomEvent, 1)
	assert.Equal(t, eventToCustomEvent["cas_count_read"].Name, Event("cas_count_read"))
	assert.Equal(t, eventToCustomEvent["cas_count_read"].Type, uint32(17))
	assert.Equal(t, eventToCustomEvent["cas_count_read"].Config, Config{0x50, 0x60})
}

func TestObtainPMUs(t *testing.T) {
	got := uncorePMUs{
		"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
		"uncore_imc_1": {name: "uncore_imc_1", typeOf: 19, cpus: []uint32{0, 1}},
	}

	actual := obtainPMUs("uncore_imc_0", got)
	assert.Equal(t, uncorePMUs{"uncore_imc_0": got["uncore_imc_0"]}, actual)

	actual = obtainPMUs("uncore_imc_1", got)
	assert.Equal(t, uncorePMUs{"uncore_imc_1": got["uncore_imc_1"]}, actual)

	actual = obtainPMUs("", got)
	assert.Equal(t, uncorePMUs{}, actual)
}

func TestUncoreParseEventName(t *testing.T) {
	eventName, pmuPrefix := parseEventName("some_event")
	assert.Equal(t, "some_event", eventName)
	assert.Empty(t, pmuPrefix)

	eventName, pmuPrefix = parseEventName("some_pmu/some_event")
	assert.Equal(t, "some_pmu", pmuPrefix)
	assert.Equal(t, "some_event", eventName)

	eventName, pmuPrefix = parseEventName("some_pmu/some_event/first_slash/second_slash")
	assert.Equal(t, "some_pmu", pmuPrefix)
	assert.Equal(t, "some_event/first_slash/second_slash", eventName)
}

func TestCheckGroup(t *testing.T) {
	var testCases = []struct {
		group          Group
		eventPMUs      map[Event]uncorePMUs
		expectedOutput string
	}{
		{
			Group{[]Event{"uncore_imc/cas_count_write"}, false},
			map[Event]uncorePMUs{},
			"the event \"uncore_imc/cas_count_write\" don't have any PMU to count with",
		},
		{
			Group{[]Event{"uncore_imc/cas_count_write", "uncore_imc/cas_count_read"}, true},
			map[Event]uncorePMUs{"uncore_imc/cas_count_write": {
				"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
				"uncore_imc_1": {name: "uncore_imc_1", typeOf: 19, cpus: []uint32{0, 1}},
			},
				"uncore_imc/cas_count_read": {
					"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
					"uncore_imc_1": {name: "uncore_imc_1", typeOf: 19, cpus: []uint32{0, 1}},
				},
			},
			"the events in group usually have to be from single PMU, try reorganizing the \"[uncore_imc/cas_count_write uncore_imc/cas_count_read]\" group",
		},
		{
			Group{[]Event{"uncore_imc_0/cas_count_write", "uncore_imc_1/cas_count_read"}, true},
			map[Event]uncorePMUs{"uncore_imc_0/cas_count_write": {
				"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
			},
				"uncore_imc_1/cas_count_read": {
					"uncore_imc_1": {name: "uncore_imc_1", typeOf: 19, cpus: []uint32{0, 1}},
				},
			},
			"the events in group usually have to be from the same PMU, try reorganizing the \"[uncore_imc_0/cas_count_write uncore_imc_1/cas_count_read]\" group",
		},
		{
			Group{[]Event{"uncore_imc/cas_count_write"}, false},
			map[Event]uncorePMUs{"uncore_imc/cas_count_write": {
				"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
				"uncore_imc_1": {name: "uncore_imc_1", typeOf: 19, cpus: []uint32{0, 1}},
			}},
			"",
		},
		{
			Group{[]Event{"uncore_imc_0/cas_count_write", "uncore_imc_0/cas_count_read"}, true},
			map[Event]uncorePMUs{"uncore_imc_0/cas_count_write": {
				"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
			},
				"uncore_imc_0/cas_count_read": {
					"uncore_imc_0": {name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
				}},
			"",
		},
	}

	for _, tc := range testCases {
		err := checkGroup(tc.group, tc.eventPMUs)
		if tc.expectedOutput == "" {
			assert.Nil(t, err)
		} else {
			assert.EqualError(t, err, tc.expectedOutput)
		}
	}
}

func TestReadPerfUncoreStat(t *testing.T) {
	file := GroupReadFormat{
		TimeEnabled: 0,
		TimeRunning: 1,
		Nr:          1,
	}

	valuesFile := Values{
		Value: 4,
		ID:    0,
	}

	expectedStat := []v1.PerfUncoreStat{{
		PerfValue: v1.PerfValue{
			ScalingRatio: 1,
			Value:        4,
			Name:         "foo",
		},
		Socket: 0,
		PMU:    "bar",
	}}
	cpuToSocket := map[int]int{
		1: 0,
		2: 0,
	}

	buf := &buffer{bytes.NewBuffer([]byte{})}
	err := binary.Write(buf, binary.LittleEndian, file)
	assert.NoError(t, err)
	err = binary.Write(buf, binary.LittleEndian, valuesFile)
	assert.NoError(t, err)

	stat, err := readPerfUncoreStat(buf, group{
		cpuFiles:   nil,
		names:      []string{"foo"},
		leaderName: "foo",
	}, 1, "bar", cpuToSocket)
	assert.NoError(t, err)
	assert.Equal(t, expectedStat, stat)
}

func getMapKeys(someMap map[string]map[int]readerCloser) []string {
	var keys []string
	for key := range someMap {
		keys = append(keys, key)
	}
	return keys
}
