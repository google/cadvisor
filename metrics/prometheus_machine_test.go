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
	"reflect"
	"testing"
	"time"

	"github.com/google/cadvisor/container"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

const machineMetricsFile = "testdata/prometheus_machine_metrics"
const machineMetricsFailureFile = "testdata/prometheus_machine_metrics_failure"

func TestPrometheusMachineCollector(t *testing.T) {
	collector := NewPrometheusMachineCollector(testSubcontainersInfoProvider{}, container.AllMetrics)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metricsFamily, err := registry.Gather()
	assert.Nil(t, err)

	var metricBuffer bytes.Buffer
	for _, metricFamily := range metricsFamily {
		_, err := expfmt.MetricFamilyToText(&metricBuffer, metricFamily)
		assert.Nil(t, err)
	}
	collectedMetrics := metricBuffer.String()

	expectedMetrics, err := ioutil.ReadFile(machineMetricsFile)
	assert.Nil(t, err)
	assert.Equal(t, string(expectedMetrics), collectedMetrics)
}

func TestPrometheusMachineCollectorWithFailure(t *testing.T) {
	provider := &erroringSubcontainersInfoProvider{
		successfulProvider: testSubcontainersInfoProvider{},
		shouldFail:         true,
	}
	collector := NewPrometheusMachineCollector(provider, container.AllMetrics)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	metricsFamily, err := registry.Gather()
	assert.Nil(t, err)

	var metricBuffer bytes.Buffer
	for _, metricFamily := range metricsFamily {
		_, err := expfmt.MetricFamilyToText(&metricBuffer, metricFamily)
		assert.Nil(t, err)
	}
	collectedMetrics := metricBuffer.String()
	expectedMetrics, err := ioutil.ReadFile(machineMetricsFailureFile)
	assert.Nil(t, err)
	assert.Equal(t, string(expectedMetrics), collectedMetrics)
}

func TestGetMemoryByType(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	capacityMetrics := getMemoryByType(machineInfo, memoryByTypeDimmCapacityKey)
	assert.Equal(t, 2, len(capacityMetrics))

	countMetrics := getMemoryByType(machineInfo, memoryByTypeDimmCountKey)
	assert.Equal(t, 2, len(countMetrics))
}

func TestGetMemoryByTypeWithWrongProperty(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	metricVals := getMemoryByType(machineInfo, "wrong_property_name")
	assert.Equal(t, 0, len(metricVals))
}

func TestGetCaches(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	metricVals := getCaches(machineInfo)

	assert.Equal(t, 25, len(metricVals))
	expectedMetricVals := []metricValue{
		{value: 32768, labels: []string{"0", "0", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32768, labels: []string{"0", "0", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262144, labels: []string{"0", "0", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"0", "1", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"0", "1", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262148, labels: []string{"0", "1", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32768, labels: []string{"0", "2", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32768, labels: []string{"0", "2", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262144, labels: []string{"0", "2", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"0", "3", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"0", "3", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262148, labels: []string{"0", "3", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32768, labels: []string{"1", "4", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32768, labels: []string{"1", "4", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262144, labels: []string{"1", "4", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"1", "5", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"1", "5", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262148, labels: []string{"1", "5", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32768, labels: []string{"1", "6", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32768, labels: []string{"1", "6", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262144, labels: []string{"1", "6", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"1", "7", "Data", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 32764, labels: []string{"1", "7", "Instruction", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 262148, labels: []string{"1", "7", "Unified", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 8388608, labels: []string{"1", "", "Unified", "3"}, timestamp: time.Unix(1395066363, 0)},
	}
	assertMetricValues(t, expectedMetricVals, metricVals, "Unexpected information about Node memory")
}

func TestGetThreadsSiblingsCount(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	metricVals := getThreadsSiblingsCount(machineInfo)

	assert.Equal(t, 16, len(metricVals))
	expectedMetricVals := []metricValue{
		{value: 2, labels: []string{"0", "0", "0"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"0", "0", "1"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"0", "1", "2"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"0", "1", "3"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"0", "2", "4"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"0", "2", "5"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"0", "3", "6"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"0", "3", "7"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "4", "8"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "4", "9"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "5", "10"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "5", "11"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "6", "12"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "6", "13"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "7", "14"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "7", "15"}, timestamp: time.Unix(1395066363, 0)},
	}
	assertMetricValues(t, expectedMetricVals, metricVals, "Unexpected information about CPU threads")
}

func TestGetNodeMemory(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	metricVals := getNodeMemory(machineInfo)

	assert.Equal(t, 2, len(metricVals))
	expectedMetricVals := []metricValue{
		{value: 33604804608, labels: []string{"0"}, timestamp: time.Unix(1395066363, 0)},
		{value: 33604804606, labels: []string{"1"}, timestamp: time.Unix(1395066363, 0)},
	}
	assertMetricValues(t, expectedMetricVals, metricVals, "Unexpected information about Node memory")
}

func TestGetHugePagesCount(t *testing.T) {
	machineInfo, err := testSubcontainersInfoProvider{}.GetMachineInfo()
	assert.Nil(t, err)

	metricVals := getHugePagesCount(machineInfo)

	assert.Equal(t, 4, len(metricVals))
	expectedMetricVals := []metricValue{
		{value: 0, labels: []string{"0", "1048576"}, timestamp: time.Unix(1395066363, 0)},
		{value: 0, labels: []string{"0", "2048"}, timestamp: time.Unix(1395066363, 0)},
		{value: 2, labels: []string{"1", "1048576"}, timestamp: time.Unix(1395066363, 0)},
		{value: 4, labels: []string{"1", "2048"}, timestamp: time.Unix(1395066363, 0)},
	}
	assertMetricValues(t, expectedMetricVals, metricVals, "Unexpected information about Node memory")
}

func assertMetricValues(t *testing.T, expected metricValues, actual metricValues, message string) {
	for i := range actual {
		assert.Truef(t, reflect.DeepEqual(expected[i], actual[i]),
			"%s expected %#v but found %#v\n", message, expected[i], actual[i])
	}
}
