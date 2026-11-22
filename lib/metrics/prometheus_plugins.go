// Copyright 2026 Google Inc. All Rights Reserved.
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
	"github.com/prometheus/client_golang/prometheus"

	"github.com/google/cadvisor/lib/container"
)

var pluginRegisteredDesc = prometheus.NewDesc(
	"cadvisor_container_plugin_registered",
	"Whether the container runtime plugin has registered its factory (1) or not (0).",
	[]string{"plugin"}, nil)

// PrometheusPluginCollector exports the registration state of container
// plugins, so operators can alert on a runtime cAdvisor can't reach.
type PrometheusPluginCollector struct {
	statuses func() map[string]container.PluginStatus
}

// NewPrometheusPluginCollector returns a collector for container.PluginStatuses.
func NewPrometheusPluginCollector() *PrometheusPluginCollector {
	return &PrometheusPluginCollector{statuses: container.PluginStatuses}
}

// Describe describes all the metrics ever exported by the collector. It
// implements prometheus.Collector.
func (c *PrometheusPluginCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- pluginRegisteredDesc
}

// Collect fetches the plugin registration states and delivers them as
// Prometheus metrics. It implements prometheus.Collector.
func (c *PrometheusPluginCollector) Collect(ch chan<- prometheus.Metric) {
	for name, status := range c.statuses() {
		value := 0.0
		if status.Registered {
			value = 1
		}
		ch <- prometheus.MustNewConstMetric(pluginRegisteredDesc, prometheus.GaugeValue, value, name)
	}
}
