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

//go:build linux

package raw

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	inotify "k8s.io/utils/inotify"

	"github.com/google/cadvisor/lib/container/common"
	"github.com/google/cadvisor/lib/watcher"
)

func TestProcessEventIgnoresDisappearedContainer(t *testing.T) {
	cgroupRoot := t.TempDir()
	containerPath := filepath.Join(cgroupRoot, "transient")
	if err := os.Mkdir(containerPath, 0o755); err != nil {
		t.Fatalf("os.Mkdir(%q) failed: %v", containerPath, err)
	}
	if err := os.Remove(containerPath); err != nil {
		t.Fatalf("os.Remove(%q) failed: %v", containerPath, err)
	}

	rawWatcher := newTestRawContainerWatcher(t, cgroupRoot)
	events := make(chan watcher.ContainerEvent, 1)
	event := &inotify.Event{
		Name: containerPath,
		Mask: inotify.InCreate | inotify.InIsdir,
	}

	if err := rawWatcher.processEvent(event, events); err != nil {
		t.Errorf("processEvent(%+v) failed: %v, want nil", event, err)
	}
	if got := len(events); got != 0 {
		t.Errorf("processEvent(%+v) emitted %d events, want 0", event, got)
	}
}

func TestProcessEventPreservesNonNotExistError(t *testing.T) {
	cgroupRoot := t.TempDir()
	containerPath := filepath.Join(cgroupRoot, "not-a-directory")
	if err := os.WriteFile(containerPath, nil, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) failed: %v", containerPath, err)
	}

	rawWatcher := newTestRawContainerWatcher(t, cgroupRoot)
	events := make(chan watcher.ContainerEvent, 1)
	event := &inotify.Event{
		Name: containerPath,
		Mask: inotify.InCreate | inotify.InIsdir,
	}

	err := rawWatcher.processEvent(event, events)
	if !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("processEvent(%+v) error = %v, want %v", event, err, syscall.ENOTDIR)
	}
}

func newTestRawContainerWatcher(t *testing.T, cgroupRoot string) *rawContainerWatcher {
	t.Helper()

	inotifyWatcher, err := common.NewInotifyWatcher()
	if err != nil {
		t.Fatalf("common.NewInotifyWatcher() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := inotifyWatcher.Close(); err != nil {
			t.Errorf("InotifyWatcher.Close() failed: %v", err)
		}
	})

	return &rawContainerWatcher{
		cgroupPaths: map[string]string{"test": cgroupRoot},
		watcher:     inotifyWatcher,
	}
}
