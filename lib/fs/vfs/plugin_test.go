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
	"path/filepath"
	"testing"

	"github.com/google/cadvisor/lib/fs"
	mount "github.com/moby/sys/mountinfo"
)

func TestCanHandle(t *testing.T) {
	tests := []struct {
		name     string
		fsType   string
		expected bool
	}{
		// Supported block filesystems
		{name: "bcachefs", fsType: "bcachefs", expected: true},
		{name: "ext2", fsType: "ext2", expected: true},
		{name: "ext3", fsType: "ext3", expected: true},
		{name: "ext4", fsType: "ext4", expected: true},
		{name: "xfs", fsType: "xfs", expected: true},
		{name: "squashfs", fsType: "squashfs", expected: true},
		{name: "f2fs", fsType: "f2fs", expected: true},
		{name: "jfs", fsType: "jfs", expected: true},
		{name: "reiserfs", fsType: "reiserfs", expected: true},
		{name: "hfs", fsType: "hfs", expected: true},
		{name: "hfsplus", fsType: "hfsplus", expected: true},
		{name: "ntfs", fsType: "ntfs", expected: true},
		{name: "vfat", fsType: "vfat", expected: true},
		{name: "fat", fsType: "fat", expected: true},
		{name: "msdos", fsType: "msdos", expected: true},
		{name: "exfat", fsType: "exfat", expected: true},
		{name: "udf", fsType: "udf", expected: true},
		{name: "iso9660", fsType: "iso9660", expected: true},

		// Excluded virtual/pseudo filesystems
		{name: "cgroup", fsType: "cgroup", expected: false},
		{name: "cgroup2", fsType: "cgroup2", expected: false},
		{name: "cpuset", fsType: "cpuset", expected: false},
		{name: "mqueue", fsType: "mqueue", expected: false},
		{name: "proc", fsType: "proc", expected: false},
		{name: "sysfs", fsType: "sysfs", expected: false},
		{name: "devtmpfs", fsType: "devtmpfs", expected: false},
		{name: "devpts", fsType: "devpts", expected: false},
		{name: "securityfs", fsType: "securityfs", expected: false},
		{name: "debugfs", fsType: "debugfs", expected: false},
		{name: "tracefs", fsType: "tracefs", expected: false},
		{name: "pstore", fsType: "pstore", expected: false},
		{name: "configfs", fsType: "configfs", expected: false},
		{name: "fusectl", fsType: "fusectl", expected: false},
		{name: "hugetlbfs", fsType: "hugetlbfs", expected: false},
		{name: "autofs", fsType: "autofs", expected: false},
		{name: "binfmt_misc", fsType: "binfmt_misc", expected: false},
		{name: "efivarfs", fsType: "efivarfs", expected: false},
		{name: "rpc_pipefs", fsType: "rpc_pipefs", expected: false},
		{name: "nsfs", fsType: "nsfs", expected: false},

		// Dedicated plugin or unsupported filesystems
		{name: "zfs", fsType: "zfs", expected: false},
		{name: "btrfs", fsType: "btrfs", expected: false},
		{name: "overlay", fsType: "overlay", expected: false},
		{name: "tmpfs", fsType: "tmpfs", expected: false},
		{name: "nfs", fsType: "nfs", expected: false},
		{name: "nfs4", fsType: "nfs4", expected: false},
		{name: "unknownfs", fsType: "unknownfs", expected: false},
		{name: "empty", fsType: "", expected: false},
	}

	plugin := NewPlugin()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := plugin.CanHandle(tc.fsType)
			if actual != tc.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tc.fsType, actual, tc.expected)
			}
		})
	}
}

func TestPluginMetadata(t *testing.T) {
	plugin := NewPlugin()
	if got := plugin.Name(); got != "vfs" {
		t.Errorf("Name() = %q, want %q", got, "vfs")
	}
	if got := plugin.Priority(); got != 0 {
		t.Errorf("Priority() = %d, want 0", got)
	}
}

func TestProcessMount(t *testing.T) {
	plugin := NewPlugin()
	mnt := &mount.Info{
		Mountpoint: "/mnt/test",
		FSType:     "bcachefs",
		Source:     "/dev/sda1",
	}

	handled, processed, err := plugin.ProcessMount(mnt)
	if err != nil {
		t.Fatalf("ProcessMount() unexpected error: %v", err)
	}
	if !handled {
		t.Errorf("ProcessMount() handled = false, want true")
	}
	if processed != mnt {
		t.Errorf("ProcessMount() returned mount %v, want original %v", processed, mnt)
	}
}

func TestGetStats(t *testing.T) {
	plugin := NewPlugin()

	t.Run("nonexistent mountpoint", func(t *testing.T) {
		nonexistent := filepath.Join(t.TempDir(), "nonexistent")
		stats, err := plugin.GetStats("/dev/sda1", fs.PartitionInfo{Mountpoint: nonexistent})
		if err != nil {
			t.Fatalf("GetStats() error = %v, want nil", err)
		}
		if stats != nil {
			t.Errorf("GetStats() = %+v, want nil", stats)
		}
	})

	t.Run("valid mountpoint", func(t *testing.T) {
		tempDir := t.TempDir()
		stats, err := plugin.GetStats("/dev/sda1", fs.PartitionInfo{Mountpoint: tempDir})
		if err != nil {
			t.Fatalf("GetStats() unexpected error: %v", err)
		}
		if stats == nil {
			t.Fatalf("GetStats() returned nil stats")
		}
		if stats.Type != fs.VFS {
			t.Errorf("stats.Type = %q, want %q", stats.Type, fs.VFS)
		}
		if stats.Capacity == 0 {
			t.Errorf("stats.Capacity = 0, want > 0")
		}
		if stats.Inodes == nil {
			t.Errorf("stats.Inodes is nil, want non-nil")
		}
		if stats.InodesFree == nil {
			t.Errorf("stats.InodesFree is nil, want non-nil")
		}
	})
}
