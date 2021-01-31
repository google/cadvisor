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

package podman

import (
	"time"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/watcher"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"
)

const podmanClientTimeout = 10 * time.Second

// NewPlugin returns an implementation of container.Plugin suitable for passing to container.RegisterPlugin()
func NewPlugin() container.Plugin {
	return &plugin{}
}

type plugin struct{}

func (p *plugin) InitializeFSContext(context *fs.Context) error {
	SetTimeout(podmanClientTimeout)
	// Try to connect to podman indefinitely on startup.
	podmanStatus := retryPodmanStatus()
	context.Podman = fs.PodmanContext{
		Root:         RootDir(),
		Driver:       podmanStatus.Driver,
		DriverStatus: podmanStatus.DriverStatus,
	}
	return nil
}

func (p *plugin) Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) (watcher.ContainerWatcher, error) {
	err := Register(factory, fsInfo, includedMetrics)
	return nil, err
}

func retryPodmanStatus() info.PodmanStatus {
	startupTimeout := podmanClientTimeout
	maxTimeout := 4 * startupTimeout
	for {
		ctx, _ := context.WithTimeout(context.Background(), startupTimeout)
		podmanStatus, err := StatusWithContext(ctx)
		if err == nil {
			return podmanStatus
		}

		switch err {
		case context.DeadlineExceeded:
			klog.Warningf("Timeout trying to communicate with podman during initialization, will retry")
		default:
			klog.V(5).Infof("Podman not connected: %v", err)
			return info.PodmanStatus{}
		}

		startupTimeout = 2 * startupTimeout
		if startupTimeout > maxTimeout {
			startupTimeout = maxTimeout
		}
	}
}
