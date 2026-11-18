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

package libcontainer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	info "github.com/google/cadvisor/lib/model"
)

// buildProcTree lays out a synthetic rootFs under dir that mirrors what
// processStatsFromProcs reads from a real host:
//
//	<dir>/cgroup/cgroup.procs      one pid per line
//	<dir>/proc/<pid>/fd/<n>        symlink per open fd
//
// Every socketEvery-th fd points at "socket:[<n>]"; the rest at "/dev/null".
// The syscall sequence (getdents64 + readlinkat per fd) is identical to a
// real /proc, so the benchmark measures the Go-side and syscall cost without
// the kernel's procfs cost, which varies by host.
func buildProcTree(tb testing.TB, dir string, pids, fdsPerPid, socketEvery int) (rootFs, cgroupPath string) {
	tb.Helper()
	rootFs = filepath.Join(dir, "root")
	cgroupPath = filepath.Join(dir, "cgroup")
	if err := os.MkdirAll(cgroupPath, 0o755); err != nil {
		tb.Fatal(err)
	}
	var procs strings.Builder
	for pid := 1; pid <= pids; pid++ {
		procs.WriteString(strconv.Itoa(pid))
		procs.WriteByte('\n')
		fdDir := filepath.Join(rootFs, "proc", strconv.Itoa(pid), "fd")
		if err := os.MkdirAll(fdDir, 0o755); err != nil {
			tb.Fatal(err)
		}
		for n := 0; n < fdsPerPid; n++ {
			target := "/dev/null"
			if socketEvery > 0 && n%socketEvery == 0 {
				target = fmt.Sprintf("socket:[%d]", 100000+n)
			}
			if err := os.Symlink(target, filepath.Join(fdDir, strconv.Itoa(n))); err != nil {
				tb.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(cgroupPath, "cgroup.procs"), []byte(procs.String()), 0o644); err != nil {
		tb.Fatal(err)
	}
	return rootFs, cgroupPath
}

var benchProcessStats info.ProcessStats

// BenchmarkProcessStatsFromProcs measures one process-stats sweep of a
// container, which cadvisor runs once per housekeeping tick per container.
// Shapes approximate a single-process Go service, a mid-size multi-process
// container and a large one.
func BenchmarkProcessStatsFromProcs(b *testing.B) {
	shapes := []struct{ pids, fds int }{
		{1, 128},
		{8, 512},
		{32, 2048},
	}
	for _, s := range shapes {
		b.Run(fmt.Sprintf("pids=%d/fds=%d", s.pids, s.fds), func(b *testing.B) {
			rootFs, cgroupPath := buildProcTree(b, b.TempDir(), s.pids, s.fds, 3)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				st, err := processStatsFromProcs(rootFs, cgroupPath, 0)
				if err != nil {
					b.Fatal(err)
				}
				benchProcessStats = st
			}
			b.ReportMetric(float64(s.pids*s.fds), "fds/op")
		})
	}
}

func TestProcessStatsFromProcsSynthetic(t *testing.T) {
	rootFs, cgroupPath := buildProcTree(t, t.TempDir(), 3, 10, 2)
	st, err := processStatsFromProcs(rootFs, cgroupPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	if st.ProcessCount != 3 || st.FdCount != 30 || st.SocketCount != 15 {
		t.Fatalf("unexpected stats: %+v", st)
	}
}
