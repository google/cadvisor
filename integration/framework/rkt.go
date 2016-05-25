// Copyright 2016 Google Inc. All Rights Reserved.
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

type RktActions interface {
	// Run the no-op pause container and return its ID.
	RunPause() string

	// Run the specified command in a busybox container and return its ID
	RunBusyBox(cmd ...string) string

	//	Version() []string
}

type rktActions struct {
	fm *realFramework
}

func (self rktActions) RunPause() string {
	return self.Run(RunArgs{
		Image: "kubernetes/pause",
	})
}

func (self rktActions) RunBusyBox(cmd ...string) string {
	return self.Run(RunArgs{
		Image: "busybox",
	}, cmd...)
}

func (self rktActions) Prepare(args RunArgs, cmd ...string) string {
	rktCommand := []string{"rkt", "prepare"}
	rktCommand = append(rktCommand, args.Args...)
	rktCommand = append(rktCommand, args.Image)
	if len(cmd) != 0 {
		rktCommand = append(rktCommand, "--")
		rktCommand = append(rktCommand, cmd...)
	}
	output, _ := self.fm.Shell().Run("sudo", rktCommand...)

	elements := strings.Fields(output)

	return elements[len(elements)-1]
}

func (self rktActions) RunPrepared(uuid string) {
	rktCommand := []string{"systemd-run", "rkt", "run-prepared", uuid}
	self.fm.Shell().Run("sudo", rktCommand...)
}

func (self rktActions) Remove(uuid string) func() {
	rktCommand := []string{"rkt", "rm", uuid}

	return func() {
		self.fm.Shell().Run("sudo", rktCommand...)
	}
}

func (self rktActions) Run(args RunArgs, cmd ...string) string {
	containerId := self.Prepare(args, cmd...)
	self.RunPrepared(containerId)

	self.fm.cleanups = append(self.fm.cleanups, self.Remove(containerId))

	return containerId
}
