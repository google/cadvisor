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

package main

import (
	"flag"
	"testing"

	"github.com/google/cadvisor/container"
	"github.com/stretchr/testify/assert"
)

func TestTcpMetricsAreDisabledByDefault(t *testing.T) {
	assert.True(t, ignoreMetrics.Has(container.NetworkTcpUsageMetrics))
	flag.Parse()
	assert.True(t, ignoreMetrics.Has(container.NetworkTcpUsageMetrics))
}

func TestAdvancedTcpMetricsAreDisabledByDefault(t *testing.T) {
	assert.True(t, ignoreMetrics.Has(container.NetworkAdvancedTcpUsageMetrics))
	flag.Parse()
	assert.True(t, ignoreMetrics.Has(container.NetworkAdvancedTcpUsageMetrics))
}

func TestUdpMetricsAreDisabledByDefault(t *testing.T) {
	assert.True(t, ignoreMetrics.Has(container.NetworkUdpUsageMetrics))
	flag.Parse()
	assert.True(t, ignoreMetrics.Has(container.NetworkUdpUsageMetrics))
}

func TestReferencedMemoryMetricsIsDisabledByDefault(t *testing.T) {
	assert.True(t, ignoreMetrics.Has(container.ReferencedMemoryMetrics))
	flag.Parse()
	assert.True(t, ignoreMetrics.Has(container.ReferencedMemoryMetrics))
}

func TestCPUTopologyMetricsAreDisabledByDefault(t *testing.T) {
	assert.True(t, ignoreMetrics.Has(container.CPUTopologyMetrics))
	flag.Parse()
	assert.True(t, ignoreMetrics.Has(container.CPUTopologyMetrics))
}

func TestMemoryNumaMetricsAreDisabledByDefault(t *testing.T) {
	assert.True(t, ignoreMetrics.Has(container.MemoryNumaMetrics))
	flag.Parse()
	assert.True(t, ignoreMetrics.Has(container.MemoryNumaMetrics))
}

func TestEnableAndIgnoreMetrics(t *testing.T) {
	tests := []struct {
		value    string
		expected []container.MetricKind
	}{
		{"", []container.MetricKind{}},
		{"disk", []container.MetricKind{container.DiskUsageMetrics}},
		{"disk,tcp,network", []container.MetricKind{container.DiskUsageMetrics, container.NetworkTcpUsageMetrics, container.NetworkUsageMetrics}},
	}

	for _, test := range tests {
		for _, sets := range []metricSetValue{enableMetrics, ignoreMetrics} {
			assert.NoError(t, sets.Set(test.value))

			assert.Equal(t, len(test.expected), len(sets.MetricSet))
			for _, expected := range test.expected {
				assert.True(t, sets.Has(expected), "Missing %s", expected)
			}
		}
	}
}

func TestToIncludedMetrics(t *testing.T) {
	ignores := []container.MetricSet{
		{
			container.CpuUsageMetrics: struct{}{},
		},
		{},
		container.AllMetrics,
	}

	expected := []container.MetricSet{
		{
			container.ProcessSchedulerMetrics:        struct{}{},
			container.PerCpuUsageMetrics:             struct{}{},
			container.MemoryUsageMetrics:             struct{}{},
			container.MemoryNumaMetrics:              struct{}{},
			container.CpuLoadMetrics:                 struct{}{},
			container.DiskIOMetrics:                  struct{}{},
			container.AcceleratorUsageMetrics:        struct{}{},
			container.DiskUsageMetrics:               struct{}{},
			container.NetworkUsageMetrics:            struct{}{},
			container.NetworkTcpUsageMetrics:         struct{}{},
			container.NetworkAdvancedTcpUsageMetrics: struct{}{},
			container.NetworkUdpUsageMetrics:         struct{}{},
			container.ProcessMetrics:                 struct{}{},
			container.AppMetrics:                     struct{}{},
			container.HugetlbUsageMetrics:            struct{}{},
			container.PerfMetrics:                    struct{}{},
			container.ReferencedMemoryMetrics:        struct{}{},
			container.CPUTopologyMetrics:             struct{}{},
			container.ResctrlMetrics:                 struct{}{},
			container.CPUSetMetrics:                  struct{}{},
			container.OOMMetrics:                     struct{}{},
		},
		container.AllMetrics,
		{},
	}

	for idx, ignore := range ignores {
		actual := toIncludedMetrics(ignore)
		assert.Equal(t, actual, expected[idx])
	}
}
