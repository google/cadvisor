package metrics

import (
	"fmt"
	"testing"

	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
	v2 "github.com/google/cadvisor/info/v2"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

var mfsTmp []*dto.MetricFamily

type benchSubcontainersInfoProvider struct {
	testSubcontainersInfoProvider

	containers map[string]*info.ContainerInfo
}

func (p benchSubcontainersInfoProvider) GetRequestedContainersInfo(string, v2.RequestOptions) (map[string]*info.ContainerInfo, error) {
	return p.containers, nil
}

func BenchmarkPrometheusCollector_Collect(b *testing.B) {
	b.ReportAllocs()

	// Generate bigger dataset for realistic situation (we can assume node has ~20 pods), which produces:
	// * 84 metric families, with 5567 series in total.
	// * 1081124 bytes (~1MB) of DTO struct.
	// Reproducing https://github.com/kubernetes/kubernetes/issues/104459.
	infoProvider := benchSubcontainersInfoProvider{containers: make(map[string]*info.ContainerInfo, 20)}
	for i:=0; i<=20; i++ {
		name := fmt.Sprintf("container-%v", i)
		infoProvider.containers[name] = genContainerInfo(name)
	}

	reg := prometheus.NewBlockingRegistry()
	c := NewPrometheusCollector(infoProvider, func(container *info.ContainerInfo) map[string]string {
		s := DefaultContainerLabels(container)
		s["zone.name"] = "hello"
		return s
	}, container.AllMetrics, now)
	reg.MustRegisterRaw(c)

	c.SetOpts(v2.RequestOptions{})
	var err error
	var done func()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mfsTmp, done, err = reg.Gather()
		require.NoError(b, err)
		done()
	}
}
