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

// Per-container manager.

package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/cadvisor/container"
	ctest "github.com/google/cadvisor/container/test"
	"github.com/google/cadvisor/info"
	itest "github.com/google/cadvisor/info/test"
	"github.com/google/cadvisor/storage"
	stest "github.com/google/cadvisor/storage/test"
)

func createContainerDataAndSetHandler(
	driver storage.StorageDriver,
	f func(*ctest.MockContainerHandler),
	t *testing.T,
) *containerData {
	factory := &ctest.FactoryForMockContainerHandler{
		Name: "factoryForMockContainer",
		PrepareContainerHandlerFunc: func(name string, handler *ctest.MockContainerHandler) {
			handler.Name = name
			f(handler)
		},
	}
	container.RegisterContainerHandlerFactory("/", factory)

	if driver == nil {
		driver = &stest.MockStorageDriver{}
	}

	ret, err := NewContainerData("/container", driver)
	if err != nil {
		t.Fatal(err)
	}
	return ret
}

func TestContainerUpdateSubcontainers(t *testing.T) {
	var handler *ctest.MockContainerHandler
	subcontainers := []info.ContainerReference{
		{Name: "/container/ee0103"},
		{Name: "/container/abcd"},
		{Name: "/container/something"},
	}
	cd := createContainerDataAndSetHandler(
		nil,
		func(h *ctest.MockContainerHandler) {
			h.On("ListContainers", container.LIST_SELF).Return(
				subcontainers,
				nil,
			)
			handler = h
		},
		t,
	)

	err := cd.updateSubcontainers()
	if err != nil {
		t.Fatal(err)
	}

	handler.AssertExpectations(t)
}

func TestContainerUpdateSubcontainersWithError(t *testing.T) {
	var handler *ctest.MockContainerHandler
	cd := createContainerDataAndSetHandler(
		nil,
		func(h *ctest.MockContainerHandler) {
			h.On("ListContainers", container.LIST_SELF).Return(
				[]info.ContainerReference{},
				fmt.Errorf("some error"),
			)
			handler = h
		},
		t,
	)

	err := cd.updateSubcontainers()
	if err == nil {
		t.Fatal("updateSubcontainers should return error")
	}

	handler.AssertExpectations(t)
}

func TestContainerUpdateStats(t *testing.T) {
	var handler *ctest.MockContainerHandler
	var ref info.ContainerReference

	driver := &stest.MockStorageDriver{}

	statsList := itest.GenerateRandomStats(1, 4, 1*time.Second)
	stats := statsList[0]

	cd := createContainerDataAndSetHandler(
		driver,
		func(h *ctest.MockContainerHandler) {
			h.On("GetStats").Return(
				stats,
				nil,
			)
			handler = h
			ref.Name = h.Name
		},
		t,
	)

	driver.On("AddStats", ref, stats).Return(nil)

	err := cd.updateStats()
	if err != nil {
		t.Fatal(err)
	}

	handler.AssertExpectations(t)
}
