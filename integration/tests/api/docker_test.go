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

package api

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sanity check the container by:
// - Checking that the specified alias is a valid one for this container.
// - Verifying that stats are not empty.
func sanityCheck(alias string, containerInfo info.ContainerInfo, t *testing.T) {
	assert.Contains(t, containerInfo.Aliases, alias, "Alias %q should be in list of aliases %v", alias, containerInfo.Aliases)
	assert.NotEmpty(t, containerInfo.Stats, "Expected container to have stats")
}

// Waits up to 5s for a container with the specified alias to appear.
func waitForContainer(alias string, fm framework.Framework) {
	err := framework.RetryForDuration(func() error {
		ret, err := fm.Cadvisor().Client().DockerContainer(alias, &info.ContainerInfoRequest{
			NumStats: 1,
		})
		if err != nil {
			return err
		}
		if len(ret.Stats) != 1 {
			return fmt.Errorf("no stats returned for container %q", alias)
		}

		return nil
	}, 5*time.Second)
	require.NoError(fm.T(), err, "Timed out waiting for container %q to be available in cAdvisor: %v", alias, err)
}

// A Docker container in /docker/<ID>
func TestDockerContainerById(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerId := fm.Docker().RunPause()

	// Wait for the container to show up.
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	require.NoError(t, err)

	sanityCheck(containerId, containerInfo, t)
}

// A Docker container in /docker/<name>
func TestDockerContainerByName(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := fmt.Sprintf("test-docker-container-by-name-%d", os.Getpid())
	fm.Docker().Run(framework.DockerRunArgs{
		Image: "kubernetes/pause",
		Args:  []string{"--name", containerName},
	})

	// Wait for the container to show up.
	waitForContainer(containerName, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerName, request)
	require.NoError(t, err)

	sanityCheck(containerName, containerInfo, t)
}

// Find the first container with the specified alias in containers.
func findContainer(alias string, containers []info.ContainerInfo, t *testing.T) info.ContainerInfo {
	for _, cont := range containers {
		for _, a := range cont.Aliases {
			if alias == a {
				return cont
			}
		}
	}
	t.Fatalf("Failed to find container %q in %+v", alias, containers)
	return info.ContainerInfo{}
}

// All Docker containers through /docker
func TestGetAllDockerContainers(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the containers to show up.
	containerId1 := fm.Docker().RunPause()
	containerId2 := fm.Docker().RunPause()
	waitForContainer(containerId1, fm)
	waitForContainer(containerId2, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containersInfo, err := fm.Cadvisor().Client().AllDockerContainers(request)
	require.NoError(t, err)

	if len(containersInfo) < 2 {
		t.Fatalf("At least 2 Docker containers should exist, received %d: %+v", len(containersInfo), containersInfo)
	}
	sanityCheck(containerId1, findContainer(containerId1, containersInfo, t), t)
	sanityCheck(containerId2, findContainer(containerId2, containersInfo, t), t)
}

// Check expected properties of a Docker container.
func TestBasicDockerContainer(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := fmt.Sprintf("test-basic-docker-container-%d", os.Getpid())
	containerId := fm.Docker().Run(framework.DockerRunArgs{
		Image: "kubernetes/pause",
		Args: []string{
			"--name", containerName,
		},
	})

	// Wait for the container to show up.
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	require.NoError(t, err)

	// Check that the contianer is known by both its name and ID.
	sanityCheck(containerId, containerInfo, t)
	sanityCheck(containerName, containerInfo, t)

	assert.Empty(t, containerInfo.Subcontainers, "Should not have subcontainers")
	assert.Len(t, containerInfo.Stats, 1, "Should have exactly one stat")
}

// TODO(vmarmol): Handle if CPU or memory is not isolated on this system.
// Check the ContainerSpec.
func TestDockerContainerSpec(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	cpuShares := uint64(2048)
	cpuMask := "0"
	memoryLimit := uint64(1 << 30) // 1GB
	containerId := fm.Docker().Run(framework.DockerRunArgs{
		Image: "kubernetes/pause",
		Args: []string{
			"--cpu-shares", strconv.FormatUint(cpuShares, 10),
			"--cpuset", cpuMask,
			"--memory", strconv.FormatUint(memoryLimit, 10),
		},
	})

	// Wait for the container to show up.
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	require.NoError(t, err)
	sanityCheck(containerId, containerInfo, t)

	assert := assert.New(t)

	assert.True(containerInfo.Spec.HasCpu, "CPU should be isolated")
	assert.Equal(containerInfo.Spec.Cpu.Limit, cpuShares, "Container should have %d shares, has %d", cpuShares, containerInfo.Spec.Cpu.Limit)
	assert.Equal(containerInfo.Spec.Cpu.Mask, cpuMask, "Cpu mask should be %q, but is %q", cpuMask, containerInfo.Spec.Cpu.Mask)
	assert.True(containerInfo.Spec.HasMemory, "Memory should be isolated")
	assert.Equal(containerInfo.Spec.Memory.Limit, memoryLimit, "Container should have memory limit of %d, has %d", memoryLimit, containerInfo.Spec.Memory.Limit)
	assert.True(containerInfo.Spec.HasNetwork, "Network should be isolated")
	assert.True(containerInfo.Spec.HasDiskIo, "Blkio should be isolated")
}

// Check the CPU ContainerStats.
func TestDockerContainerCpuStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	containerId := fm.Docker().RunBusybox("ping", "www.google.com")
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	if err != nil {
		t.Fatal(err)
	}
	sanityCheck(containerId, containerInfo, t)

	// Checks for CpuStats.
	checkCpuStats(t, containerInfo.Stats[0].Cpu)
}

// Check the memory ContainerStats.
func TestDockerContainerMemoryStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	containerId := fm.Docker().RunBusybox("ping", "www.google.com")
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	require.NoError(t, err)
	sanityCheck(containerId, containerInfo, t)

	// Checks for MemoryStats.
	checkMemoryStats(t, containerInfo.Stats[0].Memory)
}

// Check the network ContainerStats.
func TestDockerContainerNetworkStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	containerId := fm.Docker().RunBusybox("watch", "-n1", "wget", "https://www.google.com/")
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	require.NoError(t, err)
	sanityCheck(containerId, containerInfo, t)

	// Checks for NetworkStats.
	stat := containerInfo.Stats[0]
	assert := assert.New(t)
	assert.NotEqual(0, stat.Network.TxBytes, "Network tx bytes should not be zero")
	assert.NotEqual(0, stat.Network.TxPackets, "Network tx packets should not be zero")
	assert.NotEqual(0, stat.Network.RxBytes, "Network rx bytes should not be zero")
	assert.NotEqual(0, stat.Network.RxPackets, "Network rx packets should not be zero")
	assert.NotEqual(stat.Network.RxBytes, stat.Network.TxBytes, "Network tx and rx bytes should not be equal")
	assert.NotEqual(stat.Network.RxPackets, stat.Network.TxPackets, "Network tx and rx packets should not be equal")
}
