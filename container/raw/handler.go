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

// TODO(cAdvisor): Package comment.
package raw

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"code.google.com/p/go.exp/inotify"
	dockerlibcontainer "github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups"
	cgroup_fs "github.com/docker/libcontainer/cgroups/fs"
	"github.com/docker/libcontainer/network"
	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/utils"
)

type rawContainerHandler struct {
	name               string
	cgroup             *cgroups.Cgroup
	cgroupSubsystems   *cgroupSubsystems
	machineInfoFactory info.MachineInfoFactory
	watcher            *inotify.Watcher
	stopWatcher        chan error
	watches            map[string]struct{}
	fsInfo             fs.FsInfo
	networkInterface   *networkInterface
	externalMounts     []mount
}

func newRawContainerHandler(name string, cgroupSubsystems *cgroupSubsystems, machineInfoFactory info.MachineInfoFactory) (container.ContainerHandler, error) {
	fsInfo, err := fs.NewFsInfo()
	if err != nil {
		return nil, err
	}
	cHints, err := getContainerHintsFromFile(*argContainerHints)
	if err != nil {
		return nil, err
	}
	var networkInterface *networkInterface
	var externalMounts []mount
	for _, container := range cHints.AllHosts {
		if name == container.FullName {
			networkInterface = container.NetworkInterface
			externalMounts = container.Mounts
			break
		}
	}
	return &rawContainerHandler{
		name: name,
		cgroup: &cgroups.Cgroup{
			Parent: "/",
			Name:   name,
		},
		cgroupSubsystems:   cgroupSubsystems,
		machineInfoFactory: machineInfoFactory,
		stopWatcher:        make(chan error),
		watches:            make(map[string]struct{}),
		fsInfo:             fsInfo,
		networkInterface:   networkInterface,
		externalMounts:     externalMounts,
	}, nil
}

func (self *rawContainerHandler) ContainerReference() (info.ContainerReference, error) {
	// We only know the container by its one name.
	return info.ContainerReference{
		Name: self.name,
	}, nil
}

func readString(dirpath string, file string) string {
	cgroupFile := path.Join(dirpath, file)

	// Ignore non-existent files
	if !utils.FileExists(cgroupFile) {
		return ""
	}

	// Read
	out, err := ioutil.ReadFile(cgroupFile)
	if err != nil {
		glog.Errorf("raw driver: Failed to read %q: %s", cgroupFile, err)
		return ""
	}
	return string(out)
}

func readInt64(dirpath string, file string) uint64 {
	out := readString(dirpath, file)
	if out == "" {
		return 0
	}

	val, err := strconv.ParseUint(strings.TrimSpace(out), 10, 64)
	if err != nil {
		glog.Errorf("raw driver: Failed to parse int %q from file %q: %s", out, path.Join(dirpath, file), err)
		return 0
	}

	return val
}

func (self *rawContainerHandler) GetSpec() (info.ContainerSpec, error) {
	var spec info.ContainerSpec

	// The raw driver assumes unified hierarchy containers.

	// Get machine info.
	mi, err := self.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return spec, err
	}

	// CPU.
	cpuRoot, ok := self.cgroupSubsystems.mountPoints["cpu"]
	if ok {
		cpuRoot = path.Join(cpuRoot, self.name)
		if utils.FileExists(cpuRoot) {
			spec.HasCpu = true
			spec.Cpu.Limit = readInt64(cpuRoot, "cpu.shares")
		}
	}

	// Cpu Mask.
	// This will fail for non-unified hierarchies. We'll return the whole machine mask in that case.
	cpusetRoot, ok := self.cgroupSubsystems.mountPoints["cpuset"]
	if ok {
		cpusetRoot = path.Join(cpusetRoot, self.name)
		if utils.FileExists(cpusetRoot) {
			spec.HasCpu = true
			spec.Cpu.Mask = readString(cpusetRoot, "cpuset.cpus")
			if spec.Cpu.Mask == "" {
				spec.Cpu.Mask = fmt.Sprintf("0-%d", mi.NumCores-1)
			}
		}
	}

	// Memory.
	memoryRoot, ok := self.cgroupSubsystems.mountPoints["memory"]
	if ok {
		memoryRoot = path.Join(memoryRoot, self.name)
		if utils.FileExists(memoryRoot) {
			spec.HasMemory = true
			spec.Memory.Limit = readInt64(memoryRoot, "memory.limit_in_bytes")
			spec.Memory.SwapLimit = readInt64(memoryRoot, "memory.memsw.limit_in_bytes")
		}
	}

	// Fs.
	if self.name == "/" || self.externalMounts != nil {
		spec.HasFilesystem = true
	}

	//Network
	if self.networkInterface != nil {
		spec.HasNetwork = true
	}
	return spec, nil
}

func (self *rawContainerHandler) getFsStats(stats *info.ContainerStats) error {

	// Get Filesystem information only for the root cgroup.
	if self.name == "/" {
		filesystems, err := self.fsInfo.GetGlobalFsInfo()
		if err != nil {
			return err
		}
		for _, fs := range filesystems {
			stats.Filesystem = append(stats.Filesystem,
				info.FsStats{fs.Device, fs.Capacity, fs.Capacity - fs.Free,
					fs.DiskStats.ReadsCompleted,
					fs.DiskStats.ReadsMerged,
					fs.DiskStats.SectorsRead,
					fs.DiskStats.ReadTime,
					fs.DiskStats.WritesCompleted,
					fs.DiskStats.WritesMerged,
					fs.DiskStats.SectorsWritten,
					fs.DiskStats.WriteTime,
					fs.DiskStats.IoInProgress,
					fs.DiskStats.IoTime,
					fs.DiskStats.WeightedIoTime,
				})
		}
	} else if len(self.externalMounts) > 0 {
		var mountSet map[string]struct{}
		mountSet = make(map[string]struct{})
		for _, mount := range self.externalMounts {
			mountSet[mount.HostDir] = struct{}{}
		}
		filesystems, err := self.fsInfo.GetFsInfoForPath(mountSet)
		if err != nil {
			return err
		}
		for _, fs := range filesystems {
			stats.Filesystem = append(stats.Filesystem,
				info.FsStats{fs.Device, fs.Capacity, fs.Capacity - fs.Free,
					fs.DiskStats.ReadsCompleted,
					fs.DiskStats.ReadsMerged,
					fs.DiskStats.SectorsRead,
					fs.DiskStats.ReadTime,
					fs.DiskStats.WritesCompleted,
					fs.DiskStats.WritesMerged,
					fs.DiskStats.SectorsWritten,
					fs.DiskStats.WriteTime,
					fs.DiskStats.IoInProgress,
					fs.DiskStats.IoTime,
					fs.DiskStats.WeightedIoTime,
				})
		}
	}
	return nil
}

func (self *rawContainerHandler) GetStats() (*info.ContainerStats, error) {
	state := dockerlibcontainer.State{}
	if self.networkInterface != nil {
		state = dockerlibcontainer.State{
			NetworkState: network.NetworkState{
				VethHost:  self.networkInterface.VethHost,
				VethChild: self.networkInterface.VethChild,
				NsPath:    "unknown",
			},
		}
	}

	stats, err := libcontainer.GetStats(self.cgroup, &state)
	if err != nil {
		return nil, err
	}

	err = self.getFsStats(stats)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Lists all directories under "path" and outputs the results as children of "parent".
func listDirectories(dirpath string, parent string, recursive bool, output map[string]struct{}) error {
	// Ignore if this hierarchy does not exist.
	if !utils.FileExists(dirpath) {
		return nil
	}

	entries, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		// We only grab directories.
		if entry.IsDir() {
			name := path.Join(parent, entry.Name())
			output[name] = struct{}{}

			// List subcontainers if asked to.
			if recursive {
				err := listDirectories(path.Join(dirpath, entry.Name()), name, true, output)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (self *rawContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	containers := make(map[string]struct{})
	for _, subsystem := range self.cgroupSubsystems.mounts {
		err := listDirectories(path.Join(subsystem.Mountpoint, self.name), self.name, listType == container.ListRecursive, containers)
		if err != nil {
			return nil, err
		}
	}

	// Make into container references.
	ret := make([]info.ContainerReference, 0, len(containers))
	for cont := range containers {
		ret = append(ret, info.ContainerReference{
			Name: cont,
		})
	}

	return ret, nil
}

func (self *rawContainerHandler) ListThreads(listType container.ListType) ([]int, error) {
	// TODO(vmarmol): Implement
	return nil, nil
}

func (self *rawContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return cgroup_fs.GetPids(self.cgroup)
}

func (self *rawContainerHandler) watchDirectory(dir string, containerName string) error {
	err := self.watcher.AddWatch(dir, inotify.IN_CREATE|inotify.IN_DELETE|inotify.IN_MOVE)
	if err != nil {
		return err
	}
	self.watches[containerName] = struct{}{}

	// Watch subdirectories as well.
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			err = self.watchDirectory(path.Join(dir, entry.Name()), path.Join(containerName, entry.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *rawContainerHandler) processEvent(event *inotify.Event, events chan container.SubcontainerEvent) error {
	// Convert the inotify event type to a container create or delete.
	var eventType container.SubcontainerEventType
	switch {
	case (event.Mask & inotify.IN_CREATE) > 0:
		eventType = container.SubcontainerAdd
	case (event.Mask & inotify.IN_DELETE) > 0:
		eventType = container.SubcontainerDelete
	case (event.Mask & inotify.IN_MOVED_FROM) > 0:
		eventType = container.SubcontainerDelete
	case (event.Mask & inotify.IN_MOVED_TO) > 0:
		eventType = container.SubcontainerAdd
	default:
		// Ignore other events.
		return nil
	}

	// Derive the container name from the path name.
	var containerName string
	for _, mount := range self.cgroupSubsystems.mounts {
		mountLocation := path.Clean(mount.Mountpoint) + "/"
		if strings.HasPrefix(event.Name, mountLocation) {
			containerName = event.Name[len(mountLocation)-1:]
			break
		}
	}
	if containerName == "" {
		return fmt.Errorf("unable to detect container from watch event on directory %q", event.Name)
	}

	// Maintain the watch for the new or deleted container.
	switch {
	case eventType == container.SubcontainerAdd:
		// If we've already seen this event, return.
		if _, ok := self.watches[containerName]; ok {
			return nil
		}

		// New container was created, watch it.
		err := self.watchDirectory(event.Name, containerName)
		if err != nil {
			return err
		}
	case eventType == container.SubcontainerDelete:
		// If we've already seen this event, return.
		if _, ok := self.watches[containerName]; !ok {
			return nil
		}
		delete(self.watches, containerName)

		// Container was deleted, stop watching for it.
		err := self.watcher.RemoveWatch(event.Name)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown event type %v", eventType)
	}

	// Deliver the event.
	events <- container.SubcontainerEvent{
		EventType: eventType,
		Name:      containerName,
	}

	return nil
}

func (self *rawContainerHandler) WatchSubcontainers(events chan container.SubcontainerEvent) error {
	// Lazily initialize the watcher so we don't use it when not asked to.
	if self.watcher == nil {
		w, err := inotify.NewWatcher()
		if err != nil {
			return err
		}
		self.watcher = w
	}

	// Watch this container (all its cgroups) and all subdirectories.
	for _, mnt := range self.cgroupSubsystems.mounts {
		err := self.watchDirectory(path.Join(mnt.Mountpoint, self.name), self.name)
		if err != nil {
			return err
		}
	}

	// Process the events received from the kernel.
	go func() {
		for {
			select {
			case event := <-self.watcher.Event:
				err := self.processEvent(event, events)
				if err != nil {
					glog.Warningf("Error while processing event (%+v): %v", event, err)
				}
			case err := <-self.watcher.Error:
				glog.Warningf("Error while watching %q:", self.name, err)
			case <-self.stopWatcher:
				err := self.watcher.Close()
				if err == nil {
					self.stopWatcher <- err
					self.watcher = nil
					return
				}
			}
		}
	}()

	return nil
}

func (self *rawContainerHandler) StopWatchingSubcontainers() error {
	if self.watcher == nil {
		return fmt.Errorf("can't stop watch that has not started for container %q", self.name)
	}

	// Rendezvous with the watcher thread.
	self.stopWatcher <- nil
	return <-self.stopWatcher
}
