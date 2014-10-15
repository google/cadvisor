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
	"reflect"
	"testing"
	"time"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/info"
	itest "github.com/google/cadvisor/info/test"
	stest "github.com/google/cadvisor/storage/test"
)

const containerName = "/container"

// Create a containerData instance for a test. Optionsl storage driver may be specified (one is made otherwise).
func newTestContainerData(t *testing.T) (*containerData, *container.MockContainerHandler, *stest.MockStorageDriver) {
	mockHandler := container.NewMockContainerHandler(containerName)
	mockDriver := &stest.MockStorageDriver{}
	ret, err := newContainerData(containerName, mockDriver, mockHandler, false)
	if err != nil {
		t.Fatal(err)
	}
	return ret, mockHandler, mockDriver
}

func TestUpdateSubcontainers(t *testing.T) {
	subcontainers := []info.ContainerReference{
		{Name: "/container/ee0103"},
		{Name: "/container/abcd"},
		{Name: "/container/something"},
	}
	cd, mockHandler, _ := newTestContainerData(t)
	mockHandler.On("ListContainers", container.ListSelf).Return(
		subcontainers,
		nil,
	)

	err := cd.updateSubcontainers()
	if err != nil {
		t.Fatal(err)
	}

	if len(cd.info.Subcontainers) != len(subcontainers) {
		t.Errorf("Received %v subcontainers, should be %v", len(cd.info.Subcontainers), len(subcontainers))
	}

	for _, sub := range cd.info.Subcontainers {
		found := false
		for _, sub2 := range subcontainers {
			if sub.Name == sub2.Name {
				found = true
			}
		}
		if !found {
			t.Errorf("Received unknown sub container %v", sub)
		}
	}

	mockHandler.AssertExpectations(t)
}

func TestUpdateSubcontainersWithError(t *testing.T) {
	cd, mockHandler, _ := newTestContainerData(t)
	mockHandler.On("ListContainers", container.ListSelf).Return(
		[]info.ContainerReference{},
		fmt.Errorf("some error"),
	)

	err := cd.updateSubcontainers()
	if err == nil {
		t.Fatal("updateSubcontainers should return error")
	}
	if len(cd.info.Subcontainers) != 0 {
		t.Errorf("Received %v subcontainers, should be 0", len(cd.info.Subcontainers))
	}

	mockHandler.AssertExpectations(t)
}

func TestUpdateStats(t *testing.T) {
	statsList := itest.GenerateRandomStats(1, 4, 1*time.Second)
	stats := statsList[0]

	cd, mockHandler, mockDriver := newTestContainerData(t)
	mockHandler.On("GetStats").Return(
		stats,
		nil,
	)

	mockDriver.On("AddStats", info.ContainerReference{Name: mockHandler.Name}, stats).Return(nil)

	err := cd.updateStats()
	if err != nil {
		t.Fatal(err)
	}

	mockHandler.AssertExpectations(t)
}

func TestUpdateSpec(t *testing.T) {
	spec := itest.GenerateRandomContainerSpec(4)
	cd, mockHandler, _ := newTestContainerData(t)
	mockHandler.On("GetSpec").Return(
		spec,
		nil,
	)

	err := cd.updateSpec()
	if err != nil {
		t.Fatal(err)
	}

	mockHandler.AssertExpectations(t)
}

func TestGetInfo(t *testing.T) {
	spec := itest.GenerateRandomContainerSpec(4)
	subcontainers := []info.ContainerReference{
		{Name: "/container/ee0103"},
		{Name: "/container/abcd"},
		{Name: "/container/something"},
	}
	cd, mockHandler, _ := newTestContainerData(t)
	mockHandler.On("GetSpec").Return(
		spec,
		nil,
	)
	mockHandler.On("ListContainers", container.ListSelf).Return(
		subcontainers,
		nil,
	)
	mockHandler.Aliases = []string{"a1", "a2"}

	info, err := cd.GetInfo()
	if err != nil {
		t.Fatal(err)
	}

	mockHandler.AssertExpectations(t)

	if len(info.Subcontainers) != len(subcontainers) {
		t.Errorf("Received %v subcontainers, should be %v", len(info.Subcontainers), len(subcontainers))
	}

	for _, sub := range info.Subcontainers {
		found := false
		for _, sub2 := range subcontainers {
			if sub.Name == sub2.Name {
				found = true
			}
		}
		if !found {
			t.Errorf("Received unknown sub container %v", sub)
		}
	}

	if !reflect.DeepEqual(spec, info.Spec) {
		t.Errorf("received wrong container spec")
	}

	if info.Name != mockHandler.Name {
		t.Errorf("received wrong container name: received %v; should be %v", info.Name, mockHandler.Name)
	}
}
