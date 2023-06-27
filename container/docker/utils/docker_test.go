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

package utils

import "testing"

func TestIsContainerName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "/system.slice/var-lib-docker-overlay-9f086b233ab7c786bf8b40b164680b658a8f00e94323868e288d6ce20bc92193-merged.mount",
			expected: false,
		},
		{
			name:     "/system.slice/docker-72e5a5ff5eef3c4222a6551b992b9360a99122f77d2229783f0ee0946dfd800e.scope",
			expected: true,
		},
	}
	for _, test := range tests {
		if actual := IsContainerName(test.name); actual != test.expected {
			t.Errorf("%s: expected: %v, actual: %v", test.name, test.expected, actual)
		}
	}
}
