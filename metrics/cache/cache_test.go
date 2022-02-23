// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestCachedTGatherer(t *testing.T) {
	c := NewCachedTGatherer()
	mfs, done, err := c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()
	if got := mfsToString(mfs); got != "" {
		t.Error("unexpected metric family", got)
	}

	if err := c.Update(false, []Insert{
		{
			Key:       Key{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}},
			Help:      "help a",
			ValueType: prometheus.GaugeValue,
			Value:     1,
		},
		{
			Key:       Key{FQName: "b", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}},
			Help:      "help b",
			ValueType: prometheus.GaugeValue,
			Value:     1,
		},
		{
			Key:       Key{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc2"}},
			Help:      "help a2",
			ValueType: prometheus.CounterValue,
			Value:     2,
		},
		{
			Key:       Key{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc3"}},
			Help:      "help a2",
			ValueType: prometheus.CounterValue,
			Value:     2,
		},
	}, []Key{
		{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc3"}}, // Does not make much sense, but deletion works as expected.
	}); err != nil {
		t.Error("update:", err)
	}

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	const expected = "name:\"a\" help:\"help a2\" type:COUNTER metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > " +
		"gauge:<value:1 > > metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc2\" > counter:<value:2 > > ,name:\"b\" help:\"help b\" " +
		"type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:1 > > "
	if got := mfsToString(mfs); got != expected {
		t.Error("unexpected metric family, got", got)
	}

	// Update with exactly same insertion should have the same effect.
	if err := c.Update(false, []Insert{
		{
			Key:       Key{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}},
			Help:      "help a",
			ValueType: prometheus.GaugeValue,
			Value:     1,
		},
		{
			Key:       Key{FQName: "b", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}},
			Help:      "help b",
			ValueType: prometheus.GaugeValue,
			Value:     1,
		},
		{
			Key:       Key{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc2"}},
			Help:      "help a2",
			ValueType: prometheus.CounterValue,
			Value:     2,
		},
		{
			Key:       Key{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc3"}},
			Help:      "help a2",
			ValueType: prometheus.CounterValue,
			Value:     2,
		},
	}, []Key{
		{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc3"}}, // Does not make much sense, but deletion works as expected.
	}); err != nil {
		t.Error("update:", err)
	}

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	if got := mfsToString(mfs); got != expected {
		t.Error("unexpected metric family, got", got)
	}

	// Update one element.
	if err := c.Update(false, []Insert{
		{
			Key:       Key{FQName: "a", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}},
			Help:      "help a12321",
			ValueType: prometheus.CounterValue,
			Value:     9999,
		},
	}, nil); err != nil {
		t.Error("update:", err)
	}

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	if got := mfsToString(mfs); got != "name:\"a\" help:\"help a12321\" type:COUNTER metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" >"+
		" counter:<value:9999 > > metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc2\" > counter:<value:2 > > ,name:\"b\" help:\"help b\" "+
		"type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:1 > > " {
		t.Error("unexpected metric family, got", got)
	}

	// Rebuild cache and insert only 2 elements.
	if err := c.Update(true, []Insert{
		{
			Key:       Key{FQName: "ax", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}},
			Help:      "help ax",
			ValueType: prometheus.GaugeValue,
			Value:     1,
		},
		{
			Key:       Key{FQName: "bx", LabelNames: []string{"b", "c"}, LabelValues: []string{"valb", "valc"}},
			Help:      "help bx",
			ValueType: prometheus.GaugeValue,
			Value:     1,
		},
	}, nil); err != nil {
		t.Error("update:", err)
	}

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()

	if got := mfsToString(mfs); got != "name:\"ax\" help:\"help ax\" type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" >"+
		" gauge:<value:1 > > ,name:\"bx\" help:\"help bx\" type:GAUGE metric:<label:<name:\"b\" value:\"valb\" > label:<name:\"c\" value:\"valc\" > gauge:<value:1 > > " {
		t.Error("unexpected metric family, got", got)
	}

	if err := c.Update(true, nil, nil); err != nil {
		t.Error("update:", err)
	}

	mfs, done, err = c.Gather()
	if err != nil {
		t.Error("gather failed:", err)
	}
	done()
	if got := mfsToString(mfs); got != "" {
		t.Error("unexpected metric family", got)
	}
}

func mfsToString(mfs []*dto.MetricFamily) string {
	ret := make([]string, 0, len(mfs))
	for _, m := range mfs {
		ret = append(ret, m.String())
	}
	return strings.Join(ret, ",")
}

// export var=v1 && go test -count 5 -benchtime 100x -run '^$' -bench . -memprofile=${var}.mem.pprof -cpuprofile=${var}.cpu.pprof > ${var}.txt
func BenchmarkCachedTGatherer_Update(b *testing.B) {
	c := NewCachedTGatherer()

	// Generate larger metric payload.
	inserts := make([]Insert, 0, 1e6)

	// 1000 metrics in 1000 families.
	for i := 0; i < 1e3; i++ {
		for j := 0; j < 1e3; j++ {
			inserts = append(inserts, Insert{
				Key: Key{
					FQName:      fmt.Sprintf("realistic_longer_name_%d", i),
					LabelNames:  []string{"realistic_label_name1", "realistic_label_name2", "realistic_label_name3"},
					LabelValues: []string{"realistic_label_value1", "realistic_label_value2", fmt.Sprintf("realistic_label_value3_%d", j)}},
				Help:      "help string is usually quite large, so let's make it a bit realistic.",
				ValueType: prometheus.GaugeValue,
				Value:     float64(j),
			})
		}
	}

	if err := c.Update(false, inserts, nil); err != nil {
		b.Error("update:", err)
	}

	if len(c.metricFamiliesByName) != 1e3 || len(c.metricFamiliesByName["realistic_longer_name_123"].metricsByHash) != 1e3 {
		// Ensure we did not generate duplicates.
		panic("generated data set gave wrong numbers")
	}

	b.Run("Update of one element without reset", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err := c.Update(false, []Insert{
				{
					Key: Key{
						FQName:      "realistic_longer_name_334",
						LabelNames:  []string{"realistic_label_name1", "realistic_label_name2", "realistic_label_name3"},
						LabelValues: []string{"realistic_label_value1", "realistic_label_value2", "realistic_label_value3_2345"}},
					Help:      "CUSTOM help string is usually quite large, so let's make it a bit realistic.",
					ValueType: prometheus.CounterValue,
					Value:     1929495,
				},
			}, nil); err != nil {
				b.Error("update:", err)
			}
		}
	})

	b.Run("Update of all elements with reset", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			if err := c.Update(true, inserts, nil); err != nil {
				b.Error("update:", err)
			}
		}
	})

	b.Run("Gather", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			mfs, done, err := c.Gather()
			done()
			if err != nil {
				b.Error("update:", err)
			}
			testMfs = mfs
		}
	})
}

var testMfs []*dto.MetricFamily
