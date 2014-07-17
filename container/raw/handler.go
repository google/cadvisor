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

package raw

import (
	"github.com/docker/libcontainer/cgroups"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/info"
)

type rawContainerHandler struct {
	name string
}

func newRawContainerHandler(name string) (container.ContainerHandler, error) {
	return &rawContainerHandler{
		name: name,
	}, nil
}

func (self *rawContainerHandler) ContainerReference() (info.ContainerReference, error) {
	// We only know the container by its one name.
	return info.ContainerReference{
		Name: self.name,
	}, nil
}

func (self *rawContainerHandler) GetSpec() (*info.ContainerSpec, error) {
	// TODO(vmarmol): Implement
	return new(info.ContainerSpec), nil
}

func (self *rawContainerHandler) GetStats() (stats *info.ContainerStats, err error) {
	cgroup := &cgroups.Cgroup{
		Parent: "/",
		Name:   self.name,
	}

	return libcontainer.GetStats(cgroup, false)
}

func (self *rawContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	// TODO(vmarmol): Implement
	return make([]info.ContainerReference, 0, 0), nil
}

func (self *rawContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	// TODO(vmarmol): Implement
	return nil, nil
}

func (self *rawContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	// TODO(vmarmol): Implement
	return nil, nil
}
