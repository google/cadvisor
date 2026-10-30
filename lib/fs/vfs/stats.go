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
	"sync"
	"syscall"
	"time"
)

// statfs and statfsTimeout are variables so that tests can simulate a hung filesystem.
var (
	statfs        = syscall.Statfs
	statfsTimeout = 2 * time.Second
)

type statfsResult struct {
	total      uint64
	free       uint64
	avail      uint64
	inodes     uint64
	inodesFree uint64
	err        error
}

// statfsCall is a statfs in flight. res is set before done is closed.
type statfsCall struct {
	done chan struct{}
	res  statfsResult
}

// inflight holds the statfs in flight for each path. A statfs on a hung
// filesystem, such as a dead hard NFS mount, does not return, and its goroutine
// holds an OS thread until it does. Callers give up after statfsTimeout, so
// starting one per call would leave another thread behind every time, until the
// process reaches the runtime's thread limit. A caller that finds one in flight
// waits on it instead.
var (
	inflightMu sync.Mutex
	inflight   = map[string]*statfsCall{}
)

// startStatfs returns the statfs in flight for path, starting one if there is none.
func startStatfs(path string) *statfsCall {
	inflightMu.Lock()
	defer inflightMu.Unlock()

	if call, ok := inflight[path]; ok {
		return call
	}
	call := &statfsCall{done: make(chan struct{})}
	inflight[path] = call

	// The goroutine may outlive the callers' timeout, so it must not touch their return values.
	go func() {
		var s syscall.Statfs_t
		var res statfsResult
		if res.err = statfs(path, &s); res.err == nil {
			res.total = uint64(s.Frsize) * s.Blocks
			res.free = uint64(s.Frsize) * s.Bfree
			res.avail = uint64(s.Frsize) * s.Bavail
			res.inodes = uint64(s.Files)
			res.inodesFree = uint64(s.Ffree)
		}
		call.res = res

		inflightMu.Lock()
		delete(inflight, path)
		inflightMu.Unlock()
		close(call.done)
	}()
	return call
}

// GetVfsStats returns filesystem statistics using the statfs syscall.
// It has a timeout to prevent hanging on unresponsive filesystems.
func GetVfsStats(path string) (total uint64, free uint64, avail uint64, inodes uint64, inodesFree uint64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), statfsTimeout)
	defer cancel()

	call := startStatfs(path)

	select {
	case <-ctx.Done():
		return 0, 0, 0, 0, 0, ctx.Err()
	case <-call.done:
		res := call.res
		return res.total, res.free, res.avail, res.inodes, res.inodesFree, res.err
	}
}
