//go:build linux
// +build linux

// Copyright 2021 Google Inc. All Rights Reserved.
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

// Utilities tests.
//
// Mocked environment:
// - "container" first container with {1, 2, 3} processes.
// - "another" second container with {5, 6} processes.
//
package resctrl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/intelrdt"

	"github.com/stretchr/testify/assert"
)

func init() {
	// All the test cases in this file uses "fake" cgroups (not the real
	// cgroupfs). This setting relaxes filesystem type check in cgroups
	// package so it can work with fake cgroups.
	cgroups.TestMode = true
}

func mockAllGetContainerPids() ([]string, error) {
	return []string{"1", "2", "3", "5", "6"}, nil
}

func mockGetContainerPids() ([]string, error) {
	return []string{"1", "2", "3"}, nil
}

func mockAnotherGetContainerPids() ([]string, error) {
	return []string{"5", "6"}, nil
}

func touch(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	return file.Close()
}

func touchDir(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func fillPids(path string, pids []int) error {
	f, err := os.OpenFile(path, os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, pid := range pids {
		_, err := fmt.Fprintln(f, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

func mockResctrl() string {
	path, _ := ioutil.TempDir("", "resctrl")

	var files = []struct {
		path  string
		touch func(string) error
	}{
		// Mock root files.
		{
			filepath.Join(path, cpusFileName),
			touch,
		},
		{
			filepath.Join(path, cpusListFileName),
			touch,
		},
		{
			filepath.Join(path, infoDirName),
			touchDir,
		},
		{
			filepath.Join(path, monDataDirName),
			touchDir,
		},
		{
			filepath.Join(path, monGroupsDirName),
			touchDir,
		},
		{
			filepath.Join(path, schemataFileName),
			touch,
		},
		{
			filepath.Join(path, tasksFileName),
			touch,
		},
		// Create custom CLOSID "m1".
		{
			filepath.Join(path, "m1"),
			touchDir,
		},
		{
			filepath.Join(path, "m1", cpusFileName),
			touch,
		},
		{
			filepath.Join(path, "m1", cpusListFileName),
			touch,
		},
		{
			filepath.Join(path, "m1", monDataDirName),
			touchDir,
		},
		{
			filepath.Join(path, "m1", monGroupsDirName),
			touchDir,
		},
		{
			filepath.Join(path, "m1", schemataFileName),
			touch,
		},
		{
			filepath.Join(path, "m1", tasksFileName),
			touch,
		},
		{
			filepath.Join(path, "m1", monGroupsDirName, "test"),
			touchDir,
		},
		{
			filepath.Join(path, "m1", monGroupsDirName, "test", tasksFileName),
			touch,
		},
	}
	for _, file := range files {
		err := file.touch(file.path)
		if err != nil {
			return ""
		}
	}

	// Mock root group task file.
	err := fillPids(filepath.Join(path, tasksFileName), []int{1, 2, 3, 4})
	if err != nil {
		return ""
	}

	// Mock custom CLOSID "m1" task file.
	err = fillPids(filepath.Join(path, "m1", tasksFileName), []int{5, 6, 7, 8, 9, 10})
	if err != nil {
		return ""
	}
	// Mock custom mon group "test" task file.
	err = fillPids(filepath.Join(path, "m1", monGroupsDirName, "test", tasksFileName), []int{7, 8})
	if err != nil {
		return ""
	}

	return path
}

func mockResctrlMonData(path string) {

	_ = touchDir(filepath.Join(path, monDataDirName, "mon_L3_00"))
	_ = touchDir(filepath.Join(path, monDataDirName, "mon_L3_01"))

	var files = []struct {
		path  string
		value string
	}{
		{
			filepath.Join(path, monDataDirName, "mon_L3_00", llcOccupancyFileName),
			"1111",
		},
		{
			filepath.Join(path, monDataDirName, "mon_L3_00", mbmLocalBytesFileName),
			"2222",
		},
		{
			filepath.Join(path, monDataDirName, "mon_L3_00", mbmTotalBytesFileName),
			"3333",
		},
		{
			filepath.Join(path, monDataDirName, "mon_L3_01", llcOccupancyFileName),
			"3333",
		},
		{
			filepath.Join(path, monDataDirName, "mon_L3_01", mbmLocalBytesFileName),
			"1111",
		},
		{
			filepath.Join(path, monDataDirName, "mon_L3_01", mbmTotalBytesFileName),
			"3333",
		},
	}

	for _, file := range files {
		_ = touch(file.path)
		_ = ioutil.WriteFile(file.path, []byte(file.value), os.ModePerm)
	}
}

func mockContainersPids() string {
	path, _ := ioutil.TempDir("", "cgroup")
	// container
	_ = touchDir(filepath.Join(path, "container"))
	_ = touch(filepath.Join(path, "container", cgroups.CgroupProcesses))
	err := fillPids(filepath.Join(path, "container", cgroups.CgroupProcesses), []int{1, 2, 3})
	if err != nil {
		return ""
	}
	// another
	_ = touchDir(filepath.Join(path, "another"))
	_ = touch(filepath.Join(path, "another", cgroups.CgroupProcesses))
	err = fillPids(filepath.Join(path, "another", cgroups.CgroupProcesses), []int{5})
	if err != nil {
		return ""
	}

	return path
}

func mockProcFs() string {
	path, _ := ioutil.TempDir("", "proc")

	var files = []struct {
		path  string
		touch func(string) error
	}{
		// container
		{
			filepath.Join(path, "1", processTask, "1"),
			touchDir,
		},
		{
			filepath.Join(path, "2", processTask, "2"),
			touchDir,
		},
		{
			filepath.Join(path, "3", processTask, "3"),
			touchDir,
		},
		{
			filepath.Join(path, "4", processTask, "4"),
			touchDir,
		},
		// another
		{
			filepath.Join(path, "5", processTask, "5"),
			touchDir,
		},
		{
			filepath.Join(path, "6", processTask, "6"),
			touchDir,
		},
	}

	for _, file := range files {
		_ = file.touch(file.path)
	}

	return path
}

func checkError(t *testing.T, err error, expected string) {
	if expected != "" {
		assert.EqualError(t, err, expected)
	} else {
		assert.NoError(t, err)
	}
}

func TestPrepareMonitoringGroup(t *testing.T) {
	rootResctrl = mockResctrl()
	defer os.RemoveAll(rootResctrl)

	pidsPath = mockContainersPids()
	defer os.RemoveAll(pidsPath)

	processPath = mockProcFs()
	defer os.RemoveAll(processPath)

	var testCases = []struct {
		container        string
		getContainerPids func() ([]string, error)
		expected         string
		err              string
	}{
		{
			"container",
			mockGetContainerPids,
			filepath.Join(rootResctrl, monGroupsDirName, "cadvisor-container"),
			"",
		},
		{
			"another",
			mockAnotherGetContainerPids,
			filepath.Join(rootResctrl, "m1", monGroupsDirName, "cadvisor-another"),
			"",
		},
		{
			"/",
			mockAllGetContainerPids,
			rootResctrl,
			"",
		},
	}

	for _, test := range testCases {
		actual, err := prepareMonitoringGroup(test.container, test.getContainerPids, true)
		assert.Equal(t, test.expected, actual)
		checkError(t, err, test.err)
	}
}

func TestGetPids(t *testing.T) {
	pidsPath = mockContainersPids()
	defer os.RemoveAll(pidsPath)

	var testCases = []struct {
		container string
		expected  []int
		err       string
	}{
		{
			"",
			nil,
			noContainerNameError,
		},
		{
			"container",
			[]int{1, 2, 3},
			"",
		},
		{
			"no_container",
			nil,
			fmt.Sprintf("couldn't obtain pids for \"no_container\" container: lstat %v: no such file or directory", filepath.Join(pidsPath, "no_container")),
		},
	}

	for _, test := range testCases {
		actual, err := getPids(test.container)
		assert.Equal(t, test.expected, actual)
		checkError(t, err, test.err)
	}
}

func TestGetAllProcessThreads(t *testing.T) {
	mockProcFs := func() string {
		path, _ := ioutil.TempDir("", "proc")

		var files = []struct {
			path  string
			touch func(string) error
		}{
			// correct
			{
				filepath.Join(path, "4215", processTask, "4215"),
				touchDir,
			},
			{
				filepath.Join(path, "4215", processTask, "4216"),
				touchDir,
			},
			{
				filepath.Join(path, "4215", processTask, "4217"),
				touchDir,
			},
			{
				filepath.Join(path, "4215", processTask, "4218"),
				touchDir,
			},
			// invalid
			{
				filepath.Join(path, "301", processTask, "301"),
				touchDir,
			},
			{
				filepath.Join(path, "301", processTask, "incorrect"),
				touchDir,
			},
		}

		for _, file := range files {
			_ = file.touch(file.path)
		}

		return path
	}

	mockedProcFs := mockProcFs()
	defer os.RemoveAll(mockedProcFs)

	var testCases = []struct {
		path     string
		expected []int
		err      string
	}{
		{
			filepath.Join(mockedProcFs, "4215", processTask),
			[]int{4215, 4216, 4217, 4218},
			"",
		},
		{
			filepath.Join(mockedProcFs, "301", processTask),
			nil,
			"couldn't parse \"incorrect\" dir: strconv.Atoi: parsing \"incorrect\": invalid syntax",
		},
	}

	for _, test := range testCases {
		actual, err := getAllProcessThreads(test.path)
		assert.Equal(t, test.expected, actual)
		checkError(t, err, test.err)
	}
}

func TestFindGroup(t *testing.T) {
	rootResctrl = mockResctrl()
	defer os.RemoveAll(rootResctrl)

	var testCases = []struct {
		path         string
		pids         []string
		includeGroup bool
		exclusive    bool
		expected     string
		err          string
	}{
		{
			rootResctrl,
			[]string{"1", "2", "3", "4"},
			true,
			false,
			rootResctrl,
			"",
		},
		{
			rootResctrl,
			[]string{},
			true,
			false,
			"",
			"there are no pids passed",
		},
		{
			rootResctrl,
			[]string{"5", "6"},
			true,
			false,
			filepath.Join(rootResctrl, "m1"),
			"",
		},
		{
			rootResctrl,
			[]string{"11", "12"},
			true,
			false,
			"",
			"",
		},
		{
			filepath.Join(rootResctrl, "m1", monGroupsDirName),
			[]string{"5", "6"},
			false,
			true,
			"",
			"",
		},
		{
			filepath.Join(rootResctrl, "m1", monGroupsDirName),
			[]string{"7", "8"},
			false,
			true,
			filepath.Join(rootResctrl, "m1", monGroupsDirName, "test"),
			"",
		},
		{
			filepath.Join(rootResctrl, "m1", monGroupsDirName),
			[]string{"7"},
			false,
			true,
			"",
			"group should have container pids only",
		},
	}
	for _, test := range testCases {
		actual, err := findGroup(test.path, test.pids, test.includeGroup, test.exclusive)
		assert.Equal(t, test.expected, actual)
		checkError(t, err, test.err)
	}
}

func TestArePIDsInGroup(t *testing.T) {
	rootResctrl = mockResctrl()
	defer os.RemoveAll(rootResctrl)

	var testCases = []struct {
		expected  bool
		err       string
		path      string
		pids      []string
		exclusive bool
	}{
		{
			true,
			"",
			rootResctrl,
			[]string{"1", "2"},
			false,
		},
		{
			false,
			"there should be all pids in group",
			rootResctrl,
			[]string{"4", "5"},
			false,
		},
		{
			false,
			"",
			filepath.Join(rootResctrl, "m1"),
			[]string{"1"},
			false,
		},
		{
			false,
			fmt.Sprintf("couldn't read tasks file from %q path: open %s: no such file or directory", filepath.Join(rootResctrl, monitoringGroupDir, tasksFileName), filepath.Join(rootResctrl, monitoringGroupDir, tasksFileName)),
			filepath.Join(rootResctrl, monitoringGroupDir),
			[]string{"1", "2"},
			false,
		},
		{
			false,
			fmt.Sprintf("couldn't obtain pids from %q path: %v", rootResctrl, noPidsPassedError),
			rootResctrl,
			nil,
			false,
		},
	}

	for _, test := range testCases {
		actual, err := arePIDsInGroup(test.path, test.pids, test.exclusive)
		assert.Equal(t, test.expected, actual)
		checkError(t, err, test.err)
	}
}

func TestGetStats(t *testing.T) {
	rootResctrl = mockResctrl()
	defer os.RemoveAll(rootResctrl)

	pidsPath = mockContainersPids()
	defer os.RemoveAll(pidsPath)

	processPath = mockProcFs()
	defer os.RemoveAll(processPath)

	enabledCMT, enabledMBM = true, true

	var testCases = []struct {
		container string
		expected  intelrdt.Stats
		err       string
	}{
		{
			"container",
			intelrdt.Stats{
				MBMStats: &[]intelrdt.MBMNumaNodeStats{
					{
						MBMTotalBytes: 3333,
						MBMLocalBytes: 2222,
					},
					{
						MBMTotalBytes: 3333,
						MBMLocalBytes: 1111,
					},
				},
				CMTStats: &[]intelrdt.CMTNumaNodeStats{
					{
						LLCOccupancy: 1111,
					},
					{
						LLCOccupancy: 3333,
					},
				},
			},
			"",
		},
		{
			"another",
			intelrdt.Stats{
				MBMStats: &[]intelrdt.MBMNumaNodeStats{
					{
						MBMTotalBytes: 3333,
						MBMLocalBytes: 2222,
					},
					{
						MBMTotalBytes: 3333,
						MBMLocalBytes: 1111,
					},
				},
				CMTStats: &[]intelrdt.CMTNumaNodeStats{
					{
						LLCOccupancy: 1111,
					},
					{
						LLCOccupancy: 3333,
					},
				},
			},
			"",
		},
		{
			"/",
			intelrdt.Stats{
				MBMStats: &[]intelrdt.MBMNumaNodeStats{
					{
						MBMTotalBytes: 3333,
						MBMLocalBytes: 2222,
					},
					{
						MBMTotalBytes: 3333,
						MBMLocalBytes: 1111,
					},
				},
				CMTStats: &[]intelrdt.CMTNumaNodeStats{
					{
						LLCOccupancy: 1111,
					},
					{
						LLCOccupancy: 3333,
					},
				},
			},
			"",
		},
	}

	for _, test := range testCases {
		containerPath, _ := prepareMonitoringGroup(test.container, mockGetContainerPids, true)
		mockResctrlMonData(containerPath)
		actual, err := getIntelRDTStatsFrom(containerPath, "")
		checkError(t, err, test.err)
		assert.Equal(t, test.expected.CMTStats, actual.CMTStats)
		assert.Equal(t, test.expected.MBMStats, actual.MBMStats)
	}
}

func TestReadTasksFile(t *testing.T) {
	var testCases = []struct {
		tasksFile string
		expected  map[string]struct{}
		err       string
	}{
		{"testing/tasks_two",
			map[string]struct{}{
				"12": {},
				"77": {},
			},
			"",
		},
		{"testing/tasks_one",
			map[string]struct{}{
				"2": {},
			},
			"",
		},
		{"testing/tasks_empty",
			map[string]struct{}{},
			"",
		},
	}

	for _, test := range testCases {
		actual, err := readTasksFile(test.tasksFile)
		assert.Equal(t, test.expected, actual)
		checkError(t, err, test.err)
	}
}

func mockCadvisorMonGroup(controllerGroupPath string) int {
	paths := []string{
		path.Join(controllerGroupPath, strings.Join([]string{monGroupPrefix, "a"}, "-")),
		path.Join(controllerGroupPath, strings.Join([]string{monGroupPrefix, "b"}, "-")),
		path.Join(controllerGroupPath, strings.Join([]string{monGroupPrefix, "c"}, "-")),
	}
	num := 0
	for _, p := range paths {
		err := os.MkdirAll(p, os.ModePerm)
		if err != nil {
			num++
		}
	}
	return num
}

func mockOtherMonGroup(controllerGroupPath string) int {
	paths := []string{
		path.Join(controllerGroupPath, strings.Join([]string{"a"}, "-")),
		path.Join(controllerGroupPath, strings.Join([]string{"b"}, "-")),
		path.Join(controllerGroupPath, strings.Join([]string{"c"}, "-")),
	}
	num := 0
	for _, p := range paths {
		err := os.MkdirAll(p, os.ModePerm)
		if err != nil {
			num++
		}
	}
	return num
}

func TestRefreshCadvisorMonGroup(t *testing.T) {
	controllerGroupPath, _ := ioutil.TempDir("", "cadvisor-test")
	defer os.RemoveAll(controllerGroupPath)
	testGroup := &monGroupMap{}
	numOfCMon := mockCadvisorMonGroup(controllerGroupPath)
	mockOtherMonGroup(controllerGroupPath)
	testGroup.refreshMonGroup(controllerGroupPath)
	assert.Equal(t, numOfCMon, len(testGroup.monGroups))
}

func TestAddCadvisorMonGroup(t *testing.T) {
	testGroup := &monGroupMap{}
	var testCases = []struct {
		monGroupPath string
		key          string
		expected     string
	}{
		{
			path.Join("testing", strings.Join([]string{monGroupPrefix, "1"}, "-")),
			strings.Join([]string{monGroupPrefix, "1"}, "-"),
			path.Join("testing", strings.Join([]string{monGroupPrefix, "1"}, "-")),
		}, {
			path.Join("testing", strings.Join([]string{monGroupPrefix, "2"}, "-")),
			strings.Join([]string{monGroupPrefix, "2"}, "-"),
			path.Join("testing", strings.Join([]string{monGroupPrefix, "2"}, "-")),
		}, {
			path.Join("testing", strings.Join([]string{monGroupPrefix, "3"}, "-")),
			strings.Join([]string{monGroupPrefix, "3"}, "-"),
			path.Join("testing", strings.Join([]string{monGroupPrefix, "3"}, "-")),
		},
	}
	for _, testCase := range testCases {
		testGroup.addMonGroup(testCase.monGroupPath)
		assert.Equal(t, testGroup.monGroups[testCase.key].path, testCase.monGroupPath, "testing addMonGroup")
	}
}

func TestClearCadvisorMonGroup(t *testing.T) {
	interval := 1 * time.Hour
	testGroup := &monGroupMap{}
	var testCases = []struct {
		monGroupPath string
		key          string
		updateTime   time.Time
		isExist      bool
	}{
		{
			path.Join("testing", strings.Join([]string{monGroupPrefix, "1"}, "-")),
			strings.Join([]string{monGroupPrefix, "1"}, "-"),
			time.Now(),
			true,
		}, {
			path.Join("testing", strings.Join([]string{monGroupPrefix, "1"}, "-")),
			strings.Join([]string{monGroupPrefix, "1"}, "-"),
			time.Now().Add(-4 * interval),
			false,
		}, {
			path.Join("testing", strings.Join([]string{"1"}, "-")),
			strings.Join([]string{"1"}, "-"),
			time.Now().Add(-4 * interval),
			true,
		},
	}
	for _, testCase := range testCases {
		testGroup.monGroups = make(map[string]*monGroupInfo, 0)
		testGroup.monGroups[testCase.key] = &monGroupInfo{
			path:       testCase.monGroupPath,
			updateTime: testCase.updateTime,
		}
		testGroup.clearMonGroup(interval)
		_, exist := testGroup.monGroups[testCase.key]
		assert.Equal(t, exist, testCase.isExist, "testing clearMonGroup")
	}
}
