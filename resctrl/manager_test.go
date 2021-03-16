// +build linux

// Copyright 2021 Google Inc. All Rights Reserved.
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

// Manager tests.
package resctrl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	var testCases = []struct {
		isResctrlInitialized bool
		enabledCMT           bool
		enabledMBM           bool
		err                  string
		expected             Manager
	}{
		{
			true,
			true,
			false,
			"",
			&manager{interval: 0},
		},
		{
			true,
			false,
			true,
			"",
			&manager{interval: 0},
		},
		{
			true,
			true,
			true,
			"",
			&manager{interval: 0},
		},
		{
			false,
			true,
			false,
			"the resctrl isn't initialized",
			&NoopManager{},
		},
		{
			false,
			false,
			true,
			"the resctrl isn't initialized",
			&NoopManager{},
		},
		{
			false,
			true,
			true,
			"the resctrl isn't initialized",
			&NoopManager{},
		},
		{
			false,
			false,
			false,
			"the resctrl isn't initialized",
			&NoopManager{},
		},
		{
			true,
			false,
			false,
			"there are no monitoring features available",
			&NoopManager{},
		},
	}

	for _, test := range testCases {
		setup := func() error {
			isResctrlInitialized = test.isResctrlInitialized
			enabledCMT = test.enabledCMT
			enabledMBM = test.enabledMBM
			return nil
		}
		got, err := NewManager(0, setup)
		assert.Equal(t, got, test.expected)
		checkError(t, err, test.err)
	}
}

func TestGetCollector(t *testing.T) {
	rootResctrl = mockResctrl()
	defer os.RemoveAll(rootResctrl)

	pidsPath = mockContainersPids()
	defer os.RemoveAll(pidsPath)

	processPath = mockProcFs()
	defer os.RemoveAll(processPath)

	expectedID := "container"

	setup := func() error {
		isResctrlInitialized = true
		enabledCMT = true
		enabledMBM = true
		return nil
	}
	manager, err := NewManager(0, setup)
	assert.NoError(t, err)

	_, err = manager.GetCollector(expectedID, mockGetContainerPids, 2)
	assert.NoError(t, err)
}
