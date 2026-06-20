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

// Package processlist implements the v2 /ps endpoint's process listing for the
// full cAdvisor binary. It is injected into the lean library manager via
// manager.ProcessListProvider (the kubelet leaves that nil and lists no
// processes). The logic — shelling out to `ps` and filtering by the container's
// cgroup — lives here rather than in the library to keep the library lean.
package processlist

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	model "github.com/google/cadvisor/lib/model"

	"k8s.io/klog/v2"
)

// cgroup type chosen to fetch the cgroup path of a process. Memory is one of the
// default cgroups enabled for most containers; some systems (e.g. Raspberry Pi
// 4) disable the memory controller by default, so fall back to cpu.
var cgroupMemoryPathRegExp = regexp.MustCompile(`memory[^:]*:(.*?)[,;$]`)
var cgroupCPUPathRegExp = regexp.MustCompile(`cpu[^:]*:(.*?)[,;$]`)

// List returns the processes running in the named container. It satisfies the
// signature of manager.ProcessListProvider. containerName is the container's
// cgroup name ("/" for the root container); isRoot reports whether it is the
// root container.
func List(containerName string, isRoot bool, cadvisorContainer string, inHostNamespace bool) ([]model.ProcessInfo, error) {
	format := "user,pid,ppid,stime,pcpu,pmem,rss,vsz,stat,time,comm,psr,cgroup"
	out, err := getPsOutput(inHostNamespace, format)
	if err != nil {
		return nil, err
	}
	return parseProcessList(out, containerName, isRoot, cadvisorContainer, inHostNamespace)
}

func getPsOutput(inHostNamespace bool, format string) ([]byte, error) {
	args := []string{}
	command := "ps"
	if !inHostNamespace {
		command = "/usr/sbin/chroot"
		args = append(args, "/rootfs", "ps")
	}
	args = append(args, "-e", "-o", format)
	out, err := exec.Command(command, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute %q command: %v", command, err)
	}
	return out, err
}

func parseProcessList(out []byte, containerName string, isRoot bool, cadvisorContainer string, inHostNamespace bool) ([]model.ProcessInfo, error) {
	rootfs := "/"
	if !inHostNamespace {
		rootfs = "/rootfs"
	}
	processes := []model.ProcessInfo{}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines[1:] {
		processInfo, err := parsePsLine(line, containerName, isRoot, cadvisorContainer, inHostNamespace)
		if err != nil {
			return nil, fmt.Errorf("could not parse line %s: %v", line, err)
		}
		if processInfo == nil {
			continue
		}

		dirPath := path.Join(rootfs, "/proc", strconv.Itoa(processInfo.Pid), "fd")
		fds, err := os.ReadDir(dirPath)
		if err != nil {
			klog.V(4).Infof("error while listing directory %q to measure fd count: %v", dirPath, err)
			continue
		}
		processInfo.FdCount = len(fds)

		processes = append(processes, *processInfo)
	}
	return processes, nil
}

func parsePsLine(line, containerName string, isRoot bool, cadvisorContainer string, inHostNamespace bool) (*model.ProcessInfo, error) {
	const expectedFields = 13
	if len(line) == 0 {
		return nil, nil
	}

	info := model.ProcessInfo{}
	var err error

	fields := strings.Fields(line)
	if len(fields) < expectedFields {
		return nil, fmt.Errorf("expected at least %d fields, found %d: output: %q", expectedFields, len(fields), line)
	}
	info.User = fields[0]
	info.StartTime = fields[3]
	info.Status = fields[8]
	info.RunningTime = fields[9]

	info.Pid, err = strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("invalid pid %q: %v", fields[1], err)
	}
	info.Ppid, err = strconv.Atoi(fields[2])
	if err != nil {
		return nil, fmt.Errorf("invalid ppid %q: %v", fields[2], err)
	}

	percentCPU, err := strconv.ParseFloat(fields[4], 32)
	if err != nil {
		return nil, fmt.Errorf("invalid cpu percent %q: %v", fields[4], err)
	}
	info.PercentCpu = float32(percentCPU)
	percentMem, err := strconv.ParseFloat(fields[5], 32)
	if err != nil {
		return nil, fmt.Errorf("invalid memory percent %q: %v", fields[5], err)
	}
	info.PercentMemory = float32(percentMem)

	info.RSS, err = strconv.ParseUint(fields[6], 0, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid rss %q: %v", fields[6], err)
	}
	info.VirtualSize, err = strconv.ParseUint(fields[7], 0, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid virtual size %q: %v", fields[7], err)
	}
	// convert to bytes
	info.RSS *= 1024
	info.VirtualSize *= 1024

	// According to `man ps`, comm may contain spaces; join the middle fields.
	info.Cmd = strings.Join(fields[10:len(fields)-2], " ")

	// The last two fields are psr and cgroup; subslice to handle comm with spaces.
	lastTwoFields := fields[len(fields)-2:]
	info.Psr, err = strconv.Atoi(lastTwoFields[0])
	if err != nil {
		return nil, fmt.Errorf("invalid psr %q: %v", lastTwoFields[0], err)
	}
	info.CgroupPath = getCgroupPath(lastTwoFields[1])

	// Remove the ps command we just ran from the cadvisor container (cosmetic).
	if !inHostNamespace && cadvisorContainer == info.CgroupPath && info.Cmd == "ps" {
		return nil, nil
	}

	// Do not report processes from other containers when a non-root container is requested.
	if !isRoot && info.CgroupPath != containerName {
		return nil, nil
	}

	// Remove cgroup information when a non-root container is requested.
	if !isRoot {
		info.CgroupPath = ""
	}
	return &info, nil
}

func getCgroupPath(cgroups string) string {
	if cgroups == "-" {
		return "/"
	}
	if strings.HasPrefix(cgroups, "0::") {
		return cgroups[3:]
	}
	matches := cgroupMemoryPathRegExp.FindSubmatch([]byte(cgroups))
	if len(matches) != 2 {
		klog.V(3).Infof("failed to get memory cgroup path from %q, will try cpu cgroup path", cgroups)
		matches = cgroupCPUPathRegExp.FindSubmatch([]byte(cgroups))
		if len(matches) != 2 {
			klog.V(3).Infof("failed to get cpu cgroup path from %q; assuming root cgroup", cgroups)
			return "/"
		}
	}
	return string(matches[1])
}
