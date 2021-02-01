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

// Provides global docker information.
package podman

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"golang.org/x/net/context"

	"github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/machine"
)

var podmanTimeout = 10 * time.Second

func defaultContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), podmanTimeout)
	return ctx
}

func SetTimeout(timeout time.Duration) {
	podmanTimeout = timeout
}

func Status() (v1.PodmanStatus, error) {
	return StatusWithContext(defaultContext())
}

func StatusWithContext(ctx context.Context) (v1.PodmanStatus, error) {
	client, err := Client()
	if err != nil {
		return v1.PodmanStatus{}, fmt.Errorf("unable to communicate with docker daemon: %v", err)
	}
	podmanInfo, err := client.Info(ctx)
	if err != nil {
		return v1.PodmanStatus{}, err
	}
	return StatusFromPodmanInfo(podmanInfo)
}

func StatusFromPodmanInfo(podmanInfo dockertypes.Info) (v1.PodmanStatus, error) {
	out := v1.PodmanStatus{}
	out.KernelVersion = machine.KernelVersion()
	out.OS = podmanInfo.OperatingSystem
	out.Hostname = podmanInfo.Name
	out.RootDir = podmanInfo.DockerRootDir
	out.Driver = podmanInfo.Driver
	out.NumImages = podmanInfo.Images
	out.NumContainers = podmanInfo.Containers
	out.DriverStatus = make(map[string]string, len(podmanInfo.DriverStatus))
	for _, v := range podmanInfo.DriverStatus {
		out.DriverStatus[v[0]] = v[1]
	}
	var err error
	ver, err := VersionString()
	if err != nil {
		return out, err
	}
	out.Version = ver
	ver, err = APIVersionString()
	if err != nil {
		return out, err
	}
	out.APIVersion = ver
	return out, nil
}

// Checks whether the podmanInfo reflects a valid docker setup, and returns it if it does, or an
// error otherwise.
func ValidateInfo() (*dockertypes.Info, error) {
	client, err := Client()
	if err != nil {
		return nil, fmt.Errorf("unable to communicate with docker daemon: %v", err)
	}

	podmanInfo, err := client.Info(defaultContext())
	if err != nil {
		return nil, fmt.Errorf("failed to detect Docker info: %v", err)
	}

	// Fall back to version API if ServerVersion is not set in info.
	if podmanInfo.ServerVersion == "" {
		version, err := client.ServerVersion(defaultContext())
		if err != nil {
			return nil, fmt.Errorf("unable to get docker version: %v", err)
		}
		podmanInfo.ServerVersion = version.Version
	}
	version, err := parseVersion(podmanInfo.ServerVersion, versionRe, 3)
	if err != nil {
		return nil, err
	}

	if version[0] < 1 {
		return nil, fmt.Errorf("cAdvisor requires docker version %v or above but we have found version %v reported as %q", []int{1, 0, 0}, version, podmanInfo.ServerVersion)
	}

	if podmanInfo.Driver == "" {
		return nil, fmt.Errorf("failed to find docker storage driver")
	}

	return &podmanInfo, nil
}

func APIVersion() ([]int, error) {
	ver, err := APIVersionString()
	if err != nil {
		return nil, err
	}
	return parseVersion(ver, apiVersionRe, 2)
}

func VersionString() (string, error) {
	dockerVersion := "Unknown"
	client, err := Client()
	if err == nil {
		version, err := client.ServerVersion(defaultContext())
		if err == nil {
			dockerVersion = version.Version
		}
	}
	return dockerVersion, err
}

func APIVersionString() (string, error) {
	apiVersion := "Unknown"
	client, err := Client()
	if err == nil {
		version, err := client.ServerVersion(defaultContext())
		if err == nil {
			apiVersion = version.APIVersion
		}
	}
	return apiVersion, err
}

func parseVersion(versionString string, regex *regexp.Regexp, length int) ([]int, error) {
	matches := regex.FindAllStringSubmatch(versionString, -1)
	if len(matches) != 1 {
		return nil, fmt.Errorf("version string \"%v\" doesn't match expected regular expression: \"%v\"", versionString, regex.String())
	}
	versionStringArray := matches[0][1:]
	versionArray := make([]int, length)
	for index, versionStr := range versionStringArray {
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, fmt.Errorf("error while parsing \"%v\" in \"%v\"", versionStr, versionString)
		}
		versionArray[index] = version
	}
	return versionArray, nil
}
