// Copyright 2018 Google Inc. All Rights Reserved.
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

package mesos

import (
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsContainerName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "/system.slice/var-lib-mesos-provisioner-containers-04e20821-d67d3-4bf7-96b4-7d4495f50b28-backends-overlay-rootfses-6d97be39-7359-4bb7-a46b-e55c6771da81.mount",
			expected: false,
		},
		{
			name:     "/mesos/04e20821-67d3-4bf7-96b4-7d4495f50b28",
			expected: true,
		},
	}
	for _, test := range tests {
		if actual := isContainerName(test.name); actual != test.expected {
			t.Errorf("%s: expected: %v, actual: %v", test.name, test.expected, actual)
		}
	}
}
func TestCanHandleAndAccept(t *testing.T) {
	as := assert.New(t)
	testContainers := make(map[string]*containerInfo)
	var pid uint32 = 123
	testContainer := &containerInfo{
		cntr: &mContainer{
			ContainerStatus: &mesos.ContainerStatus{
				ExecutorPID: &pid,
			},
		},
	}

	testContainers["04e20821-67d3-4bf7-96b4-7d4495f50b28"] = testContainer

	f := &mesosFactory{
		machineInfoFactory: nil,
		cgroupSubsystems:   containerlibcontainer.CgroupSubsystems{},
		fsInfo:             nil,
		includedMetrics:    nil,
		client:             fakeMesosAgentClient(testContainers, nil),
	}
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "/mesos/04e20821-67d3-4bf7-96b4-7d4495f50b28",
			expected: true,
		},
		{
			name:     "/system.slice/var-lib-mesos-provisioner-containers-04e20821-d67d3-4bf7-96b4-7d4495f50b28-backends-overlay-rootfses-6d97be39-7359-4bb7-a46b-e55c6771da81.mount",
			expected: false,
		},
	}

	for _, test := range tests {
		b1, b2, err := f.CanHandleAndAccept(test.name)
		as.Nil(err)
		as.Equal(b1, test.expected)
		as.Equal(b2, test.expected)
	}
}
