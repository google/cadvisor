// Copyright 2023 Google Inc. All Rights Reserved.
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

package isulad

import (
	"flag"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"k8s.io/klog/v2"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/docker"
	dockerutil "github.com/google/cadvisor/container/docker/utils"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/devicemapper"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/watcher"
)

const (
	rootDirRetries     = 5
	rootDirRetryPeriod = time.Second
)

var ArgIsuladEndpoint = flag.String("isulad", "/var/run/isulad.sock", "isulad endpoint")

const isuladNamespace = "isulad"

var (
	rootDir     string
	rootDirOnce sync.Once
)

// Regexp that identifies isulad cgroups, containers started with
// --cgroup-parent have another prefix than 'isulad'
var isuladCgroupRegexp = regexp.MustCompile(`([a-z0-9]{64})`)

func RootDir() string {
	rootDirOnce.Do(func() {
		for i := 0; i < rootDirRetries; i++ {
			client, err := Client()
			if err != nil {
				klog.Errorf("failed to create isulad client: %v", err)
				break
			}
			in, err := client.Info(context.Background())
			if err == nil && in.DockerRootDir != "" {
				rootDir = in.DockerRootDir
				break
			} else {
				time.Sleep(rootDirRetryPeriod)
			}
		}
	})
	return rootDir
}

type isuladFactory struct {
	machineInfoFactory info.MachineInfoFactory

	storageDriver docker.StorageDriver
	storageDir    string

	client IsuladClient
	// Information about the mounted cgroup subsystems.
	cgroupSubsystems map[string]string
	// Information about mounted filesystems.
	fsInfo          fs.FsInfo
	includedMetrics container.MetricSet

	thinPoolName    string
	thinPoolWatcher *devicemapper.ThinPoolWatcher
}

func (f *isuladFactory) String() string {
	return isuladNamespace
}

func (f *isuladFactory) NewContainerHandler(name string, metadataEnvAllowList []string, inHostNamespace bool) (handler container.ContainerHandler, err error) {
	client, err := Client()
	if err != nil {
		return
	}

	return newIsuladContainerHandler(
		client,
		name,
		f.machineInfoFactory,
		f.fsInfo,
		f.storageDriver,
		f.storageDir,
		f.cgroupSubsystems,
		inHostNamespace,
		metadataEnvAllowList,
		f.includedMetrics,
		f.thinPoolName,
		f.thinPoolWatcher,
	)
}

// Returns the isulad ID from the full container name.
func ContainerNameToIsuladID(name string) string {
	id := path.Base(name)
	if matches := isuladCgroupRegexp.FindStringSubmatch(id); matches != nil {
		return matches[1]
	}
	return id
}

// isContainerName returns true if the cgroup with associated name
// corresponds to a isulad container.
func isContainerName(name string) bool {
	if strings.HasSuffix(name, ".mount") {
		return false
	}
	return isuladCgroupRegexp.MatchString(path.Base(name))
}

// Isulad can handle and accept all isulad created containers
func (f *isuladFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	// if the container is not associated with isulad, we can't handle it or accept it.
	if !isContainerName(name) {
		return false, false, nil
	}
	// Check if the container is known to isulad and it is running.
	id := ContainerNameToIsuladID(name)
	// If container and task lookup in isulad fails then we assume
	// that the container state is not known to isulad
	cont, err := f.client.InspectContainer(context.Background(), id)
	if err != nil || !cont.State.Running {
		return false, false, fmt.Errorf("failed to load container: %v", err)
	}

	return true, true, nil
}

func (f *isuladFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

// Register root container before running this function!
func Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) error {
	c, err := Client()
	if err != nil {
		return fmt.Errorf("unable to create isulad client: %v", err)
	}

	in, err := c.Info(context.Background())
	if err != nil {
		return err
	}
	if in.Driver == "" {
		return fmt.Errorf("isulad driver is not set")
	}

	cgroupSubsystems, err := libcontainer.GetCgroupSubsystems(includedMetrics)
	if err != nil {
		return fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}

	var (
		thinPoolName    string
		thinPoolWatcher *devicemapper.ThinPoolWatcher
	)
	if includedMetrics.Has(container.DiskUsageMetrics) &&
		docker.StorageDriver(in.Driver) == docker.DevicemapperStorageDriver {
		thinPoolWatcher, err = docker.StartThinPoolWatcher(in)
		if err != nil {
			klog.Errorf("devicemapper filesystem stats will not be reported: %v", err)
		}

		status, _ := docker.StatusFromDockerInfo(*in)
		thinPoolName = status.DriverStatus[dockerutil.DriverStatusPoolName]
	}

	klog.V(1).Infof("Registering isulad factory")
	f := &isuladFactory{
		cgroupSubsystems:   cgroupSubsystems,
		storageDriver:      docker.StorageDriver(in.Driver),
		storageDir:         RootDir(),
		client:             c,
		fsInfo:             fsInfo,
		machineInfoFactory: factory,
		includedMetrics:    includedMetrics,
		thinPoolName:       thinPoolName,
		thinPoolWatcher:    thinPoolWatcher,
	}

	container.RegisterContainerHandlerFactory(f, []watcher.ContainerWatchSource{watcher.Raw})
	return nil
}
