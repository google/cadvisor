// Copyright 2022 Google Inc. All Rights Reserved.
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

//go:build linux

package podman

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/cadvisor/container/docker"
)

var containersJsonFilnames = []string{
	"containers.json",
	"volatile-containers.json",
}

type containersJSON struct {
	ID    string `json:"id"`
	Layer string `json:"layer"`
	// rest in unnecessary
}

func rwLayerID(storageDriver docker.StorageDriver, storageDir string, containerID string) (string, error) {
	var containers []containersJSON

	for _, filename := range containersJsonFilnames {
		data, err := os.ReadFile(filepath.Join(storageDir, string(storageDriver)+"-containers", filename))
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}

		if data != nil {
			var buffer []containersJSON
			err = json.Unmarshal(data, &buffer)
			if err != nil {
				return "", err
			}
			containers = append(containers, buffer...)
		}
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("no containers found in containers.json or volatile-containers.json in %q", filepath.Join(storageDir, string(storageDriver)+"-containers"))
	}

	for _, c := range containers {
		if c.ID == containerID {
			return c.Layer, nil
		}
	}

	return "", fmt.Errorf("unable to determine %v rw layer id", containerID)
}
