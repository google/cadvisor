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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/container/docker/utils"
	v1 "github.com/google/cadvisor/info/v1"
)

const (
	Namespace = "podman"
)

var timeout = 10 * time.Second

func validateResponse(gotError error, response *http.Response) error {
	var err error
	switch {
	case response == nil:
		err = fmt.Errorf("response not present")
	case response.StatusCode == http.StatusNotFound:
		err = fmt.Errorf("item not found")
	case response.StatusCode == http.StatusNotImplemented:
		err = fmt.Errorf("query not implemented")
	default:
		return gotError
	}

	if gotError != nil {
		err = errors.Wrap(gotError, err.Error())
	}

	return err
}

func apiGetRequest(url string, item interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := client(&ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := conn.Client.Do(req)
	err = validateResponse(err, resp)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err

	}

	err = json.Unmarshal(data, item)
	if err != nil {
		return err
	}

	return ctx.Err()
}

func Images() ([]v1.DockerImage, error) {
	var summaries []dockertypes.ImageSummary
	err := apiGetRequest("http://d/v1.0.0/images/json", &summaries)
	if err != nil {
		return nil, err
	}
	return utils.SummariesToImages(summaries)
}

func Status() (v1.DockerStatus, error) {
	podmanInfo, err := GetInfo()
	if err != nil {
		return v1.DockerStatus{}, err
	}

	return docker.StatusFromDockerInfo(*podmanInfo)
}

func GetInfo() (*dockertypes.Info, error) {
	var info dockertypes.Info
	err := apiGetRequest("http://d/v1.0.0/info", &info)
	return &info, err
}

func VersionString() (string, error) {
	var version dockertypes.Version
	err := apiGetRequest("http://d/v1.0.0/version", &version)
	if err != nil {
		return "Unknown", err
	}

	return version.Version, nil
}

func InspectContainer(id string) (dockertypes.ContainerJSON, error) {
	var data dockertypes.ContainerJSON
	err := apiGetRequest(fmt.Sprintf("http://d/v1.0.0/containers/%s/json", id), &data)
	return data, err
}
