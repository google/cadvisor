// Copyright 2015 Google Inc. All Rights Reserved.
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

package collector

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/cadvisor/info/v1"

	containertest "github.com/google/cadvisor/container/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheus(t *testing.T) {
	assert := assert.New(t)

	//Create a prometheus collector using the config file 'sample_config_prometheus.json'
	configFile, err := ioutil.ReadFile("config/sample_config_prometheus.json")
	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	collector, err := NewPrometheusCollector("Prometheus", configFile, 100, containerHandler, http.DefaultClient)
	assert.NoError(err)
	assert.Equal(collector.name, "Prometheus")
	assert.Equal(collector.configFile.Endpoint.URL, "http://localhost:8080/metrics")

	tempServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		text := "# HELP go_gc_duration_seconds A summary of the GC invocation durations.\n"
		text += "# TYPE go_gc_duration_seconds summary\n"
		text += "go_gc_duration_seconds{quantile=\"0\"} 5.8348000000000004e-05\n"
		text += "go_gc_duration_seconds{quantile=\"1\"} 0.000499764\n"
		text += "# HELP go_goroutines Number of goroutines that currently exist.\n"
		text += "# TYPE go_goroutines gauge\n"
		text += "go_goroutines 16\n"
		text += "# HELP empty_metric A metric without any values\n"
		text += "# TYPE empty_metric counter\n"
		text += "\n"
		fmt.Fprintln(w, text)
	}))

	defer tempServer.Close()

	collector.configFile.Endpoint.URL = tempServer.URL

	var spec []v1.MetricSpec
	require.NotPanics(t, func() { spec = collector.GetSpec() })
	assert.Len(spec, 2)
	assert.Equal(spec[0].Name, "go_gc_duration_seconds")
	assert.Equal(spec[1].Name, "go_goroutines")

	metrics := map[string][]v1.MetricVal{}
	_, metrics, errMetric := collector.Collect(metrics)

	assert.NoError(errMetric)

	go_gc_duration := metrics["go_gc_duration_seconds"]
	assert.Equal(go_gc_duration[0].FloatValue, 5.8348000000000004e-05)
	assert.Equal(go_gc_duration[1].FloatValue, 0.000499764)

	goRoutines := metrics["go_goroutines"]
	assert.Equal(goRoutines[0].FloatValue, 16)
}

func TestPrometheusEndpointConfig(t *testing.T) {
	assert := assert.New(t)

	//Create a prometheus collector using the config file 'sample_config_prometheus.json'
	configFile, err := ioutil.ReadFile("config/sample_config_prometheus_endpoint_config.json")
	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	containerHandler.On("GetContainerIPAddress").Return(
		"222.222.222.222",
	)

	collector, err := NewPrometheusCollector("Prometheus", configFile, 100, containerHandler, http.DefaultClient)
	assert.NoError(err)
	assert.Equal(collector.name, "Prometheus")
	assert.Equal(collector.configFile.Endpoint.URL, "http://222.222.222.222:8081/METRICS")
}

func TestPrometheusShortResponse(t *testing.T) {
	assert := assert.New(t)

	//Create a prometheus collector using the config file 'sample_config_prometheus.json'
	configFile, err := ioutil.ReadFile("config/sample_config_prometheus.json")
	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	collector, err := NewPrometheusCollector("Prometheus", configFile, 100, containerHandler, http.DefaultClient)
	assert.NoError(err)
	assert.Equal(collector.name, "Prometheus")
	assert.Equal(collector.configFile.Endpoint.URL, "http://localhost:8080/metrics")

	tempServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := "# HELP empty_metric A metric without any values"
		fmt.Fprint(w, text)
	}))

	defer tempServer.Close()

	collector.configFile.Endpoint.URL = tempServer.URL

	assert.NotPanics(func() { collector.GetSpec() })
}

func TestPrometheusMetricCountLimit(t *testing.T) {
	assert := assert.New(t)

	//Create a prometheus collector using the config file 'sample_config_prometheus.json'
	configFile, err := ioutil.ReadFile("config/sample_config_prometheus.json")
	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	collector, err := NewPrometheusCollector("Prometheus", configFile, 10, containerHandler, http.DefaultClient)
	assert.NoError(err)
	assert.Equal(collector.name, "Prometheus")
	assert.Equal(collector.configFile.Endpoint.URL, "http://localhost:8080/metrics")

	tempServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 30; i++ {
			fmt.Fprintf(w, "# HELP m%d Number of goroutines that currently exist.\n", i)
			fmt.Fprintf(w, "# TYPE m%d gauge\n", i)
			fmt.Fprintf(w, "m%d %d", i, i)
		}
	}))
	defer tempServer.Close()

	collector.configFile.Endpoint.URL = tempServer.URL
	metrics := map[string][]v1.MetricVal{}
	_, result, errMetric := collector.Collect(metrics)

	assert.Error(errMetric)
	assert.Equal(len(metrics), 0)
	assert.Nil(result)
}

func TestPrometheusFiltersMetrics(t *testing.T) {
	assert := assert.New(t)

	//Create a prometheus collector using the config file 'sample_config_prometheus_filtered.json'
	configFile, err := ioutil.ReadFile("config/sample_config_prometheus_filtered.json")
	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	collector, err := NewPrometheusCollector("Prometheus", configFile, 100, containerHandler, http.DefaultClient)
	assert.NoError(err)
	assert.Equal(collector.name, "Prometheus")
	assert.Equal(collector.configFile.Endpoint.URL, "http://localhost:8080/metrics")

	tempServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		text := "# HELP go_gc_duration_seconds A summary of the GC invocation durations.\n"
		text += "# TYPE go_gc_duration_seconds summary\n"
		text += "go_gc_duration_seconds{quantile=\"0\"} 5.8348000000000004e-05\n"
		text += "go_gc_duration_seconds{quantile=\"1\"} 0.000499764\n"
		text += "# HELP go_goroutines Number of goroutines that currently exist.\n"
		text += "# TYPE go_goroutines gauge\n"
		text += "go_goroutines 16"
		fmt.Fprintln(w, text)
	}))

	defer tempServer.Close()

	collector.configFile.Endpoint.URL = tempServer.URL
	metrics := map[string][]v1.MetricVal{}
	_, metrics, errMetric := collector.Collect(metrics)

	assert.NoError(errMetric)
	assert.Len(metrics, 1)

	goRoutines := metrics["go_goroutines"]
	assert.Equal(goRoutines[0].FloatValue, 16)
}

func TestPrometheusFiltersMetricsCountLimit(t *testing.T) {
	assert := assert.New(t)

	//Create a prometheus collector using the config file 'sample_config_prometheus_filtered.json'
	configFile, err := ioutil.ReadFile("config/sample_config_prometheus_filtered.json")
	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	_, err = NewPrometheusCollector("Prometheus", configFile, 1, containerHandler, http.DefaultClient)
	assert.Error(err)
}
