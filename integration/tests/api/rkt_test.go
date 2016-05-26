// Copyright 2016 Google Inc. All Rights Reserved.
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

	"github.com/google/cadvisor/container/rkt"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"

	"fmt"
	"github.com/google/cadvisor/info/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	rktTimeout = 15 * time.Second
)

// A Rkt container by id
func TestRktContainerById(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	containerId := fm.Rkt().RunPause()

	// Wait for the container to show up.
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	require.NoError(t, err)

	sanityCheck(containerId, containerInfo, t)
}

// All Rkt containers
func TestGetAllRktContainers(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	containerId1 := fm.Rkt().RunPause()
	containerId2 := fm.Rkt().RunPause()

	// Wait for the containers to show up.
	waitForContainerWithTimeout(rkt.RktNamespace, containerId1, rktTimeout, fm)
	waitForContainerWithTimeout(rkt.RktNamespace, containerId2, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containersInfo, err := fm.Cadvisor().Client().AllNamespacedContainers(rkt.RktNamespace, request)
	require.NoError(t, err)

	if len(containersInfo) < 2 {
		t.Fatalf("At least 2 Rkt containers should exist, received %d: %+v", len(containersInfo), containersInfo)
	}
	sanityCheck(containerId1, findContainer(containerId1, containersInfo, t), t)
	sanityCheck(containerId2, findContainer(containerId2, containersInfo, t), t)
}

// Check expected properties of a Rkt pod container.
func TestBasicRktPod(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	containerId := fm.Rkt().RunPause()

	// Wait for the container to show up.
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	require.NoError(t, err)

	// Check that the contianer is known by both its name and ID.
	sanityCheck(containerId, containerInfo, t)

	assert.Len(t, containerInfo.Subcontainers, 1, "Should have exactly one subcontainers")
	assert.Len(t, containerInfo.Stats, 1, "Should have exactly one stat")
}

// A Rkt container by id
func TestRktContainerByContainerId(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	containerId := fm.Rkt().RunPause()

	containerId = containerId + ":pause"

	// Wait for the container to show up.
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	require.NoError(t, err)

	sanityCheck(containerId, containerInfo, t)
}

// Check expected properties of a Rkt app container.
func TestBasicRktContainer(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	containerId := fm.Rkt().RunPause()
	containerId = containerId + ":pause"

	// Wait for the container to show up.
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	require.NoError(t, err)

	// Check that the contianer is known by both its name and ID.
	sanityCheck(containerId, containerInfo, t)

	assert.Empty(t, containerInfo.Subcontainers, "Should have no subcontainers")
	assert.Len(t, containerInfo.Stats, 1, "Should have exactly one stat")
}

// Check the CPU ContainerStats.
func TestRktContainerCpuStats(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	args := framework.RunArgs{
		Image:     "docker://busybox",
		InnerArgs: []string{"--exec=ping", "--", "216.58.194.164"},
	}
	containerId := fm.Rkt().Run(args)
	containerId = containerId + ":busybox"
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	if err != nil {
		t.Fatal(err)
	}
	sanityCheck(containerId, containerInfo, t)

	// Checks for CpuStats.
	checkCpuStats(t, containerInfo.Stats[0].Cpu)
}

// Check the memory ContainerStats.
func TestRktContainerMemoryStats(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	args := framework.RunArgs{
		Image:     "docker://busybox",
		InnerArgs: []string{"--exec=ping", "--", "216.58.194.164"},
	}
	containerId := fm.Rkt().Run(args)
	containerId = containerId + ":busybox"
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	require.NoError(t, err)
	sanityCheck(containerId, containerInfo, t)

	// Checks for MemoryStats.
	checkMemoryStats(t, containerInfo.Stats[0].Memory)
}

// Check the network ContainerStats.
func TestRktPodNetworkStats(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	// Wait for the container to show up.
	args := framework.RunArgs{
		Image:     "docker://busybox",
		InnerArgs: []string{"--exec=watch", "--", "-n1", "wget", "http://216.58.194.164/"},
	}
	containerId := fm.Rkt().Run(args)
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	time.Sleep(10 * time.Second)
	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
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

func TestRktFilesystemStats(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	const (
		ddUsage       = uint64(1 << 3) // 1 KB
		sleepDuration = 10 * time.Second
	)
	// Wait for the container to show up.
	// FIXME: Tests should be bundled and run on the remote host instead of being run over ssh.
	// Escaping bash over ssh is ugly.
	// Once github issue 1130 is fixed, this logic can be removed.
	rktCmd := fmt.Sprintf("dd if=/dev/zero of=/file count=2 bs=%d & ping 216.58.194.164", ddUsage)
	if fm.Hostname().Host != "localhost" {
		rktCmd = fmt.Sprintf("'%s'", rktCmd)
	}
	args := framework.RunArgs{
		Image:     "docker://busybox",
		InnerArgs: []string{"--exec=/bin/sh", "--", "-c", rktCmd},
	}
	containerId := fm.Rkt().Run(args)
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)
	request := &v2.RequestOptions{
		IdType: v2.TypeRkt,
		Count:  1,
	}
	needsBaseUsageCheck := true
	pass := false

	// We need to wait for the `dd` operation to complete.
	for i := 0; i < 10; i++ {
		containerInfo, err := fm.Cadvisor().ClientV2().Stats(containerId, request)
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
		sanityCheckV2(containerId, info, t)

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

func testRkt() bool {
	_, err := rkt.Client()

	return err == nil
}
