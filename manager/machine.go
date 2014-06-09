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

package manager

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"

	"github.com/google/cadvisor/info"
)

var numCpuRegexp = regexp.MustCompile("processor\\t*: +[0-9]+")
var memoryCapacityRegexp = regexp.MustCompile("MemTotal: *([0-9]+) kB")

func getMachineInfo() (*info.MachineInfo, error) {
	// Get the number of CPUs from /proc/cpuinfo.
	out, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	numCores := len(numCpuRegexp.FindAll(out, -1))
	if numCores == 0 {
		return nil, fmt.Errorf("failed to count cores in output: %s", string(out))
	}

	// Get the amount of usable memory from /proc/meminfo.
	out, err = ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	matches := memoryCapacityRegexp.FindSubmatch(out)
	if len(matches) != 2 {
		return nil, fmt.Errorf("failed to find memory capacity in output: %s", string(out))
	}
	memoryCapacity, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		return nil, err
	}

	// Capacity is in KB, convert it to bytes.
	memoryCapacity = memoryCapacity * 1024

	return &info.MachineInfo{
		NumCores:       numCores,
		MemoryCapacity: memoryCapacity,
	}, nil
}
