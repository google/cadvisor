// Copyright 2020 Google Inc. All Rights Reserved.
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

package opentelemetry

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/cadvisor/version"
	"os"

	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/propagation"
	apiTrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/plugin/othttp"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"
	"net/http"
	"strconv"
	"strings"
)

const traceServiceName = "cadvisor"

var traceProbability = flag.Float64("trace_probability", 1, "used for sampler (0=never, 1=always and probability is 0.1 to 0.99)")

func InitTrace(traceEndpoint, collectorCert string) *otlp.Exporter {
	klog.Infof("write telemetry to %s", traceEndpoint)
	exporterOptions := []otlp.ExporterOption{otlp.WithAddress(traceEndpoint)}

	if collectorCert != "" {
		creds, err := credentials.NewClientTLSFromFile(collectorCert, "")
		if err != nil {
			klog.Fatal(err)
		}
		exporterOptions = append(exporterOptions, otlp.WithTLSCredentials(creds))
	} else {
		exporterOptions = append(exporterOptions, otlp.WithInsecure())
	}

	exporter, err := otlp.NewExporter(exporterOptions...)
	if err != nil {
		klog.Fatal(err)
	}

	sampler := getTraceSamplerProbability() //*traceSampler, *traceSamplerParam)
	providerOpts := []trace.ProviderOption{
		trace.WithBatcher(exporter),
		trace.WithConfig(trace.Config{DefaultSampler: sampler}),
		trace.WithResourceAttributes(getTraceTags(traceTags)...),
	}

	traceProvider, err := trace.NewProvider(providerOpts...)
	if err != nil {
		klog.Fatal(err)
	}
	global.SetTraceProvider(traceProvider)

	apiB3Single, apiB3, traceCtx, corrCtx := apiTrace.B3{SingleHeader: false}, apiTrace.B3{SingleHeader: false}, apiTrace.DefaultHTTPPropagator(), correlation.DefaultHTTPPropagator()
	extractor := extractorFilter{[]propagation.HTTPExtractor{apiB3Single, apiB3, traceCtx, corrCtx}}
	global.SetPropagators(propagation.New(propagation.WithInjectors(apiB3, traceCtx, corrCtx), propagation.WithExtractors(extractor)))
	return exporter
}

type extractorFilter struct {
	extractors []propagation.HTTPExtractor
}

func (e extractorFilter) Extract(orgCtx context.Context, supplier propagation.HTTPSupplier) context.Context {
	for _, extractor := range e.extractors {
		if newCtx := extractor.Extract(orgCtx, supplier); apiTrace.RemoteSpanContextFromContext(newCtx).IsValid() {
			return newCtx
		}
	}
	return orgCtx
}

func getTraceSamplerProbability() trace.Sampler {
	probability := *traceProbability
	if probability < 0 && probability > 1 {
		klog.Errorf("invalid probability value %f, probability is allowed between 0.0 and 1.0", probability)
	}
	if probability == 0 {
		klog.Info("set never sampler")
		return trace.NeverSample()
	} else if probability == 1 {
		klog.Info("set always sampler")
		return trace.AlwaysSample()
	} else {
		klog.Info("set probability sampler")
		return trace.ProbabilitySampler(probability)
	}
}

func WrapperServerMux(mux *http.ServeMux) *wrapperMux {
	return &wrapperMux{mux: mux}
}

type wrapperMux struct {
	mux *http.ServeMux
}

func (m *wrapperMux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	m.mux.ServeHTTP(writer, request)
}

func (m *wrapperMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	wH := wrapperHandler{callback: handler}
	m.mux.Handle(pattern, othttp.NewHandler(&wH, pattern))
}

func (m *wrapperMux) Handler(r *http.Request) (http.Handler, string) {
	h, p := m.mux.Handler(r)
	return othttp.NewHandler(h, p), p
}

func (m *wrapperMux) Handle(pattern string, handler http.Handler) {
	m.mux.Handle(pattern, othttp.NewHandler(handler, pattern))
}

type wrapperHandler struct {
	callback func(http.ResponseWriter, *http.Request)
}

func (w *wrapperHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if w.callback != nil {
		w.callback(writer, request)
	} else {
		panic(fmt.Errorf("not yet implemented"))
	}
}

func getTraceTags(tags []string) []kv.KeyValue {
	keyValue := []kv.KeyValue{
		kv.String("service.name", traceServiceName),
		kv.String("service.version", fmt.Sprintf("semver: %s git:%s", version.Info["version"], version.Info["revision"])),
	}

	if hostname, err := os.Hostname(); err != nil {
		keyValue = append(keyValue, kv.String("host.hostname", hostname))
	}
	for _, tag := range tags {
		if strings.Contains(tag, "=") {
			continue
		}
		parts := strings.Split(tag, "=")
		tKey, val := parts[0], parts[1]
		if b, e := strconv.ParseBool(val); e != nil {
			keyValue = append(keyValue, kv.Bool(tKey, b))
		} else if f32, e := strconv.ParseFloat(val, 32); e != nil {
			keyValue = append(keyValue, kv.Float32(tKey, float32(f32)))
		} else if f64, e := strconv.ParseFloat(val, 64); e != nil {
			keyValue = append(keyValue, kv.Float64(tKey, f64))
		} else if ui32, e := strconv.ParseUint(val, 0, 32); e != nil {
			keyValue = append(keyValue, kv.Uint32(tKey, uint32(ui32)))
		} else if ui64, e := strconv.ParseUint(val, 0, 64); e != nil {
			keyValue = append(keyValue, kv.Uint64(tKey, ui64))
		} else if ui00, e := strconv.ParseUint(val, 0, 0); e != nil {
			keyValue = append(keyValue, kv.Uint(tKey, uint(ui00)))
		} else if i32, e := strconv.ParseInt(val, 0, 32); e != nil {
			keyValue = append(keyValue, kv.Int32(tKey, int32(i32)))
		} else if i64, e := strconv.ParseInt(val, 0, 64); e != nil {
			keyValue = append(keyValue, kv.Int64(tKey, i64))
		} else if i00, e := strconv.ParseInt(val, 0, 0); e != nil {
			keyValue = append(keyValue, kv.Int(tKey, int(i00)))
		} else {
			keyValue = append(keyValue, kv.String(tKey, val))
		}
	}

	klog.Infof("add tracings tags: %s", ToString(keyValue))
	return keyValue
}

func ToString(keyValues []kv.KeyValue) string {
	result := ""
	for _, keyValue := range keyValues {
		if result != "" {
			result += ", "
		}

		key := keyValue.Key
		value := keyValue.Value.Emit()
		result += fmt.Sprintf("core.KeyValue{Key: %s, Value: %s}", key, value)
	}

	return fmt.Sprintf("[]core.KeyValue{ %s }", result)
}
