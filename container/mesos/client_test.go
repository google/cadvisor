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

import "fmt"

type FakeMesosAgentClient struct {
	cntrInfo map[string]*containerInfo
	err      error
}

func (c *FakeMesosAgentClient) ContainerInfo(id string) (*containerInfo, error) {
	if c.err != nil {
		return nil, c.err
	}
	cInfo, ok := c.cntrInfo[id]
	if !ok {
		return nil, fmt.Errorf("can't locate container %s", id)
	}
	return cInfo, nil
}

func (c *FakeMesosAgentClient) ContainerPid(id string) (int, error) {
	if c.err != nil {
		return invalidPID, c.err
	}
	cInfo, ok := c.cntrInfo[id]
	if !ok {
		return invalidPID, fmt.Errorf("can't locate container %s", id)
	}

	if cInfo.cntr.ContainerStatus == nil {
		return invalidPID, fmt.Errorf("error fetching Pid")
	}

	pid := int(*cInfo.cntr.ContainerStatus.ExecutorPID)
	return pid, nil
}

func fakeMesosAgentClient(cntrInfo map[string]*containerInfo, err error) mesosAgentClient {
	return &FakeMesosAgentClient{
		err:      err,
		cntrInfo: cntrInfo,
	}
}
