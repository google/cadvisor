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

	pmuSet := []pmu{
		actual["uncore_imc_0"],
		actual["uncore_imc_1"],
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
			Events: [][]Event{
				{"cache-misses"},
			},
		},
		Uncore: Events{
			Events: [][]Event{
				{"uncore_imc_0/cas_count_read"},
				{"uncore_imc/cas_count_write"},
			},
			CustomEvents: []CustomEvent{
				{18, Config{0x01, 0x02}, "uncore_imc_0/cas_count_read"},
				{0, Config{0x01, 0x03}, "uncore_imc/cas_count_write"},
			},
		},
	}

	collector := &uncoreCollector{}
	collector.perfEventOpen = func(attr *unix.PerfEventAttr, pid int, cpu int, groupFd int, flags int) (fd int, err error) {
		return 0, nil
	}

	err = collector.setup(events, path)
	// There are no errors.
	assert.Nil(t, err)

	// For "cas_count_write", collector has two registered PMUs,
	// `uncore_imc_0 (of 18 type) and `uncore_imc_1` (of 19 type).
	// Both of them has two cpus which corresponds to sockets.
	assert.Equal(t, len(collector.cpuFiles["uncore_imc/cas_count_write"]["uncore_imc_0"]), 2)
	assert.Equal(t, len(collector.cpuFiles["uncore_imc/cas_count_write"]["uncore_imc_1"]), 2)

	// For "cas_count_read", has only one registered PMU and it's `uncore_imc_0` (of 18 type) with two cpus which
	// correspond to two sockets.
	assert.Equal(t, len(collector.cpuFiles["uncore_imc_0/cas_count_read"]), 1)
	assert.Equal(t, len(collector.cpuFiles["uncore_imc_0/cas_count_read"]["uncore_imc_0"]), 2)

	// For "cache-misses" it shouldn't register any PMU.
	assert.Nil(t, collector.cpuFiles["cache-misses"])
}

func TestParseUncoreEvents(t *testing.T) {
	events := PerfEvents{
		Uncore: Events{
			Events: [][]Event{
				{"cas_count_read"},
				{"cas_count_write"},
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

	expected := []pmu{
		{name: "uncore_imc_0", typeOf: 18, cpus: []uint32{0, 1}},
		{name: "uncore_imc_1", typeOf: 19, cpus: []uint32{0, 1}},
	}

	actual := obtainPMUs("uncore_imc_0", got)
	assert.Equal(t, []pmu{expected[0]}, actual)

	actual = obtainPMUs("uncore_imc_1", got)
	assert.Equal(t, []pmu{expected[1]}, actual)

	actual = obtainPMUs("", got)
	assert.Equal(t, []pmu(nil), actual)
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

func TestReadPerfUncoreStat(t *testing.T) {
	file := ReadFormat{
		Value:       4,
		TimeEnabled: 0,
		TimeRunning: 1,
		ID:          0,
	}
	expectedStat := v1.PerfUncoreStat{
		PerfValue: v1.PerfValue{
			ScalingRatio: 1,
			Value:        4,
			Name:         "foo",
		},
		Socket: 0,
		PMU:    "bar",
	}
	topology := []v1.Node{{
		Id:        0,
		HugePages: nil,
		Cores: []v1.Core{{
			Id:       1,
			Threads:  []int{1, 2},
			SocketID: 0,
		}},
	}}

	buf := &buffer{bytes.NewBuffer([]byte{})}
	err := binary.Write(buf, binary.LittleEndian, file)
	assert.NoError(t, err)

	stat, err := readPerfUncoreStat(buf, "foo", 1, "bar", topology)
	assert.NoError(t, err)
	assert.Equal(t, expectedStat, *stat)
}
