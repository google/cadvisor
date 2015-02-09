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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
)

const (
	apiResource = "/api/"
)

func RegisterHandlers(m manager.Manager) error {
	apiVersions := getApiVersions()
	supportedApiVersions := make(map[string]ApiVersion, len(apiVersions))
	for _, v := range apiVersions {
		supportedApiVersions[v.Version()] = v
	}

	http.HandleFunc(apiResource, func(w http.ResponseWriter, r *http.Request) {
		err := handleRequest(supportedApiVersions, m, w, r)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	return nil
}

// Captures the API version, requestType [optional], and remaining request [optional].
var apiRegexp = regexp.MustCompile("/api/([^/]+)/?([^/]+)?(.*)")

const (
	apiVersion = iota + 1
	apiRequestType
	apiRequestArgs
)

func handleRequest(supportedApiVersions map[string]ApiVersion, m manager.Manager, w http.ResponseWriter, r *http.Request) error {
	start := time.Now()
	defer func() {
		glog.V(2).Infof("Request took %s", time.Since(start))
	}()

	request := r.URL.Path

	const apiPrefix = "/api"
	if !strings.HasPrefix(request, apiPrefix) {
		return fmt.Errorf("incomplete API request %q", request)
	}

	// If the request doesn't have an API version, list those.
	if request == apiPrefix || request == apiResource {
		versions := make([]string, 0, len(supportedApiVersions))
		for v := range supportedApiVersions {
			versions = append(versions, v)
		}
		sort.Strings(versions)
		fmt.Fprintf(w, "Supported API versions: %s", strings.Join(versions, ","))
		return nil
	}

	// Verify that we have all the elements we expect:
	// /<version>/<request type>[/<args...>]
	requestElements := apiRegexp.FindStringSubmatch(request)
	if len(requestElements) == 0 {
		return fmt.Errorf("malformed request %q", request)
	}
	version := requestElements[apiVersion]
	requestType := requestElements[apiRequestType]
	requestArgs := strings.Split(requestElements[apiRequestArgs], "/")

	// Check supported versions.
	versionHandler, ok := supportedApiVersions[version]
	if !ok {
		return fmt.Errorf("unsupported API version %q", version)
	}

	// If no request type, list possible request types.
	if requestType == "" {
		requestTypes := versionHandler.SupportedRequestTypes()
		sort.Strings(requestTypes)
		fmt.Fprintf(w, "Supported request types: %q", strings.Join(requestTypes, ","))
		return nil
	}

	// Trim the first empty element from the request.
	if len(requestArgs) > 0 && requestArgs[0] == "" {
		requestArgs = requestArgs[1:]
	}
	return versionHandler.HandleRequest(requestType, requestArgs, m, w, r)
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

func getContainerName(request []string) string {
	return path.Join("/", strings.Join(request, "/"))
}
