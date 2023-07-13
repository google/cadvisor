// Copyright 2022 Google Inc. All Rights Reserved.
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

package podman

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateResponse(t *testing.T) {
	for _, tc := range []struct {
		response *http.Response
		err      error
		expected string
	}{
		{
			response: nil,
			err:      nil,
			expected: "response not present",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
			err:      errors.New("some error"),
			expected: "item not found: some error",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusNotImplemented,
			},
			err:      errors.New("some error"),
			expected: "query not implemented: some error",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusOK,
			},
			err:      errors.New("some error"),
			expected: "some error",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusOK,
			},
			err:      nil,
			expected: "",
		},
	} {
		err := validateResponse(tc.err, tc.response)
		if tc.expected != "" {
			assert.EqualError(t, err, tc.expected)
		} else {
			assert.NoError(t, err)
		}
	}
}
