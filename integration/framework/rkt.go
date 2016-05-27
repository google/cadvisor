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

import (
	"math/rand"
	"strings"
	"time"
)

type RktActions interface {
	// Run the no-op pause container and return its ID.
	RunPause() string

	// Run the specified command in a busybox container and return its ID
	RunBusybox(cmd ...string) string

	Run(args RktRunArgs) string

	//	Version() []string
}

type RktImage struct {
	Image   string
	RktArgs []string
	Cmd     string
	CmdArgs []string
}

type RktRunArgs struct {
	// Arguments to systemd run
	SystemddArgs []string

	// Generic arguments to rkt cli
	RuntimeArgs []string

	// Arguments to rkt run that should impact entire pod
	GlobalPodArgs []string

	Images []RktImage
}

type rktActions struct {
	fm *realFramework
}

func (self rktActions) RunPause() string {
	return self.Run(RktRunArgs{
		Images: []RktImage{
			{
				Image: "docker://kubernetes/pause",
			},
		},
	})
}

func (self rktActions) RunBusybox(cmd ...string) string {
	image := RktImage{Image: "docker://busybox"}
	if len(cmd) > 0 {
		image.Cmd = cmd[0]
	}

	if len(cmd) > 1 {
		image.CmdArgs = cmd[1:]
	}

	return self.Run(RktRunArgs{
		Images: []RktImage{image},
	})
}

func (self rktActions) Prepare(args RktRunArgs) string {
	rktCommand := []string{"rkt", "prepare", "--insecure-options=image"}
	rktCommand = append(rktCommand, args.RuntimeArgs...)
	for _, image := range args.Images {
		rktCommand = append(rktCommand, image.Image)
		if image.Cmd != "" {
			rktCommand = append(rktCommand, "--exec="+image.Cmd)
		}
		if len(image.CmdArgs) != 0 {
			rktCommand = append(rktCommand, "--")
			rktCommand = append(rktCommand, image.CmdArgs...)
		}
	}

	output, _ := self.fm.Shell().Run("sudo", rktCommand...)

	elements := strings.Fields(output)

	return elements[len(elements)-1]
}

func (self rktActions) RunPrepared(uuid string, randUnit string, systemdArgs []string) {
	rktCommand := []string{"systemd-run", "--unit=cadvisor-" + randUnit}
	rktCommand = append(rktCommand, systemdArgs...)
	rktCommand = append(rktCommand, []string{"rkt", "run-prepared", uuid}...)
	self.fm.Shell().Run("sudo", rktCommand...)
}

func (self rktActions) Remove(uuid string, randUnit string) func() {
	systemdCmd := []string{"systemctl", "stop", "cadvisor-" + randUnit}
	rktCommand := []string{"rkt", "rm", uuid}

	return func() {
		self.fm.Shell().Run("sudo", systemdCmd...)
		self.fm.Shell().Run("sudo", rktCommand...)
	}
}

func (self rktActions) Run(args RktRunArgs) string {
	randUnit := RandomString(6)

	containerId := self.Prepare(args)

	self.RunPrepared(containerId, randUnit, args.SystemddArgs)

	self.fm.cleanups = append(self.fm.cleanups, self.Remove(containerId, randUnit))

	return containerId
}

func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
