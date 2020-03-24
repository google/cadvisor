// Copyright 2020 Google Inc. All Rights Reserved.
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
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

const machineMetricsFile = "testdata/prometheus_machine_metrics"
const machineMetricsFailureFile = "testdata/prometheus_machine_metrics_failure"

func TestPrometheusMachineCollector(t *testing.T) {
	collector := NewPrometheusMachineCollector(testSubcontainersInfoProvider{})
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metricsFamily, err := registry.Gather()
	assert.Nil(t, err)

	var metricBuffer bytes.Buffer
	for _, metricFamily := range metricsFamily {
		_, err := expfmt.MetricFamilyToText(&metricBuffer, metricFamily)
		assert.Nil(t, err)
	}
	collectedMetrics := string(metricBuffer.Bytes())
	expectedMetrics, err := ioutil.ReadFile(machineMetricsFile)
	assert.Nil(t, err)
	assert.Equal(t, string(expectedMetrics), collectedMetrics)
}

func TestPrometheusMachineCollectorWithFailure(t *testing.T) {
	provider := &erroringSubcontainersInfoProvider{
		successfulProvider: testSubcontainersInfoProvider{},
		shouldFail:         true,
	}
	collector := NewPrometheusMachineCollector(provider)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metricsFamily, err := registry.Gather()
	assert.Nil(t, err)

	var metricBuffer bytes.Buffer
	for _, metricFamily := range metricsFamily {
		_, err := expfmt.MetricFamilyToText(&metricBuffer, metricFamily)
		assert.Nil(t, err)
	}
	collectedMetrics := string(metricBuffer.Bytes())
	expectedMetrics, err := ioutil.ReadFile(machineMetricsFailureFile)
	assert.Nil(t, err)
	assert.Equal(t, string(expectedMetrics), collectedMetrics)
}

func TestGetMemoryByType(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	capacityMetrics := getMemoryByType(machineInfo, memoryByTypeDimmCapacityKey)
	assert.Equal(t, 3, len(capacityMetrics))

	countMetrics := getMemoryByType(machineInfo, memoryByTypeDimmCountKey)
	assert.Equal(t, 3, len(countMetrics))
}

func TestGetMemoryByTypeWithWrongProperty(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	metricVals := getMemoryByType(machineInfo, "wrong_property_name")
	assert.Equal(t, 0, len(metricVals))
}
