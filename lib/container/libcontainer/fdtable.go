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

package libcontainer

import (
	"errors"
	"os"
	"path"
	"strconv"
	"sync/atomic"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

// kcmpFilesType is KCMP_FILES from <linux/kcmp.h>.
const kcmpFilesType = 2

// kcmpFiles compares the file descriptor tables of pid1 and pid2 with
// kcmp(2). It returns 0 if both processes share one table, 1 if pid1's table
// orders before pid2's and 2 if it orders after. The order is arbitrary but
// stable for the life of both tables, which lets callers binary search.
// It is a variable so tests can replace it.
var kcmpFiles = func(pid1, pid2 int) (int, error) {
	r, _, errno := unix.Syscall6(unix.SYS_KCMP, uintptr(pid1), uintptr(pid2), kcmpFilesType, 0, 0, 0)
	if errno != 0 {
		return 0, errno
	}
	return int(r), nil
}

// kcmpUnavailable is set once kcmp(2) returns ENOSYS (a kernel built without
// CONFIG_KCMP, or a seccomp filter that rejects it), so later passes stop
// trying.
var kcmpUnavailable atomic.Bool

// fdTableSet records the distinct file descriptor tables already counted in
// one pass over a cgroup's processes. Processes created with CLONE_FILES and
// without CLONE_THREAD (for example gVisor's systrap stub processes) appear
// separately in cgroup.procs but share one table, so walking each of their
// /proc/<pid>/fd directories counts the same descriptors many times.
//
// reps holds one pid per distinct table, sorted by kcmp order, so each lookup
// costs O(log n) kcmp calls.
type fdTableSet struct {
	reps []int
}

// find reports whether pid shares a file descriptor table with a pid already
// in the set. When it does not, idx is where pid belongs in reps. ok is false
// when the comparison could not be made; the caller then counts pid's
// descriptors without recording it, which is the behavior before
// deduplication existed.
func (s *fdTableSet) find(pid int) (shared bool, idx int, ok bool) {
	lo, hi := 0, len(s.reps)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		r, err := kcmpFiles(pid, s.reps[mid])
		if err != nil {
			if errors.Is(err, unix.ENOSYS) {
				kcmpUnavailable.Store(true)
			}
			klog.V(5).Infof("kcmp(%d, %d, KCMP_FILES) failed: %v", pid, s.reps[mid], err)
			return false, 0, false
		}
		switch r {
		case 0:
			return true, 0, true
		case 1:
			hi = mid
		case 2:
			lo = mid + 1
		default:
			// Equal-but-unordered (3) is not returned for KCMP_FILES today;
			// treat anything unexpected as "cannot compare".
			return false, 0, false
		}
	}
	return false, lo, true
}

// insert records pid as the representative of a new table at idx, as
// returned by find.
func (s *fdTableSet) insert(idx, pid int) {
	s.reps = append(s.reps, 0)
	copy(s.reps[idx+1:], s.reps[idx:])
	s.reps[idx] = pid
}

// canDedupeFdTables reports whether pids read from rootFs/proc can be passed
// to kcmp(2). kcmp resolves pids in the caller's pid namespace, so a /proc
// from another namespace (for example the host's /proc mounted into a
// container that has its own pid namespace) would compare the wrong
// processes.
func canDedupeFdTables(rootFs string) bool {
	if kcmpUnavailable.Load() {
		return false
	}
	self, err := os.Readlink(path.Join(rootFs, "/proc/self"))
	if err != nil {
		return false
	}
	return self == strconv.Itoa(os.Getpid())
}
