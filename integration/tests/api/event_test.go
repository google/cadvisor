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
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"
	"github.com/stretchr/testify/require"
)

func TestStreamingEventInformationIsReturned(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	einfo := make(chan *info.Event)
	go func() {
		err := fm.Cadvisor().Client().EventStreamingInfo("?oom_events=true&stream=true", einfo)
		t.Logf("tried to stream events but got error %v", err)
		require.NoError(t, err)
	}()

	containerName := fmt.Sprintf("test-basic-docker-container-%d", os.Getpid())
	containerId := fm.Docker().RunStress(framework.DockerRunArgs{
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

	waitForStreamingEvent(containerId, "?deletion_events=true&stream=true", t, fm, info.EventContainerDeletion)
	waitForStaticEvent(containerId, "?creation_events=true", t, fm, info.EventContainerCreation)
}

func waitForStreamingEvent(containerId string, urlRequest string, t *testing.T, fm framework.Framework, typeEvent info.EventType) {
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(60 * time.Second)
		timeout <- true
	}()

	einfo := make(chan *info.Event)
	go func() {
		err := fm.Cadvisor().Client().EventStreamingInfo(urlRequest, einfo)
		require.NoError(t, err)
	}()
	for {
		select {
		case ev := <-einfo:
			if ev.EventType == typeEvent {
				if strings.Contains(strings.Trim(ev.ContainerName, "/ "), strings.Trim(containerId, "/ ")) {
					return
				}
			}
		case <-timeout:
			t.Fatal(
				"timeout happened before destruction event was detected")
		}
	}
}

func waitForStaticEvent(containerId string, urlRequest string, t *testing.T, fm framework.Framework, typeEvent info.EventType) {
	einfo, err := fm.Cadvisor().Client().EventStaticInfo(urlRequest)
	require.NoError(t, err)

	found := false
	for _, ev := range einfo {
		if ev.EventType == typeEvent {
			if strings.Contains(strings.Trim(ev.ContainerName, "/ "), strings.Trim(containerId, "/ ")) {
				found = true
				break
			}
		}
	}
	require.True(t, found)
}
