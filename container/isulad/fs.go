// Copyright 2023 Google Inc. All Rights Reserved.
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

package isulad

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/cadvisor/container/docker"
)

const (
	containerJSONFilename = "container.json"
)

type containersJSON struct {
	ID    string `json:"id"`
	Layer string `json:"layer"`
	// rest in unnecessary
}

func rwLayerID(storageDriver docker.StorageDriver, storageDir string, containerID string) (string, error) {
	data, err := os.ReadFile(filepath.Join(storageDir, "storage", string(storageDriver)+"-containers", containerID, containerJSONFilename))
	if err != nil {
		return "", err
	}
	var container containersJSON
	err = json.Unmarshal(data, &container)
	if err != nil {
		return "", err
	}

	return container.Layer, nil
}
