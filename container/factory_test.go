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

package container_test

import (
	"testing"

	"github.com/google/cadvisor/container"
	containertest "github.com/google/cadvisor/container/testing"
	"github.com/google/cadvisor/watcher"

	"github.com/stretchr/testify/mock"
)

type mockContainerHandlerFactory struct {
	mock.Mock
	Name           string
	CanHandleValue bool
	CanAcceptValue bool
}

func (f *mockContainerHandlerFactory) String() string {
	return f.Name
}

func (f *mockContainerHandlerFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

func (f *mockContainerHandlerFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	return f.CanHandleValue, f.CanAcceptValue, nil
}

func (f *mockContainerHandlerFactory) NewContainerHandler(name string, metadataEnvAllowList []string, isHostNamespace bool) (container.ContainerHandler, error) {
	args := f.Called(name)
	return args.Get(0).(container.ContainerHandler), args.Error(1)
}

const testContainerName = "/test"

var testMetadataEnvAllowList = []string{}

var mockFactory containertest.FactoryForMockContainerHandler

func TestNewContainerHandler_FirstMatches(t *testing.T) {
	// Register one allways yes factory.
	allwaysYes := &mockContainerHandlerFactory{
		Name:           "yes",
		CanHandleValue: true,
		CanAcceptValue: true,
	}
	factories := container.Factories{watcher.Raw: []container.ContainerHandlerFactory{allwaysYes}}

	// The yes factory should be asked to create the ContainerHandler.
	mockContainer, err := mockFactory.NewContainerHandler(testContainerName, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	allwaysYes.On("NewContainerHandler", testContainerName).Return(mockContainer, nil)

	cont, _, err := container.NewContainerHandler(factories, testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	if cont == nil {
		t.Error("Expected container to not be nil")
	}
}

func TestNewContainerHandler_SecondMatches(t *testing.T) {
	// Register one allways no and one always yes factory.
	allwaysNo := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	allwaysYes := &mockContainerHandlerFactory{
		Name:           "yes",
		CanHandleValue: true,
		CanAcceptValue: true,
	}
	factories := container.Factories{watcher.Raw: []container.ContainerHandlerFactory{allwaysNo, allwaysYes}}

	// The yes factory should be asked to create the ContainerHandler.
	mockContainer, err := mockFactory.NewContainerHandler(testContainerName, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	allwaysYes.On("NewContainerHandler", testContainerName).Return(mockContainer, nil)

	cont, _, err := container.NewContainerHandler(factories, testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	if cont == nil {
		t.Error("Expected container to not be nil")
	}
}

func TestNewContainerHandler_NoneMatch(t *testing.T) {
	// Register two allways no factories.
	allwaysNo1 := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	allwaysNo2 := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	factories := container.Factories{watcher.Raw: []container.ContainerHandlerFactory{allwaysNo1, allwaysNo2}}

	_, _, err := container.NewContainerHandler(factories, testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err == nil {
		t.Error("Expected NewContainerHandler to fail")
	}
}

func TestNewContainerHandler_Accept(t *testing.T) {
	// Register handler that can handle the container, but can't accept it.
	cannotHandle := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	cannotAccept := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: true,
		CanAcceptValue: false,
	}
	factories := container.Factories{watcher.Raw: []container.ContainerHandlerFactory{cannotHandle, cannotAccept}}

	_, accept, err := container.NewContainerHandler(factories, testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error("Expected NewContainerHandler to succeed")
	}
	if accept == true {
		t.Error("Expected NewContainerHandler to ignore the container.")
	}
}
