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

package sysfs

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNodes(t *testing.T) {
	//overwrite global variable
	nodeDir = "./testdata/"

	sysFs := NewRealSysFs()
	nodesDirs, err := sysFs.GetNodesPaths()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(nodesDirs))
	assert.Contains(t, nodesDirs, "testdata/node0")
	assert.Contains(t, nodesDirs, "testdata/node1")
}

func TestGetNodesWithNonExistingDir(t *testing.T) {
	//overwrite global variable
	nodeDir = "./testdata/NonExistingDir/"

	sysFs := NewRealSysFs()
	nodesDirs, err := sysFs.GetNodesPaths()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(nodesDirs))
}

func TestGetCPUsPaths(t *testing.T) {
	sysFs := NewRealSysFs()
	cpuDirs, err := sysFs.GetCPUsPaths("./testdata/node0")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(cpuDirs))
	assert.Contains(t, cpuDirs, "testdata/node0/cpu0")
	assert.Contains(t, cpuDirs, "testdata/node0/cpu1")
}

func TestGetCPUsPathsFromNodeWithoutCPU(t *testing.T) {
	sysFs := NewRealSysFs()
	cpuDirs, err := sysFs.GetCPUsPaths("./testdata/node1")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(cpuDirs))
}

func TestGetCoreID(t *testing.T) {
	sysFs := NewRealSysFs()
	rawCoreID, err := sysFs.GetCoreID("./testdata/node0/cpu0")
	assert.Nil(t, err)

	coreID, err := strconv.Atoi(rawCoreID)
	assert.Nil(t, err)
	assert.Equal(t, 0, coreID)
}

func TestGetCoreIDWhenFileIsMissing(t *testing.T) {
	sysFs := NewRealSysFs()
	rawCoreID, err := sysFs.GetCoreID("./testdata/node0/cpu1")
	assert.NotNil(t, err)
	assert.Equal(t, "", rawCoreID)
}

func TestGetMemInfo(t *testing.T) {
	sysFs := NewRealSysFs()
	memInfo, err := sysFs.GetMemInfo("./testdata/node0")
	assert.Nil(t, err)
	assert.Equal(t, "Node 0 MemTotal:       32817192 kB", memInfo)
}

func TestGetMemInfoWhenFileIsMissing(t *testing.T) {
	sysFs := NewRealSysFs()
	memInfo, err := sysFs.GetMemInfo("./testdata/node1")
	assert.NotNil(t, err)
	assert.Equal(t, "", memInfo)
}

func TestGetHugePagesInfo(t *testing.T) {
	sysFs := NewRealSysFs()
	hugePages, err := sysFs.GetHugePagesInfo("./testdata/node0/hugepages")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(hugePages))
	assert.Equal(t, "hugepages-1048576kB", hugePages[0].Name())
	assert.Equal(t, "hugepages-2048kB", hugePages[1].Name())
}

func TestGetHugePagesInfoWhenDirIsMissing(t *testing.T) {
	sysFs := NewRealSysFs()
	hugePages, err := sysFs.GetHugePagesInfo("./testdata/node1/hugepages")
	assert.NotNil(t, err)
	assert.Equal(t, []os.FileInfo([]os.FileInfo(nil)), hugePages)
}

func TestGetHugePagesNr(t *testing.T) {
	sysFs := NewRealSysFs()
	rawHugePageNr, err := sysFs.GetHugePagesNr("./testdata/node0/hugepages/", "hugepages-1048576kB")
	assert.Nil(t, err)

	hugePageNr, err := strconv.Atoi(rawHugePageNr)
	assert.Nil(t, err)
	assert.Equal(t, 1, hugePageNr)
}

func TestGetHugePagesNrWhenFileIsMissing(t *testing.T) {
	sysFs := NewRealSysFs()
	rawHugePageNr, err := sysFs.GetHugePagesNr("./testdata/node1/hugepages/", "hugepages-1048576kB")
	assert.NotNil(t, err)
	assert.Equal(t, "", rawHugePageNr)
}

func TestIsCPUOnline(t *testing.T) {
	sysFs := &realSysFs{
		cpuPath: "./testdata_epyc7402_nohyperthreading",
	}
	online := sysFs.IsCPUOnline("./testdata_epyc7402_nohyperthreading/cpu14")
	assert.True(t, online)

	online = sysFs.IsCPUOnline("./testdata_epyc7402_nohyperthreading/cpu13")
	assert.False(t, online)
}

func TestIsCPUOnlineNoFileAndCPU0MustBeOnline(t *testing.T) {
	// Test on x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = false

	sysFs := &realSysFs{
		cpuPath: "./testdata/missing_online/node0",
	}
	online := sysFs.IsCPUOnline("./testdata/missing_online/node0/cpu33")
	assert.True(t, online)
}

func TestIsCPU0OnlineOnx86(t *testing.T) {
	// Test on x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = true

	sysFs := NewRealSysFs()
	online := sysFs.IsCPUOnline("path/is/irrelevant/it/is/zero/that/matters/cpu0")
	assert.True(t, online)
}

func TestCPU0OfflineOnNotx86(t *testing.T) {
	// Test on arch other than x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = false

	sysFs := &realSysFs{
		cpuPath: "./testdata_graviton2",
	}
	online := sysFs.IsCPUOnline("./testdata_graviton2/cpu0")
	assert.False(t, online)
}

func TestIsCpuOnlineRaspberryPi4(t *testing.T) {
	// Test on arch other than x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = false

	sysFS := &realSysFs{
		cpuPath: "./testdata_rpi4",
	}
	online := sysFS.IsCPUOnline("./testdata_rpi4/cpu0")
	assert.True(t, online)

	online = sysFS.IsCPUOnline("./testdata_rpi4/cpu1")
	assert.True(t, online)

	online = sysFS.IsCPUOnline("./testdata_rpi4/cpu2")
	assert.True(t, online)

	online = sysFS.IsCPUOnline("./testdata_rpi4/cpu3")
	assert.True(t, online)
}

func TestIsCpuOnlineGraviton2(t *testing.T) {
	// Test on arch other than x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = false

	sysFS := &realSysFs{
		cpuPath: "./testdata_graviton2",
	}
	online := sysFS.IsCPUOnline("./testdata_graviton2/cpu0")
	assert.False(t, online)

	online = sysFS.IsCPUOnline("./testdata_graviton2/cpu1")
	assert.True(t, online)

	online = sysFS.IsCPUOnline("./testdata_graviton2/cpu2")
	assert.True(t, online)

	online = sysFS.IsCPUOnline("./testdata_graviton2/cpu3")
	assert.True(t, online)
}

func TestGetUniqueCPUPropertyCountOnRaspberryPi4(t *testing.T) {
	// Test on arch other than x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = false

	count := GetUniqueCPUPropertyCount("./testdata_rpi4/", CPUPhysicalPackageID)
	assert.Equal(t, 1, count)

	count = GetUniqueCPUPropertyCount("./testdata_rpi4/", CPUCoreID)
	assert.Equal(t, 4, count)
}

func TestGetUniqueCPUPropertyCountOnEpyc7402(t *testing.T) {
	// Test on x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = true

	count := GetUniqueCPUPropertyCount("./testdata_epyc7402/", CPUPhysicalPackageID)
	assert.Equal(t, 1, count)

	count = GetUniqueCPUPropertyCount("./testdata_epyc7402/", CPUCoreID)
	assert.Equal(t, 24, count)
}

func TestGetUniqueCPUPropertyCountOnEpyc7402NoHyperThreading(t *testing.T) {
	// Test on x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = true

	count := GetUniqueCPUPropertyCount("./testdata_epyc7402_nohyperthreading/", CPUPhysicalPackageID)
	assert.Equal(t, 1, count)

	count = GetUniqueCPUPropertyCount("./testdata_epyc7402_nohyperthreading/", CPUCoreID)
	assert.Equal(t, 17, count)
}

func TestGetUniqueCPUPropertyCountOnXeon4214(t *testing.T) {
	// Test on x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = true

	count := GetUniqueCPUPropertyCount("./testdata_xeon4214_2socket/", CPUPhysicalPackageID)
	assert.Equal(t, 2, count)

	count = GetUniqueCPUPropertyCount("./testdata_xeon4214_2socket/", CPUCoreID)
	assert.Equal(t, 24, count)
}

func TestGetUniqueCPUPropertyCountOnXeon5218NoHyperThreadingNoHotplug(t *testing.T) {
	// Test on x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = true

	count := GetUniqueCPUPropertyCount("./testdata_xeon5218_nohyperthread_2socket_nohotplug/", CPUPhysicalPackageID)
	assert.Equal(t, 2, count)

	count = GetUniqueCPUPropertyCount("./testdata_xeon5218_nohyperthread_2socket_nohotplug/", CPUCoreID)
	assert.Equal(t, 32, count)
}

func TestUniqueCPUPropertyOnSingleSocketMultipleNUMAsSystem(t *testing.T) {
	// Test on x86.
	origIsX86 := isX86
	defer func() {
		isX86 = origIsX86
	}()
	isX86 = true

	count := GetUniqueCPUPropertyCount("./testdata_single_socket_many_NUMAs/", CPUPhysicalPackageID)
	assert.Equal(t, 16, count)

	count = GetUniqueCPUPropertyCount("./testdata_single_socket_many_NUMAs/", CPUCoreID)
	assert.Equal(t, 16, count)
}

func TestGetDistances(t *testing.T) {
	sysFs := NewRealSysFs()
	distances, err := sysFs.GetDistances("./testdata/node0")
	assert.Nil(t, err)
	assert.Equal(t, "10 11", distances)
}

func TestGetDistancesFileIsMissing(t *testing.T) {
	sysFs := NewRealSysFs()
	distances, err := sysFs.GetDistances("./testdata/node1")
	assert.NotNil(t, err)
	assert.Equal(t, "", distances)
}
