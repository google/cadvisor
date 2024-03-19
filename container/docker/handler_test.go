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
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"

	info "github.com/google/cadvisor/info/v1"
)

type mockDockerClientForExitCode struct {
	dclient.APIClient
	inspectResp container.InspectResponse
	inspectErr  error
}

func (m *mockDockerClientForExitCode) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return m.inspectResp, m.inspectErr
}

func TestStorageDirDetectionWithOldVersions(t *testing.T) {
	as := assert.New(t)
	rwLayer, err := getRwLayerID("abcd", "/", AufsStorageDriver, []int{1, 9, 0})
	as.Nil(err)
	as.Equal(rwLayer, "abcd")
}

func TestStorageDirDetectionWithNewVersions(t *testing.T) {
	as := assert.New(t)
	testDir := t.TempDir()
	containerID := "abcd"
	randomizedID := "xyz"
	randomIDPath := path.Join(testDir, "image/aufs/layerdb/mounts/", containerID)
	as.Nil(os.MkdirAll(randomIDPath, os.ModePerm))
	as.Nil(os.WriteFile(path.Join(randomIDPath, "mount-id"), []byte(randomizedID), os.ModePerm))
	rwLayer, err := getRwLayerID(containerID, testDir, "aufs", []int{1, 10, 0})
	as.Nil(err)
	as.Equal(rwLayer, randomizedID)
	rwLayer, err = getRwLayerID(containerID, testDir, "aufs", []int{1, 10, 0})
	as.Nil(err)
	as.Equal(rwLayer, randomizedID)

}

func rawMetadataEnvMatch(dockerEnvWhiteList string, cntConfig container.Config) map[string]string {
	metadataEnvAllowList := strings.Split(dockerEnvWhiteList, ",")
	handlerEnvs := make(map[string]string)

	// split env vars to get metadata map.
	for _, exposedEnv := range metadataEnvAllowList {
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
	metadataEnvAllowList := strings.Split(dockerEnvWhiteList, ",")
	handlerEnvs := make(map[string]string)

	// split env vars to get metadata map.
	for _, exposedEnv := range metadataEnvAllowList {
		if exposedEnv == "" {
			// if no dockerEnvWhitelist provided, len(metadataEnvAllowList) == 1, metadataEnvAllowList[0] == ""
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

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name         string
		running      bool
		exitCode     int
		inspectErr   error
		expectErr    bool
		errContains  string
		expectedCode int
	}{
		{
			name:         "successful exit code 0",
			running:      false,
			exitCode:     0,
			inspectErr:   nil,
			expectErr:    false,
			expectedCode: 0,
		},
		{
			name:         "successful exit code 1",
			running:      false,
			exitCode:     1,
			inspectErr:   nil,
			expectErr:    false,
			expectedCode: 1,
		},
		{
			name:         "container still running",
			running:      true,
			exitCode:     0,
			inspectErr:   nil,
			expectErr:    true,
			errContains:  "still running",
			expectedCode: -1,
		},
		{
			name:         "inspect fails",
			running:      false,
			exitCode:     0,
			inspectErr:   assert.AnError,
			expectErr:    true,
			errContains:  "failed to inspect",
			expectedCode: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := assert.New(t)

			inspectResp := container.InspectResponse{
				ContainerJSONBase: &container.ContainerJSONBase{
					State: &container.State{
						Running:  tt.running,
						ExitCode: tt.exitCode,
					},
				},
			}

			mockClient := &mockDockerClientForExitCode{
				inspectResp: inspectResp,
				inspectErr:  tt.inspectErr,
			}

			h := &containerHandler{
				client: mockClient,
				reference: info.ContainerReference{
					Id: "test-container-id",
				},
			}

			code, err := h.GetExitCode()

			if tt.expectErr {
				as.Error(err)
				if tt.errContains != "" {
					as.Contains(err.Error(), tt.errContains)
				}
				as.Equal(tt.expectedCode, code)
			} else {
				as.NoError(err)
				as.Equal(tt.expectedCode, code)
			}
		})
	}
}
