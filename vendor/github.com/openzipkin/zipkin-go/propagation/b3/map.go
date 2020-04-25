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

package b3

import (
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
)

// Map allows serialization and deserialization of SpanContext into a standard Go map.
type Map map[string]string

// Extract implements Extractor
func (m *Map) Extract() (*model.SpanContext, error) {
	var (
		traceIDHeader      = (*m)[TraceID]
		spanIDHeader       = (*m)[SpanID]
		parentSpanIDHeader = (*m)[ParentSpanID]
		sampledHeader      = (*m)[Sampled]
		flagsHeader        = (*m)[Flags]
		singleHeader       = (*m)[Context]
	)

	var (
		sc   *model.SpanContext
		sErr error
		mErr error
	)
	if singleHeader != "" {
		sc, sErr = ParseSingleHeader(singleHeader)
		if sErr == nil {
			return sc, nil
		}
	}

	sc, mErr = ParseHeaders(
		traceIDHeader, spanIDHeader, parentSpanIDHeader,
		sampledHeader, flagsHeader,
	)

	if mErr != nil && sErr != nil {
		return nil, sErr
	}

	return sc, mErr

}

// Inject implements Injector
func (m *Map) Inject(opts ...InjectOption) propagation.Injector {
	options := InjectOptions{shouldInjectMultiHeader: true}
	for _, opt := range opts {
		opt(&options)
	}

	return func(sc model.SpanContext) error {
		if (model.SpanContext{}) == sc {
			return ErrEmptyContext
		}

		if options.shouldInjectMultiHeader {
			if sc.Debug {
				(*m)[Flags] = "1"
			} else if sc.Sampled != nil {
				// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
				// so don't also send "X-B3-Sampled: 1".
				if *sc.Sampled {
					(*m)[Sampled] = "1"
				} else {
					(*m)[Sampled] = "0"
				}
			}

			if !sc.TraceID.Empty() && sc.ID > 0 {
				(*m)[TraceID] = sc.TraceID.String()
				(*m)[SpanID] = sc.ID.String()
				if sc.ParentID != nil {
					(*m)[ParentSpanID] = sc.ParentID.String()
				}
			}
		}

		if options.shouldInjectSingleHeader {
			(*m)[Context] = BuildSingleHeader(sc)
		}

		return nil
	}
}
