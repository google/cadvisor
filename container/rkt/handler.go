// Copyright 2016 Google Inc. All Rights Reserved.
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

// Handler for "rkt" containers.
package rkt

import (
	"fmt"
	"os"
	"path"
	"time"

	rktapi "github.com/coreos/rkt/api/v1alpha"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils"
	"golang.org/x/net/context"

	"github.com/golang/glog"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	cgroupfs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/configs"
)

type rktContainerHandler struct {
	rktClient rktapi.PublicAPIClient
	// Name of the container for this handler.
	name               string
	cgroupSubsystems   *libcontainer.CgroupSubsystems
	machineInfoFactory info.MachineInfoFactory

	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	// Manager of this container's cgroups.
	cgroupManager cgroups.Manager

	// Whether this container has network isolation enabled.
	hasNetwork bool

	fsInfo         fs.FsInfo
	externalMounts []common.Mount

	rootFs string

	isPod bool

	aliases []string

	pid int

	rootfsStorageDir string

	// Filesystem handler.
	fsHandler common.FsHandler

	ignoreMetrics container.MetricSet
}

func (self *rktContainerHandler) GetCgroupPaths() map[string]string {
	return self.cgroupPaths
}

func (self *rktContainerHandler) GetMachineInfoFactory() info.MachineInfoFactory {
	return self.machineInfoFactory
}

func (self *rktContainerHandler) GetName() string {
	return self.name
}

func (self *rktContainerHandler) GetExternalMounts() []common.Mount {
	return self.externalMounts
}

func (self *rktContainerHandler) HasNetwork() bool {
	return self.hasNetwork && !self.ignoreMetrics.Has(container.NetworkUsageMetrics)
}

func (self *rktContainerHandler) HasFilesystem() bool {
	if !self.ignoreMetrics.Has(container.DiskUsageMetrics) {
		return true
	}
	return false
}

func newRktContainerHandler(name string, rktClient rktapi.PublicAPIClient, rktPath string, cgroupSubsystems *libcontainer.CgroupSubsystems, machineInfoFactory info.MachineInfoFactory, fsInfo fs.FsInfo, rootFs string, ignoreMetrics container.MetricSet) (container.ContainerHandler, error) {
	aliases := make([]string, 1)
	isPod := false

	parsed, err := parseName(name)
	if err != nil {
		return nil, fmt.Errorf("this should be impossible!, new handler failing, but factory allowed, name = %s", name)
	}

	//rktnetes uses containerID: rkt://fff40827-b994-4e3a-8f88-6427c2c8a5ac:nginx
	if parsed.Container == "" {
		isPod = true
		aliases = append(aliases, "rkt://"+parsed.Pod)
	} else {
		aliases = append(aliases, "rkt://"+parsed.Pod+":"+parsed.Container)
	}

	pid := os.Getpid()
	if parsed.Container == "" {
		resp, err := rktClient.InspectPod(context.Background(), &rktapi.InspectPodRequest{
			Id: parsed.Pod,
		})
		if err != nil {
			return nil, err
		}
		pid = int(resp.Pod.Pid)
	} else {
		glog.Infof("skipping as Container")
	}

	cgroupPaths := common.MakeCgroupPaths(cgroupSubsystems.MountPoints, name)

	cHints, err := common.GetContainerHintsFromFile(*common.ArgContainerHints)
	if err != nil {
		return nil, err
	}

	// Generate the equivalent cgroup manager for this container.
	cgroupManager := &cgroupfs.Manager{
		Cgroups: &configs.Cgroup{
			Name: name,
		},
		Paths: cgroupPaths,
	}

	hasNetwork := false
	if isPod {
		hasNetwork = true
	}

	//SJP: unsure the point of this code, if it event does anything today?
	var externalMounts []common.Mount
	for _, container := range cHints.AllHosts {
		if name == container.FullName {
			externalMounts = container.Mounts
			break
		}
	}

	rootfsStorageDir := getRootFs(rktPath, parsed)

	handler := &rktContainerHandler{
		name:               name,
		rktClient:          rktClient,
		cgroupSubsystems:   cgroupSubsystems,
		machineInfoFactory: machineInfoFactory,
		cgroupPaths:        cgroupPaths,
		cgroupManager:      cgroupManager,
		fsInfo:             fsInfo,
		hasNetwork:         hasNetwork,
		externalMounts:     externalMounts,
		rootFs:             rootFs,
		isPod:              isPod,
		aliases:            aliases,
		pid:                pid,
		rootfsStorageDir:   rootfsStorageDir,
		ignoreMetrics:      ignoreMetrics,
	}

	if !ignoreMetrics.Has(container.DiskUsageMetrics) {
		handler.fsHandler = common.NewFsHandler(time.Minute, rootfsStorageDir, "", fsInfo)
	}

	return handler, nil
}

func (self *rktContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return info.ContainerReference{
		Name:      self.name,
		Aliases:   self.aliases,
		Namespace: RktNamespace,
	}, nil
}

//SJP: Should a Rkt containe have have htis?
func (self *rktContainerHandler) GetRootNetworkDevices() ([]info.NetInfo, error) {
	nd := []info.NetInfo{}
	return nd, nil
}

func (self *rktContainerHandler) Start() {
	self.fsHandler.Start()
}

func (self *rktContainerHandler) Cleanup() {
	self.fsHandler.Stop()
}

func (self *rktContainerHandler) GetSpec() (info.ContainerSpec, error) {
	return common.GetSpec(self)
}

func (self *rktContainerHandler) getFsStats(stats *info.ContainerStats) error {
	if self.ignoreMetrics.Has(container.DiskUsageMetrics) {
		return nil
	}

	deviceInfo, err := self.fsInfo.GetDirFsDevice(self.rootfsStorageDir)
	if err != nil {
		return err
	}

	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return err
	}
	var limit uint64 = 0

	// SJP: Docker does not impose any filesystem limits for containers. So it uses capacity as limit.
	// Doing the same for Rkt.  is this true?
	for _, fs := range mi.Filesystems {
		if fs.Device == deviceInfo.Device {
			limit = fs.Capacity
			break
		}
	}

	fsStat := info.FsStats{Device: deviceInfo.Device, Limit: limit}

	fsStat.BaseUsage, fsStat.Usage = self.fsHandler.Usage()

	stats.Filesystem = append(stats.Filesystem, fsStat)

	return nil
}

func (self *rktContainerHandler) GetStats() (*info.ContainerStats, error) {
	stats, err := libcontainer.GetStats(self.cgroupManager, self.rootFs, self.pid, self.ignoreMetrics)
	if err != nil {
		return stats, err
	}

	// Get filesystem stats.
	err = self.getFsStats(stats)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (self *rktContainerHandler) GetCgroupPath(resource string) (string, error) {
	path, ok := self.cgroupPaths[resource]
	if !ok {
		return "", fmt.Errorf("could not find path for resource %q for container %q\n", resource, self.name)
	}
	return path, nil
}

//TODO{SJP} need to figure out what to put here
func (self *rktContainerHandler) GetContainerLabels() map[string]string {
	return map[string]string{}
}

func (self *rktContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	containers := make(map[string]struct{})

	// Rkt containers do not have subcontainers, only the "Pod" does.
	if self.isPod == false {
		var ret []info.ContainerReference
		return ret, nil
	}

	// Turn the system.slice cgroups  into the Pod's subcontainers
	for _, cgroupPath := range self.cgroupPaths {
		err := common.ListDirectories(path.Join(cgroupPath, "system.slice"), path.Join(self.name, "system.slice"), listType == container.ListRecursive, containers)
		if err != nil {
			return nil, err
		}
	}

	// Create the container references. for the Pod's subcontainers
	ret := make([]info.ContainerReference, 0, len(containers))
	for cont := range containers {
		aliases := make([]string, 1)
		parsed, err := parseName(cont)
		if err != nil {
			return nil, fmt.Errorf("this should be impossible!, unable to parse rkt subcontainer name = %s", cont)
		}
		aliases = append(aliases, parsed.Pod+":"+parsed.Container)

		ret = append(ret, info.ContainerReference{
			Name:      cont,
			Aliases:   aliases,
			Namespace: RktNamespace,
		})
	}

	return ret, nil
}

func (self *rktContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	// TODO(vmarmol): Implement
	return nil, nil
}

func (self *rktContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return libcontainer.GetProcesses(self.cgroupManager)
}

func (self *rktContainerHandler) WatchSubcontainers(events chan container.SubcontainerEvent) error {
	return fmt.Errorf("watch is unimplemented in the Rkt container driver")
}

func (self *rktContainerHandler) StopWatchingSubcontainers() error {
	// No-op for Rkt driver.
	return nil
}

func (self *rktContainerHandler) Exists() bool {
	// If any cgroup exists, the container is still alive.
	for _, cgroupPath := range self.cgroupPaths {
		if utils.FileExists(cgroupPath) {
			return true
		}
	}
	return false
}
