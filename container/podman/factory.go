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
	"path"
	"time"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/docker"
	dockerutil "github.com/google/cadvisor/container/docker/utils"
	"github.com/google/cadvisor/devicemapper"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/zfs"
)

const (
	rootDirRetries     = 5
	rootDirRetryPeriod = time.Second
	containerBaseName  = "container"
)

type podmanFactory struct {
	// Information about the mounted cgroup subsystems.
	machineInfoFactory info.MachineInfoFactory

	storageDriver docker.StorageDriver
	storageDir    string

	cgroupSubsystem map[string]string

	fsInfo fs.FsInfo

	metrics container.MetricSet

	thinPoolName    string
	thinPoolWatcher *devicemapper.ThinPoolWatcher

	zfsWatcher *zfs.ZfsWatcher

	podmanOptions *Options
}

type Options struct {
	podmanEndpoint string
	dockerOptions  *docker.Options
}

func DefaultOptions() *Options {
	return &Options{
		podmanEndpoint: "unix:///var/run/podman/podman.sock",
		dockerOptions:  docker.DefaultOptions(),
	}
}

func (f *podmanFactory) CanHandleAndAccept(name string) (handle bool, accept bool, err error) {
	// Rootless
	if path.Base(name) == containerBaseName {
		name, _ = path.Split(name)
	}
	if !dockerutil.IsContainerName(name) {
		return false, false, nil
	}

	id := dockerutil.ContainerNameToId(name)

	ctnr, err := f.podmanOptions.InspectContainer(id)
	if err != nil || !ctnr.State.Running {
		return false, true, fmt.Errorf("error inspecting container: %v", err)
	}

	return true, true, nil
}

func (f *podmanFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

func (f *podmanFactory) String() string {
	return "podman"
}

func (f *podmanFactory) NewContainerHandler(name string, metadataEnvAllowList []string, inHostNamespace bool) (handler container.ContainerHandler, err error) {
	return newPodmanContainerHandler(name, f.machineInfoFactory, f.fsInfo,
		f.storageDriver, f.storageDir, f.cgroupSubsystem, inHostNamespace,
		metadataEnvAllowList, f.metrics, f.thinPoolName, f.thinPoolWatcher, f.zfsWatcher, f.podmanOptions)
}
