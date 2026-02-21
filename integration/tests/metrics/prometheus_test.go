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

// Package metrics provides integration tests for cAdvisor's Prometheus /metrics endpoint.
package metrics

import (
	"strings"
	"testing"

	"github.com/google/cadvisor/integration/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsEndpointReturns200 verifies the /metrics endpoint is accessible
// and returns a 200 status code.
func TestMetricsEndpointReturns200(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	text, err := client.Fetch()
	require.NoError(t, err, "Failed to fetch /metrics")
	require.NotEmpty(t, text, "/metrics returned empty response")
}

// TestMetricsEndpointReturnsValidPrometheusFormat verifies the response
// can be parsed as valid Prometheus text format.
func TestMetricsEndpointReturnsValidPrometheusFormat(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err, "Failed to parse Prometheus metrics")
	require.NotEmpty(t, families, "No metric families found")
}

// TestCadvisorVersionInfoExists verifies the cadvisor_version_info metric is present
// and has expected labels.
func TestCadvisorVersionInfoExists(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	versionInfo, ok := framework.GetMetricFamily(families, "cadvisor_version_info")
	require.True(t, ok, "cadvisor_version_info metric should exist")
	require.NotEmpty(t, versionInfo.GetMetric(), "cadvisor_version_info should have at least one sample")

	// Check that expected labels are present
	metric := versionInfo.GetMetric()[0]
	labels := make(map[string]bool)
	for _, lp := range metric.GetLabel() {
		labels[lp.GetName()] = true
	}

	assert.True(t, labels["cadvisorVersion"], "Should have cadvisorVersion label")
	assert.True(t, labels["kernelVersion"], "Should have kernelVersion label")
	assert.True(t, labels["osVersion"], "Should have osVersion label")
}

// TestCoreCPUMetricsExist verifies essential CPU metrics are present.
func TestCoreCPUMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	cpuMetrics := []string{
		"container_cpu_usage_seconds_total",
		"container_cpu_user_seconds_total",
		"container_cpu_system_seconds_total",
	}

	for _, name := range cpuMetrics {
		assert.True(t, framework.HasMetric(families, name),
			"CPU metric %q should exist", name)
	}

	// Verify container_cpu_usage_seconds_total is a counter
	cpuUsage, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)
	assert.Equal(t, "COUNTER", framework.GetMetricType(cpuUsage),
		"container_cpu_usage_seconds_total should be a counter")
}

// TestCoreMemoryMetricsExist verifies essential memory metrics are present.
func TestCoreMemoryMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	memoryMetrics := []string{
		"container_memory_usage_bytes",
		"container_memory_working_set_bytes",
		"container_memory_cache",
		"container_memory_rss",
	}

	for _, name := range memoryMetrics {
		assert.True(t, framework.HasMetric(families, name),
			"Memory metric %q should exist", name)
	}

	// Verify container_memory_usage_bytes is a gauge
	memUsage, ok := framework.GetMetricFamily(families, "container_memory_usage_bytes")
	require.True(t, ok)
	assert.Equal(t, "GAUGE", framework.GetMetricType(memUsage),
		"container_memory_usage_bytes should be a gauge")
}

// TestCoreNetworkMetricsExist verifies essential network metrics are present.
func TestCoreNetworkMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	networkMetrics := []string{
		"container_network_receive_bytes_total",
		"container_network_transmit_bytes_total",
		"container_network_receive_packets_total",
		"container_network_transmit_packets_total",
	}

	for _, name := range networkMetrics {
		assert.True(t, framework.HasMetric(families, name),
			"Network metric %q should exist", name)
	}
}

// TestCoreFilesystemMetricsExist verifies essential filesystem metrics are present.
func TestCoreFilesystemMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	fsMetrics := []string{
		"container_fs_usage_bytes",
		"container_fs_limit_bytes",
	}

	for _, name := range fsMetrics {
		assert.True(t, framework.HasMetric(families, name),
			"Filesystem metric %q should exist", name)
	}
}

// TestContainerSpecMetricsExist verifies container specification metrics are present.
func TestContainerSpecMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	specMetrics := []string{
		"container_start_time_seconds",
		"container_spec_memory_limit_bytes",
	}

	for _, name := range specMetrics {
		assert.True(t, framework.HasMetric(families, name),
			"Spec metric %q should exist", name)
	}
}

// TestMachineMetricsExist verifies machine-level metrics are present with reasonable values.
func TestMachineMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	machineMetrics := []string{
		"machine_cpu_cores",
		"machine_cpu_physical_cores",
		"machine_memory_bytes",
	}

	for _, name := range machineMetrics {
		assert.True(t, framework.HasMetric(families, name),
			"Machine metric %q should exist", name)
	}

	// Verify machine_cpu_cores has a reasonable value (1 to 1024 cores)
	cpuCores, ok := framework.GetMetricFamily(families, "machine_cpu_cores")
	require.True(t, ok)
	require.NotEmpty(t, cpuCores.GetMetric())

	cores := framework.GetGaugeValue(cpuCores.GetMetric()[0])
	assert.GreaterOrEqual(t, cores, float64(1), "Should have at least 1 CPU core")
	assert.LessOrEqual(t, cores, float64(1024), "CPU core count seems unreasonably high")

	// Verify machine_memory_bytes has a reasonable value (100MB to 100TB)
	memBytes, ok := framework.GetMetricFamily(families, "machine_memory_bytes")
	require.True(t, ok)
	require.NotEmpty(t, memBytes.GetMetric())

	mem := framework.GetGaugeValue(memBytes.GetMetric()[0])
	assert.GreaterOrEqual(t, mem, float64(100*1024*1024), "Machine memory should be at least 100MB")
	assert.LessOrEqual(t, mem, float64(100*1024*1024*1024*1024), "Machine memory seems unreasonably high")
}

// TestMetricsHaveCorrectTypes verifies that metric types are correct.
func TestMetricsHaveCorrectTypes(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Counters should be COUNTER type
	counterMetrics := []string{
		"container_cpu_usage_seconds_total",
		"container_cpu_user_seconds_total",
		"container_cpu_system_seconds_total",
		"container_network_receive_bytes_total",
		"container_network_transmit_bytes_total",
	}
	for _, name := range counterMetrics {
		if mf, ok := families[name]; ok {
			assert.Equal(t, "COUNTER", framework.GetMetricType(mf),
				"%s should be a counter", name)
		}
	}

	// Gauges should be GAUGE type
	gaugeMetrics := []string{
		"container_memory_usage_bytes",
		"container_memory_working_set_bytes",
		"machine_cpu_cores",
		"machine_memory_bytes",
	}
	for _, name := range gaugeMetrics {
		if mf, ok := families[name]; ok {
			assert.Equal(t, "GAUGE", framework.GetMetricType(mf),
				"%s should be a gauge", name)
		}
	}
}

// TestMetricsHaveHelpText verifies metrics have help descriptions.
func TestMetricsHaveHelpText(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check a few key metrics have non-empty help text
	metricsToCheck := []string{
		"container_cpu_usage_seconds_total",
		"container_memory_usage_bytes",
		"machine_memory_bytes",
	}

	for _, name := range metricsToCheck {
		if mf, ok := families[name]; ok {
			help := mf.GetHelp()
			assert.NotEmpty(t, help, "Metric %s should have help text", name)
		}
	}
}

// TestContainerLastSeenExists verifies the container_last_seen metric is present.
func TestContainerLastSeenExists(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	assert.True(t, framework.HasMetric(families, "container_last_seen"),
		"container_last_seen metric should exist")
}

// TestRootContainerMetricsExist verifies the root container (/) has metrics.
func TestRootContainerMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// The root container should appear in CPU metrics with id="/"
	cpuMetric, ok := framework.GetMetricFamily(families, "container_cpu_usage_seconds_total")
	require.True(t, ok)

	// Look for root container
	found := false
	for _, metric := range cpuMetric.GetMetric() {
		id := framework.GetLabelValue(metric, "id")
		if id == "/" {
			found = true
			// Root container should have positive CPU usage
			value := framework.GetCounterValue(metric)
			assert.GreaterOrEqual(t, value, float64(0), "Root container CPU should be >= 0")
			break
		}
	}
	assert.True(t, found, "Root container (/) should appear in CPU metrics")
}

// TestGoRuntimeMetricsExist verifies Go runtime metrics are exposed.
func TestGoRuntimeMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Check for at least some Go runtime metrics
	goMetrics := []string{
		"go_goroutines",
		"go_memstats_alloc_bytes",
	}

	foundCount := 0
	for _, name := range goMetrics {
		if framework.HasMetric(families, name) {
			foundCount++
		}
	}

	assert.GreaterOrEqual(t, foundCount, 1, "Should have at least one Go runtime metric")
}

// TestProcessMetricsExist verifies process-level metrics are exposed.
func TestProcessMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// These are process collector metrics for the cAdvisor process itself
	processMetrics := []string{
		"process_cpu_seconds_total",
		"process_resident_memory_bytes",
	}

	foundCount := 0
	for _, name := range processMetrics {
		if framework.HasMetric(families, name) {
			foundCount++
		}
	}

	assert.GreaterOrEqual(t, foundCount, 1, "Should have at least one process metric")
}

// TestMetricsContainExpectedLabels verifies container metrics have the 'id' label.
func TestMetricsContainExpectedLabels(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// Container metrics should have 'id' label
	containerMetrics := []string{
		"container_cpu_usage_seconds_total",
		"container_memory_usage_bytes",
	}

	for _, name := range containerMetrics {
		mf, ok := families[name]
		if !ok {
			continue
		}
		if len(mf.GetMetric()) == 0 {
			continue
		}

		// Check first metric has 'id' label
		metric := mf.GetMetric()[0]
		id := framework.GetLabelValue(metric, "id")
		assert.NotEmpty(t, id, "Metric %s should have 'id' label", name)
	}
}

// TestDiskIOMetricsExist verifies disk I/O metrics are present.
func TestDiskIOMetricsExist(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	families, err := client.FetchAndParse()
	require.NoError(t, err)

	// At least some disk I/O metrics should exist
	diskIOMetrics := []string{
		"container_fs_reads_bytes_total",
		"container_fs_writes_bytes_total",
		"container_fs_reads_total",
		"container_fs_writes_total",
	}

	foundCount := 0
	for _, name := range diskIOMetrics {
		if framework.HasMetric(families, name) {
			foundCount++
		}
	}

	// Some environments may not have all disk I/O metrics
	// but we should have at least one
	assert.GreaterOrEqual(t, foundCount, 1, "Should have at least one disk I/O metric")
}

// TestMetricsResponseContainsComments verifies the response contains # HELP and # TYPE.
func TestMetricsResponseContainsComments(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	client := framework.NewMetricsClient(fm.Hostname())
	text, err := client.Fetch()
	require.NoError(t, err)

	assert.True(t, strings.Contains(text, "# HELP"),
		"Metrics response should contain # HELP comments")
	assert.True(t, strings.Contains(text, "# TYPE"),
		"Metrics response should contain # TYPE comments")
}
