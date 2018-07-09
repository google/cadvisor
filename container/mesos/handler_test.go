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
	"fmt"
	"testing"

	"github.com/google/cadvisor/container"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/stretchr/testify/assert"
)

func PopulateContainer() *mContainer {
	var pid uint32 = 123
	cntr := &mContainer{
		ContainerStatus: &mesos.ContainerStatus{ExecutorPID: &pid},
	}
	return cntr
}

func TestContainerReference(t *testing.T) {
	as := assert.New(t)
	type testCase struct {
		client             mesosAgentClient
		name               string
		machineInfoFactory info.MachineInfoFactory
		fsInfo             fs.FsInfo
		cgroupSubsystems   *containerlibcontainer.CgroupSubsystems
		inHostNamespace    bool
		includedMetrics    container.MetricSet

		hasErr         bool
		errContains    string
		checkReference *info.ContainerReference
	}
	for _, ts := range []testCase{
		{
			fakeMesosAgentClient(nil, fmt.Errorf("no client returned")),
			"/mesos/04e20821-67d3-4bf7-96b4-7d4495f50b28",
			nil,
			nil,
			&containerlibcontainer.CgroupSubsystems{},
			false,
			nil,

			true,
			"no client returned",
			nil,
		},
		{
			fakeMesosAgentClient(nil, nil),
			"/mesos/04e20821-67d3-4bf7-96b4-7d4495f50b28",
			nil,
			nil,
			&containerlibcontainer.CgroupSubsystems{},
			false,
			nil,

			true,
			"can't locate container 04e20821-67d3-4bf7-96b4-7d4495f50b28",
			nil,
		},
		{
			fakeMesosAgentClient(map[string]*containerInfo{"04e20821-67d3-4bf7-96b4-7d4495f50b28": {cntr: PopulateContainer()}}, nil),
			"/mesos/04e20821-67d3-4bf7-96b4-7d4495f50b28",
			nil,
			nil,
			&containerlibcontainer.CgroupSubsystems{},
			false,
			nil,

			false,
			"",
			&info.ContainerReference{
				Id:        "04e20821-67d3-4bf7-96b4-7d4495f50b28",
				Name:      "/mesos/04e20821-67d3-4bf7-96b4-7d4495f50b28",
				Aliases:   []string{"04e20821-67d3-4bf7-96b4-7d4495f50b28", "/mesos/04e20821-67d3-4bf7-96b4-7d4495f50b28"},
				Namespace: MesosNamespace,
			},
		},
	} {
		handler, err := newMesosContainerHandler(ts.name, ts.cgroupSubsystems, ts.machineInfoFactory, ts.fsInfo, ts.includedMetrics, ts.inHostNamespace, ts.client)
		if ts.hasErr {
			as.NotNil(err)
			if ts.errContains != "" {
				as.Contains(err.Error(), ts.errContains)
			}
		}
		if ts.checkReference != nil {
			cr, err := handler.ContainerReference()
			as.Nil(err)
			as.Equal(*ts.checkReference, cr)
		}
	}
}
