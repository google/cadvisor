// Copyright 2025 Google Inc. All Rights Reserved.
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
	"testing"

	"github.com/google/cadvisor/lib/container"
	info "github.com/google/cadvisor/lib/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrometheusCollectorCachedDescsKeepExtraLabels guards the cached descs
// against sharing one backing array. When two raw labels collide after
// sanitizing ("foo.bar" and "foo_bar" both become "foo_bar") one is dropped,
// which leaves spare capacity in the base label slice; building every desc
// with append(baseLabels, extraLabels...) would then make them alias each
// other, and the last extra label written would show up on every cached
// desc (container_cpu_usage_seconds_total{interface="cpu00"}).
func TestPrometheusCollectorCachedDescsKeepExtraLabels(t *testing.T) {
	c := NewPrometheusCollector(testSubcontainersInfoProvider{}, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["foo.bar"] = "a"
		s["foo_bar"] = "b"
		return s
	}, container.AllMetrics, now, info.RequestOptions{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	wantExtra := make(map[string][]string, len(c.containerMetrics))
	for _, cm := range c.containerMetrics {
		wantExtra[cm.name] = cm.extraLabels
	}

	// Scrape twice: the first scrape builds the cache, the second reads it.
	for scrape := 1; scrape <= 2; scrape++ {
		mfs, err := reg.Gather()
		require.NoError(t, err, "scrape %d", scrape)
		checked := 0
		for _, mf := range mfs {
			extra, ok := wantExtra[mf.GetName()]
			if !ok || len(extra) == 0 {
				continue
			}
			for _, m := range mf.GetMetric() {
				names := make(map[string]struct{}, len(m.GetLabel()))
				for _, lp := range m.GetLabel() {
					names[lp.GetName()] = struct{}{}
				}
				_, hasFooBar := names["foo_bar"]
				assert.Truef(t, hasFooBar, "scrape %d: %s lost the collided base label; labels %v", scrape, mf.GetName(), m.GetLabel())
				for _, e := range extra {
					_, has := names[e]
					assert.Truef(t, has, "scrape %d: %s is missing its extra label %q; labels %v", scrape, mf.GetName(), e, m.GetLabel())
				}
				checked++
			}
		}
		require.NotZero(t, checked, "scrape %d emitted no metrics with extra labels", scrape)
	}
}

// TestDescsForKeepsOnlyTheLatestLayout checks the cache is bounded: a new
// label layout replaces the previous one instead of accumulating next to it.
func TestDescsForKeepsOnlyTheLatestLayout(t *testing.T) {
	c := NewPrometheusCollector(testSubcontainersInfoProvider{}, DefaultContainerLabels, container.AllMetrics, now, info.RequestOptions{})

	a := c.descsFor([]string{"id", "name"})
	require.Same(t, a, c.descsFor([]string{"id", "name"}), "same layout must hit the cache")

	b := c.descsFor([]string{"id", "name", "zone"})
	require.NotSame(t, a, b)
	assert.Same(t, b, c.descCache.Load(), "the new layout must be the only cached entry")

	a2 := c.descsFor([]string{"id", "name"})
	assert.NotSame(t, a, a2, "an evicted layout is rebuilt, not retained")
	assert.Same(t, a2, c.descCache.Load())
}
