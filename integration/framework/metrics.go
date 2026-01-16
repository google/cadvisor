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

package framework

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// MetricsClient provides methods for fetching and parsing Prometheus metrics
// from cAdvisor's /metrics endpoint.
type MetricsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMetricsClient creates a new client for the /metrics endpoint.
func NewMetricsClient(hostname HostnameInfo) *MetricsClient {
	return &MetricsClient{
		baseURL: hostname.FullHostname(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Fetch retrieves raw metrics text from the /metrics endpoint.
func (m *MetricsClient) Fetch() (string, error) {
	return m.FetchWithParams("")
}

// FetchWithParams retrieves metrics with optional query parameters.
// Parameters can be "type=docker" or "type=name" to filter containers.
func (m *MetricsClient) FetchWithParams(params string) (string, error) {
	url := m.baseURL + "metrics"
	if params != "" {
		url += "?" + params
	}

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("metrics endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// Parse converts Prometheus text format to metric families.
func (m *MetricsClient) Parse(metricsText string) (map[string]*dto.MetricFamily, error) {
	parser := expfmt.TextParser{}
	return parser.TextToMetricFamilies(strings.NewReader(metricsText))
}

// FetchAndParse combines Fetch and Parse into one call.
func (m *MetricsClient) FetchAndParse() (map[string]*dto.MetricFamily, error) {
	text, err := m.Fetch()
	if err != nil {
		return nil, err
	}
	return m.Parse(text)
}

// HasMetric checks if a metric family exists by name.
func HasMetric(families map[string]*dto.MetricFamily, name string) bool {
	_, ok := families[name]
	return ok
}

// GetMetricFamily returns a specific metric family by name.
func GetMetricFamily(families map[string]*dto.MetricFamily, name string) (*dto.MetricFamily, bool) {
	mf, ok := families[name]
	return mf, ok
}

// FindMetricWithLabels finds a metric matching all specified labels.
// Returns nil if no matching metric is found.
func FindMetricWithLabels(mf *dto.MetricFamily, labels map[string]string) *dto.Metric {
	if mf == nil {
		return nil
	}
	for _, metric := range mf.GetMetric() {
		if matchesLabels(metric, labels) {
			return metric
		}
	}
	return nil
}

// FindMetricsWithLabelSubstring finds all metrics where the specified label
// contains the given substring.
func FindMetricsWithLabelSubstring(mf *dto.MetricFamily, labelName, substring string) []*dto.Metric {
	if mf == nil {
		return nil
	}
	var result []*dto.Metric
	for _, metric := range mf.GetMetric() {
		for _, lp := range metric.GetLabel() {
			if lp.GetName() == labelName && strings.Contains(lp.GetValue(), substring) {
				result = append(result, metric)
				break
			}
		}
	}
	return result
}

// GetGaugeValue extracts the value from a gauge metric.
func GetGaugeValue(metric *dto.Metric) float64 {
	if metric == nil || metric.GetGauge() == nil {
		return 0
	}
	return metric.GetGauge().GetValue()
}

// GetCounterValue extracts the value from a counter metric.
func GetCounterValue(metric *dto.Metric) float64 {
	if metric == nil || metric.GetCounter() == nil {
		return 0
	}
	return metric.GetCounter().GetValue()
}

// GetLabelValue returns the value of a specific label from a metric.
// Returns empty string if label is not found.
func GetLabelValue(metric *dto.Metric, labelName string) string {
	if metric == nil {
		return ""
	}
	for _, lp := range metric.GetLabel() {
		if lp.GetName() == labelName {
			return lp.GetValue()
		}
	}
	return ""
}

// ContainsLabelValue checks if any metric in the family has the label
// containing the given substring.
func ContainsLabelValue(mf *dto.MetricFamily, labelName, substring string) bool {
	if mf == nil {
		return false
	}
	for _, metric := range mf.GetMetric() {
		for _, lp := range metric.GetLabel() {
			if lp.GetName() == labelName && strings.Contains(lp.GetValue(), substring) {
				return true
			}
		}
	}
	return false
}

// GetMetricType returns the type of a metric family as a string.
func GetMetricType(mf *dto.MetricFamily) string {
	if mf == nil {
		return "unknown"
	}
	return mf.GetType().String()
}

// matchesLabels checks if a metric has all the specified labels with exact values.
func matchesLabels(metric *dto.Metric, targetLabels map[string]string) bool {
	if metric == nil {
		return false
	}
	labelMap := make(map[string]string)
	for _, lp := range metric.GetLabel() {
		labelMap[lp.GetName()] = lp.GetValue()
	}
	for k, v := range targetLabels {
		if labelMap[k] != v {
			return false
		}
	}
	return true
}

// CountMetrics returns the number of metric samples in a metric family.
func CountMetrics(mf *dto.MetricFamily) int {
	if mf == nil {
		return 0
	}
	return len(mf.GetMetric())
}

// GetAllLabelValues returns all unique values for a given label name across
// all metrics in the family.
func GetAllLabelValues(mf *dto.MetricFamily, labelName string) []string {
	if mf == nil {
		return nil
	}
	seen := make(map[string]bool)
	var values []string
	for _, metric := range mf.GetMetric() {
		for _, lp := range metric.GetLabel() {
			if lp.GetName() == labelName {
				val := lp.GetValue()
				if !seen[val] {
					seen[val] = true
					values = append(values, val)
				}
			}
		}
	}
	return values
}
