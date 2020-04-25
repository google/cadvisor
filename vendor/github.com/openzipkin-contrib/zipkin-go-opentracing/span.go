// Copyright 2019 The OpenZipkin Authors
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

package zipkintracer

import (
	"fmt"
	"time"

	otobserver "github.com/opentracing-contrib/go-observer"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/openzipkin/zipkin-go"
)

// FinisherWithDuration allows to finish span with given duration
type FinisherWithDuration interface {
	FinishedWithDuration(d time.Duration)
}

type spanImpl struct {
	tracer     *tracerImpl
	zipkinSpan zipkin.Span
	startTime  time.Time
	observer   otobserver.SpanObserver
}

func (s *spanImpl) SetOperationName(operationName string) opentracing.Span {
	if s.observer != nil {
		s.observer.OnSetOperationName(operationName)
	}

	s.zipkinSpan.SetName(operationName)
	return s
}

func (s *spanImpl) SetTag(key string, value interface{}) opentracing.Span {
	if s.observer != nil {
		s.observer.OnSetTag(key, value)
	}

	if key == string(ext.SamplingPriority) {
		// there are no means for now to change the sampling decision
		// but when finishedSpanHandler is in place we could change this.
		return s
	}

	if key == string(ext.SpanKind) ||
		key == string(ext.PeerService) ||
		key == string(ext.PeerHostIPv4) ||
		key == string(ext.PeerHostIPv6) ||
		key == string(ext.PeerPort) {
		// this tags are translated into kind and remoteEndpoint which can
		// only be set on span creation
		return s
	}

	s.zipkinSpan.Tag(key, fmt.Sprint(value))
	return s
}

func (s *spanImpl) LogKV(keyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(keyValues...)
	if err != nil {
		return
	}

	for _, field := range fields {
		s.zipkinSpan.Annotate(time.Now(), field.String())
	}
}

func (s *spanImpl) LogFields(fields ...log.Field) {
	s.logFields(time.Now(), fields...)
}

func (s *spanImpl) logFields(t time.Time, fields ...log.Field) {
	for _, field := range fields {
		s.zipkinSpan.Annotate(t, field.String())
	}
}

func (s *spanImpl) LogEvent(event string) {
	s.Log(opentracing.LogData{
		Event: event,
	})
}

func (s *spanImpl) LogEventWithPayload(event string, payload interface{}) {
	s.Log(opentracing.LogData{
		Event:   event,
		Payload: payload,
	})
}

func (s *spanImpl) Log(ld opentracing.LogData) {
	if ld.Timestamp.IsZero() {
		ld.Timestamp = time.Now()
	}

	s.zipkinSpan.Annotate(ld.Timestamp, fmt.Sprintf("%s:%s", ld.Event, ld.Payload))
}

func (s *spanImpl) Finish() {
	if s.observer != nil {
		s.observer.OnFinish(opentracing.FinishOptions{})
	}

	s.zipkinSpan.Finish()
}

func (s *spanImpl) FinishWithOptions(opts opentracing.FinishOptions) {
	if s.observer != nil {
		s.observer.OnFinish(opts)
	}

	for _, lr := range opts.LogRecords {
		s.logFields(lr.Timestamp, lr.Fields...)
	}

	if !opts.FinishTime.IsZero() {
		f, ok := s.zipkinSpan.(FinisherWithDuration)
		if !ok {
			return
		}
		f.FinishedWithDuration(opts.FinishTime.Sub(s.startTime))
		return
	}

	s.Finish()
}

func (s *spanImpl) Tracer() opentracing.Tracer {
	return s.tracer
}

func (s *spanImpl) Context() opentracing.SpanContext {
	return SpanContext(s.zipkinSpan.Context())
}

func (s *spanImpl) SetBaggageItem(key, val string) opentracing.Span {
	return s
}

func (s *spanImpl) BaggageItem(key string) string {
	return ""
}
