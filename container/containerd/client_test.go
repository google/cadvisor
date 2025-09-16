// Copyright 2017 Google Inc. All Rights Reserved.
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

package containerd

import (
	"context"
	"fmt"

	"github.com/google/cadvisor/container/containerd/containers"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type containerdClientMock struct {
	cntrs     map[string]*containers.Container
	returnErr error
}

func (c *containerdClientMock) LoadContainer(ctx context.Context, id string) (*containers.Container, error) {
	if c.returnErr != nil {
		return nil, c.returnErr
	}
	cntr, ok := c.cntrs[id]
	if !ok {
		return nil, fmt.Errorf("unable to find container %q", id)
	}
	return cntr, nil
}

func (c *containerdClientMock) Version(ctx context.Context) (string, error) {
	return "test-v0.0.0", nil
}

func (c *containerdClientMock) TaskPid(ctx context.Context, id string) (uint32, error) {
	return 2389, nil
}

func (c *containerdClientMock) ContainerStats(ctx context.Context, id string) (*runtimeapi.ContainerStats, error) {
	if c.returnErr != nil {
		return nil, c.returnErr
	}
	// Return mock stats with filesystem usage
	return &runtimeapi.ContainerStats{
		Attributes: &runtimeapi.ContainerAttributes{
			Id: id,
		},
		WritableLayer: &runtimeapi.FilesystemUsage{
			UsedBytes:  &runtimeapi.UInt64Value{Value: 1024 * 1024}, // 1MB
			InodesUsed: &runtimeapi.UInt64Value{Value: 100},
		},
	}, nil
}

func mockcontainerdClient(cntrs map[string]*containers.Container, returnErr error) ContainerdClient {
	return &containerdClientMock{
		cntrs:     cntrs,
		returnErr: returnErr,
	}
}
