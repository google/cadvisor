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

package testing

import (
	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"

	"github.com/stretchr/testify/mock"
)

// This struct mocks a container handler.
type MockContainerHandler struct {
	mock.Mock
	Name    string
	Aliases []string
}

func NewMockContainerHandler(containerName string) *MockContainerHandler {
	return &MockContainerHandler{
		Name: containerName,
	}
}

// If self.Name is not empty, then ContainerReference() will return self.Name and self.Aliases.
// Otherwise, it will use the value provided by .On().Return().
func (h *MockContainerHandler) ContainerReference() (info.ContainerReference, error) {
	if len(h.Name) > 0 {
		var aliases []string
		if len(h.Aliases) > 0 {
			aliases = make([]string, len(h.Aliases))
			copy(aliases, h.Aliases)
		}
		return info.ContainerReference{
			Name:    h.Name,
			Aliases: aliases,
		}, nil
	}
	args := h.Called()
	return args.Get(0).(info.ContainerReference), args.Error(1)
}

func (h *MockContainerHandler) Start() {}

func (h *MockContainerHandler) Cleanup() {}

func (h *MockContainerHandler) GetSpec() (info.ContainerSpec, error) {
	args := h.Called()
	return args.Get(0).(info.ContainerSpec), args.Error(1)
}

func (h *MockContainerHandler) GetStats() (*info.ContainerStats, error) {
	args := h.Called()
	return args.Get(0).(*info.ContainerStats), args.Error(1)
}

func (h *MockContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	args := h.Called(listType)
	return args.Get(0).([]info.ContainerReference), args.Error(1)
}

func (h *MockContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	args := h.Called(listType)
	return args.Get(0).([]int), args.Error(1)
}

func (h *MockContainerHandler) Exists() bool {
	args := h.Called()
	return args.Get(0).(bool)
}

func (h *MockContainerHandler) GetCgroupPath(path string) (string, error) {
	args := h.Called(path)
	return args.Get(0).(string), args.Error(1)
}

func (h *MockContainerHandler) GetContainerLabels() map[string]string {
	args := h.Called()
	return args.Get(0).(map[string]string)
}

func (h *MockContainerHandler) Type() container.ContainerType {
	args := h.Called()
	return args.Get(0).(container.ContainerType)
}

func (h *MockContainerHandler) GetContainerIPAddress() string {
	args := h.Called()
	return args.Get(0).(string)
}

type FactoryForMockContainerHandler struct {
	Name                        string
	PrepareContainerHandlerFunc func(name string, handler *MockContainerHandler)
}

func (h *FactoryForMockContainerHandler) String() string {
	return h.Name
}

func (h *FactoryForMockContainerHandler) NewContainerHandler(name string, inHostNamespace bool) (container.ContainerHandler, error) {
	handler := &MockContainerHandler{}
	if h.PrepareContainerHandlerFunc != nil {
		h.PrepareContainerHandlerFunc(name, handler)
	}
	return handler, nil
}

func (h *FactoryForMockContainerHandler) CanHandle(name string) bool {
	return true
}
