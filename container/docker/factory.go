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
	"strconv"
	"strings"

	"github.com/docker/libcontainer/cgroups/systemd"
	"github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/info"
)

var ArgDockerEndpoint = flag.String("docker", "unix:///var/run/docker.sock", "docker endpoint")

// Basepath to all container specific information that libcontainer stores.
var dockerRootDir = flag.String("docker_root", "/var/lib/docker", "Absolute path to the Docker state root directory (default: /var/lib/docker)")

// Whether the system is using Systemd.
var useSystemd bool

func init() {
	useSystemd = systemd.UseSystemd()
	if useSystemd {
		glog.Infof("System is using systemd")
	}
}

type dockerFactory struct {
	machineInfoFactory info.MachineInfoFactory

	// Whether docker is running with AUFS storage driver.
	usesAufsDriver bool

	client *docker.Client
}

func (self *dockerFactory) String() string {
	return "docker"
}

func (self *dockerFactory) NewContainerHandler(name string) (handler container.ContainerHandler, err error) {
	client, err := docker.NewClient(*ArgDockerEndpoint)
	if err != nil {
		return
	}
	handler, err = newDockerContainerHandler(
		client,
		name,
		self.machineInfoFactory,
		*dockerRootDir,
		self.usesAufsDriver,
	)
	return
}

// Returns whether the specified full container name corresponds to a Docker container.
func IsDockerContainerName(name string) bool {
	if useSystemd {
		// In systemd systems the containers are: /system.slice/docker-{ID}
		return strings.HasPrefix(name, "/system.slice/docker-")
	} else {
		return strings.HasPrefix(name, "/docker/")
	}
}

// Returns a list of possible full container names for the specified Docker container name.
func FullDockerContainerNames(dockerName string) []string {
	names := make([]string, 0, 2)

	// Add the full container name.
	if useSystemd {
		names = append(names, path.Join("/system.slice", fmt.Sprintf("docker-%s.scope", dockerName)))
	} else {
		names = append(names, path.Join("/docker", dockerName))
	}

	// Add the Docker alias name.
	return append(names, path.Join("/docker", dockerName))
}

// Docker handles all containers under /docker
func (self *dockerFactory) CanHandle(name string) (bool, error) {
	if name == "/docker" {
		// We need the docker driver to handle /docker. Otherwise the aggregation at the API level will break.
		return true, nil
	} else if !IsDockerContainerName(name) {
		return false, nil
	}

	// Check if the container is known to docker and it is active.
	id := containerNameToDockerId(name)

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := self.client.InspectContainer(id)
	if err != nil || !ctnr.State.Running {
		return false, fmt.Errorf("error inspecting container: %v", err)
	}

	return true, nil
}

func parseDockerVersion(full_version_string string) ([]int, error) {
	version_regexp_string := "(\\d+)\\.(\\d+)\\.(\\d+)"
	version_re := regexp.MustCompile(version_regexp_string)
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

// Register root container before running this function!
func Register(factory info.MachineInfoFactory) error {
	client, err := docker.NewClient(*ArgDockerEndpoint)
	if err != nil {
		return fmt.Errorf("unable to communicate with docker daemon: %v", err)
	}
	if version, err := client.Version(); err != nil {
		return fmt.Errorf("unable to communicate with docker daemon: %v", err)
	} else {
		expected_version := []int{0, 11, 1}
		version_string := version.Get("Version")
		version, err := parseDockerVersion(version_string)
		if err != nil {
			return fmt.Errorf("couldn't parse docker version: %v", err)
		}
		for index, number := range version {
			if number > expected_version[index] {
				break
			} else if number < expected_version[index] {
				return fmt.Errorf("cAdvisor requires docker version above %v but we have found version %v reported as \"%v\"", expected_version, version, version_string)
			}
		}
	}

	// Check that the libcontainer execdriver is used.
	information, err := client.Info()
	if err != nil {
		return fmt.Errorf("failed to detect Docker info: %v", err)
	}
	usesNativeDriver := false
	for _, val := range *information {
		if strings.Contains(val, "ExecutionDriver=") && strings.Contains(val, "native") {
			usesNativeDriver = true
			break
		}
	}
	if !usesNativeDriver {
		return fmt.Errorf("Docker found, but not using native exec driver")
	}

	usesAufsDriver := false
	for _, val := range *information {
		if strings.Contains(val, "Driver=") && strings.Contains(val, "aufs") {
			usesAufsDriver = true
			break
		}
	}

	f := &dockerFactory{
		machineInfoFactory: factory,
		client:             client,
		usesAufsDriver:     usesAufsDriver,
	}
	container.RegisterContainerHandlerFactory(f)
	return nil
}
