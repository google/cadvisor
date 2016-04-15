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

package libcontainer

import (
	"path"

	"github.com/google/cadvisor/utils"
)

const (
	// Relative path to the libcontainer execdriver directory.
	libcontainerExecDriverPath = "execdriver/native"
	// Absolute path of the containerd directory.
	containerdPath = "/run/containerd"
)

// Gets the path to the libcontainer configuration.
func configPath(dockerRun, containerID string) string {
	return path.Join(dockerRun, libcontainerExecDriverPath, containerID, "state.json")
}

func containerdConfigPath(dockerRun, containerID string) string {
	return path.Join(containerdPath, containerID, "state.json")
}

// Gets the path to the old libcontainer configuration.
func oldConfigPath(dockerRoot, containerID string) string {
	return path.Join(dockerRoot, libcontainerExecDriverPath, containerID, "container.json")
}

// Gets whether the specified container exists.
func Exists(dockerRoot, dockerRun, containerID string) bool {
	// New or old config must exist for the container to be considered alive.
	return utils.FileExists(containerdConfigPath(dockerRun, containerID)) ||
		utils.FileExists(configPath(dockerRun, containerID)) ||
		utils.FileExists(oldConfigPath(dockerRoot, containerID))
}
