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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/cadvisor/accelerators"
	"github.com/google/cadvisor/cache/memory"
	"github.com/google/cadvisor/container"
	_ "github.com/google/cadvisor/container/containerd"
	"github.com/google/cadvisor/container/crio"
	"github.com/google/cadvisor/container/docker"
	_ "github.com/google/cadvisor/container/mesos"
	_ "github.com/google/cadvisor/container/raw"
	"github.com/google/cadvisor/container/rkt"
	_ "github.com/google/cadvisor/container/systemd"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/machine"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/watcher"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"k8s.io/klog"
)

const dockerClientTimeout = 10 * time.Second

// New takes a memory storage and returns a new manager.
func New(memoryCache *memory.InMemoryCache, sysfs sysfs.SysFs, maxHousekeepingInterval time.Duration, allowDynamicHousekeeping bool, includedMetricsSet container.MetricSet, collectorHttpClient *http.Client, rawContainerCgroupPathPrefixWhiteList []string) (manager.Manager, error) {
	if memoryCache == nil {
		return nil, fmt.Errorf("manager requires memory storage")
	}

	// Detect the container we are running on.
	selfContainer, err := cgroups.GetOwnCgroupPath("cpu")
	if err != nil {
		return nil, err
	}
	klog.V(2).Infof("cAdvisor running in container: %q", selfContainer)

	var (
		dockerStatus info.DockerStatus
		rktPath      string
	)
	docker.SetTimeout(dockerClientTimeout)
	// Try to connect to docker indefinitely on startup.
	dockerStatus = retryDockerStatus()

	if tmpRktPath, err := rkt.RktPath(); err != nil {
		klog.V(5).Infof("Rkt not connected: %v", err)
	} else {
		rktPath = tmpRktPath
	}

	crioClient, err := crio.Client()
	if err != nil {
		return nil, err
	}
	crioInfo, err := crioClient.Info()
	if err != nil {
		klog.V(5).Infof("CRI-O not connected: %v", err)
	}

	context := fs.Context{
		Docker: fs.DockerContext{
			Root:         docker.RootDir(),
			Driver:       dockerStatus.Driver,
			DriverStatus: dockerStatus.DriverStatus,
		},
		RktPath: rktPath,
		Crio: fs.CrioContext{
			Root: crioInfo.StorageRoot,
		},
	}
	fsInfo, err := fs.NewFsInfo(context)
	if err != nil {
		return nil, err
	}

	// If cAdvisor was started with host's rootfs mounted, assume that its running
	// in its own namespaces.
	inHostNamespace := false
	if _, err := os.Stat("/rootfs/proc"); os.IsNotExist(err) {
		inHostNamespace = true
	}

	// Register for new subcontainers.
	eventsChannel := make(chan watcher.ContainerEvent, 16)

	machineInfo, err := machine.Info(sysfs, fsInfo, inHostNamespace)
	if err != nil {
		return nil, err
	}
	klog.V(1).Infof("Machine: %+v", *machineInfo)

	newManager := manager.New(
		memoryCache,
		fsInfo,
		sysfs,
		*machineInfo,
		make([]chan error, 0, 2),
		selfContainer,
		inHostNamespace,
		time.Now(),
		maxHousekeepingInterval,
		allowDynamicHousekeeping,
		includedMetricsSet,
		[]watcher.ContainerWatcher{},
		eventsChannel,
		collectorHttpClient,
		&accelerators.NvidiaManager{},
		rawContainerCgroupPathPrefixWhiteList)

	versionInfo, err := newManager.GetVersionInfo()
	if err != nil {
		return nil, err
	}
	klog.V(1).Infof("Version: %+v", *versionInfo)
	return newManager, nil
}

func retryDockerStatus() info.DockerStatus {
	startupTimeout := dockerClientTimeout
	maxTimeout := 4 * startupTimeout
	for {
		ctx, e := context.WithTimeout(context.Background(), startupTimeout)
		if e != nil {
			klog.V(5).Infof("error during timeout: %v", e)
		}
		dockerStatus, err := docker.StatusWithContext(ctx)
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
