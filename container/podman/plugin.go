// Copyright 2021 Google Inc. All Rights Reserved.
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
	"fmt"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"k8s.io/klog/v2"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/docker"
	dockerutil "github.com/google/cadvisor/container/docker/utils"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/devicemapper"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/watcher"
	"github.com/google/cadvisor/zfs"
)

func NewPlugin() container.Plugin {
	return &plugin{}
}

type plugin struct{}

func (p *plugin) InitializeFSContext(context *fs.Context) error {
	context.Podman = fs.PodmanContext{
		Root:         "",
		Driver:       "",
		DriverStatus: map[string]string{},
	}

	return nil
}

func (p *plugin) Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) (watcher.ContainerWatcher, error) {
	return Register(factory, fsInfo, includedMetrics)
}

func Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, metrics container.MetricSet) (watcher.ContainerWatcher, error) {
	cgroupSubsystem, err := libcontainer.GetCgroupSubsystems(metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}

	validatedInfo, err := docker.ValidateInfo(GetInfo, VersionString)
	if err != nil {
		return nil, fmt.Errorf("failed to validate Podman info: %v", err)
	}

	var (
		thinPoolName    string
		thinPoolWatcher *devicemapper.ThinPoolWatcher
		zfsWatcher      *zfs.ZfsWatcher
	)
	if metrics.Has(container.DiskUsageMetrics) {
		switch docker.StorageDriver(validatedInfo.Driver) {
		case docker.DevicemapperStorageDriver:
			thinPoolWatcher, err = docker.StartThinPoolWatcher(validatedInfo)
			if err != nil {
				klog.Errorf("devicemapper filesystem stats will not be reported: %v", err)
			}

			status, _ := docker.StatusFromDockerInfo(*validatedInfo)
			thinPoolName = status.DriverStatus[dockerutil.DriverStatusPoolName]
		case docker.ZfsStorageDriver:
			zfsWatcher, err = docker.StartZfsWatcher(validatedInfo)
			if err != nil {
				klog.Errorf("zfs filesystem stats will not be reported: %v", err)
			}
		}
	}

	// Register Podman container handler factory.
	klog.V(1).Info("Registering Podman factory")
	f := &podmanFactory{
		machineInfoFactory: factory,
		storageDriver:      docker.StorageDriver(validatedInfo.Driver),
		storageDir:         RootDir(),
		cgroupSubsystem:    cgroupSubsystem,
		fsInfo:             fsInfo,
		metrics:            metrics,
		thinPoolName:       thinPoolName,
		thinPoolWatcher:    thinPoolWatcher,
		zfsWatcher:         zfsWatcher,
	}

	container.RegisterContainerHandlerFactory(f, []watcher.ContainerWatchSource{watcher.Raw})

	if !cgroups.IsCgroup2UnifiedMode() {
		klog.Warning("Podman rootless containers not working with cgroups v1!")
	}

	return nil, nil
}
