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
	"github.com/google/cadvisor/info/v2"
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

// Sanity check the container by:
// - Checking that the specified alias is a valid one for this container.
// - Verifying that stats are not empty.
func sanityCheckV2(alias string, info v2.ContainerInfo, t *testing.T) {
	assert.Contains(t, info.Spec.Aliases, alias, "Alias %q should be in list of aliases %v", alias, info.Spec.Aliases)
	assert.NotEmpty(t, info.Stats, "Expected container to have stats")
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

	containerID := fm.Docker().RunPause()

	// Wait for the container to show up.
	waitForContainer(containerID, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerID, request)
	require.NoError(t, err)

	sanityCheck(containerID, containerInfo, t)
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
	containerID1 := fm.Docker().RunPause()
	containerID2 := fm.Docker().RunPause()
	waitForContainer(containerID1, fm)
	waitForContainer(containerID2, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containersInfo, err := fm.Cadvisor().Client().AllDockerContainers(request)
	require.NoError(t, err)

	if len(containersInfo) < 2 {
		t.Fatalf("At least 2 Docker containers should exist, received %d: %+v", len(containersInfo), containersInfo)
	}
	sanityCheck(containerID1, findContainer(containerID1, containersInfo, t), t)
	sanityCheck(containerID2, findContainer(containerID2, containersInfo, t), t)
}

// Check expected properties of a Docker container.
func TestBasicDockerContainer(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := fmt.Sprintf("test-basic-docker-container-%d", os.Getpid())
	containerID := fm.Docker().Run(framework.DockerRunArgs{
		Image: "kubernetes/pause",
		Args: []string{
			"--name", containerName,
		},
	})

	// Wait for the container to show up.
	waitForContainer(containerID, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerID, request)
	require.NoError(t, err)

	// Check that the contianer is known by both its name and ID.
	sanityCheck(containerID, containerInfo, t)
	sanityCheck(containerName, containerInfo, t)

	assert.Empty(t, containerInfo.Subcontainers, "Should not have subcontainers")
	assert.Len(t, containerInfo.Stats, 1, "Should have exactly one stat")
}

// TODO(vmarmol): Handle if CPU or memory is not isolated on this system.
// Check the ContainerSpec.
func TestDockerContainerSpec(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	var (
		cpuShares   = uint64(2048)
		cpuMask     = "0"
		memoryLimit = uint64(1 << 30) // 1GB
		image       = "kubernetes/pause"
		env         = map[string]string{"test_var": "FOO"}
		labels      = map[string]string{"bar": "baz"}
	)

	containerID := fm.Docker().Run(framework.DockerRunArgs{
		Image: image,
		Args: []string{
			"--cpu-shares", strconv.FormatUint(cpuShares, 10),
			"--cpuset-cpus", cpuMask,
			"--memory", strconv.FormatUint(memoryLimit, 10),
			"--env", "TEST_VAR=FOO",
			"--label", "bar=baz",
		},
	})

	// Wait for the container to show up.
	waitForContainer(containerID, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerID, request)
	require.NoError(t, err)
	sanityCheck(containerID, containerInfo, t)

	assert := assert.New(t)

	assert.True(containerInfo.Spec.HasCpu, "CPU should be isolated")
	assert.Equal(cpuShares, containerInfo.Spec.Cpu.Limit, "Container should have %d shares, has %d", cpuShares, containerInfo.Spec.Cpu.Limit)
	assert.Equal(cpuMask, containerInfo.Spec.Cpu.Mask, "Cpu mask should be %q, but is %q", cpuMask, containerInfo.Spec.Cpu.Mask)
	assert.True(containerInfo.Spec.HasMemory, "Memory should be isolated")
	assert.Equal(memoryLimit, containerInfo.Spec.Memory.Limit, "Container should have memory limit of %d, has %d", memoryLimit, containerInfo.Spec.Memory.Limit)
	assert.True(containerInfo.Spec.HasNetwork, "Network should be isolated")
	assert.True(containerInfo.Spec.HasDiskIo, "Blkio should be isolated")

	assert.Equal(image, containerInfo.Spec.Image, "Spec should include container image")
	assert.Equal(env, containerInfo.Spec.Envs, "Spec should include environment variables")
	assert.Equal(labels, containerInfo.Spec.Labels, "Spec should include labels")
}

// Check the CPU ContainerStats.
func TestDockerContainerCpuStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	containerID := fm.Docker().RunBusybox("ping", "www.google.com")
	waitForContainer(containerID, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerID, request)
	if err != nil {
		t.Fatal(err)
	}
	sanityCheck(containerID, containerInfo, t)

	// Checks for CpuStats.
	checkCPUStats(t, containerInfo.Stats[0].Cpu)
}

// Check the memory ContainerStats.
func TestDockerContainerMemoryStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	containerID := fm.Docker().RunBusybox("ping", "www.google.com")
	waitForContainer(containerID, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerID, request)
	require.NoError(t, err)
	sanityCheck(containerID, containerInfo, t)

	// Checks for MemoryStats.
	checkMemoryStats(t, containerInfo.Stats[0].Memory)
}

// Check the network ContainerStats.
func TestDockerContainerNetworkStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	containerID := fm.Docker().RunBusybox("watch", "-n1", "wget", "http://www.google.com/")
	waitForContainer(containerID, fm)

	// Wait for at least one additional housekeeping interval
	time.Sleep(20 * time.Second)
	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerID, request)
	require.NoError(t, err)
	sanityCheck(containerID, containerInfo, t)

	stat := containerInfo.Stats[0]
	ifaceStats := stat.Network.InterfaceStats
	// macOS we have more than one interface, since traffic is
	// only on eth0 we need to pick that one
	if len(stat.Network.Interfaces) > 0 {
		for _, iface := range stat.Network.Interfaces {
			if iface.Name == "eth0" {
				ifaceStats = iface
			}
		}
	}

	// Checks for NetworkStats.
	assert := assert.New(t)
	assert.NotEqual(0, ifaceStats.TxBytes, "Network tx bytes should not be zero")
	assert.NotEqual(0, ifaceStats.TxPackets, "Network tx packets should not be zero")
	assert.NotEqual(0, ifaceStats.RxBytes, "Network rx bytes should not be zero")
	assert.NotEqual(0, ifaceStats.RxPackets, "Network rx packets should not be zero")
	assert.NotEqual(ifaceStats.RxBytes, ifaceStats.TxBytes, fmt.Sprintf("Network tx (%d) and rx (%d) bytes should not be equal", ifaceStats.TxBytes, ifaceStats.RxBytes))
	assert.NotEqual(ifaceStats.RxPackets, ifaceStats.TxPackets, fmt.Sprintf("Network tx (%d) and rx (%d) packets should not be equal", ifaceStats.TxPackets, ifaceStats.RxPackets))
}

func TestDockerFilesystemStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	storageDriver := fm.Docker().StorageDriver()
	if storageDriver == framework.DeviceMapper {
		// Filesystem stats not supported with devicemapper, yet
		return
	}

	const (
		ddUsage       = uint64(1 << 3) // 1 KB
		sleepDuration = 10 * time.Second
	)
	// Wait for the container to show up.
	// FIXME: Tests should be bundled and run on the remote host instead of being run over ssh.
	// Escaping bash over ssh is ugly.
	// Once github issue 1130 is fixed, this logic can be removed.
	dockerCmd := fmt.Sprintf("dd if=/dev/zero of=/file count=2 bs=%d & ping google.com", ddUsage)
	if fm.Hostname().Host != "localhost" {
		dockerCmd = fmt.Sprintf("'%s'", dockerCmd)
	}
	containerID := fm.Docker().RunBusybox("/bin/sh", "-c", dockerCmd)
	waitForContainer(containerID, fm)
	request := &v2.RequestOptions{
		IdType: v2.TypeDocker,
		Count:  1,
	}
	needsBaseUsageCheck := false
	switch storageDriver {
	case framework.Aufs, framework.Overlay, framework.Overlay2, framework.DeviceMapper:
		needsBaseUsageCheck = true
	}
	pass := false
	// We need to wait for the `dd` operation to complete.
	for i := 0; i < 10; i++ {
		containerInfo, err := fm.Cadvisor().ClientV2().Stats(containerID, request)
		if err != nil {
			t.Logf("%v stats unavailable - %v", time.Now().String(), err)
			t.Logf("retrying after %s...", sleepDuration.String())
			time.Sleep(sleepDuration)

			continue
		}
		require.Equal(t, len(containerInfo), 1)
		var info v2.ContainerInfo
		// There is only one container in containerInfo. Since it is a map with unknown key,
		// use the value blindly.
		for _, cInfo := range containerInfo {
			info = cInfo
		}
		sanityCheckV2(containerID, info, t)

		require.NotNil(t, info.Stats[0], "got info: %+v", info)
		require.NotNil(t, info.Stats[0].Filesystem, "got info: %+v", info)
		require.NotNil(t, info.Stats[0].Filesystem.TotalUsageBytes, "got info: %+v", info.Stats[0].Filesystem)
		if *info.Stats[0].Filesystem.TotalUsageBytes >= ddUsage {
			if !needsBaseUsageCheck {
				pass = true
				break
			}
			require.NotNil(t, info.Stats[0].Filesystem.BaseUsageBytes)
			if *info.Stats[0].Filesystem.BaseUsageBytes >= ddUsage {
				pass = true
				break
			}
		}
		t.Logf("expected total usage %d bytes to be greater than %d bytes", *info.Stats[0].Filesystem.TotalUsageBytes, ddUsage)
		if needsBaseUsageCheck {
			t.Logf("expected base %d bytes to be greater than %d bytes", *info.Stats[0].Filesystem.BaseUsageBytes, ddUsage)
		}
		t.Logf("retrying after %s...", sleepDuration.String())
		time.Sleep(sleepDuration)
	}

	if !pass {
		t.Fail()
	}
}
