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
