// Copyright 2022 Google Inc. All Rights Reserved.
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

package podman

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/cadvisor/lib/container"
	"github.com/google/cadvisor/lib/watcher"
)

func TestValidateResponse(t *testing.T) {
	for _, tc := range []struct {
		response *http.Response
		err      error
		expected string
	}{
		{
			response: nil,
			err:      nil,
			expected: "response not present",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusNotFound,
			},
			err:      errors.New("some error"),
			expected: "item not found: some error",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusNotImplemented,
			},
			err:      errors.New("some error"),
			expected: "query not implemented: some error",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusOK,
			},
			err:      errors.New("some error"),
			expected: "some error",
		},
		{
			response: &http.Response{
				StatusCode: http.StatusOK,
			},
			err:      nil,
			expected: "",
		},
	} {
		err := validateResponse(tc.err, tc.response)
		if tc.expected != "" {
			assert.EqualError(t, err, tc.expected)
		} else {
			assert.NoError(t, err)
		}
	}
}

// servePodman points the podman endpoint at a fake API served by handler.
func servePodman(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "podman.sock")
	lis, err := net.Listen("unix", socket)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = lis
	srv.Start()
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})

	oldEndpoint := *endpointFlag
	*endpointFlag = "unix://" + socket
	t.Cleanup(func() { *endpointFlag = oldEndpoint })
}

func TestRegisterStorageRoot(t *testing.T) {
	for _, tc := range []struct {
		name    string
		info    string
		wantDir string
		wantErr bool
	}{
		{
			name:    "from validated info",
			info:    `{"ServerVersion":"5.0.0","Driver":"overlay","DockerRootDir":"/var/lib/containers/storage"}`,
			wantDir: "/var/lib/containers/storage",
		},
		{
			name:    "missing",
			info:    `{"ServerVersion":"5.0.0","Driver":"overlay"}`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			container.ClearContainerHandlerFactories()
			t.Cleanup(container.ClearContainerHandlerFactories)
			servePodman(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/info") {
					_, _ = w.Write([]byte(tc.info))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			})

			_, err := Register(nil, nil, container.MetricSet{})
			factories := container.GetReorderedFactoryList(watcher.Raw)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Empty(t, factories)
				return
			}
			require.NoError(t, err)
			require.Len(t, factories, 1)
			assert.Equal(t, tc.wantDir, factories[0].(*podmanFactory).storageDir)
		})
	}
}
