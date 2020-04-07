// +build !libpfm !cgo

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

package perf

import (
	"github.com/google/cadvisor/stats"

	"k8s.io/klog"
)

type noopManager struct {
	stats.NoopSetupDestroy
}

func (m *noopManager) GetCollector(cgroup string) (stats.Collector, error) {
	return &noopCollector{}, nil
}

func NewManager(configFile string, numCores int) (stats.Manager, error) {
	klog.V(1).Info("cAdvisor is build without cgo and/or libpfm support. Perf event counters are not available.")
	return &noopManager{}, nil
}