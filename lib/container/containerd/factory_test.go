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

package containerd

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	versionapi "github.com/containerd/containerd/api/services/version/v1"
	"github.com/containerd/typeurl/v2"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"github.com/google/cadvisor/lib/container/containerd/containers"
)

func TestIsContainerName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "/system.slice/run-containerd-io.containerd.runtime.v1.linux-k8s.io-14ae50f1d3ada102aec3ab00168fdafb2dc0986d79ca9e8d5b75581fa89e9fea-rootfs.mount",
			expected: false,
		},
		{
			name:     "/kubepods/besteffort/podd76e26fba3bf2bfd215eb29011d55250/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
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
	testContainers := make(map[string]*containers.Container)
	testContainer := &containers.Container{
		ID:     "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
		Labels: map[string]string{"io.cri-containerd.kind": "sandbox"},
	}
	spec := &specs.Spec{Root: &specs.Root{Path: "/test/"}, Process: &specs.Process{}}
	testContainer.Spec, _ = typeurl.MarshalAnyToProto(spec)
	testContainers["40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"] = testContainer

	f := &containerdFactory{
		client:             mockcontainerdClient(testContainers, nil),
		cgroupSubsystems:   nil,
		fsInfo:             nil,
		machineInfoFactory: nil,
		includedMetrics:    nil,
	}
	for k, v := range map[string]bool{
		"/kubepods/besteffort/podd76e26fba3bf2bfd215eb29011d55250/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9":                        true,
		"/system.slice/run-containerd-io.containerd.runtime.v1.linux-k8s.io-14ae50f1d3ada102aec3ab00168fdafb2dc0986d79ca9e8d5b75581fa89e9fea-rootfs.mount": false,
	} {
		b1, b2, err := f.CanHandleAndAccept(k)
		as.Nil(err)
		as.Equal(b1, v)
		as.Equal(b2, v)
	}
}

// hangingVersionServer accepts Version calls and never answers them.
type hangingVersionServer struct {
	versionapi.UnimplementedVersionServer
}

func (hangingVersionServer) Version(ctx context.Context, _ *emptypb.Empty) (*versionapi.VersionResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRegisterBoundsVersion(t *testing.T) {
	resetClientCache(t)
	socket := filepath.Join(t.TempDir(), "containerd.sock")
	serveContainerd(t, socket, hangingVersionServer{})

	oldEndpoint := *ArgContainerdEndpoint
	*ArgContainerdEndpoint = socket
	t.Cleanup(func() { *ArgContainerdEndpoint = oldEndpoint })

	done := make(chan error, 1)
	go func() { done <- Register(nil, nil, nil) }()
	select {
	case err := <-done:
		assert.ErrorContains(t, err, "failed to fetch containerd client version")
	case <-time.After(3*connectionTimeout + 5*time.Second):
		t.Fatal("Register did not time out the Version call")
	}
}
