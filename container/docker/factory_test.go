// Copyright 2016 Google Inc. All Rights Reserved.
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

package docker

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/cadvisor/lib/container"
	"github.com/google/cadvisor/lib/watcher"
)

func TestEnsureThinLsKernelVersion(t *testing.T) {
	tests := []struct {
		version       string
		expectedError string
	}{
		{"4.4.0-31-generic", ""},
		{"4.4.1", ""},
		{"4.6.4-301.fc24.x86_64", ""},
		{"3.10.0-327.22.2.el7.x86_64", `RHEL/Centos 7.x kernel version 3.10.0-366 or later is required to use thin_ls - you have "3.10.0-327.22.2.el7.x86_64"`},
		{"3.10.0-366.el7.x86_64", ""},
		{"3.10.0-366.el7_3.x86_64", ""},
		{"3.10.0.el7.abc", `unable to determine RHEL/Centos 7.x kernel release from "3.10.0.el7.abc"`},
		{"3.10.0-abc.el7.blarg", `unable to determine RHEL/Centos 7.x kernel release from "3.10.0-abc.el7.blarg"`},
		{"3.10.0-367.el7.x86_64", ""},
		{"3.10.0-366.x86_64", `kernel version 4.4.0 or later is required to use thin_ls - you have "3.10.0-366.x86_64"`},
		{"3.10.1-1.el7.x86_64", ""},
		{"2.0.36", `kernel version 4.4.0 or later is required to use thin_ls - you have "2.0.36"`},
		{"2.1", `error parsing kernel version: "2.1" is not a semver`},
	}

	for _, test := range tests {
		err := ensureThinLsKernelVersion(test.version)
		if err != nil {
			if len(test.expectedError) == 0 {
				t.Errorf("%s: expected no error, got %v", test.version, err)
			} else if err.Error() != test.expectedError {
				t.Errorf("%s: expected error %v, got %v", test.version, test.expectedError, err)
			}
		} else if err == nil && len(test.expectedError) > 0 {
			t.Errorf("%s: expected error %v", test.version, test.expectedError)
		}
	}
}

func TestCanHandleAndAcceptBoundsInspect(t *testing.T) {
	f := &dockerFactory{client: newHangingDockerClient(t)}

	type result struct {
		handle, accept bool
		err            error
	}
	done := make(chan result, 1)
	go func() {
		handle, accept, err := f.CanHandleAndAccept("/docker/" + testDockerContainerID)
		done <- result{handle, accept, err}
	}()
	select {
	case r := <-done:
		if r.handle || r.err == nil {
			t.Errorf("CanHandleAndAccept() = %v, %v, %v; want false and an error", r.handle, r.accept, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("CanHandleAndAccept did not time out the inspect call")
	}
}

func TestRegisterStorageRoot(t *testing.T) {
	// Client() caches its client, so drop it before and after the test to make
	// it connect to the fake daemon. The subtests share the daemon and switch
	// the info it returns.
	resetClient := func() {
		dockerClientOnce = sync.Once{}
		dockerClient, dockerClientErr = nil, nil
	}
	resetClient()
	t.Cleanup(resetClient)
	var info atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/info") {
			_, _ = w.Write([]byte(info.Load().(string)))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	oldEndpoint := *ArgDockerEndpoint
	*ArgDockerEndpoint = "tcp://" + srv.Listener.Addr().String()
	t.Cleanup(func() { *ArgDockerEndpoint = oldEndpoint })

	for _, tc := range []struct {
		name    string
		info    string
		wantDir string
	}{
		{
			name:    "from validated info",
			info:    `{"ServerVersion":"28.5.1","Driver":"overlay2","DockerRootDir":"/srv/docker"}`,
			wantDir: "/srv/docker",
		},
		{
			name:    "falls back to --docker_root",
			info:    `{"ServerVersion":"28.5.1","Driver":"overlay2"}`,
			wantDir: *dockerRootDirFlag,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			container.ClearContainerHandlerFactories()
			t.Cleanup(container.ClearContainerHandlerFactories)
			info.Store(tc.info)

			require.NoError(t, Register(nil, nil, container.MetricSet{}))
			factories := container.GetReorderedFactoryList(watcher.Raw)
			require.Len(t, factories, 1)
			assert.Equal(t, tc.wantDir, factories[0].(*dockerFactory).storageDir)
		})
	}
}
