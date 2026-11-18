// Copyright 2014 Google Inc. All Rights Reserved.
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
	"syscall"
	"time"
)

// statfs and statfsTimeout are variables so that tests can simulate a hung filesystem.
var (
	statfs        = syscall.Statfs
	statfsTimeout = 2 * time.Second
)

// GetVfsStats returns filesystem statistics using the statfs syscall.
// It has a timeout to prevent hanging on unresponsive filesystems.
func GetVfsStats(path string) (total uint64, free uint64, avail uint64, inodes uint64, inodesFree uint64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), statfsTimeout)
	defer cancel()

	type result struct {
		total      uint64
		free       uint64
		avail      uint64
		inodes     uint64
		inodesFree uint64
		err        error
	}

	resultChan := make(chan result, 1)

	// The goroutine may outlive the timeout below, so it must not touch the named return values.
	go func() {
		var s syscall.Statfs_t
		var res result
		if res.err = statfs(path, &s); res.err == nil {
			res.total = uint64(s.Frsize) * s.Blocks
			res.free = uint64(s.Frsize) * s.Bfree
			res.avail = uint64(s.Frsize) * s.Bavail
			res.inodes = uint64(s.Files)
			res.inodesFree = uint64(s.Ffree)
		}
		resultChan <- res
	}()

	select {
	case <-ctx.Done():
		return 0, 0, 0, 0, 0, ctx.Err()
	case res := <-resultChan:
		return res.total, res.free, res.avail, res.inodes, res.inodesFree, res.err
	}
}
