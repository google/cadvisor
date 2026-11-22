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

//go:build linux

// Handler for containerd containers.
package containerd

import (
	"context"
	"testing"
	"time"

	"github.com/containerd/typeurl/v2"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"

	"github.com/google/cadvisor/lib/container"
	"github.com/google/cadvisor/lib/container/containerd/containers"
	"github.com/google/cadvisor/lib/fs"
	info "github.com/google/cadvisor/lib/model"
)

func init() {
	typeurl.Register(&specs.Spec{}, "types.contianerd.io/opencontainers/runtime-spec", "v1", "Spec")
}

type mockedMachineInfo struct{}

func (m *mockedMachineInfo) GetMachineInfo() (*info.MachineInfo, error) {
	return &info.MachineInfo{}, nil
}

func (m *mockedMachineInfo) GetVersionInfo() (*info.VersionInfo, error) {
	return &info.VersionInfo{}, nil
}

func TestHandler(t *testing.T) {
	as := assert.New(t)
	type testCase struct {
		client               ContainerdClient
		name                 string
		machineInfoFactory   info.MachineInfoFactory
		fsInfo               fs.FsInfo
		cgroupSubsystems     map[string]string
		inHostNamespace      bool
		metadataEnvAllowList []string
		includedMetrics      container.MetricSet

		hasErr         bool
		errContains    string
		checkReference *info.ContainerReference
		checkEnvVars   map[string]string
	}
	testContainers := make(map[string]*containers.Container)
	testContainer := &containers.Container{
		ID:     "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
		Labels: map[string]string{"io.cri-containerd.kind": "sandbox"},
	}
	spec := &specs.Spec{Root: &specs.Root{Path: "/test/"}, Process: &specs.Process{Env: []string{"TEST_REGION=FRA", "TEST_ZONE=A", "HELLO=WORLD"}}}
	testContainer.Spec, _ = typeurl.MarshalAnyToProto(spec)
	testContainers["40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"] = testContainer
	for _, ts := range []testCase{
		{
			mockcontainerdClient(nil, nil),
			"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
			nil,
			nil,
			nil,
			false,
			nil,
			nil,
			true,
			"unable to find container \"40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9\"",
			nil,
			nil,
		},
		{
			mockcontainerdClient(testContainers, nil),
			"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
			&mockedMachineInfo{},
			nil,
			nil,
			false,
			nil,
			nil,
			false,
			"",
			&info.ContainerReference{
				Id:        "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Name:      "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Aliases:   []string{"40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9", "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"},
				Namespace: k8sContainerdNamespace,
			},
			map[string]string{},
		},
		{
			mockcontainerdClient(testContainers, nil),
			"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
			&mockedMachineInfo{},
			nil,
			nil,
			false,
			[]string{"TEST"},
			nil,
			false,
			"",
			&info.ContainerReference{
				Id:        "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Name:      "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Aliases:   []string{"40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9", "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"},
				Namespace: k8sContainerdNamespace,
			},
			map[string]string{"TEST_REGION": "FRA", "TEST_ZONE": "A"},
		},
	} {
		handler, err := newContainerdContainerHandler(ts.client, ts.name, ts.machineInfoFactory, ts.fsInfo, ts.cgroupSubsystems, ts.inHostNamespace, ts.metadataEnvAllowList, ts.includedMetrics)
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
		if ts.checkEnvVars != nil {
			sp, err := handler.GetSpec()
			as.Nil(err)
			as.Equal(ts.checkEnvVars, sp.Envs)
		}
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name         string
		exitStatus   uint32
		returnErr    error
		expectErr    bool
		errContains  string
		expectedCode int
	}{
		{
			name:         "successful exit code 0",
			exitStatus:   0,
			returnErr:    nil,
			expectErr:    false,
			expectedCode: 0,
		},
		{
			name:         "successful exit code 1",
			exitStatus:   1,
			returnErr:    nil,
			expectErr:    false,
			expectedCode: 1,
		},
		{
			name:         "task not stopped",
			exitStatus:   0,
			returnErr:    assert.AnError,
			expectErr:    true,
			expectedCode: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := assert.New(t)

			mockClient := &containerdClientMock{
				returnErr:  tt.returnErr,
				exitStatus: tt.exitStatus,
			}

			h := &containerdContainerHandler{
				client: mockClient,
				reference: info.ContainerReference{
					Id: "test-container-id",
				},
			}

			code, err := h.GetExitCode()

			if tt.expectErr {
				as.Error(err)
				as.Equal(tt.expectedCode, code)
			} else {
				as.NoError(err)
				as.Equal(tt.expectedCode, code)
			}
		})
	}
}

const testContainerdContainerName = "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"

type hangingLoadContainerClient struct {
	containerdClientMock
}

func (c *hangingLoadContainerClient) LoadContainer(ctx context.Context, id string) (*containers.Container, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestHandlerBoundsLoadContainer(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		_, err := newContainerdContainerHandler(&hangingLoadContainerClient{}, testContainerdContainerName, &mockedMachineInfo{}, nil, nil, false, nil, nil)
		done <- err
	}()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(connectionTimeout + 5*time.Second):
		t.Fatal("newContainerdContainerHandler did not time out LoadContainer")
	}
}

// slowTaskClient reports the task as not found a few times, and records how
// much time each TaskPid call had left on its context.
type slowTaskClient struct {
	containerdClientMock
	notFound  int
	remaining []time.Duration
}

func (c *slowTaskClient) TaskPid(ctx context.Context, id string) (uint32, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Hour)
	}
	c.remaining = append(c.remaining, time.Until(deadline))
	if len(c.remaining) <= c.notFound {
		return 0, ErrNotFound
	}
	return 2389, nil
}

func TestHandlerTaskPidRetryGetsFreshDeadlines(t *testing.T) {
	testContainer := &containers.Container{ID: "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"}
	testContainer.Spec, _ = typeurl.MarshalAnyToProto(&specs.Spec{Root: &specs.Root{Path: "/test/"}, Process: &specs.Process{}})
	mock := mockcontainerdClient(map[string]*containers.Container{testContainer.ID: testContainer}, nil).(*containerdClientMock)

	// The backoff sleeps 100+200+400+800ms before the fifth attempt, more than
	// connectionTimeout, so a deadline shared across attempts would expire.
	client := &slowTaskClient{containerdClientMock: *mock, notFound: 4}
	_, err := newContainerdContainerHandler(client, testContainerdContainerName, &mockedMachineInfo{}, nil, nil, false, nil, nil)
	assert.NoError(t, err)
	assert.Len(t, client.remaining, 5)
	for i, remaining := range client.remaining {
		assert.Greater(t, remaining, connectionTimeout/2, "attempt %d", i+1)
		assert.LessOrEqual(t, remaining, connectionTimeout, "attempt %d", i+1)
	}
}
