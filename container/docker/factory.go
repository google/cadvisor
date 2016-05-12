// Copyright 2014 Google Inc. All Rights Reserved.
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
	"flag"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"

	docker "github.com/docker/engine-api/client"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

var ArgDockerEndpoint = flag.String("docker", "unix:///var/run/docker.sock", "docker endpoint")

// The namespace under which Docker aliases are unique.
const DockerNamespace = "docker"

// Regexp that identifies docker cgroups, containers started with
// --cgroup-parent have another prefix than 'docker'
var dockerCgroupRegexp = regexp.MustCompile(`([a-z0-9]{64})`)

var dockerEnvWhitelist = flag.String("docker_env_metadata_whitelist", "", "a comma-separated list of environment variable keys that needs to be collected for docker containers")

var (
	// Basepath to all container specific information that libcontainer stores.
	dockerRootDir string

	dockerRootDirFlag = flag.String("docker_root", "/var/lib/docker", "DEPRECATED: docker root is read from docker info (this is a fallback, default: /var/lib/docker)")

	dockerRootDirOnce sync.Once
)

func RootDir() string {
	dockerRootDirOnce.Do(func() {
		status, err := Status()
		if err == nil && status.RootDir != "" {
			dockerRootDir = status.RootDir
		} else {
			dockerRootDir = *dockerRootDirFlag
		}
	})
	return dockerRootDir
}

type storageDriver string

const (
	// TODO: Add support for devicemapper storage usage.
	devicemapperStorageDriver storageDriver = "devicemapper"
	aufsStorageDriver         storageDriver = "aufs"
	overlayStorageDriver      storageDriver = "overlay"
	zfsStorageDriver          storageDriver = "zfs"
)

type dockerFactory struct {
	machineInfoFactory info.MachineInfoFactory

	storageDriver storageDriver
	storageDir    string

	client *docker.Client

	// Information about the mounted cgroup subsystems.
	cgroupSubsystems libcontainer.CgroupSubsystems

	// Information about mounted filesystems.
	fsInfo fs.FsInfo

	dockerVersion []int

	ignoreMetrics container.MetricSet
}

func (self *dockerFactory) String() string {
	return DockerNamespace
}

func (self *dockerFactory) NewContainerHandler(name string, inHostNamespace bool) (handler container.ContainerHandler, err error) {
	handle, accept, err := self.canHandleAndAccept(name)
	if handle != true {
		return nil, nil
	}
	if accept != true {
		ignore := container.NewIgnoreHandler(name)
		return ignore, nil
	}
	if err != nil {
		return nil, err
	}

	client, err := Client()
	if err != nil {
		return nil, err
	}

	metadataEnvs := strings.Split(*dockerEnvWhitelist, ",")

	handler, err = newDockerContainerHandler(
		client,
		name,
		self.machineInfoFactory,
		self.fsInfo,
		self.storageDriver,
		self.storageDir,
		&self.cgroupSubsystems,
		inHostNamespace,
		metadataEnvs,
		self.dockerVersion,
		self.ignoreMetrics,
	)
	return handler, err
}

// Returns the Docker ID from the full container name.
func ContainerNameToDockerId(name string) string {
	id := path.Base(name)

	if matches := dockerCgroupRegexp.FindStringSubmatch(id); matches != nil {
		return matches[1]
	}

	return id
}

func isContainerName(name string) bool {
	return dockerCgroupRegexp.MatchString(path.Base(name))
}

// Docker handles all containers under /docker
func (self *dockerFactory) canHandleAndAccept(name string) (bool, bool, error) {
	// docker factory accepts all containers it can handle.
	canAccept := true

	if !isContainerName(name) {
		return false, canAccept, fmt.Errorf("invalid container name")
	}

	// Check if the container is known to docker and it is active.
	id := ContainerNameToDockerId(name)

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := self.client.ContainerInspect(context.Background(), id)
	if err != nil || !ctnr.State.Running {
		return false, canAccept, fmt.Errorf("error inspecting container: %v", err)
	}

	return true, canAccept, nil
}

func (self *dockerFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

var (
	version_regexp_string = `(\d+)\.(\d+)\.(\d+)`
	version_re            = regexp.MustCompile(version_regexp_string)
)

// Register root container before running this function!
func Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, ignoreMetrics container.MetricSet) error {
	client, err := Client()
	if err != nil {
		return fmt.Errorf("unable to communicate with docker daemon: %v", err)
	}

	dockerInfo, err := ValidateInfo()
	if err != nil {
		return fmt.Errorf("failed to validate Docker info: %v", err)
	}

	// Version already validated above, assume no error here.
	dockerVersion, _ := parseDockerVersion(dockerInfo.ServerVersion)

	cgroupSubsystems, err := libcontainer.GetCgroupSubsystems()
	if err != nil {
		return fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}

	glog.Infof("Registering Docker factory")
	f := &dockerFactory{
		cgroupSubsystems:   cgroupSubsystems,
		client:             client,
		dockerVersion:      dockerVersion,
		fsInfo:             fsInfo,
		machineInfoFactory: factory,
		storageDriver:      storageDriver(dockerInfo.Driver),
		storageDir:         RootDir(),
		ignoreMetrics:      ignoreMetrics,
	}

	container.RegisterContainerHandlerFactory(f)
	return nil
}
