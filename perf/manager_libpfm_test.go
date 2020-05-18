// +build libpfm,cgo

// Copyright 2020 Google Inc. All Rights Reserved.
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

// Manager of perf events for containers.
package perf

import (
	"testing"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/stats"

	"github.com/stretchr/testify/assert"
)

func TestNoConfigFilePassed(t *testing.T) {
	manager, err := NewManager("", 1, []info.Node{})

	assert.Nil(t, err)
	_, ok := manager.(*stats.NoopManager)
	assert.True(t, ok)
}

func TestNonExistentFile(t *testing.T) {
	manager, err := NewManager("this-file-is-so-non-existent", 1, []info.Node{})

	assert.NotNil(t, err)
	assert.Nil(t, manager)
}

func TestMalformedJsonFile(t *testing.T) {
	manager, err := NewManager("testing/this-is-some-random.json", 1, []info.Node{})

	assert.NotNil(t, err)
	assert.Nil(t, manager)
}

func TestGroupedEvents(t *testing.T) {
	manager, err := NewManager("testing/grouped.json", 1, []info.Node{})

	assert.NotNil(t, err)
	assert.Nil(t, manager)
}

func TestNewManager(t *testing.T) {
	managerInstance, err := NewManager("testing/perf.json", 1, []info.Node{})

	assert.Nil(t, err)
	_, ok := managerInstance.(*manager)
	assert.True(t, ok)
}
