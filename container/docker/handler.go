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

package docker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"strings"
	"time"

	"github.com/docker/libcontainer/cgroups"
	"github.com/docker/libcontainer/cgroups/fs"
	"github.com/dotcloud/docker/nat"
	"github.com/fsouza/go-dockerclient"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/info"
)

type dockerContainerHandler struct {
	client             *docker.Client
	name               string
	machineInfoFactory info.MachineInfoFactory
	container.NoStatsSummary
}

func (self *dockerContainerHandler) splitName() (string, string, error) {
	parent, id := path.Split(self.name)
	cgroupSelf, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", "", err
	}
	scanner := bufio.NewScanner(cgroupSelf)

	subsys := []string{"memory", "cpu"}
	nestedLevels := 0
	for scanner.Scan() {
		line := scanner.Text()
		elems := strings.Split(line, ":")
		if len(elems) < 3 {
			continue
		}
		for _, s := range subsys {
			if elems[1] == s {
				// count how many nested docker containers are there.
				nestedLevels = strings.Count(elems[2], "/docker")
				break
			}
		}
	}
	if nestedLevels > 0 {
		// we are running inside a docker container
		upperLevel := strings.Repeat("../../", nestedLevels)
		//parent = strings.Join([]string{parent, upperLevel}, "/")
		parent = fmt.Sprintf("%v%v", upperLevel, parent)
	}
	return parent, id, nil
}

func (self *dockerContainerHandler) isDockerRoot() bool {
	// TODO(dengnan): Should we consider other cases?
	return self.name == "/docker"
}

func (self *dockerContainerHandler) isRootContainer() bool {
	return self.name == "/"
}

func (self *dockerContainerHandler) isDockerContainer() bool {
	return (!self.isDockerRoot()) && (!self.isRootContainer())
}

type dockerPortBinding struct {
	HostIp   string
	HostPort string
}

type dockerPort string
type dockerPortMap map[dockerPort][]dockerPortBinding

type dockerNetworkSettings struct {
	IPAddress   string
	IPPrefixLen int
	Gateway     string
	Bridge      string
	Ports       dockerPortMap
}

type dockerContainerConfig struct {
	Hostname        string
	Domainname      string
	User            string
	Memory          int64  // Memory limit (in bytes)
	MemorySwap      int64  // Total memory usage (memory + swap); set `-1' to disable swap
	CpuShares       int64  // CPU shares (relative weight vs. other containers)
	Cpuset          string // Cpuset 0-2, 0,1
	AttachStdin     bool
	AttachStdout    bool
	AttachStderr    bool
	PortSpecs       []string // Deprecated - Can be in the format of 8080/tcp
	ExposedPorts    map[nat.Port]struct{}
	Tty             bool // Attach standard streams to a tty, including stdin if it is not closed.
	OpenStdin       bool // Open stdin
	StdinOnce       bool // If true, close stdin after the 1 attached client disconnects.
	Env             []string
	Cmd             []string
	Image           string // Name of the image as it was passed by the operator (eg. could be symbolic)
	Volumes         map[string]struct{}
	WorkingDir      string
	Entrypoint      []string
	NetworkDisabled bool
	OnBuild         []string
}

type dockerState struct {
	Running    bool
	Pid        int
	ExitCode   int
	StartedAt  time.Time
	FinishedAt time.Time
}

type dockerContainerSpec struct {
	ID string

	Created time.Time

	Path string
	Args []string

	Config *dockerContainerConfig
	State  dockerState
	Image  string

	NetworkSettings *dockerNetworkSettings

	ResolvConfPath string
	HostnamePath   string
	HostsPath      string
	Name           string
	Driver         string
	ExecDriver     string

	MountLabel, ProcessLabel string

	Volumes map[string]string
	// Store rw/ro in a separate structure to preserve reverse-compatibility on-disk.
	// Easier than migrating older container configs :)
	VolumesRW map[string]bool
	// contains filtered or unexported fields
}

func readDockerSpec(id string) (spec *dockerContainerSpec, err error) {
	dir := "/var/lib/docker/containers"
	configPath := path.Join(dir, id, "config.json")
	f, err := os.Open(configPath)
	if err != nil {
		return
	}
	defer f.Close()
	d := json.NewDecoder(f)
	ret := new(dockerContainerSpec)
	err = d.Decode(ret)
	if err != nil {
		return
	}
	spec = ret
	return
}

func dockerConfigToContainerSpec(config *dockerContainerSpec, mi *info.MachineInfo) *info.ContainerSpec {
	spec := new(info.ContainerSpec)
	spec.Memory = new(info.MemorySpec)
	spec.Memory.Limit = math.MaxUint64
	spec.Memory.SwapLimit = math.MaxUint64
	if config.Config.Memory > 0 {
		spec.Memory.Limit = uint64(config.Config.Memory)
	}
	if config.Config.MemorySwap > 0 {
		spec.Memory.SwapLimit = uint64(config.Config.MemorySwap - config.Config.Memory)
	}
	if mi != nil {
		spec.Cpu = new(info.CpuSpec)
		spec.Cpu.Limit = math.MaxUint64
		n := mi.NumCores / 64
		if mi.NumCores%64 > 0 {
			n++
		}
		spec.Cpu.Mask.Data = make([]uint64, n)
		for i := 0; i < n; i++ {
			spec.Cpu.Mask.Data[i] = math.MaxUint64
		}
	}
	return spec
}

func (self *dockerContainerHandler) GetSpec() (spec *info.ContainerSpec, err error) {
	if !self.isDockerContainer() {
		spec = new(info.ContainerSpec)
		return
	}
	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return
	}
	_, id, err := self.splitName()
	if err != nil {
		return
	}
	dspec, err := readDockerSpec(id)
	if err != nil {
		return
	}

	spec = dockerConfigToContainerSpec(dspec, mi)
	return
}

func libcontainerToContainerStats(s *cgroups.Stats, mi *info.MachineInfo) *info.ContainerStats {
	ret := new(info.ContainerStats)
	ret.Timestamp = time.Now()
	ret.Cpu = new(info.CpuStats)
	ret.Cpu.Usage.User = s.CpuStats.CpuUsage.UsageInUsermode
	ret.Cpu.Usage.System = s.CpuStats.CpuUsage.UsageInKernelmode
	n := len(s.CpuStats.CpuUsage.PercpuUsage)
	ret.Cpu.Usage.PerCpu = make([]uint64, n)

	ret.Cpu.Usage.Total = 0
	for i := 0; i < n; i++ {
		ret.Cpu.Usage.PerCpu[i] = s.CpuStats.CpuUsage.PercpuUsage[i]
		ret.Cpu.Usage.Total += s.CpuStats.CpuUsage.PercpuUsage[i]
	}
	ret.Memory = new(info.MemoryStats)
	ret.Memory.Usage = s.MemoryStats.Usage
	if v, ok := s.MemoryStats.Stats["pgfault"]; ok {
		ret.Memory.ContainerData.Pgfault = v
		ret.Memory.HierarchicalData.Pgfault = v
	}
	if v, ok := s.MemoryStats.Stats["pgmajfault"]; ok {
		ret.Memory.ContainerData.Pgmajfault = v
		ret.Memory.HierarchicalData.Pgmajfault = v
	}
	return ret
}

func (self *dockerContainerHandler) GetStats() (stats *info.ContainerStats, err error) {
	if !self.isDockerContainer() {
		// Return empty stats for root containers.
		stats = new(info.ContainerStats)
		stats.Timestamp = time.Now()
		return
	}
	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return
	}
	parent, id, err := self.splitName()
	if err != nil {
		return
	}
	cg := &cgroups.Cgroup{
		Parent: parent,
		Name:   id,
	}
	s, err := fs.GetStats(cg)
	if err != nil {
		return
	}
	stats = libcontainerToContainerStats(s, mi)
	return
}

func (self *dockerContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	if self.isDockerContainer() {
		return nil, nil
	}
	if self.isRootContainer() && listType == container.LIST_SELF {
		return []info.ContainerReference{info.ContainerReference{Name: "/docker"}}, nil
	}
	opt := docker.ListContainersOptions{
		All: true,
	}
	containers, err := self.client.ListContainers(opt)
	if err != nil {
		return nil, err
	}
	ret := make([]info.ContainerReference, 0, len(containers)+1)
	for _, c := range containers {
		if !strings.HasPrefix(c.Status, "Up ") {
			continue
		}
		path := fmt.Sprintf("/docker/%v", c.ID)
		aliases := c.Names
		ref := info.ContainerReference{
			Name:    path,
			Aliases: aliases,
		}
		ret = append(ret, ref)
	}
	if self.isRootContainer() {
		ret = append(ret, info.ContainerReference{Name: "/docker"})
	}
	return ret, nil
}

func (self *dockerContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	return nil, nil
}

func (self *dockerContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return nil, nil
}
