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

package machine

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func TestGetNumaMemoryStats(t *testing.T) {
	testfile := "./testdata/numa_meminfo"
	test_numa_meminfo, err := ioutil.ReadFile(testfile)
	if err != nil {
		t.Fatalf("unable to read input test file %s", testfile)
	}
	size, free, err := GetNumaMemoryStats(test_numa_meminfo)
	if err != nil {
		t.Errorf("failed to get Memory Stats for sample numa_meminfo %s: %v", string(test_numa_meminfo), err)
	}

	if size != 33455420 {
		t.Errorf("Expected memory size 33455420, found %d", size)
	}
	if free != 14443652 {
		t.Errorf("Expected memory free 14443652, found %d", free)
	}
}

func TestGetNumaCpuDetails(t *testing.T) {
	testfile := "./testdata/numa_cpumap"
	test_numa_cpumap, err := ioutil.ReadFile(testfile)
	if err != nil {
		t.Fatalf("unable to read input test file %s", testfile)
	}
	numCores, cores, err := GetNumaCpuDetails(test_numa_cpumap)
	if err != nil {
		t.Errorf("failed to get cpu details for sample numa_cpumap %s: %v", string(test_numa_cpumap), err)
	}

	if numCores != 12 {
		t.Errorf("Expected number of cores is 12, found %d", numCores)
	}
	expected_cores := []string{"0", "2", "4", "6", "8", "10", "12", "14", "16", "18", "20", "22"}
	if !reflect.DeepEqual(cores, expected_cores) {
		t.Errorf("Expected number of cores is %v, found %v", cores, expected_cores)
	}
}
