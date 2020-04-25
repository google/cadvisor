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
	"strconv"
	"strings"

	"github.com/openzipkin/zipkin-go/model"
)

// ParseHeaders takes values found from B3 Headers and tries to reconstruct a
// SpanContext.
func ParseHeaders(
	hdrTraceID, hdrSpanID, hdrParentSpanID, hdrSampled, hdrFlags string,
) (*model.SpanContext, error) {
	var (
		err           error
		spanID        uint64
		requiredCount int
		sc            = &model.SpanContext{}
	)

	// correct values for an existing sampled header are "0" and "1".
	// For legacy support and  being lenient to other tracing implementations we
	// allow "true" and "false" as inputs for interop purposes.
	switch strings.ToLower(hdrSampled) {
	case "0", "false":
		sampled := false
		sc.Sampled = &sampled
	case "1", "true":
		sampled := true
		sc.Sampled = &sampled
	case "":
		// sc.Sampled = nil
	default:
		return nil, ErrInvalidSampledHeader
	}

	// The only accepted value for Flags is "1". This will set Debug to true. All
	// other values and omission of header will be ignored.
	if hdrFlags == "1" {
		sc.Debug = true
		sc.Sampled = nil
	}

	if hdrTraceID != "" {
		requiredCount++
		if sc.TraceID, err = model.TraceIDFromHex(hdrTraceID); err != nil {
			return nil, ErrInvalidTraceIDHeader
		}
	}

	if hdrSpanID != "" {
		requiredCount++
		if spanID, err = strconv.ParseUint(hdrSpanID, 16, 64); err != nil {
			return nil, ErrInvalidSpanIDHeader
		}
		sc.ID = model.ID(spanID)
	}

	if requiredCount != 0 && requiredCount != 2 {
		return nil, ErrInvalidScope
	}

	if hdrParentSpanID != "" {
		if requiredCount == 0 {
			return nil, ErrInvalidScopeParent
		}
		if spanID, err = strconv.ParseUint(hdrParentSpanID, 16, 64); err != nil {
			return nil, ErrInvalidParentSpanIDHeader
		}
		parentSpanID := model.ID(spanID)
		sc.ParentID = &parentSpanID
	}

	return sc, nil
}

// ParseSingleHeader takes values found from B3 Single Header and tries to reconstruct a
// SpanContext.
func ParseSingleHeader(contextHeader string) (*model.SpanContext, error) {
	if contextHeader == "" {
		return nil, ErrEmptyContext
	}

	var (
		sc       = model.SpanContext{}
		sampling string
	)

	headerLen := len(contextHeader)

	if headerLen == 1 {
		sampling = contextHeader
	} else if headerLen == 16 || headerLen == 32 {
		return nil, ErrInvalidScope
	} else if headerLen >= 16+16+1 {
		var high, low uint64
		pos := 0
		if string(contextHeader[16]) != "-" {
			// traceID must be 128 bits
			var err error
			high, err = strconv.ParseUint(contextHeader[0:16], 16, 64)
			if err != nil {
				return nil, ErrInvalidTraceIDValue
			}
			pos = 16
		}

		low, err := strconv.ParseUint(contextHeader[pos+1:pos+16], 16, 64)
		if err != nil {
			return nil, ErrInvalidTraceIDValue
		}

		sc.TraceID = model.TraceID{High: high, Low: low}

		rawID, err := strconv.ParseUint(contextHeader[pos+16+1:pos+16+1+16], 16, 64)
		if err != nil {
			return nil, ErrInvalidSpanIDValue
		}

		sc.ID = model.ID(rawID)

		if headerLen > pos+16+1+16 {
			if headerLen == pos+16+1+16+1 {
				return nil, ErrInvalidSampledByte
			}

			if headerLen == pos+16+1+16+1+1 {
				sampling = string(contextHeader[pos+16+1+16+1])
			} else if headerLen == pos+16+1+16+1+16 {
				return nil, ErrInvalidScopeParentSingle
			} else if headerLen == pos+16+1+16+1+1+1+16 {
				sampling = string(contextHeader[pos+16+1+16+1])

				rawParentID, err := strconv.ParseUint(contextHeader[pos+16+1+16+1+1+1:], 16, 64)
				if err != nil {
					return nil, ErrInvalidParentSpanIDValue
				}

				parentID := model.ID(rawParentID)
				sc.ParentID = &parentID
			} else {
				return nil, ErrInvalidParentSpanIDValue
			}
		}
	} else {
		return nil, ErrInvalidTraceIDValue
	}
	switch sampling {
	case "d":
		sc.Debug = true
	case "1":
		trueVal := true
		sc.Sampled = &trueVal
	case "0":
		falseVal := false
		sc.Sampled = &falseVal
	case "":
	default:
		return nil, ErrInvalidSampledByte
	}

	return &sc, nil
}

// BuildSingleHeader takes the values from the SpanContext and builds the B3 header
func BuildSingleHeader(sc model.SpanContext) string {
	header := []string{}
	if !sc.TraceID.Empty() && sc.ID > 0 {
		header = append(header, sc.TraceID.String(), sc.ID.String())
	}

	if sc.Debug {
		header = append(header, "d")
	} else if sc.Sampled != nil {
		if *sc.Sampled {
			header = append(header, "1")
		} else {
			header = append(header, "0")
		}
	}

	if sc.ParentID != nil {
		header = append(header, sc.ParentID.String())
	}

	return strings.Join(header, "-")
}
