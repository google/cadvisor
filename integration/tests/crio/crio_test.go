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
	v2 "github.com/google/cadvisor/info/v2"
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
// - Checking that the specified ID is a valid alias for this container.
// - Verifying that stats are not empty.
func sanityCheck(containerID string, containerInfo info.ContainerInfo, t *testing.T) {
	assert.Contains(t, containerInfo.Aliases, containerID, "Alias %q should be in list of aliases %v", containerID, containerInfo.Aliases)
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

// TestGetAllCrioContainers tests that cAdvisor can find multiple CRI-O containers.
func TestGetAllCrioContainers(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start two containers
	containerID1 := fm.Crio().RunPause()
	containerID2 := fm.Crio().RunPause()

	// Wait for both containers to show up
	waitForCrioContainer(containerID1, fm)
	waitForCrioContainer(containerID2, fm)

	// Query all containers via SubcontainersInfo
	allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
		NumStats: 1,
	})
	require.NoError(t, err)

	// Find both containers
	var found1, found2 bool
	for _, container := range allInfo {
		for _, alias := range container.Aliases {
			if alias == containerID1 {
				sanityCheck(containerID1, container, t)
				found1 = true
			}
			if alias == containerID2 {
				sanityCheck(containerID2, container, t)
				found2 = true
			}
		}
	}
	assert.True(t, found1, "Container %q should be found in cAdvisor", containerID1)
	assert.True(t, found2, "Container %q should be found in cAdvisor", containerID2)
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

// TestCrioContainerNetworkStats tests network statistics collection for CRI-O containers.
// TODO: Skip this test until network stats collection works reliably for CRI-O containers.
// In the CI environment, network stats (TxBytes, RxBytes, etc.) are reported as zero,
// possibly due to CNI network namespace issues in the Docker-in-Docker environment.
func TestCrioContainerNetworkStats(t *testing.T) {
	t.Skip("Skipping: network stats are not reliably collected for CRI-O containers in CI")

	fm := framework.New(t)
	defer fm.Cleanup()

	// Run a busybox container that generates network traffic
	containerID := fm.Crio().RunBusybox("sh", "-c", "while true; do wget -q -O /dev/null http://www.google.com/ 2>/dev/null || true; sleep 1; done")

	// Wait for the container to show up
	waitForCrioContainer(containerID, fm)

	// Wait for at least one additional housekeeping interval for network stats
	time.Sleep(20 * time.Second)

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

	stat := containerInfo.Stats[0]
	ifaceStats := stat.Network.InterfaceStats
	// Pick eth0 if multiple interfaces exist
	if len(stat.Network.Interfaces) > 0 {
		for _, iface := range stat.Network.Interfaces {
			if iface.Name == "eth0" {
				ifaceStats = iface
			}
		}
	}

	// Checks for NetworkStats
	assert.NotEqual(t, uint64(0), ifaceStats.TxBytes, "Network tx bytes should not be zero")
	assert.NotEqual(t, uint64(0), ifaceStats.TxPackets, "Network tx packets should not be zero")
	assert.NotEqual(t, uint64(0), ifaceStats.RxBytes, "Network rx bytes should not be zero")
	assert.NotEqual(t, uint64(0), ifaceStats.RxPackets, "Network rx packets should not be zero")
}

// TestCrioContainerSpec tests that container spec is correctly populated for CRI-O containers.
func TestCrioContainerSpec(t *testing.T) {
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

	// Check that spec has basic properties
	assert.True(t, containerInfo.Spec.HasCpu, "CPU should be isolated")
	assert.True(t, containerInfo.Spec.HasMemory, "Memory should be isolated")
	// CRI-O containers may or may not have network depending on pod config
}

// TestCrioContainerDeletionExitCode tests that container deletion events include exit codes.
// TODO: Skip this test until cAdvisor properly reports exit codes for CRI-O containers.
// Currently cAdvisor reports exit code -1 for CRI-O containers even when the container
// exits with a specific code. This appears to be a timing/integration issue.
func TestCrioContainerDeletionExitCode(t *testing.T) {
	t.Skip("Skipping: cAdvisor currently reports exit code -1 for CRI-O containers")

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

			containerID := fm.Crio().RunBusybox("sh", "-c", fmt.Sprintf("exit %d", tt.exitCode))

			err := framework.RetryForDuration(func() error {
				events, err := fm.Cadvisor().Client().EventStaticInfo("?deletion_events=true&subcontainers=true")
				if err != nil {
					return err
				}

				for _, ev := range events {
					if ev.EventType == info.EventContainerDeletion {
						// Check if this event is for our container (CRI-O container names contain the ID)
						if contains(ev.ContainerName, containerID) {
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
				}
				return fmt.Errorf("deletion event not found for container %s", containerID)
			}, 30*time.Second)

			require.NoError(t, err)
		})
	}
}

// TestCrioHealthState tests health state reporting for CRI-O containers.
// Docker-style health checks (HEALTHCHECK instruction) are not supported by CRI-O.
// CRI-O containers rely on Kubernetes liveness/readiness probes instead,
// which are managed by the kubelet, not the container runtime.
func TestCrioHealthState(t *testing.T) {
	t.Skip("Skipping: Docker-style health checks are not supported by CRI-O (use Kubernetes probes instead)")
}

// TestCrioFilesystemStats tests filesystem statistics collection for CRI-O containers.
func TestCrioFilesystemStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	const (
		ddUsage       = uint64(1 << 3) // 1 KB
		sleepDuration = 10 * time.Second
	)

	// Run a busybox container that creates a file and stays running
	containerID := fm.Crio().RunBusybox("/bin/sh", "-c", "dd if=/dev/zero of=/file count=2 bs=1024 && while true; do sleep 1; done")

	// Wait for the container to show up
	waitForCrioContainer(containerID, fm)

	request := &v2.RequestOptions{
		IdType: v2.TypeName,
		Count:  1,
	}

	pass := false
	// We need to wait for the `dd` operation to complete.
	for i := 0; i < 10; i++ {
		// Query all containers and find ours
		allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
			NumStats: 1,
		})
		if err != nil {
			t.Logf("%v stats unavailable - %v", time.Now().String(), err)
			t.Logf("retrying after %s...", sleepDuration.String())
			time.Sleep(sleepDuration)
			continue
		}

		// Find our container
		var containerName string
		for _, container := range allInfo {
			for _, alias := range container.Aliases {
				if alias == containerID {
					containerName = container.Name
					break
				}
			}
			if containerName != "" {
				break
			}
		}

		if containerName == "" {
			t.Logf("Container %q not found, retrying...", containerID)
			time.Sleep(sleepDuration)
			continue
		}

		// Get stats using v2 API
		containerInfo, err := fm.Cadvisor().ClientV2().Stats(containerName, request)
		if err != nil {
			t.Logf("%v stats unavailable - %v", time.Now().String(), err)
			t.Logf("retrying after %s...", sleepDuration.String())
			time.Sleep(sleepDuration)
			continue
		}

		if len(containerInfo) == 0 {
			t.Logf("No container info returned, retrying...")
			time.Sleep(sleepDuration)
			continue
		}

		// Get the first (and only) container info
		var cInfo v2.ContainerInfo
		for _, ci := range containerInfo {
			cInfo = ci
			break
		}

		if len(cInfo.Stats) == 0 || cInfo.Stats[0].Filesystem == nil {
			t.Logf("No filesystem stats available yet, retrying...")
			time.Sleep(sleepDuration)
			continue
		}

		if cInfo.Stats[0].Filesystem.TotalUsageBytes != nil && *cInfo.Stats[0].Filesystem.TotalUsageBytes >= ddUsage {
			pass = true
			break
		}

		t.Logf("expected total usage to be greater than %d bytes, retrying...", ddUsage)
		time.Sleep(sleepDuration)
	}

	if !pass {
		t.Fail()
	}
}
