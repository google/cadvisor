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
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	containersapi "github.com/containerd/containerd/api/services/containers/v1"
	tasksapi "github.com/containerd/containerd/api/services/tasks/v1"
	versionapi "github.com/containerd/containerd/api/services/version/v1"
	tasktypes "github.com/containerd/containerd/api/types/task"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"github.com/google/cadvisor/lib/container/containerd/containers"
	"github.com/google/cadvisor/lib/container/containerd/namespaces"
	"github.com/google/cadvisor/lib/container/containerd/pkg/dialer"
)

type client struct {
	// namespace is set on each call's context, unless the caller set one.
	namespace        string
	containerService containersapi.ContainersClient
	taskService      tasksapi.TasksClient
	versionService   versionapi.VersionClient
}

type ContainerdClient interface {
	LoadContainer(ctx context.Context, id string) (*containers.Container, error)
	TaskPid(ctx context.Context, id string) (uint32, error)
	LoadTaskProcess(ctx context.Context, id string) (*tasktypes.Process, error)
	TaskExitStatus(ctx context.Context, id string) (uint32, error)
	Version(ctx context.Context) (string, error)
}

var (
	ErrTaskIsInUnknownState = errors.New("containerd task is in unknown state") // used when process reported in containerd task is in Unknown State
	// ErrNotFound is returned when containerd reports the requested object does
	// not exist (gRPC codes.NotFound). It replaces the dependency on containerd's
	// errdefs package; the sole consumer is the retriable-error check in handler.go.
	ErrNotFound = errors.New("not found")
)

// toNative normalizes a gRPC error from the containerd API. A codes.NotFound
// status is wrapped as ErrNotFound so callers can detect it with errors.Is;
// every other error is returned unchanged (its message is preserved).
func toNative(err error) error {
	if status.Code(err) == codes.NotFound {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	return err
}

// conns holds one connection per containerd address, shared by the clients of
// every namespace. A failed dial isn't cached, so a containerd that starts
// later is picked up when registration is retried.
var (
	connsMu sync.Mutex
	conns   = map[string]*grpc.ClientConn{}
)

const (
	maxBackoffDelay   = 3 * time.Second
	baseBackoffDelay  = 100 * time.Millisecond
	connectionTimeout = 2 * time.Second
	maxMsgSize        = 16 * 1024 * 1024 // 16MB
)

// Client returns a containerd client whose calls use the given namespace,
// dialing containerd if no connection to address has succeeded yet.
func Client(address, namespace string) (ContainerdClient, error) {
	conn, err := connection(address)
	if err != nil {
		return nil, err
	}
	return &client{
		namespace:        namespace,
		containerService: containersapi.NewContainersClient(conn),
		taskService:      tasksapi.NewTasksClient(conn),
		versionService:   versionapi.NewVersionClient(conn),
	}, nil
}

func connection(address string) (*grpc.ClientConn, error) {
	connsMu.Lock()
	defer connsMu.Unlock()
	if conn, ok := conns[address]; ok {
		return conn, nil
	}
	conn, err := dial(address)
	if err != nil {
		return nil, err
	}
	conns[address] = conn
	return conn, nil
}

func dial(address string) (*grpc.ClientConn, error) {
	tryConn, err := net.DialTimeout("unix", address, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("containerd: cannot unix dial containerd api service: %v", err)
	}
	tryConn.Close()

	connParams := grpc.ConnectParams{
		Backoff: backoff.DefaultConfig,
	}
	connParams.Backoff.BaseDelay = baseBackoffDelay
	connParams.Backoff.MaxDelay = maxBackoffDelay
	//nolint:staticcheck // SA1019
	gopts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer.ContextDialer),
		grpc.WithBlock(),
		grpc.WithConnectParams(connParams),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	//nolint:staticcheck // SA1019
	return grpc.DialContext(ctx, dialer.DialAddress(address), gopts...)
}

// withNamespace sets the client's namespace on ctx, unless the caller already
// set one.
func (c *client) withNamespace(ctx context.Context) context.Context {
	if _, ok := namespaces.Namespace(ctx); ok {
		return ctx
	}
	return namespaces.WithNamespace(ctx, c.namespace)
}

func (c *client) LoadContainer(ctx context.Context, id string) (*containers.Container, error) {
	r, err := c.containerService.Get(c.withNamespace(ctx), &containersapi.GetContainerRequest{
		ID: id,
	})
	if err != nil {
		return nil, toNative(err)
	}
	return containerFromProto(r.Container), nil
}

func (c *client) TaskPid(ctx context.Context, id string) (uint32, error) {
	response, err := c.taskService.Get(c.withNamespace(ctx), &tasksapi.GetRequest{
		ContainerID: id,
	})
	if err != nil {
		return 0, toNative(err)
	}
	if response.Process.Status == tasktypes.Status_UNKNOWN {
		return 0, ErrTaskIsInUnknownState
	}
	return response.Process.Pid, nil
}

func (c *client) LoadTaskProcess(ctx context.Context, id string) (*tasktypes.Process, error) {
	response, err := c.taskService.Get(c.withNamespace(ctx), &tasksapi.GetRequest{
		ContainerID: id,
	})
	if err != nil {
		return nil, toNative(err)
	}

	return response.Process, nil
}

func (c *client) TaskExitStatus(ctx context.Context, id string) (uint32, error) {
	response, err := c.taskService.Get(c.withNamespace(ctx), &tasksapi.GetRequest{
		ContainerID: id,
	})
	if err != nil {
		return 0, toNative(err)
	}
	if response.Process.Status != tasktypes.Status_STOPPED {
		return 0, fmt.Errorf("container %s has not exited (status: %v)", id, response.Process.Status)
	}
	return response.Process.ExitStatus, nil
}

func (c *client) Version(ctx context.Context) (string, error) {
	response, err := c.versionService.Version(c.withNamespace(ctx), &emptypb.Empty{})
	if err != nil {
		return "", toNative(err)
	}
	return response.Version, nil
}

func containerFromProto(containerpb *containersapi.Container) *containers.Container {
	var runtime containers.RuntimeInfo
	var createdAt time.Time
	// TODO: is nil check required for containerpb
	if containerpb.Runtime != nil {
		runtime = containers.RuntimeInfo{
			Name:    containerpb.Runtime.Name,
			Options: containerpb.Runtime.Options,
		}
	}
	if containerpb.GetCreatedAt() != nil {
		createdAt = containerpb.GetCreatedAt().AsTime()
	}
	return &containers.Container{
		ID:          containerpb.ID,
		Labels:      containerpb.Labels,
		Image:       containerpb.Image,
		Runtime:     runtime,
		Spec:        containerpb.Spec,
		Snapshotter: containerpb.Snapshotter,
		SnapshotKey: containerpb.SnapshotKey,
		Extensions:  containerpb.Extensions,
		CreatedAt:   createdAt,
	}
}
