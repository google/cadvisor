// Copyright 2026 Google Inc. All Rights Reserved.
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
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	versionapi "github.com/containerd/containerd/api/services/version/v1"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"github.com/google/cadvisor/container/containerd/namespaces"
)

type testVersionServer struct {
	versionapi.UnimplementedVersionServer

	mu         sync.Mutex
	namespaces []string
}

func (s *testVersionServer) Version(ctx context.Context, _ *emptypb.Empty) (*versionapi.VersionResponse, error) {
	namespace, _ := namespaces.Namespace(ctx)

	s.mu.Lock()
	s.namespaces = append(s.namespaces, namespace)
	s.mu.Unlock()

	return &versionapi.VersionResponse{Version: namespace}, nil
}

func (s *testVersionServer) recordedNamespaces() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, len(s.namespaces))
	copy(out, s.namespaces)
	return out
}

func TestClientCacheSeparatesNamespaces(t *testing.T) {
	resetClientCacheForTest(t)

	socketDir, err := os.MkdirTemp("/tmp", "cadvisor-containerd-")
	if err != nil {
		t.Fatalf("os.MkdirTemp failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(socketDir); err != nil {
			t.Logf("failed to clean up socket dir %q: %v", socketDir, err)
		}
	})
	socketPath := filepath.Join(socketDir, "containerd.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("net.Listen(%q) failed: %v", socketPath, err)
	}
	server := grpc.NewServer()
	versionServer := &testVersionServer{}
	versionapi.RegisterVersionServer(server, versionServer)
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("test containerd server stopped: %v", err)
		}
	}()
	t.Cleanup(server.Stop)

	mobyClient, err := Client(socketPath, "moby")
	if err != nil {
		t.Fatalf("Client(%q, moby) failed: %v", socketPath, err)
	}
	mobyVersion, err := mobyClient.Version(context.Background())
	if err != nil {
		t.Fatalf("moby Version() failed: %v", err)
	}
	if mobyVersion != "moby" {
		t.Fatalf("moby Version() = %q, want moby", mobyVersion)
	}

	k8sClient, err := Client(socketPath, "k8s.io")
	if err != nil {
		t.Fatalf("Client(%q, k8s.io) failed: %v", socketPath, err)
	}
	if mobyClient == k8sClient {
		t.Fatalf("Client returned the same cached client for distinct namespaces")
	}
	k8sVersion, err := k8sClient.Version(context.Background())
	if err != nil {
		t.Fatalf("k8s Version() failed: %v", err)
	}
	if k8sVersion != "k8s.io" {
		t.Fatalf("k8s Version() = %q, want k8s.io", k8sVersion)
	}

	wantNamespaces := []string{"moby", "k8s.io"}
	if gotNamespaces := versionServer.recordedNamespaces(); !reflect.DeepEqual(gotNamespaces, wantNamespaces) {
		t.Fatalf("recorded namespaces = %v, want %v", gotNamespaces, wantNamespaces)
	}
}

func resetClientCacheForTest(t *testing.T) {
	t.Helper()

	ctrdClientsMu.Lock()
	previous := ctrdClients
	ctrdClients = map[clientCacheKey]*cachedClient{}
	ctrdClientsMu.Unlock()

	t.Cleanup(func() {
		ctrdClientsMu.Lock()
		ctrdClients = previous
		ctrdClientsMu.Unlock()
	})
}
