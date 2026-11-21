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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// fakeTable describes one file descriptor table in a fake /proc.
type fakeTable struct {
	fds     int
	sockets int
}

// fakeProc builds a fake root filesystem and cgroup directory. pidTables maps
// each pid to the table it uses; pids sharing a table get identical fd
// directories, like processes created with CLONE_FILES. It returns rootFs and
// the cgroup path.
func fakeProc(t testing.TB, pids []int, pidTables map[int]int, tables map[int]fakeTable, selfPid int) (string, string) {
	t.Helper()
	root := t.TempDir()
	procDir := filepath.Join(root, "proc")
	require.NoError(t, os.MkdirAll(procDir, 0o755))
	require.NoError(t, os.Symlink(strconv.Itoa(selfPid), filepath.Join(procDir, "self")))

	var procs strings.Builder
	for _, pid := range pids {
		fmt.Fprintf(&procs, "%d\n", pid)
		tbl := tables[pidTables[pid]]
		fdDir := filepath.Join(procDir, strconv.Itoa(pid), "fd")
		require.NoError(t, os.MkdirAll(fdDir, 0o755))
		for fd := 0; fd < tbl.fds; fd++ {
			target := "/dev/null"
			if fd < tbl.sockets {
				target = fmt.Sprintf("socket:[%d]", 1000+fd)
			}
			require.NoError(t, os.Symlink(target, filepath.Join(fdDir, strconv.Itoa(fd))))
		}
	}

	cgroupDir := filepath.Join(root, "cgroup")
	require.NoError(t, os.MkdirAll(cgroupDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cgroupDir, "cgroup.procs"), []byte(procs.String()), 0o644))
	return root, cgroupDir
}

// fakeKcmp replaces kcmpFiles with a function that orders pids by the table
// they use, the way the kernel orders file tables. It returns a pointer to
// the call count.
func fakeKcmp(t testing.TB, pidTables map[int]int) *int {
	t.Helper()
	calls := 0
	orig := kcmpFiles
	kcmpFiles = func(pid1, pid2 int) (int, error) {
		calls++
		a, ok1 := pidTables[pid1]
		b, ok2 := pidTables[pid2]
		if !ok1 || !ok2 {
			return 0, unix.ESRCH
		}
		switch {
		case a == b:
			return 0, nil
		case a < b:
			return 1, nil
		default:
			return 2, nil
		}
	}
	t.Cleanup(func() {
		kcmpFiles = orig
		kcmpUnavailable.Store(false)
	})
	return &calls
}

func TestProcessStatsFromProcsSharedFdTables(t *testing.T) {
	// Pids 10, 11 and 12 share table 1 (like a sandbox's stub processes);
	// 20 and 30 each have their own.
	pids := []int{10, 20, 11, 30, 12}
	pidTables := map[int]int{10: 1, 11: 1, 12: 1, 20: 2, 30: 3}
	tables := map[int]fakeTable{1: {fds: 5, sockets: 2}, 2: {fds: 3, sockets: 1}, 3: {fds: 4, sockets: 0}}
	rootFs, cgroupPath := fakeProc(t, pids, pidTables, tables, os.Getpid())
	fakeKcmp(t, pidTables)

	stats, err := processStatsFromProcs(rootFs, cgroupPath, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(5), stats.ProcessCount)
	assert.Equal(t, uint64(5+3+4), stats.FdCount)
	assert.Equal(t, uint64(2+1+0), stats.SocketCount)
}

func TestProcessStatsFromProcsDistinctFdTables(t *testing.T) {
	pids := []int{10, 11, 12}
	pidTables := map[int]int{10: 3, 11: 1, 12: 2}
	tables := map[int]fakeTable{1: {fds: 1, sockets: 1}, 2: {fds: 2, sockets: 0}, 3: {fds: 3, sockets: 2}}
	rootFs, cgroupPath := fakeProc(t, pids, pidTables, tables, os.Getpid())
	fakeKcmp(t, pidTables)

	stats, err := processStatsFromProcs(rootFs, cgroupPath, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), stats.ProcessCount)
	assert.Equal(t, uint64(6), stats.FdCount)
	assert.Equal(t, uint64(3), stats.SocketCount)
}

func TestProcessStatsFromProcsKcmpCallsAreLogarithmic(t *testing.T) {
	// 64 processes, each with its own table: every lookup is a miss, so the
	// set grows to 64 entries. A linear scan would need 64*63/2 = 2016 calls.
	n := 64
	pids := make([]int, n)
	pidTables := make(map[int]int, n)
	tables := make(map[int]fakeTable, n)
	for i := range pids {
		pids[i] = 100 + i
		// Scramble the table order relative to the pid order.
		pidTables[pids[i]] = (i * 37) % n
		tables[(i*37)%n] = fakeTable{fds: 1}
	}
	rootFs, cgroupPath := fakeProc(t, pids, pidTables, tables, os.Getpid())
	calls := fakeKcmp(t, pidTables)

	stats, err := processStatsFromProcs(rootFs, cgroupPath, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(n), stats.FdCount)
	assert.LessOrEqual(t, *calls, n*7) // ceil(log2(64))+1 per pid
}

func TestProcessStatsFromProcsKcmpErrors(t *testing.T) {
	pids := []int{10, 11, 12}
	pidTables := map[int]int{10: 1, 11: 1, 12: 1}
	tables := map[int]fakeTable{1: {fds: 4, sockets: 1}}

	for _, tc := range []struct {
		name            string
		err             error
		wantUnavailable bool
	}{
		{name: "ENOSYS", err: unix.ENOSYS, wantUnavailable: true},
		{name: "EPERM", err: unix.EPERM},
		{name: "ESRCH", err: unix.ESRCH},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rootFs, cgroupPath := fakeProc(t, pids, pidTables, tables, os.Getpid())
			fakeKcmp(t, pidTables)
			kcmpFiles = func(int, int) (int, error) { return 0, tc.err }

			// Without a working kcmp, every process is counted, as before
			// deduplication existed.
			stats, err := processStatsFromProcs(rootFs, cgroupPath, 0)
			require.NoError(t, err)
			assert.Equal(t, uint64(12), stats.FdCount)
			assert.Equal(t, uint64(3), stats.SocketCount)
			assert.Equal(t, tc.wantUnavailable, kcmpUnavailable.Load())
		})
	}
}

func TestProcessStatsFromProcsSkipsKcmpAfterENOSYS(t *testing.T) {
	pids := []int{10, 11}
	pidTables := map[int]int{10: 1, 11: 1}
	tables := map[int]fakeTable{1: {fds: 2}}
	rootFs, cgroupPath := fakeProc(t, pids, pidTables, tables, os.Getpid())
	calls := fakeKcmp(t, pidTables)
	kcmpUnavailable.Store(true)

	stats, err := processStatsFromProcs(rootFs, cgroupPath, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(4), stats.FdCount)
	assert.Equal(t, 0, *calls)
}

func TestProcessStatsFromProcsForeignPidNamespace(t *testing.T) {
	// /proc/self resolving to another pid means the /proc being read belongs
	// to a different pid namespace, so its pids must not be passed to kcmp.
	pids := []int{10, 11}
	pidTables := map[int]int{10: 1, 11: 1}
	tables := map[int]fakeTable{1: {fds: 2, sockets: 1}}
	rootFs, cgroupPath := fakeProc(t, pids, pidTables, tables, os.Getpid()+1)
	calls := fakeKcmp(t, pidTables)

	stats, err := processStatsFromProcs(rootFs, cgroupPath, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(4), stats.FdCount)
	assert.Equal(t, uint64(2), stats.SocketCount)
	assert.Equal(t, 0, *calls)
}

func TestKcmpFilesSyscall(t *testing.T) {
	self := os.Getpid()
	r, err := kcmpFiles(self, self)
	if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EPERM) {
		t.Skipf("kcmp(2) not available: %v", err)
	}
	require.NoError(t, err)
	assert.Equal(t, 0, r, "a process shares its own file table")

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start child: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	r, err = kcmpFiles(self, cmd.Process.Pid)
	require.NoError(t, err)
	assert.Contains(t, []int{1, 2}, r, "a child started with exec has its own file table")
	r2, err := kcmpFiles(cmd.Process.Pid, self)
	require.NoError(t, err)
	assert.Equal(t, 3-r, r2, "the order is antisymmetric")
}

// BenchmarkProcessStatsFromProcsSharedFdTable models a sandbox whose stub
// processes share one large file table.
func BenchmarkProcessStatsFromProcsSharedFdTable(b *testing.B) {
	const procs, fds = 32, 1000
	pids := make([]int, procs)
	pidTables := make(map[int]int, procs)
	for i := range pids {
		pids[i] = 100 + i
		pidTables[pids[i]] = 1
	}
	tables := map[int]fakeTable{1: {fds: fds, sockets: fds / 10}}

	for _, dedupe := range []bool{false, true} {
		b.Run(fmt.Sprintf("dedupe=%t", dedupe), func(b *testing.B) {
			selfPid := os.Getpid()
			if !dedupe {
				selfPid++
			}
			rootFs, cgroupPath := fakeProc(b, pids, pidTables, tables, selfPid)
			fakeKcmp(b, pidTables)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := processStatsFromProcs(rootFs, cgroupPath, 0); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
