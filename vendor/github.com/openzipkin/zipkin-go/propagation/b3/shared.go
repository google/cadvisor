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

import "errors"

// Common Header Extraction / Injection errors
var (
	ErrInvalidSampledByte        = errors.New("invalid B3 Sampled found")
	ErrInvalidSampledHeader      = errors.New("invalid B3 Sampled header found")
	ErrInvalidFlagsHeader        = errors.New("invalid B3 Flags header found")
	ErrInvalidTraceIDHeader      = errors.New("invalid B3 TraceID header found")
	ErrInvalidSpanIDHeader       = errors.New("invalid B3 SpanID header found")
	ErrInvalidParentSpanIDHeader = errors.New("invalid B3 ParentSpanID header found")
	ErrInvalidScope              = errors.New("require either both TraceID and SpanID or none")
	ErrInvalidScopeParent        = errors.New("ParentSpanID requires both TraceID and SpanID to be available")
	ErrInvalidScopeParentSingle  = errors.New("ParentSpanID requires TraceID, SpanID and Sampled to be available")
	ErrEmptyContext              = errors.New("empty request context")
	ErrInvalidTraceIDValue       = errors.New("invalid B3 TraceID value found")
	ErrInvalidSpanIDValue        = errors.New("invalid B3 SpanID value found")
	ErrInvalidParentSpanIDValue  = errors.New("invalid B3 ParentSpanID value found")
)

// Default B3 Header keys
const (
	TraceID      = "x-b3-traceid"
	SpanID       = "x-b3-spanid"
	ParentSpanID = "x-b3-parentspanid"
	Sampled      = "x-b3-sampled"
	Flags        = "x-b3-flags"
	Context      = "b3"
)
