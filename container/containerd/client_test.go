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

	types "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/containers"
	criapi "github.com/google/cadvisor/cri-api/pkg/apis/runtime/v1alpha2"
)

type containerdClientMock struct {
	cntrs     map[string]*containers.Container
	status    *criapi.ContainerStatus
	stats     *criapi.ContainerStats
	mounts    []*types.Mount
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

func (c *containerdClientMock) ContainerStatus(ctx context.Context, id string) (*criapi.ContainerStatus, error) {
	return c.status, nil
}

func (c *containerdClientMock) ContainerStats(ctx context.Context, id string) (*criapi.ContainerStats, error) {
	return c.stats, nil
}

func (c *containerdClientMock) SnapshotMounts(ctx context.Context, snapshotter, key string) ([]*types.Mount, error) {
	return c.mounts, nil
}

func mockcontainerdClient(cntrs map[string]*containers.Container, status *criapi.ContainerStatus, stats *criapi.ContainerStats, mounts []*types.Mount, returnErr error) ContainerdClient {
	return &containerdClientMock{
		cntrs:     cntrs,
		status:    status,
		stats:     stats,
		mounts:    mounts,
		returnErr: returnErr,
	}
}
