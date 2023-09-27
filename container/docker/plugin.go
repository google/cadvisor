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

package docker

import (
	"sync"
	"time"

	"golang.org/x/net/context"
	"k8s.io/klog/v2"

	dclient "github.com/docker/docker/client"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
)

const dockerClientTimeout = 10 * time.Second

func NewPluginWithOptions(o *Options) container.Plugin {
	return &plugin{options: o}
}

type Options struct {
	DockerEndpoint string
	DockerTLS      bool
	DockerCert     string
	DockerKey      string
	DockerCA       string

	dockerClientOnce sync.Once
	dockerClient     *dclient.Client
	dockerClientErr  error

	dockerRootDirOnce sync.Once
	dockerRootDir     string
}

func DefaultOptions() *Options {
	return &Options{
		DockerEndpoint: "unix:///var/run/docker.sock",
		DockerTLS:      false,
		DockerCert:     "cert.pem",
		DockerKey:      "key.pem",
		DockerCA:       "ca.pem",
	}
}

type plugin struct {
	options *Options
}

func (p *plugin) InitializeFSContext(context *fs.Context) error {
	SetTimeout(dockerClientTimeout)
	// Try to connect to docker indefinitely on startup.
	dockerStatus := p.retryDockerStatus()
	context.Docker = fs.DockerContext{
		Root:         p.options.RootDir(),
		Driver:       dockerStatus.Driver,
		DriverStatus: dockerStatus.DriverStatus,
	}
	return nil
}

func (p *plugin) Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) (container.Factories, error) {
	return Register(p.options, factory, fsInfo, includedMetrics)
}

func (p *plugin) retryDockerStatus() info.DockerStatus {
	startupTimeout := dockerClientTimeout
	maxTimeout := 4 * startupTimeout
	for {
		ctx, _ := context.WithTimeout(context.Background(), startupTimeout)
		dockerStatus, err := p.options.StatusWithContext(ctx)
		if err == nil {
			return dockerStatus
		}

		switch err {
		case context.DeadlineExceeded:
			klog.Warningf("Timeout trying to communicate with docker during initialization, will retry")
		default:
			klog.V(5).Infof("Docker not connected: %v", err)
			return info.DockerStatus{}
		}

		startupTimeout = 2 * startupTimeout
		if startupTimeout > maxTimeout {
			startupTimeout = maxTimeout
		}
	}
}
