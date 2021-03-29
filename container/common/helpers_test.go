// Copyright 2018 Google Inc. All Rights Reserved.
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

package common

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	info "github.com/google/cadvisor/info/v1"
	v2 "github.com/google/cadvisor/info/v2"
	"github.com/stretchr/testify/assert"
)

func BenchmarkListDirectories(b *testing.B) {
	for i := 0; i < b.N; i++ {
		output := make(map[string]struct{})
		if err := ListDirectories("/sys/fs/cgroup", "", true, output); err != nil {
			b.Fatal(err)
		}
	}
}

func TestConvertCpuWeightToCpuLimit(t *testing.T) {
	limit, err := convertCPUWeightToCPULimit(1)
	if err != nil {
		t.Fatalf("Error in convertCPUWeightToCPULimit: %s", err)
	}
	if limit != 2 {
		t.Fatalf("convertCPUWeightToCPULimit(1) != 2")
	}
	limit, err = convertCPUWeightToCPULimit(10000)
	if err != nil {
		t.Fatalf("Error in convertCPUWeightToCPULimit: %s", err)
	}
	if limit != 262144 {
		t.Fatalf("convertCPUWeightToCPULimit(10000) != 262144")
	}
	_, err = convertCPUWeightToCPULimit(0)
	if err == nil {
		t.Fatalf("convertCPUWeightToCPULimit(0) must raise an error")
	}
	_, err = convertCPUWeightToCPULimit(10001)
	if err == nil {
		t.Fatalf("convertCPUWeightToCPULimit(10001) must raise an error")
	}
}

func TestParseUint64String(t *testing.T) {
	if parseUint64String("1000") != 1000 {
		t.Fatalf("parseUint64String(\"1000\") != 1000")
	}
	if parseUint64String("-1") != 0 {
		t.Fatalf("parseUint64String(\"-1\") != 0")
	}
	if parseUint64String("0") != 0 {
		t.Fatalf("parseUint64String(\"0\") != 0")
	}
	if parseUint64String("not-a-number") != 0 {
		t.Fatalf("parseUint64String(\"not-a-number\") != 0")
	}
	if parseUint64String(" 1000 ") != 0 {
		t.Fatalf("parseUint64String(\" 1000 \") != 0")
	}
	if parseUint64String("18446744073709551615") != 18446744073709551615 {
		t.Fatalf("parseUint64String(\"18446744073709551615\") != 18446744073709551615")
	}
}

type mockInfoProvider struct {
	options v2.RequestOptions
}

func (m *mockInfoProvider) GetRequestedContainersInfo(containerName string, options v2.RequestOptions) (map[string]*info.ContainerInfo, error) {
	m.options = options
	return map[string]*info.ContainerInfo{}, nil
}

func (m *mockInfoProvider) GetVersionInfo() (*info.VersionInfo, error) {
	return nil, errors.New("not supported")
}

func (m *mockInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	return &info.MachineInfo{
		NumCores: 7,
	}, nil
}

func TestGetSpecCgroupV1(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %s", err)
	}

	cgroupPaths := map[string]string{
		"memory": filepath.Join(root, "test_resources/cgroup_v1/test1/memory"),
		"cpu":    filepath.Join(root, "test_resources/cgroup_v1/test1/cpu"),
		"cpuset": filepath.Join(root, "test_resources/cgroup_v1/test1/cpuset"),
		"pids":   filepath.Join(root, "test_resources/cgroup_v1/test1/pids"),
	}

	spec, err := getSpecInternal(cgroupPaths, &mockInfoProvider{}, false, false, false)
	assert.Nil(t, err)

	assert.True(t, spec.HasMemory)
	assert.EqualValues(t, spec.Memory.Limit, 123456789)
	assert.EqualValues(t, spec.Memory.SwapLimit, 13579)
	assert.EqualValues(t, spec.Memory.Reservation, 24680)

	assert.True(t, spec.HasCpu)
	assert.EqualValues(t, spec.Cpu.Limit, 1025)
	assert.EqualValues(t, spec.Cpu.Period, 100010)
	assert.EqualValues(t, spec.Cpu.Quota, 20000)

	assert.EqualValues(t, spec.Cpu.Mask, "0-5")

	assert.True(t, spec.HasProcesses)
	assert.EqualValues(t, spec.Processes.Limit, 1027)

	assert.False(t, spec.HasHugetlb)
	assert.False(t, spec.HasDiskIo)
}

func TestGetSpecCgroupV2(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %s", err)
	}

	cgroupPaths := map[string]string{
		"memory": filepath.Join(root, "test_resources/cgroup_v2/test1"),
		"cpu":    filepath.Join(root, "test_resources/cgroup_v2/test1"),
		"cpuset": filepath.Join(root, "test_resources/cgroup_v2/test1"),
		"pids":   filepath.Join(root, "test_resources/cgroup_v2/test1"),
	}

	spec, err := getSpecInternal(cgroupPaths, &mockInfoProvider{}, false, false, true)
	assert.Nil(t, err)

	assert.True(t, spec.HasMemory)
	assert.EqualValues(t, spec.Memory.Limit, 123456789)
	assert.EqualValues(t, spec.Memory.SwapLimit, 13579)
	assert.EqualValues(t, spec.Memory.Reservation, 24680)

	assert.True(t, spec.HasCpu)
	assert.EqualValues(t, spec.Cpu.Limit, 1286)
	assert.EqualValues(t, spec.Cpu.Period, 100010)
	assert.EqualValues(t, spec.Cpu.Quota, 20000)

	assert.EqualValues(t, spec.Cpu.Mask, "0-5")

	assert.True(t, spec.HasProcesses)
	assert.EqualValues(t, spec.Processes.Limit, 1027)

	assert.False(t, spec.HasHugetlb)
	assert.False(t, spec.HasDiskIo)
}

func TestGetSpecCgroupV2Max(t *testing.T) {
	root, err := os.Getwd()
	assert.Nil(t, err)

	cgroupPaths := map[string]string{
		"memory": filepath.Join(root, "test_resources/cgroup_v2/test2"),
		"cpu":    filepath.Join(root, "test_resources/cgroup_v2/test2"),
		"pids":   filepath.Join(root, "test_resources/cgroup_v2/test2"),
	}

	spec, err := getSpecInternal(cgroupPaths, &mockInfoProvider{}, false, false, true)
	assert.Nil(t, err)

	max := uint64(math.MaxUint64)

	assert.True(t, spec.HasMemory)
	assert.EqualValues(t, spec.Memory.Limit, max)
	assert.EqualValues(t, spec.Memory.SwapLimit, max)
	assert.EqualValues(t, spec.Memory.Reservation, max)

	assert.True(t, spec.HasCpu)
	assert.EqualValues(t, spec.Cpu.Limit, 1286)
	assert.EqualValues(t, spec.Cpu.Period, 100010)
	assert.EqualValues(t, spec.Cpu.Quota, 0)

	assert.EqualValues(t, spec.Processes.Limit, max)
}
