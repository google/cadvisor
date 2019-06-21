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
	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/agent"
	"github.com/stretchr/testify/assert"
	"testing"
)

func PopulateFrameworks(fwID string) *agent.Response_GetFrameworks {
	fws := &agent.Response_GetFrameworks{}
	fws.Frameworks = make([]agent.Response_GetFrameworks_Framework, 1)
	fw := &agent.Response_GetFrameworks_Framework{}
	fw.FrameworkInfo = mesos.FrameworkInfo{
		ID:   &mesos.FrameworkID{Value: fwID},
		Name: "TestFramework",
	}
	fws.Frameworks[0] = *fw
	return fws
}

func PopulateExecutors(exID string) *agent.Response_GetExecutors {
	execs := &agent.Response_GetExecutors{}
	execs.Executors = make([]agent.Response_GetExecutors_Executor, 1)
	exec := &agent.Response_GetExecutors_Executor{}
	source := "source1"
	exec.ExecutorInfo = mesos.ExecutorInfo{
		ExecutorID: mesos.ExecutorID{Value: exID},
		Source:     &source,
	}
	execs.Executors[0] = *exec
	return execs
}

func PopulateTasks(taskID string, exID string) *agent.Response_GetTasks {
	tasks := &agent.Response_GetTasks{}
	tasks.LaunchedTasks = make([]mesos.Task, 1)

	task := mesos.Task{
		TaskID: mesos.TaskID{Value: taskID},
	}
	if len(exID) > 0 {
		task.ExecutorID = &mesos.ExecutorID{Value: exID}
	}

	task.Resources = make([]mesos.Resource, 1)
	resource := mesos.Resource{
		Name:      cpus,
		Revocable: nil,
	}
	task.Resources[0] = resource

	task.Labels = &mesos.Labels{
		Labels: make([]mesos.Label, 1),
	}
	labelValue := "value1"
	label := mesos.Label{
		Key:   "key1",
		Value: &labelValue,
	}
	task.Labels.Labels[0] = label

	tasks.LaunchedTasks[0] = task
	return tasks
}

func TestFetchLabels(t *testing.T) {
	type testCase struct {
		frameworkID    string
		executorID     string
		agentState     *agent.Response_GetState
		expectedError  error
		expectedLabels map[string]string
	}

	for _, ts := range []testCase{
		{
			frameworkID: "fw-id1",
			executorID:  "exec-id1",
			agentState: &agent.Response_GetState{
				GetFrameworks: PopulateFrameworks("fw-id1"),
				GetExecutors:  PopulateExecutors("exec-id1"),
				GetTasks:      PopulateTasks("task-id1", "exec-id1"),
			},
			expectedError: nil,
			expectedLabels: map[string]string{
				framework:    "TestFramework",
				source:       "source1",
				schedulerSLA: nonRevocable,
				"key1":       "value1",
			},
		},
		{
			frameworkID: "fw-id1",
			executorID:  "task-id1",
			agentState: &agent.Response_GetState{
				GetFrameworks: PopulateFrameworks("fw-id1"),
				GetExecutors:  PopulateExecutors("task-id1"),
				GetTasks:      PopulateTasks("task-id1", ""),
			},
			expectedError: nil,
			expectedLabels: map[string]string{
				framework:    "TestFramework",
				source:       "source1",
				schedulerSLA: nonRevocable,
				"key1":       "value1",
			},
		},
		{
			frameworkID: "fw-id2",
			executorID:  "exec-id1",
			agentState: &agent.Response_GetState{
				GetFrameworks: PopulateFrameworks("fw-id1"),
				GetExecutors:  PopulateExecutors("exec-id1"),
				GetTasks:      PopulateTasks("task-id1", "exec-id1"),
			},
			expectedError:  fmt.Errorf("framework ID \"fw-id2\" not found: unable to find framework id fw-id2"),
			expectedLabels: map[string]string{},
		},
	} {

		var s state
		s.st = ts.agentState

		actualLabels, err := s.FetchLabels(ts.frameworkID, ts.executorID)
		if ts.expectedError == nil {
			assert.Nil(t, err)
		} else {
			assert.Equal(t, ts.expectedError.Error(), err.Error())
		}
		assert.Equal(t, ts.expectedLabels, actualLabels)
	}
}
