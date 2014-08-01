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

// Handler for /api/

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
)

const (
	apiResource      = "/api/"
	containersApi    = "containers"
	subcontainersApi = "subcontainers"
	machineApi       = "machine"
)

var supportedApiVersions map[string]struct{} = map[string]struct{}{
	"v1.0": struct{}{},
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
		log.Printf("Api - Machine")

		// Get the MachineInfo
		machineInfo, err := m.GetMachineInfo()
		if err != nil {
			return err
		}

		out, err := json.Marshal(machineInfo)
		if err != nil {
			fmt.Fprintf(w, "Failed to marshall MachineInfo with error: %s", err)
		}
		w.Write(out)
	case requestType == containersApi:
		// The container name is the path after the requestType.
		containerName := path.Join("/", strings.Join(requestArgs, "/"))

		log.Printf("Api - Container(%s)", containerName)

		var query info.ContainerInfoRequest

		// If a user does not specify number of stats/samples he wants,
		// it's 64 by default
		query.NumStats = 64
		query.NumSamples = 64
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&query)
		if err != nil && err != io.EOF {
			return fmt.Errorf("unable to decode the json value: ", err)
		}
		// Get the container.
		cont, err := m.GetContainerInfo(containerName, &query)
		if err != nil {
			fmt.Fprintf(w, "Failed to get container \"%s\" with error: %s", containerName, err)
			return err
		}

		// Only output the container as JSON.
		out, err := json.Marshal(cont)
		if err != nil {
			fmt.Fprintf(w, "Failed to marshall container %q with error: %s", containerName, err)
		}
		w.Write(out)
	default:
		return fmt.Errorf("unknown API request type %q", requestType)
	}

	log.Printf("Request took %s", time.Since(start))
	return nil
}
