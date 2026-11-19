// Copyright 2026 Google Inc. All Rights Reserved.
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
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/google/cadvisor/lib/container"
)

func TestPrometheusPluginCollector(t *testing.T) {
	c := &PrometheusPluginCollector{statuses: func() map[string]container.PluginStatus {
		return map[string]container.PluginStatus{
			"docker": {Registered: true, Attempts: 1},
			"podman": {Attempts: 3, LastError: errors.New("connection refused")},
		}
	}}

	want := `
# HELP cadvisor_container_plugin_registered Whether the container runtime plugin has registered its factory (1) or not (0).
# TYPE cadvisor_container_plugin_registered gauge
cadvisor_container_plugin_registered{plugin="docker"} 1
cadvisor_container_plugin_registered{plugin="podman"} 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(want)); err != nil {
		t.Error(err)
	}
}
