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
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
	v2 "github.com/google/cadvisor/info/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	clock "k8s.io/utils/clock/testing"
)

var now = clock.NewFakeClock(time.Unix(1395066363, 0))

func TestPrometheusCollector(t *testing.T) {
	c := NewPrometheusCollector(testSubcontainersInfoProvider{}, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["zone.name"] = "hello"
		return s
	}, container.AllMetrics, now, v2.RequestOptions{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics")
}

func TestPrometheusCollectorWithWhiteList(t *testing.T) {
	c := NewPrometheusCollector(testSubcontainersInfoProvider{}, func(container *info.ContainerInfo) map[string]string {
		whitelistedLabels := []string{
			"no_one_match",
		}
		containerLabelFunc := BaseContainerLabels(whitelistedLabels)
		s := containerLabelFunc(container)
		s["zone.name"] = "hello"
		return s
	}, container.AllMetrics, now, v2.RequestOptions{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics_whitelist_filtered")
}

func TestPrometheusCollectorWithPerfAggregated(t *testing.T) {
	metrics := container.MetricSet{
		container.PerfMetrics: struct{}{},
	}
	c := NewPrometheusCollector(testSubcontainersInfoProvider{}, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["zone.name"] = "hello"
		return s
	}, metrics, now, v2.RequestOptions{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics_perf_aggregated")
}

func testPrometheusCollector(t *testing.T, gatherer prometheus.Gatherer, metricsFile string) {
	wantMetrics, err := os.Open(metricsFile)
	if err != nil {
		t.Fatalf("unable to read input test file %s", metricsFile)
	}

	err = testutil.GatherAndCompare(gatherer, wantMetrics)
	if err != nil {
		t.Fatalf("Metric comparison failed: %s", err)
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
	}, container.AllMetrics, now, v2.RequestOptions{})
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics_failure")

	provider.shouldFail = false

	testPrometheusCollector(t, reg, "testdata/prometheus_metrics")
}

func TestNewPrometheusCollectorWithPerf(t *testing.T) {
	c := NewPrometheusCollector(&mockInfoProvider{}, mockLabelFunc, container.MetricSet{container.PerfMetrics: struct{}{}}, now, v2.RequestOptions{})
	assert.Len(t, c.containerMetrics, 5)
	names := []string{}
	for _, m := range c.containerMetrics {
		names = append(names, m.name)
	}
	assert.Contains(t, names, "container_last_seen")
	assert.Contains(t, names, "container_perf_events_total")
	assert.Contains(t, names, "container_perf_events_scaling_ratio")
	assert.Contains(t, names, "container_perf_uncore_events_total")
	assert.Contains(t, names, "container_perf_uncore_events_scaling_ratio")
}

func TestNewPrometheusCollectorWithRequestOptions(t *testing.T) {
	p := mockInfoProvider{
		containerInfo: map[string]*info.ContainerInfo{},
	}
	opts := v2.RequestOptions{
		IdType: "docker",
	}
	c := NewPrometheusCollector(&p, mockLabelFunc, container.AllMetrics, now, opts)
	ch := make(chan prometheus.Metric, 10)
	c.Collect(ch)
	assert.Equal(t, p.options, opts)
}

type mockInfoProvider struct {
	containerInfo map[string]*info.ContainerInfo
	options       v2.RequestOptions
}

func (m *mockInfoProvider) GetRequestedContainersInfo(containerName string, options v2.RequestOptions) (map[string]*info.ContainerInfo, error) {
	m.options = options
	return m.containerInfo, nil
}

func (m *mockInfoProvider) GetVersionInfo() (*info.VersionInfo, error) {
	return &info.VersionInfo{}, nil
}

func (m *mockInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	return nil, errors.New("not supported")
}

func mockLabelFunc(*info.ContainerInfo) map[string]string {
	return map[string]string{}
}

func TestGetPerCpuCorePerfEvents(t *testing.T) {
	containerStats := &info.ContainerStats{
		Timestamp: time.Unix(1395066367, 0),
		PerfStats: []info.PerfStat{
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 1.0,
					Value:        123,
					Name:         "instructions",
				},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.5,
					Value:        456,
					Name:         "instructions",
				},
				Cpu: 1,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.7,
					Value:        321,
					Name:         "instructions_retired"},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.3,
					Value:        789,
					Name:         "instructions_retired"},
				Cpu: 1,
			},
		},
	}
	metricVals := getPerCPUCorePerfEvents(containerStats)
	assert.Equal(t, 4, len(metricVals))
	values := []float64{}
	for _, metric := range metricVals {
		values = append(values, metric.value)
	}
	assert.Contains(t, values, 123.0)
	assert.Contains(t, values, 456.0)
	assert.Contains(t, values, 321.0)
	assert.Contains(t, values, 789.0)
}

func TestGetPerCpuCoreScalingRatio(t *testing.T) {
	containerStats := &info.ContainerStats{
		Timestamp: time.Unix(1395066367, 0),
		PerfStats: []info.PerfStat{
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 1.0,
					Value:        123,
					Name:         "instructions"},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.5,
					Value:        456,
					Name:         "instructions"},
				Cpu: 1,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.7,
					Value:        321,
					Name:         "instructions_retired"},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.3,
					Value:        789,
					Name:         "instructions_retired"},
				Cpu: 1,
			},
		},
	}
	metricVals := getPerCPUCoreScalingRatio(containerStats)
	assert.Equal(t, 4, len(metricVals))
	values := []float64{}
	for _, metric := range metricVals {
		values = append(values, metric.value)
	}
	assert.Contains(t, values, 1.0)
	assert.Contains(t, values, 0.5)
	assert.Contains(t, values, 0.7)
	assert.Contains(t, values, 0.3)
}

func TestGetAggCorePerfEvents(t *testing.T) {
	containerStats := &info.ContainerStats{
		Timestamp: time.Unix(1395066367, 0),
		PerfStats: []info.PerfStat{
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 1.0,
					Value:        123,
					Name:         "instructions"},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.5,
					Value:        456,
					Name:         "instructions"},
				Cpu: 1,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.7,
					Value:        321,
					Name:         "instructions_retired"},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.3,
					Value:        789,
					Name:         "instructions_retired"},
				Cpu: 1,
			},
		},
	}
	metricVals := getAggregatedCorePerfEvents(containerStats)
	assert.Equal(t, 2, len(metricVals))
	values := []float64{}
	for _, metric := range metricVals {
		values = append(values, metric.value)
	}
	assert.Contains(t, values, 579.0)
	assert.Contains(t, values, 1110.0)
}

func TestGetMinCoreScalingRatio(t *testing.T) {
	containerStats := &info.ContainerStats{
		Timestamp: time.Unix(1395066367, 0),
		PerfStats: []info.PerfStat{
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 1.0,
					Value:        123,
					Name:         "instructions"},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.5,
					Value:        456,
					Name:         "instructions"},
				Cpu: 1,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.7,
					Value:        321,
					Name:         "instructions_retired"},
				Cpu: 0,
			},
			{
				PerfValue: info.PerfValue{
					ScalingRatio: 0.3,
					Value:        789,
					Name:         "instructions_retired"},
				Cpu: 1,
			},
		},
	}
	metricVals := getMinCoreScalingRatio(containerStats)
	assert.Equal(t, 2, len(metricVals))
	values := []float64{}
	for _, metric := range metricVals {
		values = append(values, metric.value)
	}
	assert.Contains(t, values, 0.5)
	assert.Contains(t, values, 0.3)
}

func BenchmarkPrometheusCollector(b *testing.B) {
	p := mockInfoProvider{
		containerInfo: map[string]*info.ContainerInfo{
			"8cb3438991e90a4b415fd7920bd8b67f59e90c62ae82b89c081ac320f31cf427": {
				ContainerReference: info.ContainerReference{
					Id:        "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod3d0df1aa_a48c_4cff_88be_9222491bdcb7.slice/cri-containerd-8cb3438991e90a4b415fd7920bd8b67f59e90c62ae82b89c081ac320f31cf427.scope",
					Name:      "alertmanager",
					Namespace: "observability",
				},
				Spec: info.ContainerSpec{
					Image: "quay.io/prometheus/alertmanager:v0.26.0",
				},
				Stats: []*info.ContainerStats{{
					Timestamp: time.Now(),
					Cpu: info.CpuStats{
						Usage: info.CpuUsage{
							Total:  1000000000,
							User:   100000000,
							System: 900000000,
							PerCpu: []uint64{100000000, 900000000},
						},
						CFS: info.CpuCFS{
							Periods:          100,
							ThrottledPeriods: 10,
							ThrottledTime:    100000000,
						},
						Schedstat: info.CpuSchedstat{
							RunTime:      100000000,
							RunqueueTime: 100000000,
							RunPeriods:   100,
						},
						LoadAverage: 5,
					},
				}},
			}},
	}
	opts := v2.RequestOptions{
		IdType: "docker",
	}

	labels := map[string]string{
		"container": "alertmanager",
		"id":        "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod3d0df1aa_a48c_4cff_88be_9222491bdcb7.slice/cri-containerd-8cb3438991e90a4b415fd7920bd8b67f59e90c62ae82b89c081ac320f31cf427.scope",
		"image":     "quay.io/prometheus/alertmanager:v0.26.0",
		"name":      "8cb3438991e90a4b415fd7920bd8b67f59e90c62ae82b89c081ac320f31cf427",
		"namespace": "observability",
		"pod":       "alertmanager-main-1",
	}

	labelFunc := func(*info.ContainerInfo) map[string]string {
		return labels
	}

	c := NewPrometheusCollector(&p, labelFunc, container.AllMetrics, now, opts)
	ch := make(chan prometheus.Metric)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-ch:
				_ = m
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Collect(ch)
	}
}
