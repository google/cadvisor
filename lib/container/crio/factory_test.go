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

package crio

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanHandleAndAccept(t *testing.T) {
	tests := []struct {
		name          string
		cgroupDriver  string
		path          string
		wantCanHandle bool
		wantCanAccept bool
	}{
		// Systemd behavior - sandbox containers (without .scope) are filtered
		{
			name:          "systemd: sandbox container without .scope",
			cgroupDriver:  "systemd",
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f",
			wantCanHandle: true,
			wantCanAccept: false,
		},
		{
			name:          "systemd: regular container with .scope",
			cgroupDriver:  "systemd",
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.scope",
			wantCanHandle: true,
			wantCanAccept: true,
		},
		// Non-systemd (cgroupfs) behavior - all valid containers accepted
		{
			name:          "cgroupfs: container without .scope",
			cgroupDriver:  "cgroupfs",
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f",
			wantCanHandle: true,
			wantCanAccept: true,
		},
		{
			name:          "cgroupfs: container with .scope",
			cgroupDriver:  "cgroupfs",
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.scope",
			wantCanHandle: true,
			wantCanAccept: true,
		},
		// Common cases (same behavior for both systemd and cgroupfs)
		{
			name:          "system-systemd component",
			cgroupDriver:  "systemd",
			path:          "/system.slice/system-systemd\\x2dcoredump.slice",
			wantCanHandle: true,
			wantCanAccept: false,
		},
		{
			name:          "mount cgroup",
			cgroupDriver:  "systemd",
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.mount",
			wantCanHandle: false,
			wantCanAccept: false,
		},
		{
			name:          "crio-conmon container",
			cgroupDriver:  "systemd",
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-conmon-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f",
			wantCanHandle: false,
			wantCanAccept: false,
		},
		{
			name:          "invalid container ID",
			cgroupDriver:  "systemd",
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75",
			wantCanHandle: false,
			wantCanAccept: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &crioFactory{
				cgroupDriver: tt.cgroupDriver,
			}

			canHandle, canAccept, err := f.CanHandleAndAccept(tt.path)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCanHandle, canHandle, "canHandle mismatch")
			assert.Equal(t, tt.wantCanAccept, canAccept, "canAccept mismatch")
		})
	}
}

// newHangingCrioClient returns a client for a CRI-O socket that accepts
// requests but never answers them, and lowers crioRegistrationTimeout.
func newHangingCrioClient(t *testing.T) CrioClient {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "crio.sock")
	lis, err := net.Listen("unix", socket)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	srv.Listener = lis
	srv.Start()
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})

	tr := new(http.Transport)
	require.NoError(t, configureUnixTransport(tr, "unix", socket))

	oldTimeout := crioRegistrationTimeout
	crioRegistrationTimeout = 100 * time.Millisecond
	t.Cleanup(func() { crioRegistrationTimeout = oldTimeout })

	// No client timeout, as with the default --crio_client_timeout=0.
	return &crioClientImpl{client: &http.Client{Transport: tr}}
}

func TestRegistrationCallsAreBounded(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(CrioClient) error
	}{
		{"Info", func(c CrioClient) error {
			_, err := boundedInfo(c)
			return err
		}},
		{"ContainerInfo", func(c CrioClient) error {
			_, err := boundedContainerInfo(c, "id")
			return err
		}},
		{"handler construction", func(c CrioClient) error {
			_, err := newCrioContainerHandler(c, "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f", nil, nil, "", "", nil, false, nil, nil)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newHangingCrioClient(t)
			done := make(chan error, 1)
			go func() { done <- tc.call(client) }()
			select {
			case err := <-done:
				assert.ErrorIs(t, err, context.DeadlineExceeded)
			case <-time.After(5 * time.Second):
				t.Fatal("the call did not time out")
			}
		})
	}
}
