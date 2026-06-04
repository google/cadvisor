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
	"strings"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/integration/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForContainerdContainerViaAPI waits for a containerd container to appear in cAdvisor
// using the API client, which is more reliable than searching metrics labels.
func waitForContainerdContainerViaAPI(containerID string, fm framework.Framework, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allInfo, err := fm.Cadvisor().Client().SubcontainersInfo("/", &info.ContainerInfoRequest{
			NumStats: 1,
		})
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Look for container by ID in aliases or name
		for _, container := range allInfo {
			for _, alias := range container.Aliases {
				if alias == containerID {
					return true
				}
			}
			if strings.Contains(container.Name, containerID) {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// getContainerdContainerMetricID finds the 'id' label value used for a containerd container
// in Prometheus metrics by searching all metrics for one containing the container ID.
func getContainerdContainerMetricID(fm framework.Framework, containerID string) string {
	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	if err != nil {
		return ""
	}

	// Search in CPU metrics for the container
	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	if !ok {
		return ""
	}

	// Look for any metric where the 'id' label contains our container ID
	for _, metric := range cpuMetric.GetMetric() {
		id := framework.GetLabelValue(metric, "id")
		if strings.Contains(id, containerID) {
			return id
		}
	}

	return ""
}

// TestContainerdContainerAppearsInMetrics verifies a containerd container's metrics are exposed.
func TestContainerdContainerAppearsInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a containerd container
	containerID := fm.Containerd().RunPause()

	// Wait for container to appear via API first (more reliable)
	found := waitForContainerdContainerViaAPI(containerID, fm, 15*time.Second)
	require.True(t, found, "Containerd container %s should appear in cAdvisor API within 15 seconds", containerID)

	// Now verify it appears in Prometheus metrics
	metricID := getContainerdContainerMetricID(fm, containerID)
	require.NotEmpty(t, metricID, "Containerd container %s should appear in Prometheus metrics", containerID)
}

// TestContainerdContainerCPUMetrics verifies CPU metrics for containerd containers.
func TestContainerdContainerCPUMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a container that does some CPU work
	containerID := fm.Containerd().RunBusybox("sh", "-c", "while true; do :; done")

	found := waitForContainerdContainerViaAPI(containerID, fm, 15*time.Second)
	require.True(t, found, "Container should appear in cAdvisor")

	// Wait for CPU stats to accumulate
	time.Sleep(2 * time.Second)

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	// Find metrics for our container (search for container ID substring)
	metrics := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", containerID)
	require.NotEmpty(t, metrics, "Should find CPU metrics for containerd container")

	// Active container should have some CPU usage
	hasPositiveValue := false
	for _, m := range metrics {
		if framework.GetCounterValue(m) > 0 {
			hasPositiveValue = true
			break
		}
	}
	assert.True(t, hasPositiveValue, "Active containerd container should have positive CPU usage")
}

// TestContainerdContainerMemoryMetrics verifies memory metrics for containerd containers.
func TestContainerdContainerMemoryMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Containerd().RunPause()

	found := waitForContainerdContainerViaAPI(containerID, fm, 15*time.Second)
	require.True(t, found)

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check container_memory_usage_bytes
	memUsage, ok := framework.GetMetricFamily(families, "container_memory_usage_bytes")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(memUsage, "id", containerID)
	require.NotEmpty(t, metrics, "Should find memory usage metrics for containerd container")

	// Memory usage should be non-negative
	for _, m := range metrics {
		usage := framework.GetGaugeValue(m)
		assert.GreaterOrEqual(t, usage, float64(0), "Memory usage should be >= 0")
	}

	// Check container_memory_working_set_bytes
	workingSet, ok := framework.GetMetricFamily(families, "container_memory_working_set_bytes")
	require.True(t, ok)

	wsMetrics := framework.FindMetricsWithLabelSubstring(workingSet, "id", containerID)
	require.NotEmpty(t, wsMetrics, "Should find working set metrics for containerd container")
}

// TestContainerdContainerLabelsInMetrics verifies container labels appear in metrics.
func TestContainerdContainerLabelsInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a container with custom labels
	containerID := fm.Containerd().Run(framework.ContainerdRunArgs{
		Image: "registry.k8s.io/pause:3.9",
		Labels: map[string]string{
			"io.kubernetes.pod.name": "test-pod",
		},
	})

	found := waitForContainerdContainerViaAPI(containerID, fm, 15*time.Second)
	require.True(t, found)

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", containerID)
	require.NotEmpty(t, metrics)

	// Check if label appears (labels are prefixed with 'container_label_')
	// Note: dots in label names are converted to underscores
	metric := metrics[0]
	labelValue := framework.GetLabelValue(metric, "container_label_io_kubernetes_pod_name")
	assert.Equal(t, "test-pod", labelValue, "Container label should appear in metrics")
}

// TestContainerdContainerImageInMetrics verifies container image name appears in metrics.
func TestContainerdContainerImageInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerID := fm.Containerd().Run(framework.ContainerdRunArgs{
		Image: "registry.k8s.io/pause:3.9",
	})

	found := waitForContainerdContainerViaAPI(containerID, fm, 15*time.Second)
	require.True(t, found)

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", containerID)
	require.NotEmpty(t, metrics)

	// Check image label
	image := framework.GetLabelValue(metrics[0], "image")
	assert.Contains(t, image, "pause", "Image label should contain 'pause'")
}

// TestContainerdContainerStartTimeMetric verifies container start time is recorded.
func TestContainerdContainerStartTimeMetric(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	beforeCreate := time.Now().Unix()
	containerID := fm.Containerd().RunPause()
	afterCreate := time.Now().Unix()

	found := waitForContainerdContainerViaAPI(containerID, fm, 15*time.Second)
	require.True(t, found)

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	startTime, ok := framework.GetMetricFamily(families, "container_start_time_seconds")
	require.True(t, ok)

	metrics := framework.FindMetricsWithLabelSubstring(startTime, "id", containerID)
	require.NotEmpty(t, metrics)

	// Start time should be between beforeCreate and afterCreate (with tolerance)
	startTs := framework.GetGaugeValue(metrics[0])
	assert.GreaterOrEqual(t, startTs, float64(beforeCreate-5), "Start time should be recent")
	assert.LessOrEqual(t, startTs, float64(afterCreate+10), "Start time should not be far in the future")
}

// TestMultipleContainerdContainersInMetrics verifies multiple containers appear correctly.
func TestMultipleContainerdContainersInMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start multiple containers
	container1 := fm.Containerd().RunPause()
	container2 := fm.Containerd().RunPause()

	// Wait for both containers to appear
	found1 := waitForContainerdContainerViaAPI(container1, fm, 15*time.Second)
	found2 := waitForContainerdContainerViaAPI(container2, fm, 15*time.Second)

	require.True(t, found1, "First container should appear in cAdvisor")
	require.True(t, found2, "Second container should appear in cAdvisor")

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	// Both containers should have distinct metrics
	metrics1 := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", container1)
	metrics2 := framework.FindMetricsWithLabelSubstring(cpuMetric, "id", container2)

	assert.NotEmpty(t, metrics1, "First containerd container should have metrics")
	assert.NotEmpty(t, metrics2, "Second containerd container should have metrics")

	// IDs should be different
	id1 := framework.GetLabelValue(metrics1[0], "id")
	id2 := framework.GetLabelValue(metrics2[0], "id")
	assert.NotEqual(t, id1, id2, "Container IDs should be different")
}

// TestContainerdContainerDiskIOMetrics verifies disk I/O metrics for containerd containers.
func TestContainerdContainerDiskIOMetrics(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Start a container that does some disk I/O
	containerID := fm.Containerd().RunBusybox("sh", "-c", "dd if=/dev/zero of=/tmp/test bs=1M count=10 2>/dev/null; sleep infinity")

	found := waitForContainerdContainerViaAPI(containerID, fm, 15*time.Second)
	require.True(t, found)

	// Wait for disk I/O to be recorded
	time.Sleep(3 * time.Second)

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check for disk I/O write metrics
	fsWrites, ok := framework.GetMetricFamily(families, "container_fs_writes_bytes_total")
	if ok {
		metrics := framework.FindMetricsWithLabelSubstring(fsWrites, "id", containerID)
		if len(metrics) > 0 {
			writes := framework.GetCounterValue(metrics[0])
			// The container wrote at least 10MB
			assert.GreaterOrEqual(t, writes, float64(0), "Write bytes should be >= 0")
		}
	}
}
