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
	"encoding/json"
	"flag"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils"

	"github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
	"github.com/opencontainers/runc/libcontainer/cgroups"
)

var ArgDockerEndpoint = flag.String("docker", "unix:///var/run/docker.sock", "docker endpoint")

// The namespace under which Docker aliases are unique.
var DockerNamespace = "docker"

// Basepath to all container specific information that libcontainer stores.
// TODO: Deprecate this flag
var dockerRootDir = flag.String("docker_root", "/var/lib/docker", "Absolute path to the Docker state root directory (default: /var/lib/docker)")
var dockerRunDir = flag.String("docker_run", "/var/run/docker", "Absolute path to the Docker run directory (default: /var/run/docker)")

// Regexp that identifies docker cgroups, containers started with
// --cgroup-parent have another prefix than 'docker'
var dockerCgroupRegexp = regexp.MustCompile(`.+-([a-z0-9]{64})\.scope$`)

var noSystemd = flag.Bool("nosystemd", false, "Explicitly disable systemd support for Docker containers")

var dockerEnvWhitelist = flag.String("docker_env_metadata_whitelist", "", "a comma-separated list of environment variable keys that needs to be collected for docker containers")

// TODO(vmarmol): Export run dir too for newer Dockers.
// Directory holding Docker container state information.
func DockerStateDir() string {
	return libcontainer.DockerStateDir(*dockerRootDir)
}

// Whether the system is using Systemd.
var useSystemd = false
var check = sync.Once{}

const (
	dockerRootDirKey = "Root Dir"
)

func UseSystemd() bool {
	check.Do(func() {
		if *noSystemd {
			return
		}
		// Check for system.slice in systemd and cpu cgroup.
		for _, cgroupType := range []string{"name=systemd", "cpu"} {
			mnt, err := cgroups.FindCgroupMountpoint(cgroupType)
			if err == nil {
				// systemd presence does not mean systemd controls cgroups.
				// If system.slice cgroup exists, then systemd is taking control.
				// This breaks if user creates system.slice manually :)
				if utils.FileExists(path.Join(mnt, "system.slice")) {
					useSystemd = true
					break
				}
			}
		}
	})
	return useSystemd
}

func RootDir() string {
	return *dockerRootDir
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

	storageDriver    storageDriver
	storageDriverDir string

	client *docker.Client

	// Information about the mounted cgroup subsystems.
	cgroupSubsystems libcontainer.CgroupSubsystems

	// Information about mounted filesystems.
	fsInfo fs.FsInfo
}

func (self *dockerFactory) String() string {
	return DockerNamespace
}

func (self *dockerFactory) NewContainerHandler(name string, inHostNamespace bool) (handler container.ContainerHandler, err error) {
	client, err := Client()
	if err != nil {
		return
	}

	metadataEnvs := strings.Split(*dockerEnvWhitelist, ",")

	handler, err = newDockerContainerHandler(
		client,
		name,
		self.machineInfoFactory,
		self.fsInfo,
		self.storageDriver,
		self.storageDriverDir,
		&self.cgroupSubsystems,
		inHostNamespace,
		metadataEnvs,
	)
	return
}

// Returns the Docker ID from the full container name.
func ContainerNameToDockerId(name string) string {
	id := path.Base(name)

	// Turn systemd cgroup name into Docker ID.
	if UseSystemd() {
		if matches := dockerCgroupRegexp.FindStringSubmatch(id); matches != nil {
			id = matches[1]
		}
	}

	return id
}

func isContainerName(name string) bool {
	if UseSystemd() {
		return dockerCgroupRegexp.MatchString(path.Base(name))
	}
	return true
}

// Docker handles all containers under /docker
func (self *dockerFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	// docker factory accepts all containers it can handle.
	canAccept := true

	if !isContainerName(name) {
		return false, canAccept, fmt.Errorf("invalid container name")
	}

	// Check if the container is known to docker and it is active.
	id := ContainerNameToDockerId(name)

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := self.client.InspectContainer(id)
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

func parseDockerVersion(full_version_string string) ([]int, error) {
	matches := version_re.FindAllStringSubmatch(full_version_string, -1)
	if len(matches) != 1 {
		return nil, fmt.Errorf("version string \"%v\" doesn't match expected regular expression: \"%v\"", full_version_string, version_regexp_string)
	}
	version_string_array := matches[0][1:]
	version_array := make([]int, 3)
	for index, version_string := range version_string_array {
		version, err := strconv.Atoi(version_string)
		if err != nil {
			return nil, fmt.Errorf("error while parsing \"%v\" in \"%v\"", version_string, full_version_string)
		}
		version_array[index] = version
	}
	return version_array, nil
}

func getStorageDir(dockerInfo *docker.Env) (string, error) {
	storageDriverInfo := dockerInfo.GetList("DriverStatus")
	if len(storageDriverInfo) == 0 {
		return "", fmt.Errorf("failed to find docker storage driver options")
	}

	var storageDir string
	for _, data := range storageDriverInfo {
		if data == "" {
			continue
		}
		var parts [][]string
		if err := json.Unmarshal([]byte(data), &parts); err != nil {
			return "", fmt.Errorf("failed to parse docker storage driver options - %+v", data)
		}
		for _, part := range parts {
			if len(part) != 2 {
				return "", fmt.Errorf("failed to parse docker storage driver options - %+v", part)
			}
			if part[0] == dockerRootDirKey {
				storageDir = part[1]
				break
			}
		}
	}
	if storageDir == "" {
		return "", fmt.Errorf("failed to find docker storage directory from docker info. Expected key %q in storage driver info %v", dockerRootDirKey, storageDriverInfo)
	}
	return storageDir, nil
}

// Register root container before running this function!
func Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo) error {
	if UseSystemd() {
		glog.Infof("System is using systemd")
	}

	client, err := Client()
	if err != nil {
		return fmt.Errorf("unable to communicate with docker daemon: %v", err)
	}
	if version, err := client.Version(); err != nil {
		return fmt.Errorf("unable to communicate with docker daemon: %v", err)
	} else {
		expected_version := []int{1, 0, 0}
		version_string := version.Get("Version")
		version, err := parseDockerVersion(version_string)
		if err != nil {
			return fmt.Errorf("couldn't parse docker version: %v", err)
		}
		for index, number := range version {
			if number > expected_version[index] {
				break
			} else if number < expected_version[index] {
				return fmt.Errorf("cAdvisor requires docker version %v or above but we have found version %v reported as \"%v\"", expected_version, version, version_string)
			}
		}
	}

	information, err := client.Info()
	if err != nil {
		return fmt.Errorf("failed to detect Docker info: %v", err)
	}

	// Check that the libcontainer execdriver is used.
	execDriver := information.Get("ExecutionDriver")
	if !strings.HasPrefix(execDriver, "native") {
		return fmt.Errorf("docker found, but not using native exec driver")
	}

	sd := information.Get("Driver")
	if sd == "" {
		return fmt.Errorf("failed to find docker storage driver")
	}

	storageDir, err := getStorageDir(information)
	if err != nil {
		return err
	}
	cgroupSubsystems, err := libcontainer.GetCgroupSubsystems()
	if err != nil {
		return fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}

	glog.Infof("Registering Docker factory")
	f := &dockerFactory{
		machineInfoFactory: factory,
		client:             client,
		storageDriver:      storageDriver(sd),
		storageDriverDir:   storageDir,
		cgroupSubsystems:   cgroupSubsystems,
		fsInfo:             fsInfo,
	}
	container.RegisterContainerHandlerFactory(f)
	return nil
}
