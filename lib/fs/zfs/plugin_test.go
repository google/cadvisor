// Copyright 2024 Google Inc. All Rights Reserved.
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

package zfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/cadvisor/lib/fs"
)

func TestGetStatsFallsBackWhenDevMissing(t *testing.T) {
	oldPath := zfsDevicePath
	zfsDevicePath = filepath.Join(t.TempDir(), "missing-zfs")
	defer func() { zfsDevicePath = oldPath }()

	_, err := NewPlugin().GetStats("tank/root", fs.PartitionInfo{Mountpoint: "/"})
	if !errors.Is(err, fs.ErrFallbackToVFS) {
		t.Fatalf("GetStats() error = %v, want ErrFallbackToVFS", err)
	}
}

func TestGetStatsFallsBackOnZfsError(t *testing.T) {
	// Simulate /dev/zfs present but unusable (e.g. EPERM in an unprivileged
	// container): device node exists, yet zfs list fails.
	dev := filepath.Join(t.TempDir(), "zfs")
	if err := os.WriteFile(dev, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	oldPath, oldFn := zfsDevicePath, getZfsStats
	zfsDevicePath = dev
	getZfsStats = func(string) (uint64, uint64, uint64, error) {
		return 0, 0, 0, os.ErrPermission
	}
	defer func() {
		zfsDevicePath = oldPath
		getZfsStats = oldFn
	}()

	_, err := NewPlugin().GetStats("tank/root", fs.PartitionInfo{Mountpoint: "/"})
	if !errors.Is(err, fs.ErrFallbackToVFS) {
		t.Fatalf("GetStats() error = %v, want ErrFallbackToVFS", err)
	}
}

func TestGetStatsSuccess(t *testing.T) {
	dev := filepath.Join(t.TempDir(), "zfs")
	if err := os.WriteFile(dev, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	oldPath, oldFn := zfsDevicePath, getZfsStats
	zfsDevicePath = dev
	getZfsStats = func(pool string) (uint64, uint64, uint64, error) {
		if pool != "tank/root" {
			t.Errorf("pool = %q, want tank/root", pool)
		}
		return 300, 100, 100, nil
	}
	defer func() {
		zfsDevicePath = oldPath
		getZfsStats = oldFn
	}()

	stats, err := NewPlugin().GetStats("tank/root", fs.PartitionInfo{Mountpoint: "/"})
	if err != nil {
		t.Fatalf("GetStats() unexpected error: %v", err)
	}
	if stats.Capacity != 300 || stats.Free != 100 || stats.Available != 100 {
		t.Errorf("stats = %+v, want capacity=300 free=100 available=100", stats)
	}
	if stats.Type != fs.ZFS {
		t.Errorf("Type = %q, want %q", stats.Type, fs.ZFS)
	}
}
