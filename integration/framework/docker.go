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

package framework

import "strings"

type DockerActions interface {
	// Run the no-op pause Docker container and return its ID.
	RunPause() string

	// Run the specified command in a Docker busybox container and return its ID.
	RunBusybox(cmd ...string) string

	// Runs a Docker container in the background. Uses the specified DockerRunArgs and command.
	// Returns the ID of the new container.
	//
	// e.g.:
	// Run(DockerRunArgs{Image: "busybox"}, "ping", "www.google.com")
	//   -> docker run busybox ping www.google.com
	Run(args RunArgs, cmd ...string) string
	RunStress(args RunArgs, cmd ...string) string

	Version() []string
	StorageDriver() string
}

type dockerActions struct {
	fm *realFramework
}

func (self dockerActions) RunPause() string {
	return self.Run(RunArgs{
		Image: "kubernetes/pause",
	})
}

// Run the specified command in a Docker busybox container.
func (self dockerActions) RunBusybox(cmd ...string) string {
	return self.Run(RunArgs{
		Image: "busybox",
	}, cmd...)
}

// TODO(vmarmol): Use the Docker remote API.
// TODO(vmarmol): Refactor a set of "RunCommand" actions.
// Runs a Docker container in the background. Uses the specified DockerRunArgs and command.
//
// e.g.:
// RunDockerContainer(DockerRunArgs{Image: "busybox"}, "ping", "www.google.com")
//   -> docker run busybox ping www.google.com
func (self dockerActions) Run(args RunArgs, cmd ...string) string {
	dockerCommand := append(append([]string{"docker", "run", "-d"}, args.Args...), args.Image)
	dockerCommand = append(dockerCommand, cmd...)
	output, _ := self.fm.Shell().Run("sudo", dockerCommand...)

	// The last line is the container ID.
	elements := strings.Fields(output)
	containerId := elements[len(elements)-1]

	self.fm.cleanups = append(self.fm.cleanups, func() {
		self.fm.Shell().Run("sudo", "docker", "rm", "-f", containerId)
	})
	return containerId
}
func (self dockerActions) Version() []string {
	dockerCommand := []string{"docker", "version", "-f", "'{{.Server.Version}}'"}
	output, _ := self.fm.Shell().Run("sudo", dockerCommand...)
	output = strings.TrimSpace(output)
	ret := strings.Split(output, ".")
	if len(ret) != 3 {
		self.fm.T().Fatalf("invalid version %v", output)
	}
	return ret
}

func (self dockerActions) StorageDriver() string {
	dockerCommand := []string{"docker", "info"}
	output, _ := self.fm.Shell().Run("sudo", dockerCommand...)
	if len(output) < 1 {
		self.fm.T().Fatalf("failed to find docker storage driver - %v", output)
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Storage Driver: ") {
			idx := strings.LastIndex(line, ": ") + 2
			driver := line[idx:]
			switch driver {
			case Aufs:
				return Aufs
			case Overlay:
				return Overlay
			case DeviceMapper:
				return DeviceMapper
			default:
				return Unknown
			}
		}
	}
	self.fm.T().Fatalf("failed to find docker storage driver from info - %v", output)
	return Unknown
}

func (self dockerActions) RunStress(args RunArgs, cmd ...string) string {
	dockerCommand := append(append(append(append([]string{"docker", "run", "-m=4M", "-d", "-t", "-i"}, args.Args...), args.Image), args.InnerArgs...), cmd...)

	output, _ := self.fm.Shell().RunStress("sudo", dockerCommand...)

	// The last line is the container ID.
	if len(output) < 1 {
		self.fm.T().Fatalf("need 1 arguments in output %v to get the name but have %v", output, len(output))
	}
	elements := strings.Fields(output)
	containerId := elements[len(elements)-1]

	self.fm.cleanups = append(self.fm.cleanups, func() {
		self.fm.Shell().Run("sudo", "docker", "rm", "-f", containerId)
	})
	return containerId
}
