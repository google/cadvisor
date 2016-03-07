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

package rkt

import (
	"fmt"
	"strings"

	rktapi "github.com/coreos/rkt/api/v1alpha"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"golang.org/x/net/context"

	"github.com/golang/glog"
)

const RktNamespace = "rkt"

type rktFactory struct {
	machineInfoFactory info.MachineInfoFactory

	cgroupSubsystems *libcontainer.CgroupSubsystems

	fsInfo fs.FsInfo

	ignoreMetrics container.MetricSet

	rktPath string
}

func (self *rktFactory) String() string {
	return "rkt"
}

func (self *rktFactory) NewContainerHandler(name string, inHostNamespace bool) (container.ContainerHandler, error) {
	client, err := Client()
	if err != nil {
		return nil, err
	}

	rootFs := "/"
	if !inHostNamespace {
		rootFs = "/rootfs"
	}
	return newRktContainerHandler(name, client, self.rktPath, self.cgroupSubsystems, self.machineInfoFactory, self.fsInfo, rootFs, self.ignoreMetrics)
}

func (self *rktFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	// TODO{SJP}: will ignore all cgroup names that don't either correspond to the machine.slice that is the pod
	// or the containers that belong to the pod
	if strings.HasPrefix(name, "/machine.slice/machine-rkt\\x2d") {
		accept, err := verifyName(name)
		return true, accept, err
	}
	return false, false, fmt.Errorf("%s not handled by rkt handler", name)
}

func (self *rktFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

func Register(machineInfoFactory info.MachineInfoFactory, fsInfo fs.FsInfo, ignoreMetrics container.MetricSet) error {
	client, err := Client()
	if err != nil {
		return fmt.Errorf("unable to communicate with Rkt api service: %v", err)
	}

	resp, err := client.GetInfo(context.Background(), &rktapi.GetInfoRequest{})
	if err != nil {
		return fmt.Errorf("couldn't GetInfo from rkt api servie: %v", err)
	}
	rktPath := resp.Info.GlobalFlags.Dir

	cgroupSubsystems, err := libcontainer.GetCgroupSubsystems()
	if err != nil {
		return fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}
	if len(cgroupSubsystems.Mounts) == 0 {
		return fmt.Errorf("failed to find supported cgroup mounts for the raw factory")
	}

	glog.Infof("Registering Rkt factory")
	factory := &rktFactory{
		machineInfoFactory: machineInfoFactory,
		fsInfo:             fsInfo,
		cgroupSubsystems:   &cgroupSubsystems,
		ignoreMetrics:      ignoreMetrics,
		rktPath:            rktPath,
	}
	container.RegisterContainerHandlerFactory(factory)
	return nil
}
