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

// Handler for Docker containers.
package docker

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"

	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
)

func TestStorageDirDetectionWithOldVersions(t *testing.T) {
	as := assert.New(t)
	rwLayer, err := getRwLayerID("abcd", "/", aufsStorageDriver, []int{1, 9, 0})
	as.Nil(err)
	as.Equal(rwLayer, "abcd")
}

func TestStorageDirDetectionWithNewVersions(t *testing.T) {
	as := assert.New(t)
	testDir, err := ioutil.TempDir("", "")
	as.Nil(err)
	containerID := "abcd"
	randomizedID := "xyz"
	randomIDPath := path.Join(testDir, "image/aufs/layerdb/mounts/", containerID)
	as.Nil(os.MkdirAll(randomIDPath, os.ModePerm))
	as.Nil(ioutil.WriteFile(path.Join(randomIDPath, "mount-id"), []byte(randomizedID), os.ModePerm))
	rwLayer, err := getRwLayerID(containerID, testDir, "aufs", []int{1, 10, 0})
	as.Nil(err)
	as.Equal(rwLayer, randomizedID)
	rwLayer, err = getRwLayerID(containerID, testDir, "aufs", []int{1, 10, 0})
	as.Nil(err)
	as.Equal(rwLayer, randomizedID)

}

func rawMetadataEnvMatch(dockerEnvWhiteList string, cntConfig container.Config) map[string]string {
	metadataEnvs := strings.Split(dockerEnvWhiteList, ",")
	handlerEnvs := make(map[string]string)

	// split env vars to get metadata map.
	for _, exposedEnv := range metadataEnvs {
		for _, envVar := range cntConfig.Env {
			if envVar != "" {
				splits := strings.SplitN(envVar, "=", 2)
				if len(splits) == 2 && splits[0] == exposedEnv {
					handlerEnvs[strings.ToLower(exposedEnv)] = splits[1]
				}
			}
		}
	}

	return handlerEnvs
}

func newMetadataEnvMatch(dockerEnvWhiteList string, cntConfig container.Config) map[string]string {
	metadataEnvs := strings.Split(dockerEnvWhiteList, ",")
	handlerEnvs := make(map[string]string)

	// split env vars to get metadata map.
	for _, exposedEnv := range metadataEnvs {
		if exposedEnv == "" {
			// if no dockerEnvWhitelist provided, len(metadataEnvs) == 1, metadataEnvs[0] == ""
			continue
		}

		for _, envVar := range cntConfig.Env {
			if envVar != "" {
				splits := strings.SplitN(envVar, "=", 2)
				if len(splits) == 2 && strings.HasPrefix(splits[0], exposedEnv) {
					handlerEnvs[strings.ToLower(splits[0])] = splits[1]
				}
			}
		}
	}

	return handlerEnvs
}

func TestDockerEnvWhitelist(t *testing.T) {
	as := assert.New(t)

	envTotalMatch := "TEST_REGION,TEST_ZONE"
	envMatchWithPrefix := "TEST_"
	envMatchWithPrefixEmpty := ""

	rawCntConfig := container.Config{Env: []string{"TEST_REGION=FRA", "TEST_ZONE=A", "HELLO=WORLD"}}
	newCntConfig := container.Config{Env: []string{"TEST_REGION=FRA", "TEST_ZONE=A", "TEST_POOL=TOOLING", "HELLO=WORLD"}}

	rawExpected := map[string]string{
		"test_region": "FRA",
		"test_zone":   "A",
	}
	newExpected := map[string]string{
		"test_region": "FRA",
		"test_zone":   "A",
		"test_pool":   "TOOLING",
	}
	emptyExpected := map[string]string{}

	rawEnvsTotalMatch := rawMetadataEnvMatch(envTotalMatch, rawCntConfig)
	newEnvsTotalMatch := newMetadataEnvMatch(envTotalMatch, rawCntConfig)

	// make sure total match does not change
	as.Equal(rawEnvsTotalMatch, newEnvsTotalMatch)
	as.Equal(rawEnvsTotalMatch, rawExpected)

	rawEnvsTotalMatch2 := rawMetadataEnvMatch(envTotalMatch, newCntConfig)
	newEnvsTotalMatch2 := newMetadataEnvMatch(envTotalMatch, newCntConfig)

	// make sure total match does not change with more envs exposed
	as.Equal(rawEnvsTotalMatch2, newEnvsTotalMatch2)
	as.Equal(rawEnvsTotalMatch2, rawExpected)

	newEnvsMatchWithPrefix := newMetadataEnvMatch(envMatchWithPrefix, rawCntConfig)
	newEnvsMatchWithPrefix2 := newMetadataEnvMatch(envMatchWithPrefix, newCntConfig)

	// make sure new method can return envs with prefix specified
	as.Equal(newEnvsMatchWithPrefix, rawExpected)
	as.Equal(newEnvsMatchWithPrefix2, newExpected)

	newEnvsMatchWithEmptyPrefix := newMetadataEnvMatch(envMatchWithPrefixEmpty, newCntConfig)
	rawEnvsMatchWithEmptyWhitelist := rawMetadataEnvMatch(envMatchWithPrefixEmpty, newCntConfig)

	// make sure empty whitelist returns nothing
	as.Equal(newEnvsMatchWithEmptyPrefix, emptyExpected)
	as.Equal(rawEnvsMatchWithEmptyWhitelist, emptyExpected)

}

func TestAddDiskStatsCheck(t *testing.T) {
	var readsCompleted, readsMerged, sectorsRead, readTime, writesCompleted, writesMerged, sectorsWritten,
		writeTime, ioInProgress, ioTime, weightedIoTime uint64 = 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11

	fileSystem := fs.Fs{
		DiskStats: fs.DiskStats{
			ReadsCompleted:  readsCompleted,
			ReadsMerged:     readsMerged,
			SectorsRead:     sectorsRead,
			ReadTime:        readTime,
			WritesCompleted: writesCompleted,
			WritesMerged:    writesMerged,
			SectorsWritten:  sectorsWritten,
			WriteTime:       writeTime,
			IoInProgress:    ioInProgress,
			IoTime:          ioTime,
			WeightedIoTime:  weightedIoTime,
		},
	}

	fileSystems := []fs.Fs{fileSystem}

	var fsStats info.FsStats
	addDiskStats(fileSystems, nil, &fsStats)
}

func TestAddDiskStats(t *testing.T) {
	// Arrange
	as := assert.New(t)
	var readsCompleted, readsMerged, sectorsRead, readTime, writesCompleted, writesMerged, sectorsWritten,
		writeTime, ioInProgress, ioTime, weightedIoTime uint64 = 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11
	var fsStats info.FsStats

	fsInfo := info.FsInfo{
		DeviceMajor: 4,
		DeviceMinor: 64,
	}

	fileSystem := fs.Fs{
		DiskStats: fs.DiskStats{
			ReadsCompleted:  readsCompleted,
			ReadsMerged:     readsMerged,
			SectorsRead:     sectorsRead,
			ReadTime:        readTime,
			WritesCompleted: writesCompleted,
			WritesMerged:    writesMerged,
			SectorsWritten:  sectorsWritten,
			WriteTime:       writeTime,
			IoInProgress:    ioInProgress,
			IoTime:          ioTime,
			WeightedIoTime:  weightedIoTime,
		},
	}

	fileSystems := []fs.Fs{fileSystem}

	// Act
	addDiskStats(fileSystems, &fsInfo, &fsStats)

	// Assert
	as.Equal(readsCompleted, fileSystem.DiskStats.ReadsCompleted, "ReadsCompleted metric should be %d but was %d", readsCompleted, fileSystem.DiskStats.ReadsCompleted)
	as.Equal(readsMerged, fileSystem.DiskStats.ReadsMerged, "ReadsMerged metric should be %d but was %d", readsMerged, fileSystem.DiskStats.ReadsMerged)
	as.Equal(sectorsRead, fileSystem.DiskStats.SectorsRead, "SectorsRead metric should be %d but was %d", sectorsRead, fileSystem.DiskStats.SectorsRead)
	as.Equal(readTime, fileSystem.DiskStats.ReadTime, "ReadTime metric should be %d but was %d", readTime, fileSystem.DiskStats.ReadTime)
	as.Equal(writesCompleted, fileSystem.DiskStats.WritesCompleted, "WritesCompleted metric should be %d but was %d", writesCompleted, fileSystem.DiskStats.WritesCompleted)
	as.Equal(writesMerged, fileSystem.DiskStats.WritesMerged, "WritesMerged metric should be %d but was %d", writesMerged, fileSystem.DiskStats.WritesMerged)
	as.Equal(sectorsWritten, fileSystem.DiskStats.SectorsWritten, "SectorsWritten metric should be %d but was %d", sectorsWritten, fileSystem.DiskStats.SectorsWritten)
	as.Equal(writeTime, fileSystem.DiskStats.WriteTime, "WriteTime metric should be %d but was %d", writeTime, fileSystem.DiskStats.WriteTime)
	as.Equal(ioInProgress, fileSystem.DiskStats.IoInProgress, "IoInProgress metric should be %d but was %d", ioInProgress, fileSystem.DiskStats.IoInProgress)
	as.Equal(ioTime, fileSystem.DiskStats.IoTime, "IoTime metric should be %d but was %d", ioTime, fileSystem.DiskStats.IoTime)
	as.Equal(weightedIoTime, fileSystem.DiskStats.WeightedIoTime, "WeightedIoTime metric should be %d but was %d", weightedIoTime, fileSystem.DiskStats.WeightedIoTime)
}
