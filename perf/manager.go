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

// Manager of perf events for containers.
package perf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/google/cadvisor/stats"

	"k8s.io/klog"
)

type manager struct {
	events Events
	stats.NoopSetupDestroy
}

func NewManager(configFile string) (stats.Manager, error) {
	if configFile == "" {
		return nil, nil
	}
	configContents, err := ioutil.ReadFile(configFile)
	if err != nil {
		klog.Errorf("Unable to read configuration file %q: %q", configFile, err)
		return nil, fmt.Errorf("Unable to read configuration file %q: %q", configFile, err)
	}
	config := Events{}
	err = json.Unmarshal(configContents, &config)
	if err != nil {
		klog.Errorf("Unable to load perf events configuration from %q: %q", configFile, err)
		return nil, fmt.Errorf("unable to load perf events cofiguration from %q: %q", configFile, err)
	}

	return &manager{events: config}, nil
}

func (m *manager) GetCollector(cgroupPath string) (stats.Collector, error) {
	return NewCollector(cgroupPath, m.events), nil
}
