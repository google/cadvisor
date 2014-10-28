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

// Package api provides a handler for /api/
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
)

const (
	apiResource      = "/api/"
	containersApi    = "containers"
	subcontainersApi = "subcontainers"
	machineApi       = "machine"
	dockerApi        = "docker"

	version1_0 = "v1.0"
	version1_1 = "v1.1"
	version1_2 = "v1.2"
)

var supportedApiVersions map[string]struct{} = map[string]struct{}{
	version1_0: {},
	version1_1: {},
	version1_2: {},
}

func RegisterHandlers(m manager.Manager) error {
	http.HandleFunc(apiResource, func(w http.ResponseWriter, r *http.Request) {
		err := handleRequest(m, w, r)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	return nil
}

func handleRequest(m manager.Manager, w http.ResponseWriter, r *http.Request) error {
	start := time.Now()

	request := r.URL.Path
	requestElements := strings.Split(r.URL.Path, "/")

	// Verify that we have all the elements we expect:
	// <empty>/api/<version>/<request type>[/<args...>]
	// [0]     [1] [2]       [3]             [4...]
	if len(requestElements) < 4 {
		return fmt.Errorf("incomplete API request %q", request)
	}

	// Get all the element parts.
	emptyElement := requestElements[0]
	apiElement := requestElements[1]
	version := requestElements[2]
	requestType := requestElements[3]
	requestArgs := []string{}
	if len(requestElements) > 4 {
		requestArgs = requestElements[4:]
	}

	// The container name is the path after the requestType.
	containerName := path.Join("/", strings.Join(requestArgs, "/"))

	// Check elements.
	if len(emptyElement) != 0 {
		return fmt.Errorf("unexpected API request format %q", request)
	}
	if apiElement != "api" {
		return fmt.Errorf("invalid API request format %q", request)
	}
	if _, ok := supportedApiVersions[version]; !ok {
		return fmt.Errorf("unsupported API version %q", version)
	}

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
	case requestType == subcontainersApi:
		if version == version1_0 {
			return fmt.Errorf("request type of %q not supported in API version %q", requestType, version)
		}

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
	case requestType == dockerApi:
		if version == version1_0 || version == version1_1 {
			return fmt.Errorf("request type of %q not supported in API version %q", requestType, version)
		}

		glog.V(2).Infof("Api - Docker(%s)", containerName)

		// Get the query request.
		query, err := getContainerInfoRequest(r.Body)
		if err != nil {
			return err
		}

		// Get the Docker containers.
		containers, err := m.DockerContainersInfo(containerName, query)
		if err != nil {
			return fmt.Errorf("failed to get docker containers for %q with error: %s", containerName, err)
		}

		// Only output the containers as JSON.
		err = writeResult(containers, w)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown API request type %q", requestType)
	}

	glog.V(2).Infof("Request took %s", time.Since(start))
	return nil
}

func writeResult(res interface{}, w http.ResponseWriter) error {
	out, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("failed to marshall response %+v with error: %s", res, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
	return nil
}

func getContainerInfoRequest(body io.ReadCloser) (*info.ContainerInfoRequest, error) {
	var query info.ContainerInfoRequest

	// Default stats and samples is 64.
	query.NumStats = 64

	decoder := json.NewDecoder(body)
	err := decoder.Decode(&query)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("unable to decode the json value: %s", err)
	}

	return &query, nil
}
