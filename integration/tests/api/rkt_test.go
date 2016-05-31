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
	"strconv"
	"testing"
	"time"

	"github.com/google/cadvisor/container/rkt"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"

	"fmt"
	"github.com/google/cadvisor/info/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"strings"
)

const (
	rktTimeout = 15 * time.Second
	googleIp   = "216.58.194.164"
)

var (
	pingGoogle = []string{"ping", googleIp}
)

func testRkt() bool {
	_, err := rkt.Client()

	return err == nil
}

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

	containerId := fm.Rkt().RunBusybox(pingGoogle...)
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

	containerId := fm.Rkt().RunBusybox(pingGoogle...)
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

	containerId := fm.Rkt().RunBusybox([]string{"watch", "-n1", "wget", "http://" + googleIp}...)
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

	containerId := fm.Rkt().RunBusybox([]string{"/bin/sh", "-c", rktCmd}...)
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

// Check the ContainerSpec.
func TestRktContainerSpec(t *testing.T) {
	if !testRkt() {
		t.SkipNow()
		return
	}

	fm := framework.New(t)
	defer fm.Cleanup()

	var (
		cpuShares = uint64(2048)
		//cpuMask     = "0"
		memoryLimit = uint64(1 << 30) // 1GB
		image_name  = "registry-1.docker.io/kubernetes/pause:latest"
		label_key   = "appc.io/docker/imageid"
	)

	image := framework.RktImage{
		Image:   "docker://" + image_name,
		RktArgs: []string{"--memory=1G"},
	}

	arg := framework.RktRunArgs{
		SystemddArgs: []string{"--property=CPUShares=" + strconv.FormatUint(cpuShares, 10), "--property=MemoryLimit=" + strconv.FormatUint(memoryLimit, 10)},
		Images:       []framework.RktImage{image},
	}

	containerId := fm.Rkt().Run(arg)
	containerId = containerId

	/*	fm.Docker().Run(framework.RunArgs{
		Image: image,
		Args: []string{
			"--cpu-shares", strconv.FormatUint(cpuShares, 10),
			cpusetArg, cpuMask,
			"--memory", strconv.FormatUint(memoryLimit, 10),
			"--env", "TEST_VAR=FOO",
			"--label", "bar=baz",
		},
	}) */

	// Wait for the container to show up.
	waitForContainerWithTimeout(rkt.RktNamespace, containerId, rktTimeout, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	podInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	require.NoError(t, err)
	sanityCheck(containerId, podInfo, t)
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId+":pause", request)
	require.NoError(t, err)
	sanityCheck(containerId+":pause", containerInfo, t)

	//assert := assert.New(t)

	assert.True(t, containerInfo.Spec.HasCpu, "CPU should be isolated")
	// TODO(sjpotter): Figure out how to convey CPU limits appropriately
	// assert.Equal(t, cpuShares, containerInfo.Spec.Cpu.Limit, "Container should have %d shares, has %d", cpuShares, containerInfo.Spec.Cpu.Limit)
	// assert.Equal(t, cpuMask, containerInfo.Spec.Cpu.Mask, "Cpu mask should be %q, but is %q", cpuMask, containerInfo.Spec.Cpu.Mask)

	assert.True(t, containerInfo.Spec.HasMemory, "Memory should be isolated")
	// TODO(sjpotter): Figure out how to convey memory limits appropriately
	// assert.Equal(t, memoryLimit, containerInfo.Spec.Memory.Limit, "Container should have memory limit of %d, has %d", memoryLimit, containerInfo.Spec.Memory.Limit)

	assert.True(t, podInfo.Spec.HasNetwork, "Network should be isolated")
	assert.False(t, containerInfo.Spec.HasNetwork, "Container Network shouldn't be isolated (shared with pod)")

	assert.True(t, containerInfo.Spec.HasDiskIo, "Blkio should be isolated")

	assert.Equal(t, image_name, containerInfo.Spec.Image, "Spec should include container image")
	assert.Contains(t, containerInfo.Spec.Labels, label_key, "Spec should include dockerimageid labels")

	// TODO(sjpotter) Figure out how to convey environment appropriately
	// assert.Equal(env, containerInfo.Spec.Envs, "Spec should include environment variables")
}

// getLen try to get length of object.
// return (false, 0) if impossible.
func getLen(x interface{}) (ok bool, length int) {
	v := reflect.ValueOf(x)
	defer func() {
		if e := recover(); e != nil {
			ok = false
		}
	}()
	return true, v.Len()
}

func includeElement(list interface{}, element interface{}) (ok, found bool) {
	listValue := reflect.ValueOf(list)
	elementValue := reflect.ValueOf(element)
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("exception thrown = %v\n", e)
			ok = false
			found = false
		}
	}()

	if reflect.TypeOf(list).Kind() == reflect.String {
		fmt.Printf("Type is string\n")
		return true, strings.Contains(listValue.String(), elementValue.String())
	}

	if reflect.TypeOf(list).Kind() == reflect.Map {
		fmt.Printf("type is map\n")
		mapKeys := listValue.MapKeys()
		fmt.Printf("mapKeys = %q\n", mapKeys)
		fmt.Printf("len(mapKeys) = %v\n", len(mapKeys))
		for i := 0; i < len(mapKeys); i++ {
			fmt.Printf("Does %v = %v: ", mapKeys[i].Interface(), element)
			if ObjectsAreEqual(mapKeys[i].Interface(), element) {
				fmt.Printf(" yes\n")
				return true, true
			}
			fmt.Printf(" no\n")
		}
		return true, false
	}

	for i := 0; i < listValue.Len(); i++ {
		fmt.Printf("type is array\n")
		if ObjectsAreEqual(listValue.Index(i).Interface(), element) {
			return true, true
		}
	}
	return true, false
}

func ObjectsAreEqual(expected, actual interface{}) bool {

	if expected == nil || actual == nil {
		return expected == actual
	}

	return reflect.DeepEqual(expected, actual)

}
