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
	"bufio"
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
	containerPrefix       = '/'
	minContainerNameLen   = 2 // "/<container_name>" e.g. "/a"
	unavailable           = "Unavailable"
)

var (
	rootResctrl             = ""
	pidsPath                = ""
	processPath             = "/proc"
	enabledMBM              = false
	enabledCMT              = false
	isResctrlInitialized    = false
	controlGroupDirectories = map[string]struct{}{
		infoDirName:      {},
		monDataDirName:   {},
		monGroupsDirName: {},
	}
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

	if len(containerName) >= minContainerNameLen && containerName[0] == containerPrefix {
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

// findControlGroup returns the path of a control group in which the pids are.
func findControlGroup(pids []string) (string, error) {
	if len(pids) == 0 {
		return "", fmt.Errorf(noPidsPassedError)
	}
	availablePaths := make([]string, 0)
	err := filepath.Walk(rootResctrl, func(path string, info os.FileInfo, err error) error {
		// Look only for directories.
		if info.IsDir() {
			// Avoid internal control group dirs.
			if _, ok := controlGroupDirectories[filepath.Base(path)]; !ok {
				availablePaths = append(availablePaths, path)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("couldn't obtain control groups paths: %w", err)
	}

	for _, path := range availablePaths {
		groupFound, err := arePIDsInControlGroup(path, pids)
		if err != nil {
			return "", err
		}
		if groupFound {
			return path, nil
		}
	}

	return "", fmt.Errorf("there is no available control group")
}

// arePIDsInControlGroup returns true if all of the pids are within control group.
func arePIDsInControlGroup(path string, pids []string) (bool, error) {
	if len(pids) == 0 {
		return false, fmt.Errorf("couldn't obtain pids from %q path: %v", path, noPidsPassedError)
	}

	tasks, err := readTasksFile(filepath.Join(path, tasksFileName))
	if err != nil {
		return false, err
	}

	for _, pidString := range pids {
		pid, err := strconv.Atoi(pidString)
		if err != nil {
			return false, err
		}
		_, ok := tasks[pid]
		if !ok {
			// There is any missing pid within group.
			return false, nil
		}
	}

	return true, nil
}

// readTasksFile returns pids map from given tasks path.
func readTasksFile(tasksPath string) (map[int]struct{}, error) {
	tasks := make(map[int]struct{})

	tasksFile, err := os.Open(tasksPath)
	if err != nil {
		return tasks, fmt.Errorf("couldn't read tasks file from %q path: %w", tasksPath, err)
	}
	defer tasksFile.Close()

	scanner := bufio.NewScanner(tasksFile)
	for scanner.Scan() {
		id, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return tasks, fmt.Errorf("couldn't convert pids from %q path: %w", tasksPath, err)
		}
		tasks[id] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return tasks, fmt.Errorf("couldn't obtain pids from %q path: %w", tasksPath, err)
	}

	return tasks, nil
}

func readStatFrom(path string, vendorID string) (uint64, error) {
	context, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}

	contextString := string(bytes.TrimSpace(context))

	if contextString == unavailable {
		err := fmt.Errorf("\"Unavailable\" value from file %q", path)
		if vendorID == "AuthenticAMD" {
			kernelBugzillaLink := "https://bugzilla.kernel.org/show_bug.cgi?id=213311"
			err = fmt.Errorf("%v, possible bug: %q", err, kernelBugzillaLink)
		}
		return 0, err
	}

	stat, err := strconv.ParseUint(contextString, 10, 64)
	if err != nil {
		return stat, fmt.Errorf("unable to parse %q as a uint from file %q", string(context), path)
	}

	return stat, nil
}

func getIntelRDTStatsFrom(path string, vendorID string) (intelrdt.Stats, error) {
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
			llcOccupancy, err := readStatFrom(filepath.Join(dir, llcOccupancyFileName), vendorID)
			if err != nil {
				return stats, err
			}
			cmtStats = append(cmtStats, intelrdt.CMTNumaNodeStats{LLCOccupancy: llcOccupancy})
		}
		if enabledMBM {
			mbmTotalBytes, err := readStatFrom(filepath.Join(dir, mbmTotalBytesFileName), vendorID)
			if err != nil {
				return stats, err
			}
			mbmLocalBytes, err := readStatFrom(filepath.Join(dir, mbmLocalBytesFileName), vendorID)
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
