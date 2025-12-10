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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"
)

func TestStreamingEventInformationIsReturned(t *testing.T) {
	// TODO(vmarmol): De-flake and re-enable.
	t.Skip()

	fm := framework.New(t)
	defer fm.Cleanup()

	// Watch for container deletions
	einfo := make(chan *info.Event)
	go func() {
		err := fm.Cadvisor().Client().EventStreamingInfo("?deletion_events=true&stream=true&subcontainers=true", einfo)
		require.NoError(t, err)
	}()

	// Create a short-lived container.
	containerID := fm.Docker().RunBusybox("sleep", "2")

	// Wait for the deletion event.
	timeout := time.After(30 * time.Second)
	done := false
	for !done {
		select {
		case ev := <-einfo:
			if ev.EventType == info.EventContainerDeletion {
				if strings.Contains(ev.ContainerName, containerID) {
					done = true
				}
			}
		case <-timeout:
			t.Errorf(
				"timeout happened before destruction event was detected for container %q", containerID)
			done = true
		}
	}

	// We should have already received a creation event.
	waitForStaticEvent(containerID, "?creation_events=true&subcontainers=true", t, fm, info.EventContainerCreation)
}

func waitForStaticEvent(containerID string, urlRequest string, t *testing.T, fm framework.Framework, typeEvent info.EventType) {
	einfo, err := fm.Cadvisor().Client().EventStaticInfo(urlRequest)
	require.NoError(t, err)
	found := false
	for _, ev := range einfo {
		if ev.EventType == typeEvent {
			if strings.Contains(ev.ContainerName, containerID) {
				found = true
				break
			}
		}
	}
	require.True(t, found)
}

func TestContainerDeletionExitCode(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
	}{
		{
			name:     "successful exit",
			exitCode: 0,
		},
		{
			name:     "error exit",
			exitCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := framework.New(t)
			defer fm.Cleanup()

			containerID := fm.Docker().RunBusybox("sh", "-c", "exit "+strconv.Itoa(tt.exitCode))

			err := framework.RetryForDuration(func() error {
				events, err := fm.Cadvisor().Client().EventStaticInfo("?deletion_events=true&subcontainers=true")
				if err != nil {
					return err
				}

				for _, ev := range events {
					if ev.EventType == info.EventContainerDeletion &&
						strings.Contains(ev.ContainerName, containerID) {
						if ev.EventData.ContainerDeletion == nil {
							return fmt.Errorf("deletion event data is nil")
						}
						if ev.EventData.ContainerDeletion.ExitCode != tt.exitCode {
							t.Errorf("expected exit code %d, got %d",
								tt.exitCode, ev.EventData.ContainerDeletion.ExitCode)
						}
						return nil
					}
				}
				return fmt.Errorf("deletion event not found for container %s", containerID)
			}, 30*time.Second)

			require.NoError(t, err)
		})
	}
}
