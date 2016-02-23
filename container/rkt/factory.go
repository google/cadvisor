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

package rkt

import (
	"fmt"
	"strings"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"

	"github.com/golang/glog"
)

var RktNamespace = "rkt"

type rktFactory struct {
	// Factory for machine information.
	machineInfoFactory info.MachineInfoFactory

	// Information about the cgroup subsystems.
	cgroupSubsystems *libcontainer.CgroupSubsystems

	// Information about mounted filesystems.
	fsInfo fs.FsInfo

	// Watcher for inotify events.
	watcher *common.InotifyWatcher
}

func (self *rktFactory) String() string {
	return "rkt"
}

func (self *rktFactory) NewContainerHandler(name string, inHostNamespace bool) (container.ContainerHandler, error) {
	rootFs := "/"
	if !inHostNamespace {
		rootFs = "/rootfs"
	}
	return newRktContainerHandler(name, self.cgroupSubsystems, self.machineInfoFactory, self.fsInfo, self.watcher, rootFs)
}

func (self *rktFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	if strings.HasPrefix(name, "/machine.slice/machine-rkt\\x2d") {
		accept, err := verifyName(name)
		return true, accept, err
	}
	return false, false, fmt.Errorf("%s not handled by rkt handler", name)
}

func (self *rktFactory) DebugInfo() map[string][]string {
	return common.DebugInfo(self.watcher.GetWatches())
}

func Register(machineInfoFactory info.MachineInfoFactory, fsInfo fs.FsInfo) error {
	cgroupSubsystems, err := libcontainer.GetCgroupSubsystems()
	if err != nil {
		return fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}
	if len(cgroupSubsystems.Mounts) == 0 {
		return fmt.Errorf("failed to find supported cgroup mounts for the raw factory")
	}

	watcher, err := common.NewInotifyWatcher()
	if err != nil {
		return err
	}

	glog.Infof("Registering Rkt factory")
	factory := &rktFactory{
		machineInfoFactory: machineInfoFactory,
		fsInfo:             fsInfo,
		cgroupSubsystems:   &cgroupSubsystems,
		watcher:            watcher,
	}
	container.RegisterContainerHandlerFactory(factory)
	return nil
}
