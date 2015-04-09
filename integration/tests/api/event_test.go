// Copyright 2015 Google Inc. All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"
	"github.com/google/cadvisor/utils/oomparser"
	"github.com/stretchr/testify/require"
)

func TestStreamingEventInformationIsReturned(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	einfo := make(chan *info.Event)
	go func() {
		err := fm.Cadvisor().Client().EventStreamingInfo("?oom_events=true", einfo)
		t.Logf("Started event streaming with error %v", err)
		require.NoError(t, err)
	}()

	containerName := fmt.Sprintf("test-basic-docker-container-%d", os.Getpid())
	fm.Docker().RunStress(framework.DockerRunArgs{
		Image: "bernardo/stress",
		Args:  []string{"--name", containerName},
		InnerArgs: []string{
			"--vm", strconv.FormatUint(4, 10),
			"--vm-keep",
			"-q",
			"-m", strconv.FormatUint(1000, 10),
			"--timeout", strconv.FormatUint(10, 10),
		},
	})

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(60 * time.Second)
		timeout <- true
	}()

	select {
	case ev := <-einfo:
		if ev.EventType == 0 {
			marshaledData, err := json.Marshal(ev.EventData)
			require.Nil(t, err)
			var oomEvent *oomparser.OomInstance
			err = json.Unmarshal(marshaledData, &oomEvent)
			require.Nil(t, err)
			require.True(t, oomEvent.ProcessName == "stress")
		}
	case <-timeout:
		t.Fatal(
			"timeout happened before event was detected")
	}
}
