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
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
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
