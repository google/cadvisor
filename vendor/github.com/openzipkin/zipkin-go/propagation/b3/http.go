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
	"net/http"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation"
)

type InjectOption func(opts *InjectOptions)

type InjectOptions struct {
	shouldInjectSingleHeader bool
	shouldInjectMultiHeader  bool
}

// WithSingleAndMultiHeader allows to include both single and multiple
// headers in the context injection
func WithSingleAndMultiHeader() InjectOption {
	return func(opts *InjectOptions) {
		opts.shouldInjectSingleHeader = true
		opts.shouldInjectMultiHeader = true
	}
}

// WithSingleHeaderOnly allows to include only single header in the context
// injection
func WithSingleHeaderOnly() InjectOption {
	return func(opts *InjectOptions) {
		opts.shouldInjectSingleHeader = true
		opts.shouldInjectMultiHeader = false
	}
}

// ExtractHTTP will extract a span.Context from the HTTP Request if found in
// B3 header format.
func ExtractHTTP(r *http.Request) propagation.Extractor {
	return func() (*model.SpanContext, error) {
		var (
			traceIDHeader      = r.Header.Get(TraceID)
			spanIDHeader       = r.Header.Get(SpanID)
			parentSpanIDHeader = r.Header.Get(ParentSpanID)
			sampledHeader      = r.Header.Get(Sampled)
			flagsHeader        = r.Header.Get(Flags)
			singleHeader       = r.Header.Get(Context)
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
}

// InjectHTTP will inject a span.Context into a HTTP Request
func InjectHTTP(r *http.Request, opts ...InjectOption) propagation.Injector {
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
				r.Header.Set(Flags, "1")
			} else if sc.Sampled != nil {
				// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
				// so don't also send "X-B3-Sampled: 1".
				if *sc.Sampled {
					r.Header.Set(Sampled, "1")
				} else {
					r.Header.Set(Sampled, "0")
				}
			}

			if !sc.TraceID.Empty() && sc.ID > 0 {
				r.Header.Set(TraceID, sc.TraceID.String())
				r.Header.Set(SpanID, sc.ID.String())
				if sc.ParentID != nil {
					r.Header.Set(ParentSpanID, sc.ParentID.String())
				}
			}
		}

		if options.shouldInjectSingleHeader {
			r.Header.Set(Context, BuildSingleHeader(sc))
		}

		return nil
	}
}
