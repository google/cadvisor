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

// Creates handler to handle containers launched by lxc (under /var/lib/docker/lxc)
package lxc

import (
	"fmt"
	"strings"
	"github.com/docker/libcontainer/cgroups"
	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/info"
)

type cgroupSubsystems struct {
	// Cgroup subsystem mounts.
	mounts []cgroups.Mount

	// Cgroup subsystem to their mount location.
	mountPoints map[string]string
}

type lxcFactory struct {
	// Factory for machine information.
	machineInfoFactory info.MachineInfoFactory
	cgroupSubsystems   *cgroupSubsystems
}

func (self *lxcFactory) String() string {
	return "lxc"
}

func (self *lxcFactory) NewContainerHandler(name string) (container.ContainerHandler, error) {
	fmt.Printf("Invoking NewLxcContainerhandler for %s by %q\n", name, self.String())
	return newLxcContainerHandler(name, self.cgroupSubsystems, self.machineInfoFactory)
}

// The lxc factory can handle any container.
func (self *lxcFactory) CanHandle(name string) (bool, error) {
	if name == "/lxc" {
		return true, nil
	} else if strings.HasPrefix(name, "/lxc/") {
		return true, nil
	}
	return false, nil
}

func Register(machineInfoFactory info.MachineInfoFactory) error {
	// Get all cgroup mounts.
	allCgroups, err := cgroups.GetCgroupMounts()
	if err != nil {
		return err
	}
	if len(allCgroups) == 0 {
		return fmt.Errorf("failed to find cgroup mounts for the lxc factory")
	}

	// Trim the mounts to only the subsystems we care about.
	supportedCgroups := make([]cgroups.Mount, 0, len(allCgroups))
	mountPoints := make(map[string]string, len(allCgroups))
	for _, mount := range allCgroups {
		for _, subsystem := range mount.Subsystems {
			if _, ok := supportedSubsystems[subsystem]; ok {
				supportedCgroups = append(supportedCgroups, mount)
				mountPoints[subsystem] = mount.Mountpoint
			}
		}
	}
	if len(supportedCgroups) == 0 {
		return fmt.Errorf("failed to find supported cgroup mounts for the lxc factory")
	}

	glog.Infof("Registering Lxc factory")
	factory := &lxcFactory{
		machineInfoFactory: machineInfoFactory,
		cgroupSubsystems: &cgroupSubsystems{
			mounts:      supportedCgroups,
			mountPoints: mountPoints,
		},
	}
	container.RegisterContainerHandlerFactory(factory)
	return nil
}

// Cgroup subsystems we support listing (should be the minimal set we need stats from).
var supportedSubsystems map[string]struct{} = map[string]struct{}{
	"cpu":     {},
	"cpuacct": {},
	"memory":  {},
	"cpuset":  {},
}
