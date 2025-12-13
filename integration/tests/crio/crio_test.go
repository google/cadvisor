// Copyright 2024 Google Inc. All Rights Reserved.
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

package crio

import (
	"fmt"
	"os"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Waits up to 10s for a CRI-O container with the specified ID to appear in cAdvisor.
// CRI-O containers may take longer to appear due to the pod sandbox model.
func waitForCrioContainer(containerID string, fm framework.Framework) {
	err := framework.RetryForDuration(func() error {
		// Query all containers via SubcontainersInfo - CRI-O containers are in "crio" namespace,
		// not "docker" namespace, so AllDockerContainers won't find them.
		allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
			NumStats: 1,
		})
		if err != nil {
			return err
		}

		// Look for container by ID (CRI-O containers appear with crio- prefix in cgroup)
		for _, container := range allInfo {
			for _, alias := range container.Aliases {
				if alias == containerID {
					return nil
				}
			}
			// Also check if the container name contains the ID
			if len(container.Name) > 0 && contains(container.Name, containerID) {
				return nil
			}
		}
		return fmt.Errorf("container %q not found in cAdvisor", containerID)
	}, 10*time.Second)
	require.NoError(fm.T(), err, "Timed out waiting for CRI-O container %q to be available in cAdvisor", containerID)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Sanity check the container by:
// - Checking that the specified ID is associated with this container.
// - Verifying that stats are not empty.
func sanityCheck(containerID string, containerInfo info.ContainerInfo, t *testing.T) {
	// Check that the container has stats
	assert.NotEmpty(t, containerInfo.Stats, "Expected container to have stats")
}

// TestCrioContainerById tests that cAdvisor can find a CRI-O container by its ID.
func TestCrioContainerById(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Crio().RunPause()

	// Wait for the container to show up in cAdvisor
	waitForCrioContainer(containerID, fm)

	// Query all containers via SubcontainersInfo - CRI-O containers are in "crio" namespace
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	var found bool
	for _, container := range allInfo {
		for _, alias := range container.Aliases {
			if alias == containerID {
				sanityCheck(containerID, container, t)
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	assert.True(t, found, "Container %q should be found in cAdvisor", containerID)
}

// TestCrioContainerByName tests that cAdvisor can find a CRI-O container by a custom name.
func TestCrioContainerByName(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := fmt.Sprintf("test-crio-container-by-name-%d", os.Getpid())
	containerID := fm.Crio().Run(framework.CrioRunArgs{
		Image: "registry.k8s.io/pause:3.9",
		Name:  containerName,
	})

	// Wait for the container to show up
	waitForCrioContainer(containerID, fm)

	// Query all containers via SubcontainersInfo - CRI-O containers are in "crio" namespace
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container by ID
	var found bool
	for _, container := range allInfo {
		for _, alias := range container.Aliases {
			if alias == containerID {
				sanityCheck(containerID, container, t)
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	assert.True(t, found, "Container with ID %q should be found in cAdvisor", containerID)
}

// TestBasicCrioContainer tests basic container properties.
func TestBasicCrioContainer(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Crio().RunPause()

	// Wait for the container to show up
	waitForCrioContainer(containerID, fm)

	// Query all containers via SubcontainersInfo - CRI-O containers are in "crio" namespace
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	var containerInfo *info.ContainerInfo
	for i, container := range allInfo {
		for _, alias := range container.Aliases {
			if alias == containerID {
				containerInfo = &allInfo[i]
				break
			}
		}
		if containerInfo != nil {
			break
		}
	}

	require.NotNil(t, containerInfo, "Container %q should be found", containerID)
	assert.NotEmpty(t, containerInfo.Stats, "Should have at least one stat")
}

// TestCrioContainerCpuStats tests CPU statistics collection for CRI-O containers.
func TestCrioContainerCpuStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Run a busybox container that does some work
	containerID := fm.Crio().RunBusybox("sh", "-c", "while true; do echo hello; sleep 1; done")

	// Wait for the container to show up
	waitForCrioContainer(containerID, fm)

	// Give the container some time to generate CPU usage
	time.Sleep(2 * time.Second)

	// Query all containers via SubcontainersInfo - CRI-O containers are in "crio" namespace
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	var containerInfo *info.ContainerInfo
	for i, container := range allInfo {
		for _, alias := range container.Aliases {
			if alias == containerID {
				containerInfo = &allInfo[i]
				break
			}
		}
		if containerInfo != nil {
			break
		}
	}

	require.NotNil(t, containerInfo, "Container %q should be found", containerID)
	require.NotEmpty(t, containerInfo.Stats, "Should have stats")

	// Check CPU stats
	stat := containerInfo.Stats[0]
	checkCPUStats(t, stat.Cpu)
}

// TestCrioContainerMemoryStats tests memory statistics collection for CRI-O containers.
func TestCrioContainerMemoryStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Run a busybox container
	containerID := fm.Crio().RunBusybox("sh", "-c", "while true; do echo hello; sleep 1; done")

	// Wait for the container to show up
	waitForCrioContainer(containerID, fm)

	// Give the container some time to use memory
	time.Sleep(2 * time.Second)

	// Query all containers via SubcontainersInfo - CRI-O containers are in "crio" namespace
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	var containerInfo *info.ContainerInfo
	for i, container := range allInfo {
		for _, alias := range container.Aliases {
			if alias == containerID {
				containerInfo = &allInfo[i]
				break
			}
		}
		if containerInfo != nil {
			break
		}
	}

	require.NotNil(t, containerInfo, "Container %q should be found", containerID)
	require.NotEmpty(t, containerInfo.Stats, "Should have stats")

	// Check memory stats
	stat := containerInfo.Stats[0]
	checkMemoryStats(t, stat.Memory)
}
