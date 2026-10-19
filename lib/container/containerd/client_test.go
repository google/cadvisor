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
	"net"
	"path/filepath"
	"testing"

	versionapi "github.com/containerd/containerd/api/services/version/v1"
	"github.com/containerd/containerd/api/types/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/google/cadvisor/lib/container/containerd/containers"
)

type containerdClientMock struct {
	cntrs      map[string]*containers.Container
	returnErr  error
	tasks      map[string]*task.Process
	exitStatus uint32
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

func (c *containerdClientMock) LoadTaskProcess(ctx context.Context, id string) (*task.Process, error) {
	if c.returnErr != nil {
		return nil, c.returnErr
	}
	task, ok := c.tasks[id]
	if !ok {
		return nil, fmt.Errorf("unable to find task for container %q", id)
	}
	return task, nil
}

func (c *containerdClientMock) TaskExitStatus(ctx context.Context, id string) (uint32, error) {
	if c.returnErr != nil {
		return 0, c.returnErr
	}
	return c.exitStatus, nil
}

func mockcontainerdClient(cntrs map[string]*containers.Container, returnErr error) ContainerdClient {
	tasks := make(map[string]*task.Process)

	for _, cntr := range cntrs {
		tasks[cntr.ID] = &task.Process{}
	}

	return &containerdClientMock{
		cntrs:     cntrs,
		returnErr: returnErr,
		tasks:     tasks,
	}
}

// resetClientCache drops the cached client before and after the test.
func resetClientCache(t *testing.T) {
	t.Helper()
	reset := func() {
		ctrdClientMu.Lock()
		ctrdClient = nil
		ctrdClientMu.Unlock()
	}
	reset()
	t.Cleanup(reset)
}

// serveContainerd serves the given version service on a unix socket.
func serveContainerd(t *testing.T, socket string, version versionapi.VersionServer) {
	t.Helper()
	lis, err := net.Listen("unix", socket)
	require.NoError(t, err)
	srv := grpc.NewServer()
	versionapi.RegisterVersionServer(srv, version)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
}

func TestClientRetriesFailedDial(t *testing.T) {
	resetClientCache(t)
	socket := filepath.Join(t.TempDir(), "containerd.sock")

	_, err := Client(socket, "k8s.io")
	require.Error(t, err)

	serveContainerd(t, socket, &versionapi.UnimplementedVersionServer{})
	c, err := Client(socket, "k8s.io")
	require.NoError(t, err)
	require.NotNil(t, c)

	again, err := Client(socket, "k8s.io")
	require.NoError(t, err)
	assert.Same(t, c, again)
}
