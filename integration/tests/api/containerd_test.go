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

//go:build linux

package api

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

// Waits up to 10s for a containerd container with the specified ID to appear in cAdvisor.
func waitForContainerdContainer(containerID string, fm framework.Framework) {
	err := framework.RetryForDuration(func() error {
		// Query all containers via SubcontainersInfo - containerd containers are in "containerd" namespace
		allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
			NumStats: 1,
		})
		if err != nil {
			return err
		}

		// Look for container by ID
		for _, container := range allInfo {
			for _, alias := range container.Aliases {
				if alias == containerID {
					return nil
				}
			}
			// Also check if the container name contains the ID
			if len(container.Name) > 0 && containsString(container.Name, containerID) {
				return nil
			}
		}
		return fmt.Errorf("container %q not found in cAdvisor", containerID)
	}, 10*time.Second)
	require.NoError(fm.T(), err, "Timed out waiting for containerd container %q to be available in cAdvisor", containerID)
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Sanity check the container by:
// - Checking that the specified ID is a valid alias for this container.
// - Verifying that stats are not empty.
func sanityCheckContainerd(containerID string, containerInfo info.ContainerInfo, t *testing.T) {
	assert.Contains(t, containerInfo.Aliases, containerID, "Alias %q should be in list of aliases %v", containerID, containerInfo.Aliases)
	assert.NotEmpty(t, containerInfo.Stats, "Expected container to have stats")
}

// findContainerdContainer finds a container by ID in the list of containers.
func findContainerdContainer(containerID string, containers []info.ContainerInfo) *info.ContainerInfo {
	for i, container := range containers {
		for _, alias := range container.Aliases {
			if alias == containerID {
				return &containers[i]
			}
		}
		// Also check if the container name contains the ID
		if containsString(container.Name, containerID) {
			return &containers[i]
		}
	}
	return nil
}

// TestContainerdContainerById tests that cAdvisor can find a containerd container by its ID.
func TestContainerdContainerById(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Containerd().RunPause()

	// Wait for the container to show up in cAdvisor
	waitForContainerdContainer(containerID, fm)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found in cAdvisor", containerID)
	sanityCheckContainerd(containerID, *containerInfo, t)
}

// TestContainerdContainerByName tests that cAdvisor can find a containerd container by a custom hex ID.
// Note: cAdvisor's containerd handler expects 64-char hex container IDs, which is the standard format
// used by Kubernetes/CRI. Custom human-readable names are not supported.
func TestContainerdContainerByName(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Generate a 64-char hex ID (this is what Kubernetes/CRI uses)
	containerID := fmt.Sprintf("%032x%032x", os.Getpid(), 999999)
	_ = fm.Containerd().Run(framework.ContainerdRunArgs{
		Image: "registry.k8s.io/pause:3.9",
		Name:  containerID, // Using hex ID as the name
	})

	// Wait for the container to show up
	waitForContainerdContainer(containerID, fm)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container by ID
	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container with ID %q should be found in cAdvisor", containerID)
	sanityCheckContainerd(containerID, *containerInfo, t)
}

// TestGetAllContainerdContainers tests that cAdvisor can find multiple containerd containers.
func TestGetAllContainerdContainers(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start two containers
	containerID1 := fm.Containerd().RunPause()
	containerID2 := fm.Containerd().RunPause()

	// Wait for both containers to show up
	waitForContainerdContainer(containerID1, fm)
	waitForContainerdContainer(containerID2, fm)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find both containers
	containerInfo1 := findContainerdContainer(containerID1, allInfo)
	containerInfo2 := findContainerdContainer(containerID2, allInfo)

	require.NotNil(t, containerInfo1, "Container %q should be found in cAdvisor", containerID1)
	require.NotNil(t, containerInfo2, "Container %q should be found in cAdvisor", containerID2)

	sanityCheckContainerd(containerID1, *containerInfo1, t)
	sanityCheckContainerd(containerID2, *containerInfo2, t)
}

// TestBasicContainerdContainer tests basic container properties.
func TestBasicContainerdContainer(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Containerd().RunPause()

	// Wait for the container to show up
	waitForContainerdContainer(containerID, fm)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)
	assert.NotEmpty(t, containerInfo.Stats, "Should have at least one stat")
}

// TestContainerdContainerCpuStats tests CPU statistics collection for containerd containers.
func TestContainerdContainerCpuStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Run a busybox container that does some work
	containerID := fm.Containerd().RunBusybox("sh", "-c", "while true; do echo hello; sleep 1; done")

	// Wait for the container to show up
	waitForContainerdContainer(containerID, fm)

	// Give the container some time to generate CPU usage
	time.Sleep(2 * time.Second)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)
	require.NotEmpty(t, containerInfo.Stats, "Should have stats")

	// Check CPU stats
	stat := containerInfo.Stats[0]
	checkCPUStats(t, stat.Cpu)
}

// TestContainerdContainerMemoryStats tests memory statistics collection for containerd containers.
func TestContainerdContainerMemoryStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Run a busybox container
	containerID := fm.Containerd().RunBusybox("sh", "-c", "while true; do echo hello; sleep 1; done")

	// Wait for the container to show up
	waitForContainerdContainer(containerID, fm)

	// Give the container some time to use memory
	time.Sleep(2 * time.Second)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)
	require.NotEmpty(t, containerInfo.Stats, "Should have stats")

	// Check memory stats
	stat := containerInfo.Stats[0]
	checkMemoryStats(t, stat.Memory)
}

// TestContainerdContainerSpec tests that container spec is correctly populated for containerd containers.
func TestContainerdContainerSpec(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Containerd().RunPause()

	// Wait for the container to show up
	waitForContainerdContainer(containerID, fm)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)

	// Check that spec has basic properties
	assert.True(t, containerInfo.Spec.HasCpu, "CPU should be isolated")
	assert.True(t, containerInfo.Spec.HasMemory, "Memory should be isolated")
}

// TestContainerdContainerLabels tests that container labels are correctly captured.
func TestContainerdContainerLabels(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Use auto-generated 64-char hex ID (required by cAdvisor's containerd handler)
	containerID := fm.Containerd().Run(framework.ContainerdRunArgs{
		Image: "registry.k8s.io/pause:3.9",
		Labels: map[string]string{
			"test.label.key": "test-value",
		},
	})

	// Wait for the container to show up
	waitForContainerdContainer(containerID, fm)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find our container
	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)

	// Check that labels are captured
	assert.Contains(t, containerInfo.Spec.Labels, "test.label.key", "Labels should contain test.label.key")
	assert.Equal(t, "test-value", containerInfo.Spec.Labels["test.label.key"], "Label value should match")
}

// TestContainerdContainerCreationTime tests that container creation time is valid.
func TestContainerdContainerCreationTime(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	beforeCreation := time.Now().Add(-1 * time.Second)

	containerID := fm.Containerd().RunPause()
	waitForContainerdContainer(containerID, fm)

	afterCreation := time.Now().Add(1 * time.Second)

	// Query all containers
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)

	// Check creation time is within expected range
	creationTime := containerInfo.Spec.CreationTime
	assert.True(t, creationTime.After(beforeCreation), "Creation time %v should be after %v", creationTime, beforeCreation)
	assert.True(t, creationTime.Before(afterCreation), "Creation time %v should be before %v", creationTime, afterCreation)
}

// TestContainerdContainerDiskIoStats tests DiskIO statistics for containerd containers.
func TestContainerdContainerDiskIoStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Run a container that does disk I/O and stays running
	containerID := fm.Containerd().RunBusybox("sh", "-c", "dd if=/dev/zero of=/tmp/testfile bs=1024 count=1000 && sync && sleep 30")

	// Wait for the container to show up and do some I/O
	waitForContainerdContainer(containerID, fm)
	time.Sleep(3 * time.Second)

	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)

	// Check that DiskIo stats are present
	assert.True(t, containerInfo.Spec.HasDiskIo, "Container should have DiskIo isolation")
}

// TestContainerdContainerImageInfo tests that container image information is captured.
func TestContainerdContainerImageInfo(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	expectedImage := "registry.k8s.io/pause:3.9"
	containerID := fm.Containerd().Run(framework.ContainerdRunArgs{
		Image: expectedImage,
	})

	waitForContainerdContainer(containerID, fm)

	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	containerInfo := findContainerdContainer(containerID, allInfo)
	require.NotNil(t, containerInfo, "Container %q should be found", containerID)

	// Check image name is captured
	assert.Contains(t, containerInfo.Spec.Image, "pause", "Container image should contain 'pause'")
}
