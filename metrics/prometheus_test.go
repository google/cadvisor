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

package metrics

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	includeRe = regexp.MustCompile(`^(?:(?:# HELP |# TYPE )?container_|cadvisor_version_info\{)`)
	ignoreRe  = regexp.MustCompile(`^container_last_seen\{`)
)

func TestPrometheusCollector(t *testing.T) {
	c := NewPrometheusCollector(testSubcontainersInfoProvider{}, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["zone.name"] = "hello"
		return s
	}, container.AllMetrics)
	prometheus.MustRegister(c)
	defer prometheus.Unregister(c)

	testPrometheusCollector(t, c, "testdata/prometheus_metrics")
}

func testPrometheusCollector(t *testing.T, c *PrometheusCollector, metricsFile string) {
	rw := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rw, &http.Request{})

	wantMetrics, err := ioutil.ReadFile(metricsFile)
	if err != nil {
		t.Fatalf("unable to read input test file %s", metricsFile)
	}

	wantLines := strings.Split(string(wantMetrics), "\n")
	gotLines := strings.Split(string(rw.Body.String()), "\n")

	// Until the Prometheus Go client library offers better testability
	// (https://github.com/prometheus/client_golang/issues/58), we simply compare
	// verbatim text-format metrics outputs, but ignore certain metric lines
	// whose value depends on the current time or local circumstances.
	for i, want := range wantLines {
		if !includeRe.MatchString(want) || ignoreRe.MatchString(want) {
			continue
		}
		if want != gotLines[i] {
			t.Fatalf("unexpected metric line\nwant: %s\nhave: %s", want, gotLines[i])
		}
	}
}

func TestPrometheusCollector_scrapeFailure(t *testing.T) {
	provider := &erroringSubcontainersInfoProvider{
		successfulProvider: testSubcontainersInfoProvider{},
		shouldFail:         true,
	}

	c := NewPrometheusCollector(provider, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["zone.name"] = "hello"
		return s
	}, container.AllMetrics)
	prometheus.MustRegister(c)
	defer prometheus.Unregister(c)

	testPrometheusCollector(t, c, "testdata/prometheus_metrics_failure")

	provider.shouldFail = false

	testPrometheusCollector(t, c, "testdata/prometheus_metrics")
}
