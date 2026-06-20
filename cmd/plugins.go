// Copyright 2024 Google Inc. All Rights Reserved.
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

package main

import (
	"time"

	"github.com/google/cadvisor/cmd/internal/appmetrics"
	"github.com/google/cadvisor/cmd/internal/processlist"
	"github.com/google/cadvisor/lib/container"
	"github.com/google/cadvisor/lib/manager"
	model "github.com/google/cadvisor/lib/model"
	"github.com/google/cadvisor/lib/stats"
	"github.com/google/cadvisor/perf"
	"github.com/google/cadvisor/resctrl"
	"github.com/google/cadvisor/summary"
	"github.com/google/cadvisor/utils/cpuload"
)

// Wire the perf_event, resctrl and summary (derived-stats) implementations into
// the lean library manager's injection seams (lib/manager/plugins.go). The
// library leaves these factories nil — the kubelet uses none of them — so
// without this the full binary would silently report no perf/resctrl metrics
// and no derived stats. init() runs before manager.New, which consumes them.
func init() {
	manager.PerfManagerFactory = perf.NewManager
	manager.ResctrlManagerFactory = func(interval time.Duration, vendorID string, inHostNamespace bool) (stats.ResctrlManager, error) {
		return resctrl.NewManager(interval, vendorID, inHostNamespace)
	}
	manager.SummaryReaderFactory = func(spec model.ContainerSpec) (manager.SummaryReader, error) {
		return summary.New(spec)
	}
	manager.ProcessListProvider = processlist.List
	manager.CollectorManagerFactory = func(handler container.ContainerHandler, readFile func(string) ([]byte, error)) (manager.CollectorManager, error) {
		return appmetrics.NewManager(handler, readFile, manager.ApplicationMetricsCountLimit())
	}
	manager.CpuLoadReaderFactory = func() (manager.CpuLoadReader, error) {
		return cpuload.New()
	}
}
