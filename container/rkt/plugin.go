// Copyright 2019 Google Inc. All Rights Reserved.
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

package rkt

import (
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/watcher"
	"k8s.io/klog"
)

// NewPlugin returns an implementation of container.Plugin suitable for passing to container.RegisterPlugin()
func NewPlugin() container.Plugin {
	return &plugin{}
}

type plugin struct{}

func (p *plugin) InitializeFSContext(context *fs.Context) error {
	if tmpRktPath, err := RktPath(); err != nil {
		klog.V(5).Infof("Rkt not connected: %v", err)
	} else {
		context.RktPath = tmpRktPath
	}
	return nil
}

func (p *plugin) Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) (watcher.ContainerWatcher, error) {
	err := Register(factory, fsInfo, includedMetrics)
	if err != nil {
		return nil, err
	}
	return NewRktContainerWatcher()
}
