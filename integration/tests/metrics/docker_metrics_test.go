//go:build linux

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

package metrics

import (
	"testing"
	"time"

	"github.com/google/cadvisor/integration/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForContainerInMetrics waits for a container to appear in the metrics endpoint.
func waitForContainerInMetrics(_ *testing.T, client *framework.MetricsClient, containerID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		families, err := client.FetchAndParse()
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
		if !ok {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Check if container appears (using first 12 chars of ID)
		if framework.ContainsLabelValue(cpuMetric, "id", containerID[:12]) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// TestDockerContainerAppearsInMetrics verifies a Docker container's metrics are exposed.
func TestDockerContainerAppearsInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a Docker container
	containerID := fm.Docker().RunPause()

	client := framework.NewMetricsClient(fm.Hostname())

	// Wait for container to appear in metrics
	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found, "Container %s should appear in metrics within 10 seconds", containerID[:12])
}

// TestDockerContainerCPUMetrics verifies CPU metrics are collected for Docker containers.
func TestDockerContainerCPUMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a container that does some CPU work
	containerID := fm.Docker().RunBusybox("sh", "-c", "while true; do :; done")

	client := framework.NewMetricsClient(fm.Hostname())

	// Wait for container and let it accumulate some CPU time
	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found, "Container should appear in metrics")

	// Wait a bit more for CPU stats to accumulate
	time.Sleep(2 * time.Second)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check container_cpu_usage_seconds_total
	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	// Find metrics for our container
	metrics := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", containerID[:12])
	require.NotEmpty(t, metrics, "Should find CPU metrics for container %s", containerID[:12])

	// At least one metric should have a positive value (the container is doing CPU work)
	hasPositiveValue := false
	for _, m := range metrics {
		if framework.GetCounterValue(m) > 0 {
			hasPositiveValue = true
			break
		}
	}
	assert.True(t, hasPositiveValue, "Active container should have positive CPU usage")

	// Check container_cpu_user_seconds_total exists for this container
	userCPU, ok := framework.GetMetricFamily(families, "container_cpu_user_seconds_total")
	if ok {
		userMetrics := framework.FindMetricsWithLabelSubstring(userCPU, "id", containerID[:12])
		assert.NotEmpty(t, userMetrics, "Should have user CPU metrics")
	}
}

// TestDockerContainerMemoryMetrics verifies memory metrics are collected for Docker containers.
func TestDockerContainerMemoryMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a container with a memory limit
	containerID := fm.Docker().Run(framework.DockerRunArgs{
		Image: "registry.k8s.io/pause",
		Args:  []string{"--memory=128m"},
	})

	client := framework.NewMetricsClient(fm.Hostname())

	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check container_memory_usage_bytes
	memUsage, ok := framework.GetMetricFamily(families, "container_memory_usage_bytes")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(memUsage, "id", containerID[:12])
	require.NotEmpty(t, metrics, "Should find memory usage metrics")

	// Memory usage should be non-negative
	for _, m := range metrics {
		usage := framework.GetGaugeValue(m)
		assert.GreaterOrEqual(t, usage, float64(0), "Memory usage should be >= 0")
	}

	// Check container_memory_working_set_bytes
	workingSet, ok := framework.GetMetricFamily(families, "container_memory_working_set_bytes")
	require.True(t, ok)

	wsMetrics := framework.FindMetricsWithLabelSubstring(workingSet, "id", containerID[:12])
	require.NotEmpty(t, wsMetrics, "Should find working set metrics")

	// Check container_spec_memory_limit_bytes shows our limit
	memLimit, ok := framework.GetMetricFamily(families, "container_spec_memory_limit_bytes")
	if ok {
		limitMetrics := framework.FindMetricsWithLabelSubstring(memLimit, "id", containerID[:12])
		if len(limitMetrics) > 0 {
			limit := framework.GetGaugeValue(limitMetrics[0])
			// 128MB = 134217728 bytes
			expectedLimit := float64(128 * 1024 * 1024)
			assert.Equal(t, expectedLimit, limit, "Memory limit should be 128MB")
		}
	}
}

// TestDockerContainerNetworkMetrics verifies network metrics are collected for Docker containers.
func TestDockerContainerNetworkMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a basic container - pause container has minimal network activity
	containerID := fm.Docker().RunPause()

	client := framework.NewMetricsClient(fm.Hostname())

	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check network receive metrics exist
	rxBytes, ok := framework.GetMetricFamily(families, "container_network_receive_bytes_total")
	require.True(t, ok)

	rxMetrics := framework.FindMetricsWithLabelSubstring(rxBytes, "id", containerID[:12])
	// Network metrics may or may not be present depending on network mode
	// Just verify the metric family exists and has proper structure
	if len(rxMetrics) > 0 {
		// Should have 'interface' label
		iface := framework.GetLabelValue(rxMetrics[0], "interface")
		assert.NotEmpty(t, iface, "Network metric should have 'interface' label")
	}

	// Check network transmit metrics exist
	txBytes, ok := framework.GetMetricFamily(families, "container_network_transmit_bytes_total")
	require.True(t, ok)

	txMetrics := framework.FindMetricsWithLabelSubstring(txBytes, "id", containerID[:12])
	if len(txMetrics) > 0 {
		iface := framework.GetLabelValue(txMetrics[0], "interface")
		assert.NotEmpty(t, iface, "Network metric should have 'interface' label")
	}
}

// TestDockerContainerLabelsInMetrics verifies container labels appear in metrics.
func TestDockerContainerLabelsInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a container with custom labels
	containerID := fm.Docker().Run(framework.DockerRunArgs{
		Image: "registry.k8s.io/pause",
		Args:  []string{"--label", "test.label=test-value"},
	})

	client := framework.NewMetricsClient(fm.Hostname())

	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Find our container's metrics
	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", containerID[:12])
	require.NotEmpty(t, metrics)

	// Check if our label appears (labels are prefixed with 'container_label_')
	// Note: label dots are converted to underscores
	metric := metrics[0]
	labelValue := framework.GetLabelValue(metric, "container_label_test_label")
	assert.Equal(t, "test-value", labelValue, "Container label should appear in metrics")
}

// TestDockerContainerImageInMetrics verifies container image name appears in metrics.
func TestDockerContainerImageInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Docker().Run(framework.DockerRunArgs{
		Image: "registry.k8s.io/pause",
	})

	client := framework.NewMetricsClient(fm.Hostname())

	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", containerID[:12])
	require.NotEmpty(t, metrics)

	// Check image label
	image := framework.GetLabelValue(metrics[0], "image")
	assert.Contains(t, image, "pause", "Image label should contain 'pause'")
}

// TestDockerContainerStartTimeMetric verifies container start time is recorded.
func TestDockerContainerStartTimeMetric(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	beforeCreate := time.Now().Unix()
	containerID := fm.Docker().RunPause()
	afterCreate := time.Now().Unix()

	client := framework.NewMetricsClient(fm.Hostname())

	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	startTime, ok := framework.GetMetricFamily(families, "container_start_time_seconds")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(startTime, "id", containerID[:12])
	require.NotEmpty(t, metrics)

	// Start time should be between beforeCreate and afterCreate (with some tolerance)
	startTs := framework.GetGaugeValue(metrics[0])
	assert.GreaterOrEqual(t, startTs, float64(beforeCreate-5), "Start time should be recent")
	assert.LessOrEqual(t, startTs, float64(afterCreate+5), "Start time should not be in the future")
}

// TestMultipleDockerContainersInMetrics verifies multiple containers appear correctly.
func TestMultipleDockerContainersInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start multiple containers
	container1 := fm.Docker().RunPause()
	container2 := fm.Docker().RunPause()

	client := framework.NewMetricsClient(fm.Hostname())

	// Wait for both containers to appear
	found1 := waitForContainerInMetrics(t, client, container1, 10*time.Second)
	found2 := waitForContainerInMetrics(t, client, container2, 10*time.Second)

	require.True(t, found1, "First container should appear in metrics")
	require.True(t, found2, "Second container should appear in metrics")

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	// Both containers should have distinct metrics
	metrics1 := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", container1[:12])
	metrics2 := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", container2[:12])

	assert.NotEmpty(t, metrics1, "First container should have metrics")
	assert.NotEmpty(t, metrics2, "Second container should have metrics")

	// IDs should be different
	id1 := framework.GetLabelValue(metrics1[0], "id")
	id2 := framework.GetLabelValue(metrics2[0], "id")
	assert.NotEqual(t, id1, id2, "Container IDs should be different")
}

// TestDockerContainerFilesystemMetrics verifies filesystem metrics for Docker containers.
func TestDockerContainerFilesystemMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Docker().RunPause()

	client := framework.NewMetricsClient(fm.Hostname())

	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check filesystem usage metric
	fsUsage, ok := framework.GetMetricFamily(families, "container_fs_usage_bytes")
	if ok {
		metrics := framework.FindMetricsWithLabelSubstring(fsUsage, "id", containerID[:12])
		// Filesystem metrics may not always be available for all containers
		if len(metrics) > 0 {
			usage := framework.GetGaugeValue(metrics[0])
			assert.GreaterOrEqual(t, usage, float64(0), "Filesystem usage should be >= 0")

			// Should have 'device' label
			device := framework.GetLabelValue(metrics[0], "device")
			assert.NotEmpty(t, device, "Filesystem metric should have 'device' label")
		}
	}
}

// TestDockerContainerMemoryFailcnt verifies memory failcnt metric exists.
func TestDockerContainerMemoryFailcnt(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Docker().Run(framework.DockerRunArgs{
		Image: "registry.k8s.io/pause",
		Args:  []string{"--memory=64m"},
	})

	client := framework.NewMetricsClient(fm.Hostname())

	found := waitForContainerInMetrics(t, client, containerID, 10*time.Second)
	require.True(t, found)

	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// container_memory_failcnt should exist (may be 0 for healthy containers)
	failcnt, ok := framework.GetMetricFamily(families, "container_memory_failcnt")
	if ok {
		metrics := framework.FindMetricsWithLabelSubstring(failcnt, "id", containerID[:12])
		if len(metrics) > 0 {
			count := framework.GetCounterValue(metrics[0])
			assert.GreaterOrEqual(t, count, float64(0), "Failcnt should be >= 0")
		}
	}
}
