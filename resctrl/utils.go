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

// Utilities.
package resctrl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/intelrdt"
)

const (
	cpuCgroup             = "cpu"
	rootContainer         = "/"
	monitoringGroupDir    = "mon_groups"
	processTask           = "task"
	cpusFileName          = "cpus"
	cpusListFileName      = "cpus_list"
	schemataFileName      = "schemata"
	tasksFileName         = "tasks"
	infoDirName           = "info"
	monDataDirName        = "mon_data"
	monGroupsDirName      = "mon_groups"
	noPidsPassedError     = "there are no pids passed"
	noContainerNameError  = "there are no container name passed"
	llcOccupancyFileName  = "llc_occupancy"
	mbmLocalBytesFileName = "mbm_local_bytes"
	mbmTotalBytesFileName = "mbm_total_bytes"
)

var (
	rootResctrl          = ""
	pidsPath             = ""
	processPath          = "/proc"
	enabledMBM           = false
	enabledCMT           = false
	isResctrlInitialized = false
)

func Setup() error {
	var err error
	rootResctrl, err = intelrdt.GetIntelRdtPath(rootContainer)
	if err != nil {
		return fmt.Errorf("unable to initialize resctrl: %v", err)
	}

	if cgroups.IsCgroup2UnifiedMode() {
		pidsPath = fs2.UnifiedMountpoint
	} else {
		pidsPath = filepath.Join(fs2.UnifiedMountpoint, cpuCgroup)
	}

	enabledMBM = intelrdt.IsMBMEnabled()
	enabledCMT = intelrdt.IsCMTEnabled()

	isResctrlInitialized = true

	return nil
}

func prepareMonitoringGroup(containerName string, getContainerPids func() ([]string, error)) (string, error) {
	if containerName == rootContainer {
		return rootResctrl, nil
	}

	pids, err := getContainerPids()
	if err != nil {
		return "", err
	}

	if len(pids) == 0 {
		return "", fmt.Errorf("couldn't obtain %q container pids, there is no pids in cgroup", containerName)
	}

	path, err := findControlGroup(pids)
	if err != nil {
		return "", fmt.Errorf("couldn't find control group matching %q container: %v", containerName, err)
	}

	if containerName[0] == '/' && (len(containerName) > 1) {
		containerName = containerName[1:]
	}

	properContainerName := strings.Replace(containerName, "/", "-", -1)
	monGroupPath := filepath.Join(path, monitoringGroupDir, properContainerName)

	// Create new mon_group if not exists.
	err = os.MkdirAll(monGroupPath, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("couldn't create monitoring group directory for %q container: %w", containerName, err)
	}

	for _, pid := range pids {
		threadFiles, err := ioutil.ReadDir(filepath.Join(processPath, pid, processTask))
		if err != nil {
			return "", err
		}

		processThreads, err := getAllProcessThreads(threadFiles)
		if err != nil {
			return "", err
		}
		for _, thread := range processThreads {
			err = intelrdt.WriteIntelRdtTasks(monGroupPath, thread)
			if err != nil {
				secondError := os.Remove(monGroupPath)
				if secondError != nil {
					return "", fmt.Errorf(
						"coudn't assign pids to %q container monitoring group: %w \n couldn't clear %q monitoring group: %v",
						containerName, err, containerName, secondError)
				}
				return "", fmt.Errorf("coudn't assign pids to %q container monitoring group: %w", containerName, err)
			}
		}
	}

	return monGroupPath, nil
}

func getPids(containerName string) ([]int, error) {
	if len(containerName) == 0 {
		// No container name passed.
		return nil, fmt.Errorf(noContainerNameError)
	}
	pids, err := cgroups.GetAllPids(filepath.Join(pidsPath, containerName))
	if err != nil {
		return nil, fmt.Errorf("couldn't obtain pids for %q container: %v", containerName, err)
	}
	return pids, nil
}

func getAllProcessThreads(threadFiles []os.FileInfo) ([]int, error) {
	processThreads := make([]int, 0)
	for _, file := range threadFiles {
		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			return nil, fmt.Errorf("couldn't parse %q file: %v", file.Name(), err)
		}
		processThreads = append(processThreads, pid)
	}

	return processThreads, nil
}

func findControlGroup(pids []string) (string, error) {
	if len(pids) == 0 {
		return "", fmt.Errorf(noPidsPassedError)
	}
	availablePaths, err := filepath.Glob(filepath.Join(rootResctrl, "*"))
	if err != nil {
		return "", err
	}

	for _, path := range availablePaths {
		switch path {
		case filepath.Join(rootResctrl, cpusFileName):
			continue
		case filepath.Join(rootResctrl, cpusListFileName):
			continue
		case filepath.Join(rootResctrl, infoDirName):
			continue
		case filepath.Join(rootResctrl, monDataDirName):
			continue
		case filepath.Join(rootResctrl, monGroupsDirName):
			continue
		case filepath.Join(rootResctrl, schemataFileName):
			continue
		case filepath.Join(rootResctrl, tasksFileName):
			inGroup, err := arePIDsInControlGroup(rootResctrl, pids)
			if err != nil {
				return "", err
			}
			if inGroup {
				return rootResctrl, nil
			}
		default:
			inGroup, err := arePIDsInControlGroup(path, pids)
			if err != nil {
				return "", err
			}
			if inGroup {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("there is no available control group")
}

func arePIDsInControlGroup(path string, pids []string) (bool, error) {
	if len(pids) == 0 {
		// No container pids passed.
		return false, fmt.Errorf("couldn't obtain pids from %q path: %v", path, noPidsPassedError)
	}

	tasksFile, err := ioutil.ReadFile(filepath.Join(path, "tasks"))
	if err != nil {
		return false, fmt.Errorf("couldn't obtain pids from %q path: %w", path, err)
	}

	if len(tasksFile) == 0 {
		return false, nil
	}

	tasks := readTasksFile(tasksFile)

	for _, pid := range pids {
		_, ok := tasks[pid]
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func readTasksFile(tasksFile []byte) map[string]bool {
	const (
		newLineASCIICode = 0xA
		zeroASCIICode    = 0x30
		nineASCIICode    = 0x39
	)
	tasks := make(map[string]bool)
	var task []byte
	for _, b := range tasksFile {
		if b == newLineASCIICode {
			if len(task) != 0 {
				tasks[string(task)] = true
				task = []byte{}
			}
			continue
		}
		if b >= zeroASCIICode && b <= nineASCIICode {
			task = append(task, b)
		}
	}

	if len(task) > 0 {
		tasks[string(task)] = true
	}

	return tasks
}

func readStatFrom(path string) (uint64, error) {
	context, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}

	stat, err := strconv.ParseUint(string(bytes.TrimSpace(context)), 10, 64)
	if err != nil {
		return stat, fmt.Errorf("unable to parse %q as a uint from file %q", string(context), path)
	}

	return stat, nil
}

func getIntelRDTStatsFrom(path string) (intelrdt.Stats, error) {
	stats := intelrdt.Stats{}

	statsDirectories, err := filepath.Glob(filepath.Join(path, monDataDirName, "*"))
	if err != nil {
		return stats, err
	}

	if len(statsDirectories) == 0 {
		return stats, fmt.Errorf("there is no mon_data stats directories: %q", path)
	}

	var cmtStats []intelrdt.CMTNumaNodeStats
	var mbmStats []intelrdt.MBMNumaNodeStats

	for _, dir := range statsDirectories {
		if enabledCMT {
			llcOccupancy, err := readStatFrom(filepath.Join(dir, llcOccupancyFileName))
			if err != nil {
				return stats, err
			}
			cmtStats = append(cmtStats, intelrdt.CMTNumaNodeStats{LLCOccupancy: llcOccupancy})
		}
		if enabledMBM {
			mbmTotalBytes, err := readStatFrom(filepath.Join(dir, mbmTotalBytesFileName))
			if err != nil {
				return stats, err
			}
			mbmLocalBytes, err := readStatFrom(filepath.Join(dir, mbmLocalBytesFileName))
			if err != nil {
				return stats, err
			}
			mbmStats = append(mbmStats, intelrdt.MBMNumaNodeStats{
				MBMTotalBytes: mbmTotalBytes,
				MBMLocalBytes: mbmLocalBytes,
			})
		}
	}

	stats.CMTStats = &cmtStats
	stats.MBMStats = &mbmStats

	return stats, nil
}
