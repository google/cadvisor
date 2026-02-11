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
	"testing"

	"github.com/stretchr/testify/assert"
)

type canHandleAndAccept struct {
	canHandle bool
	canAccept bool
}

// mockSystemd allows overriding systemd detection in tests
func mockSystemd(isSystemd bool) func() {
	original := systemdCheck
	systemdCheck = func() bool { return isSystemd }
	return func() { systemdCheck = original }
}

func TestCanHandleAndAccept(t *testing.T) {
	f := &crioFactory{
		client:             nil,
		cgroupSubsystems:   nil,
		fsInfo:             nil,
		machineInfoFactory: nil,
		storageDriver:      "",
		storageDir:         "",
		includedMetrics:    nil,
	}

	tests := []struct {
		name          string
		isSystemd     bool
		path          string
		wantCanHandle bool
		wantCanAccept bool
	}{
		// Systemd behavior - sandbox containers (without .scope) are filtered
		{
			name:          "systemd: sandbox container without .scope",
			isSystemd:     true,
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f",
			wantCanHandle: true,
			wantCanAccept: false,
		},
		{
			name:          "systemd: regular container with .scope",
			isSystemd:     true,
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.scope",
			wantCanHandle: true,
			wantCanAccept: true,
		},
		// Non-systemd (cgroupfs) behavior - all valid containers accepted
		{
			name:          "cgroupfs: container without .scope",
			isSystemd:     false,
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f",
			wantCanHandle: true,
			wantCanAccept: true,
		},
		{
			name:          "cgroupfs: container with .scope",
			isSystemd:     false,
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.scope",
			wantCanHandle: true,
			wantCanAccept: true,
		},
		// Common cases (same behavior for both systemd and cgroupfs)
		{
			name:          "system-systemd component",
			isSystemd:     true,
			path:          "/system.slice/system-systemd\\\\x2dcoredump.slice",
			wantCanHandle: true,
			wantCanAccept: false,
		},
		{
			name:          "mount cgroup",
			isSystemd:     true,
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.mount",
			wantCanHandle: false,
			wantCanAccept: false,
		},
		{
			name:          "crio-conmon container",
			isSystemd:     true,
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-conmon-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f",
			wantCanHandle: false,
			wantCanAccept: false,
		},
		{
			name:          "invalid container ID",
			isSystemd:     true,
			path:          "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75",
			wantCanHandle: false,
			wantCanAccept: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock systemd detection
			restore := mockSystemd(tt.isSystemd)
			defer restore()

			canHandle, canAccept, err := f.CanHandleAndAccept(tt.path)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCanHandle, canHandle, "canHandle mismatch")
			assert.Equal(t, tt.wantCanAccept, canAccept, "canAccept mismatch")
		})
	}
}
