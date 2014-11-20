package api

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/integration/framework"
)

// Checks whether el is in vals.
func contains(el string, vals []string) bool {
	for _, val := range vals {
		if el == val {
			return true
		}
	}
	return false
}

// Sanity check the container by:
// - Checking that the specified alias is a valid one for this container.
// - Verifying that stats are not empty.
func sanityCheck(alias string, containerInfo info.ContainerInfo, t *testing.T) {
	if !contains(alias, containerInfo.Aliases) {
		t.Errorf("Failed to find container alias %q in aliases %v", alias, containerInfo.Aliases)
	}
	if len(containerInfo.Stats) == 0 {
		t.Errorf("No container stats found: %+v", containerInfo)
	}
}

// Waits up to 5s for a container with the specified alias to appear.
func waitForContainer(alias string, fm framework.Framework) {
	err := framework.RetryForDuration(func() error {
		_, err := fm.Cadvisor().Client().DockerContainer(alias, &info.ContainerInfoRequest{})
		if err != nil {
			return err
		}

		return nil
	}, 5*time.Second)
	if err != nil {
		fm.T().Fatalf("Timed out waiting for container %q to be available in cAdvisor: %v", alias, err)
	}
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
	if err != nil {
		t.Fatal(err)
	}

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
	}, "sleep", "inf")

	// Wait for the container to show up.
	waitForContainer(containerName, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerName, request)
	if err != nil {
		t.Fatal(err)
	}

	sanityCheck(containerName, containerInfo, t)
}

// Find the first container with the specified alias in containers.
func findContainer(alias string, containers []info.ContainerInfo, t *testing.T) info.ContainerInfo {
	for _, cont := range containers {
		if contains(alias, cont.Aliases) {
			return cont
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
	if err != nil {
		t.Fatal(err)
	}

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
	}, "sleep", "inf")

	// Wait for the container to show up.
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the contianer is known by both its name and ID.
	sanityCheck(containerId, containerInfo, t)
	sanityCheck(containerName, containerInfo, t)

	if len(containerInfo.Subcontainers) != 0 {
		t.Errorf("Container has subcontainers: %+v", containerInfo)
	}

	if len(containerInfo.Stats) != 1 {
		t.Fatalf("Container has more than 1 stat, has %d: %+v", len(containerInfo.Stats), containerInfo.Stats)
	}
}

// Returns the difference between a and b.
func difference(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
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
	}, "sleep", "inf")

	// Wait for the container to show up.
	waitForContainer(containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().DockerContainer(containerId, request)
	if err != nil {
		t.Fatal(err)
	}
	sanityCheck(containerId, containerInfo, t)

	if !containerInfo.Spec.HasCpu {
		t.Errorf("CPU should be isolated: %+v", containerInfo)
	}
	if containerInfo.Spec.Cpu.Limit != cpuShares {
		t.Errorf("Container should have %d shares, has %d", cpuShares, containerInfo.Spec.Cpu.Limit)
	}
	if containerInfo.Spec.Cpu.Mask != cpuMask {
		t.Errorf("Cpu mask should be %q, but is %q", cpuMask, containerInfo.Spec.Cpu.Mask)
	}
	if !containerInfo.Spec.HasMemory {
		t.Errorf("Memory should be isolated: %+v", containerInfo)
	}
	if containerInfo.Spec.Memory.Limit != memoryLimit {
		t.Errorf("Container should have memory limit of %d, has %d", memoryLimit, containerInfo.Spec.Memory.Limit)
	}
	if !containerInfo.Spec.HasNetwork {
		t.Errorf("Network should be isolated: %+v", containerInfo)
	}
	if !containerInfo.Spec.HasFilesystem {
		t.Errorf("Filesystem should be isolated: %+v", containerInfo)
	}
}

// Expect the specified value to be non-zero.
func expectNonZero(val int, description string, t *testing.T) {
	if val < 0 {
		t.Errorf("%s should be posiive", description)
	}
	expectNonZeroU(uint64(val), description, t)
}
func expectNonZeroU(val uint64, description string, t *testing.T) {
	if val == 0 {
		t.Errorf("%s should be non-zero", description)
	}
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
	stat := containerInfo.Stats[0]

	// Checks for CpuStats.
	expectNonZeroU(stat.Cpu.Usage.Total, "CPU total usage", t)
	expectNonZero(len(stat.Cpu.Usage.PerCpu), "per-core CPU usage", t)
	totalUsage := uint64(0)
	for _, usage := range stat.Cpu.Usage.PerCpu {
		totalUsage += usage
	}
	dif := difference(totalUsage, stat.Cpu.Usage.Total)
	if dif > uint64((5 * time.Millisecond).Nanoseconds()) {
		t.Errorf("Per-core CPU usage (%d) and total usage (%d) are more than 1ms off", totalUsage, stat.Cpu.Usage.Total)
	}
	userPlusSystem := stat.Cpu.Usage.User + stat.Cpu.Usage.System
	dif = difference(totalUsage, userPlusSystem)
	if dif > uint64((25 * time.Millisecond).Nanoseconds()) {
		t.Errorf("User + system CPU usage (%d) and total usage (%d) are more than 20ms off", userPlusSystem, stat.Cpu.Usage.Total)
	}
	if stat.Cpu.Load != 0 {
		t.Errorf("Non-zero load is unexpected as it is currently unset. Do we need to update the test?")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	sanityCheck(containerId, containerInfo, t)
	stat := containerInfo.Stats[0]

	// Checks for MemoryStats.
	expectNonZeroU(stat.Memory.Usage, "memory usage", t)
	expectNonZeroU(stat.Memory.WorkingSet, "memory working set", t)
	if stat.Memory.WorkingSet > stat.Memory.Usage {
		t.Errorf("Memory working set (%d) should be at most equal to memory usage (%d)", stat.Memory.WorkingSet, stat.Memory.Usage)
	}
	// TODO(vmarmol): Add checks for ContainerData and HierarchicalData
}

// Check the network ContainerStats.
func TestDockerContainerNetworkStats(t *testing.T) {
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
	stat := containerInfo.Stats[0]

	// Checks for NetworkStats.
	expectNonZeroU(stat.Network.TxBytes, "network tx bytes", t)
	expectNonZeroU(stat.Network.TxPackets, "network tx packets", t)
	// TODO(vmarmol): Can probably do a better test with two containers pinging each other.
}
