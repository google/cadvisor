package cache

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// export var=v1 && go test -count 5 -benchtime 100x -run '^$' -bench . -memprofile=${var}.mem.pprof -cpuprofile=${var}.cpu.pprof > ${var}.txt
func BenchmarkCachedTGatherer_Insert(b *testing.B) {
	defProfRate := runtime.MemProfileRate
	runtime.MemProfileRate = 0

	c := NewCachedTGatherer()

	// Generate larger metric payload.
	inserts := make([]Metric, 0, 1e6)

	// 1000 metrics in 1000 families.
	for i := 0; i < 1e3; i++ {
		for j := 0; j < 1e3; j++ {
			inserts = append(inserts, Metric{
				FQName:      fmt.Sprintf("realistic_longer_name_%d", i),
				LabelNames:  []string{"realistic_label_name1", "realistic_label_name2", "realistic_label_name3"},
				LabelValues: []string{"realistic_label_value1", "realistic_label_value2", fmt.Sprintf("realistic_label_value3_%d", j)},
				Help:        "help string is usually quite large, so let's make it a bit realistic.",
				ValueType:   prometheus.GaugeValue,
				Value:       float64(j),
			})
		}
	}

	// Initial update.
	stop := c.StartUpdateSession()
	for _, ins := range inserts {
		if err := c.InsertInPlace(ins); err != nil {
			b.Error("update:", err)
		}
	}
	stop()

	if len(c.metricFamiliesByName) != 1e3 || len(c.metricFamiliesByName["realistic_longer_name_123"].metricsByHash) != 1e3 {
		// Ensure we did not generate duplicates.
		panic("generated data set gave wrong numbers")
	}

	// We care about memory profile starting from here.
	runtime.MemProfileRate = defProfRate

	b.Run("Update of one element in empty cache", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			c := NewCachedTGatherer()
			stop := c.StartUpdateSession()
			if err := c.InsertInPlace(inserts[0]); err != nil {
				b.Error("update:", err)
			}
			stop()
		}
	})

	b.Run("Update of all elements with reset", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			stop := c.StartUpdateSession()
			for _, ins := range inserts {
				if err := c.InsertInPlace(ins); err != nil {
					b.Error("update:", err)
				}
			}
			stop()
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
