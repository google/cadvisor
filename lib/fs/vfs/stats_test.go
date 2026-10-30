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

package vfs

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"testing/synctest"
	"time"
)

var sampleStatfs = syscall.Statfs_t{Frsize: 4096, Blocks: 100, Bfree: 40, Bavail: 30, Files: 1000, Ffree: 900}

// stubStatfs replaces the syscall and the timeout for one test.
func stubStatfs(t *testing.T, f func(string, *syscall.Statfs_t) error, timeout time.Duration) {
	t.Helper()
	origStatfs, origTimeout := statfs, statfsTimeout
	statfs, statfsTimeout = f, timeout
	t.Cleanup(func() { statfs, statfsTimeout = origStatfs, origTimeout })
}

func TestGetVfsStats(t *testing.T) {
	stubStatfs(t, func(_ string, s *syscall.Statfs_t) error {
		*s = sampleStatfs
		return nil
	}, 2*time.Second)

	total, free, avail, inodes, inodesFree, err := GetVfsStats("/mnt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := [5]uint64{total, free, avail, inodes, inodesFree}
	want := [5]uint64{409600, 163840, 122880, 1000, 900}
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestGetVfsStatsError(t *testing.T) {
	errStatfs := errors.New("statfs failed")
	stubStatfs(t, func(string, *syscall.Statfs_t) error { return errStatfs }, 2*time.Second)

	total, free, avail, inodes, inodesFree, err := GetVfsStats("/mnt")
	if !errors.Is(err, errStatfs) {
		t.Errorf("got error %v, want %v", err, errStatfs)
	}
	if got := [5]uint64{total, free, avail, inodes, inodesFree}; got != [5]uint64{} {
		t.Errorf("got %v, want all zero", got)
	}
}

// TestGetVfsStatsTimeout runs with fake time: the stub returns 50 ms after the
// 10 ms timeout, so the caller must get context.DeadlineExceeded with zero
// values while the worker still finishes and exits without touching them.
// With -race this fails on the old code, which wrote the named results from
// the worker.
func TestGetVfsStatsTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stubStatfs(t, func(_ string, s *syscall.Statfs_t) error {
			time.Sleep(50 * time.Millisecond)
			*s = sampleStatfs
			return nil
		}, 10*time.Millisecond)

		total, free, avail, inodes, inodesFree, err := GetVfsStats("/mnt")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("got error %v, want context.DeadlineExceeded", err)
		}
		if got := [5]uint64{total, free, avail, inodes, inodesFree}; got != [5]uint64{} {
			t.Errorf("got %v, want all zero", got)
		}

		// Let the worker's statfs return and the worker exit before the bubble ends.
		time.Sleep(100 * time.Millisecond)
		synctest.Wait()
	})
}

// TestGetVfsStatsHungKeepsOneCallPerPath runs with fake time. The stub does not
// return until the test releases it, as statfs does on a dead hard NFS mount.
// Repeated calls for a path must time out without starting another statfs, since
// each one would hold an OS thread for as long as the filesystem stays hung.
func TestGetVfsStatsHungKeepsOneCallPerPath(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var started atomic.Int32
		stubStatfs(t, func(_ string, s *syscall.Statfs_t) error {
			started.Add(1)
			<-release
			*s = sampleStatfs
			return nil
		}, 10*time.Millisecond)

		timesOut := func(path string) {
			t.Helper()
			if _, _, _, _, _, err := GetVfsStats(path); !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("got error %v, want context.DeadlineExceeded", err)
			}
		}

		for range 3 {
			timesOut("/mnt")
		}
		var wg sync.WaitGroup
		for range 3 {
			wg.Go(func() { timesOut("/mnt") })
		}
		wg.Wait()
		if got := started.Load(); got != 1 {
			t.Errorf("started statfs %d times for one hung path, want 1", got)
		}

		// Another path is not held up by the hung one.
		timesOut("/other")
		if got := started.Load(); got != 2 {
			t.Errorf("started statfs %d times for two hung paths, want 2", got)
		}

		// Once the hung calls return, the next call starts a new statfs and succeeds.
		close(release)
		synctest.Wait()
		total, _, _, _, _, err := GetVfsStats("/mnt")
		if err != nil {
			t.Fatalf("unexpected error after release: %v", err)
		}
		if total != 409600 {
			t.Errorf("got total %d, want 409600", total)
		}
		if got := started.Load(); got != 3 {
			t.Errorf("started statfs %d times in total, want 3", got)
		}
	})
}
