// Copyright 2015 Google Inc. All Rights Reserved.
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

package api

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
)

const (
	containersApi    = "containers"
	subcontainersApi = "subcontainers"
	machineApi       = "machine"
	dockerApi        = "docker"
)

// Interface for a cAdvisor API version
type ApiVersion interface {
	// Returns the version string.
	Version() string

	// List of supported API endpoints.
	SupportedRequestTypes() []string

	// Handles a request. The second argument is the parameters after /api/<version>/<endpoint>
	HandleRequest(requestType string, request []string, m manager.Manager, w http.ResponseWriter, r *http.Request) error
}

// Gets all supported API versions.
func getApiVersions() []ApiVersion {
	v1_0 := &version1_0{}
	v1_1 := newVersion1_1(v1_0)
	v1_2 := newVersion1_2(v1_1)
	return []ApiVersion{v1_0, v1_1, v1_2}
}

// API v1.0

type version1_0 struct {
}

func (self *version1_0) Version() string {
	return "v1.0"
}

func (self *version1_0) SupportedRequestTypes() []string {
	return []string{containersApi, machineApi}
}

func (self *version1_0) HandleRequest(requestType string, request []string, m manager.Manager, w http.ResponseWriter, r *http.Request) error {
	switch {
	case requestType == machineApi:
		glog.V(2).Infof("Api - Machine")

		// Get the MachineInfo
		machineInfo, err := m.GetMachineInfo()
		if err != nil {
			return err
		}

		err = writeResult(machineInfo, w)
		if err != nil {
			return err
		}
	case requestType == containersApi:
		containerName := getContainerName(request)
		glog.V(2).Infof("Api - Container(%s)", containerName)

		// Get the query request.
		query, err := getContainerInfoRequest(r.Body)
		if err != nil {
			return err
		}

		// Get the container.
		cont, err := m.GetContainerInfo(containerName, query)
		if err != nil {
			return fmt.Errorf("failed to get container %q with error: %s", containerName, err)
		}

		// Only output the container as JSON.
		err = writeResult(cont, w)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown request type %q", requestType)
	}
	return nil
}

// API v1.1

type version1_1 struct {
	baseVersion *version1_0
}

// v1.1 builds on v1.0.
func newVersion1_1(v *version1_0) *version1_1 {
	return &version1_1{
		baseVersion: v,
	}
}

func (self *version1_1) Version() string {
	return "v1.1"
}

func (self *version1_1) SupportedRequestTypes() []string {
	return append(self.baseVersion.SupportedRequestTypes(), subcontainersApi)
}

func (self *version1_1) HandleRequest(requestType string, request []string, m manager.Manager, w http.ResponseWriter, r *http.Request) error {
	switch {
	case requestType == subcontainersApi:
		containerName := getContainerName(request)
		glog.V(2).Infof("Api - Subcontainers(%s)", containerName)

		// Get the query request.
		query, err := getContainerInfoRequest(r.Body)
		if err != nil {
			return err
		}

		// Get the subcontainers.
		containers, err := m.SubcontainersInfo(containerName, query)
		if err != nil {
			return fmt.Errorf("failed to get subcontainers for container %q with error: %s", containerName, err)
		}

		// Only output the containers as JSON.
		err = writeResult(containers, w)
		if err != nil {
			return err
		}
		return nil
	default:
		return self.baseVersion.HandleRequest(requestType, request, m, w, r)
	}
}

// API v1.2

type version1_2 struct {
	baseVersion *version1_1
}

// v1.2 builds on v1.1.
func newVersion1_2(v *version1_1) *version1_2 {
	return &version1_2{
		baseVersion: v,
	}
}

func (self *version1_2) Version() string {
	return "v1.2"
}

func (self *version1_2) SupportedRequestTypes() []string {
	return append(self.baseVersion.SupportedRequestTypes(), dockerApi)
}

func (self *version1_2) HandleRequest(requestType string, request []string, m manager.Manager, w http.ResponseWriter, r *http.Request) error {
	switch {
	case requestType == dockerApi:
		glog.V(2).Infof("Api - Docker(%v)", request)

		// Get the query request.
		query, err := getContainerInfoRequest(r.Body)
		if err != nil {
			return err
		}

		var containers map[string]info.ContainerInfo
		// map requests for "docker/" to "docker"
		if len(request) == 1 && len(request[0]) == 0 {
			request = request[:0]
		}
		switch len(request) {
		case 0:
			// Get all Docker containers.
			containers, err = m.AllDockerContainers(query)
			if err != nil {
				return fmt.Errorf("failed to get all Docker containers with error: %v", err)
			}
		case 1:
			// Get one Docker container.
			var cont info.ContainerInfo
			cont, err = m.DockerContainer(request[0], query)
			if err != nil {
				return fmt.Errorf("failed to get Docker container %q with error: %v", request[0], err)
			}
			containers = map[string]info.ContainerInfo{
				cont.Name: cont,
			}
		default:
			return fmt.Errorf("unknown request for Docker container %v", request)
		}

		// Only output the containers as JSON.
		err = writeResult(containers, w)
		if err != nil {
			return err
		}
		return nil
	default:
		return self.baseVersion.HandleRequest(requestType, request, m, w, r)
	}
}
