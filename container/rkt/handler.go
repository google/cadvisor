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
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils"

	"github.com/golang/glog"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	cgroupfs "github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/configs"
	"golang.org/x/exp/inotify"
)

type rktContainerHandler struct {
	// Name of the container for this handler.
	name               string
	cgroupSubsystems   *libcontainer.CgroupSubsystems
	machineInfoFactory info.MachineInfoFactory

	// Inotify event watcher.
	watcher *common.InotifyWatcher

	// Signal for watcher thread to stop.
	stopWatcher chan error

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
	return self.hasNetwork
}

func (self *rktContainerHandler) HasFilesystem() bool {
	return true
}

func newRktContainerHandler(name string, cgroupSubsystems *libcontainer.CgroupSubsystems, machineInfoFactory info.MachineInfoFactory, fsInfo fs.FsInfo, watcher *common.InotifyWatcher, rootFs string) (container.ContainerHandler, error) {
	rktPath := "/var/lib/rkt"

	glog.Infof("rkt cgroup name = %q", name)
	aliases := make([]string, 1)
	isPod := false

	parsed, err := parseName(name)
	if err != nil {
		return nil, fmt.Errorf("this should be impossible!, new handler failing, but factory allowed, name = %s", name)
	}
	if parsed.Container == "" {
		isPod = true
		aliases = append(aliases, parsed.Pod)
	} else {
		aliases = append(aliases, parsed.Pod+":"+parsed.Container)
	}
	glog.Infof("aliases = %s", aliases)

	// Create the cgroup paths.
	cgroupPaths := make(map[string]string, len(cgroupSubsystems.MountPoints))
	for key, val := range cgroupSubsystems.MountPoints {
		cgroupPaths[key] = path.Join(val, name)
	}

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

	pid := os.Getpid()
	if isPod {
		pids, err := cgroups.GetPids(cgroupPaths["cpu"])
		if err == nil {
			pid = pids[0]
		} else {
			glog.Infof("couldn't get processes as %v", err)
		}
	}

	hasNetwork := false
	if isPod {
		hasNetwork = true
	}
	var externalMounts []common.Mount
	for _, container := range cHints.AllHosts {
		if name == container.FullName {
			/*libcontainerState.NetworkState = network.NetworkState{
				VethHost:  container.NetworkInterface.VethHost,
				VethChild: container.NetworkInterface.VethChild,
			}
			hasNetwork = true*/
			externalMounts = container.Mounts
			break
		}
	}

	rootfsStorageDir := GetRootFs(rktPath, parsed)

	return &rktContainerHandler{
		name:               name,
		cgroupSubsystems:   cgroupSubsystems,
		machineInfoFactory: machineInfoFactory,
		stopWatcher:        make(chan error),
		cgroupPaths:        cgroupPaths,
		cgroupManager:      cgroupManager,
		fsInfo:             fsInfo,
		hasNetwork:         hasNetwork,
		externalMounts:     externalMounts,
		watcher:            watcher,
		rootFs:             rootFs,
		isPod:              isPod,
		aliases:            aliases,
		pid:                pid,
		rootfsStorageDir:   rootfsStorageDir,
		fsHandler:          common.NewFsHandler(time.Minute, rootfsStorageDir, "", fsInfo),
	}, nil
}

func (self *rktContainerHandler) ContainerReference() (info.ContainerReference, error) {
	// We only know the container by its one name.
	return info.ContainerReference{
		Name:      self.name,
		Aliases:   self.aliases,
		Namespace: RktNamespace,
	}, nil
}

func (self *rktContainerHandler) GetRootNetworkDevices() ([]info.NetInfo, error) {
	nd := []info.NetInfo{}
	if self.name == "/" {
		mi, err := self.machineInfoFactory.GetMachineInfo()
		if err != nil {
			return nd, err
		}
		return mi.NetworkDevices, nil
	}
	return nd, nil
}

// Nothing to start up.
func (self *rktContainerHandler) Start() {
	self.fsHandler.Start()
}

// Nothing to clean up.
func (self *rktContainerHandler) Cleanup() {
	self.fsHandler.Stop()
}

func (self *rktContainerHandler) GetSpec() (info.ContainerSpec, error) {
	return common.GetSpec(self)
}

func (self *rktContainerHandler) getFsStats(stats *info.ContainerStats) error {
	if self.isPod {
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
	// Docker does not impose any filesystem limits for containers. So use capacity as limit.
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
	stats, err := libcontainer.GetStats(self.cgroupManager, self.rootFs, self.pid)
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

func (self *rktContainerHandler) GetContainerLabels() map[string]string {
	return map[string]string{}
}

func (self *rktContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	containers := make(map[string]struct{})
	if self.isPod == false {
		var ret []info.ContainerReference
		return ret, nil
	}

	for _, cgroupPath := range self.cgroupPaths {
		glog.Infof("ListContainers: self.name = %q, cgroupPath = %q, listType = %v", self.name, cgroupPath, listType)
		err := common.ListDirectories(path.Join(cgroupPath, "system.slice"), path.Join(self.name, "system.slice"), listType == container.ListRecursive, containers)
		if err != nil {
			return nil, err
		}
	}

	// Make into container references.
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

// Watches the specified directory and all subdirectories. Returns whether the path was
// already being watched and an error (if any).
func (self *rktContainerHandler) watchDirectory(dir string, containerName string) (bool, error) {
	alreadyWatching, err := self.watcher.AddWatch(containerName, dir)
	if err != nil {
		return alreadyWatching, err
	}

	// Remove the watch if further operations failed.
	cleanup := true
	defer func() {
		if cleanup {
			_, err := self.watcher.RemoveWatch(containerName, dir)
			if err != nil {
				glog.Warningf("Failed to remove inotify watch for %q: %v", dir, err)
			}
		}
	}()

	// TODO(vmarmol): We should re-do this once we're done to ensure directories were not added in the meantime.
	// Watch subdirectories as well.
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return alreadyWatching, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			// TODO(vmarmol): We don't have to fail here, maybe we can recover and try to get as many registrations as we can.
			_, err = self.watchDirectory(path.Join(dir, entry.Name()), path.Join(containerName, entry.Name()))
			if err != nil {
				return alreadyWatching, err
			}
		}
	}

	cleanup = false
	return alreadyWatching, nil
}

func (self *rktContainerHandler) processEvent(event *inotify.Event, events chan container.SubcontainerEvent) error {
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
	for _, mount := range self.cgroupSubsystems.Mounts {
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
		// New container was created, watch it.
		alreadyWatched, err := self.watchDirectory(event.Name, containerName)
		if err != nil {
			return err
		}

		// Only report container creation once.
		if alreadyWatched {
			return nil
		}
	case eventType == container.SubcontainerDelete:
		// Container was deleted, stop watching for it.
		lastWatched, err := self.watcher.RemoveWatch(containerName, event.Name)
		if err != nil {
			return err
		}

		// Only report container deletion once.
		if !lastWatched {
			return nil
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

func (self *rktContainerHandler) WatchSubcontainers(events chan container.SubcontainerEvent) error {
	// Watch this container (all its cgroups) and all subdirectories.
	for _, cgroupPath := range self.cgroupPaths {
		_, err := self.watchDirectory(cgroupPath, self.name)
		if err != nil {
			return err
		}
	}

	// Process the events received from the kernel.
	go func() {
		for {
			select {
			case event := <-self.watcher.Event():
				err := self.processEvent(event, events)
				if err != nil {
					glog.Warningf("Error while processing event (%+v): %v", event, err)
				}
			case err := <-self.watcher.Error():
				glog.Warningf("Error while watching %q:", self.name, err)
			case <-self.stopWatcher:
				err := self.watcher.Close()
				if err == nil {
					self.stopWatcher <- err
					return
				}
			}
		}
	}()

	return nil
}

func (self *rktContainerHandler) StopWatchingSubcontainers() error {
	// Rendezvous with the watcher thread.
	self.stopWatcher <- nil
	return <-self.stopWatcher
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
