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
	info "github.com/google/cadvisor/info/v1"
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/klog"
)

var baseLabelsNames = []string{"machine_id", "system_uuid", "boot_id"}

const (
	prometheusModeLabelName = "mode"
	prometheusTypeLabelName = "type"

	nvmMemoryMode    = "memory_mode"
	nvmAppDirectMode = "app_direct_mode"

	memoryByTypeDimmCountKey    = "DimmCount"
	memoryByTypeDimmCapacityKey = "Capacity"
	memoryByTypeAllType         = "all"
)

// machineMetric describes a multi-dimensional metric used for exposing a
// certain type of machine statistic.
type machineMetric struct {
	name        string
	help        string
	valueType   prometheus.ValueType
	extraLabels []string
	condition   func(machineInfo *info.MachineInfo) bool
	getValues   func(machineInfo *info.MachineInfo) metricValues
}

func (metric *machineMetric) desc(baseLabels []string) *prometheus.Desc {
	return prometheus.NewDesc(metric.name, metric.help, append(baseLabels, metric.extraLabels...), nil)
}

// PrometheusMachineCollector implements prometheus.Collector.
type PrometheusMachineCollector struct {
	infoProvider   infoProvider
	errors         prometheus.Gauge
	machineMetrics []machineMetric
}

// NewPrometheusMachineCollector returns a new PrometheusCollector.
func NewPrometheusMachineCollector(i infoProvider) *PrometheusMachineCollector {
	c := &PrometheusMachineCollector{
		infoProvider: i,
		errors: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "machine",
			Name:      "scrape_error",
			Help:      "1 if there was an error while getting machine metrics, 0 otherwise.",
		}),
		machineMetrics: []machineMetric{
			{
				name:      "machine_cpu_physical_cores",
				help:      "Number of physical CPU cores.",
				valueType: prometheus.GaugeValue,
				getValues: func(machineInfo *info.MachineInfo) metricValues {
					return metricValues{{value: float64(machineInfo.NumPhysicalCores)}}
				},
			},
			{
				name:      "machine_cpu_cores",
				help:      "Number of logical CPU cores.",
				valueType: prometheus.GaugeValue,
				getValues: func(machineInfo *info.MachineInfo) metricValues {
					return metricValues{{value: float64(machineInfo.NumCores)}}
				},
			},
			{
				name:      "machine_cpu_sockets",
				help:      "Number of CPU sockets.",
				valueType: prometheus.GaugeValue,
				getValues: func(machineInfo *info.MachineInfo) metricValues {
					return metricValues{{value: float64(machineInfo.NumSockets)}}
				},
			},
			{
				name:      "machine_memory_bytes",
				help:      "Amount of memory installed on the machine.",
				valueType: prometheus.GaugeValue,
				getValues: func(machineInfo *info.MachineInfo) metricValues {
					return metricValues{{value: float64(machineInfo.MemoryCapacity)}}
				},
			},
			{
				name:        "machine_dimm_count",
				help:        "Number of RAM DIMM (all types memory modules) value labeled by dimm type.",
				valueType:   prometheus.GaugeValue,
				extraLabels: []string{prometheusTypeLabelName},
				condition:   func(machineInfo *info.MachineInfo) bool { return len(machineInfo.MemoryByType) != 0 },
				getValues: func(machineInfo *info.MachineInfo) metricValues {
					return getMemoryByType(machineInfo, memoryByTypeDimmCountKey)
				},
			},
			{
				name:        "machine_dimm_capacity_bytes",
				help:        "Total RAM DIMM capacity (all types memory modules) value labeled by dimm type.",
				valueType:   prometheus.GaugeValue,
				extraLabels: []string{prometheusTypeLabelName},
				condition:   func(machineInfo *info.MachineInfo) bool { return len(machineInfo.MemoryByType) != 0 },
				getValues: func(machineInfo *info.MachineInfo) metricValues {
					return getMemoryByType(machineInfo, memoryByTypeDimmCapacityKey)
				},
			},
			{
				name:        "machine_nvm_capacity",
				help:        "NVM capacity value labeled by NVM mode (memory mode or app direct mode).",
				valueType:   prometheus.GaugeValue,
				extraLabels: []string{prometheusModeLabelName},
				getValues: func(machineInfo *info.MachineInfo) metricValues {
					return metricValues{
						{value: float64(machineInfo.NVMInfo.MemoryModeCapacity), labels: []string{nvmMemoryMode}},
						{value: float64(machineInfo.NVMInfo.AppDirectModeCapacity), labels: []string{nvmAppDirectMode}},
					}
				},
			},
		},
	}
	return c
}

// Describe describes all the machine metrics ever exported by cadvisor. It
// implements prometheus.PrometheusCollector.
func (collector *PrometheusMachineCollector) Describe(ch chan<- *prometheus.Desc) {
	collector.errors.Describe(ch)
	for _, metric := range collector.machineMetrics {
		ch <- metric.desc([]string{})
	}
}

// Collect fetches information about machine and delivers them as
// Prometheus metrics. It implements prometheus.PrometheusCollector.
func (collector *PrometheusMachineCollector) Collect(ch chan<- prometheus.Metric) {
	collector.errors.Set(0)
	collector.collectMachineInfo(ch)
	collector.errors.Collect(ch)
}

func (collector *PrometheusMachineCollector) collectMachineInfo(ch chan<- prometheus.Metric) {
	machineInfo, err := collector.infoProvider.GetMachineInfo()
	if err != nil {
		collector.errors.Set(1)
		klog.Warningf("Couldn't get machine info: %s", err)
		return
	}

	baseLabelsValues := []string{machineInfo.MachineID, machineInfo.SystemUUID, machineInfo.BootID}

	for _, metric := range collector.machineMetrics {
		if metric.condition != nil && !metric.condition(machineInfo) {
			continue
		}

		for _, metricValue := range metric.getValues(machineInfo) {
			labelValues := make([]string, len(baseLabelsValues))
			copy(labelValues, baseLabelsValues)
			if len(metric.extraLabels) != 0 {
				labelValues = append(labelValues, metricValue.labels...)
			}
			ch <- prometheus.MustNewConstMetric(metric.desc(baseLabelsNames),
				metric.valueType, metricValue.value, labelValues...)
		}

	}
}

func getMemoryByType(machineInfo *info.MachineInfo, property string) metricValues {
	mValues := make(metricValues, 0, len(machineInfo.MemoryByType))
	allValue := 0.0
	for memoryType, memoryInfo := range machineInfo.MemoryByType {
		propertyValue := 0.0
		switch property {
		case memoryByTypeDimmCapacityKey:
			propertyValue = float64(memoryInfo.Capacity)
		case memoryByTypeDimmCountKey:
			propertyValue = float64(memoryInfo.DimmCount)
		default:
			klog.Warningf("Incorrect propery name for MemoryByType, property %s", property)
			return metricValues{}
		}
		allValue += propertyValue
		mValues = append(mValues, metricValue{value: propertyValue, labels: []string{memoryType}})
	}
	mValues = append(mValues, metricValue{value: allValue, labels: []string{memoryByTypeAllType}})
	return mValues
}
