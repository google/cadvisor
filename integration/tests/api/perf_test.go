// +build libpfm,cgo

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

package api

import (
	"testing"
	"time"

	v1 "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"

	"github.com/stretchr/testify/assert"
)

func TestPerfEvents(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Docker().RunPause()
	waitForContainerInfo(fm, containerID)

	info, err := fm.Cadvisor().Client().DockerContainer(containerID, &v1.ContainerInfoRequest{
		NumStats: 1,
	})

	assert.Nil(t, err)
	assert.Len(t, info.Stats, 1)
	assert.Greater(t, len(info.Stats[0].PerfStats), 0, "Length of info.Stats[0].PerfStats is not greater than zero")
	for k, stat := range info.Stats[0].PerfStats {
		//Everything beyond name is non-deterministic unfortunately.
		assert.Contains(t, []string{"context-switches", "cpu-migrations-custom"}, stat.Name, "Wrong metric name for key %d: %#v", k, stat)
	}
}

func waitForContainerInfo(fm framework.Framework, containerID string) {
	err := framework.RetryForDuration(func() error {
		_, err := fm.Cadvisor().Client().DockerContainer(containerID, &v1.ContainerInfoRequest{NumStats: 1})
		if err != nil {
			return err
		}
		return nil
	}, 5*time.Second)
	assert.NoError(fm.T(), err)
}
