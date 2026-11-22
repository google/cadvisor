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
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/google/cadvisor/lib/container"
	info "github.com/google/cadvisor/lib/model"
)

// fanOutInfoProvider returns n copies of the single test container with
// distinct names, approximating a host running n containers.
type fanOutInfoProvider struct {
	testSubcontainersInfoProvider
	n int
}

func (p fanOutInfoProvider) GetRequestedContainersInfo(name string, opts info.RequestOptions) (map[string]*info.ContainerInfo, error) {
	base, err := p.testSubcontainersInfoProvider.GetRequestedContainersInfo(name, opts)
	if err != nil {
		return nil, err
	}
	src := base["testcontainer"]
	out := make(map[string]*info.ContainerInfo, p.n)
	for i := 0; i < p.n; i++ {
		c := *src
		c.Name = fmt.Sprintf("/docker/%040x", i)
		c.Aliases = []string{fmt.Sprintf("container-%d", i)}
		out[c.Name] = &c
	}
	return out, nil
}

// benchMetricSet is a typical production set: the default metrics plus the
// per-process and pressure families that are commonly enabled.
func benchMetricSet() container.MetricSet {
	s := container.MetricSet{}
	if err := s.Set("cpu,cpuLoad,cpuset,diskIO,memory,process,resctrl,pressure"); err != nil {
		panic(err)
	}
	return s
}

// BenchmarkPrometheusCollectorGather measures one full scrape of the cadvisor
// Prometheus collector for a host with n containers: Collect plus the
// registry's consistency checking, which is what a /metrics scrape pays.
func BenchmarkPrometheusCollectorGather(b *testing.B) {
	for _, n := range []int{20, 100, 400} {
		b.Run(fmt.Sprintf("containers=%d", n), func(b *testing.B) {
			c := NewPrometheusCollector(fanOutInfoProvider{n: n}, DefaultContainerLabels, benchMetricSet(), now, info.RequestOptions{})
			reg := prometheus.NewRegistry()
			reg.MustRegister(c)
			mfs, err := reg.Gather()
			if err != nil {
				b.Fatal(err)
			}
			var samples int
			for _, mf := range mfs {
				samples += len(mf.Metric)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := reg.Gather(); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(samples), "samples/op")
		})
	}
}
