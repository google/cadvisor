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

package manager

import (
	"flag"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
)

var globalHousekeepingInterval = flag.Duration("global_housekeeping_interval", 1*time.Minute, "Interval between global housekeepings")

type Manager interface {
	// Start the manager, blocks forever.
	Start() error

	// Get information about a container.
	GetContainerInfo(containerName string, query *info.ContainerInfoRequest) (*info.ContainerInfo, error)

	// Get information about all subcontainers of the specified container (includes self).
	SubcontainersInfo(containerName string, query *info.ContainerInfoRequest) ([]*info.ContainerInfo, error)

	// Get information about the machine.
	GetMachineInfo() (*info.MachineInfo, error)

	// Get version information about different components we depend on.
	GetVersionInfo() (*info.VersionInfo, error)
}

func New(driver storage.StorageDriver) (Manager, error) {
	if driver == nil {
		return nil, fmt.Errorf("nil storage driver!")
	}
	newManager := &manager{}
	newManager.containers = make(map[string]*containerData)

	machineInfo, err := getMachineInfo()
	if err != nil {
		return nil, err
	}
	newManager.machineInfo = *machineInfo
	glog.Infof("Machine: %+v", newManager.machineInfo)

	versionInfo, err := getVersionInfo()
	if err != nil {
		return nil, err
	}
	newManager.versionInfo = *versionInfo
	glog.Infof("Version: %+v", newManager.versionInfo)
	newManager.storageDriver = driver

	return newManager, nil
}

type manager struct {
	containers     map[string]*containerData
	containersLock sync.RWMutex
	storageDriver  storage.StorageDriver
	machineInfo    info.MachineInfo
	versionInfo    info.VersionInfo
}

// Start the container manager.
func (self *manager) Start() error {
	// Create root and then recover all containers.
	err := self.createContainer("/")
	if err != nil {
		return err
	}
	glog.Infof("Starting recovery of all containers")
	err = self.detectSubcontainers("/")
	if err != nil {
		return err
	}
	glog.Infof("Recovery completed")

	// Long housekeeping is either 100ms or half of the housekeeping interval.
	longHousekeeping := 100 * time.Millisecond
	if *globalHousekeepingInterval/2 < longHousekeeping {
		longHousekeeping = *globalHousekeepingInterval / 2
	}

	// Watch for new container.
	go self.watchForNewContainers()

	// Look for new containers in the main housekeeping thread.
	ticker := time.Tick(*globalHousekeepingInterval)
	for t := range ticker {
		start := time.Now()

		// Check for new containers.
		err = self.detectSubcontainers("/")
		if err != nil {
			glog.Errorf("Failed to detect containers: %s", err)
		}

		// Log if housekeeping took more than 100ms.
		duration := time.Since(start)
		if duration >= longHousekeeping {
			glog.V(1).Infof("Global Housekeeping(%d) took %s", t.Unix(), duration)
		}
	}

	return nil
}

// Get a container by name.
func (self *manager) GetContainerInfo(containerName string, query *info.ContainerInfoRequest) (*info.ContainerInfo, error) {
	var cont *containerData
	var ok bool
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()

		// Ensure we have the container.
		cont, ok = self.containers[containerName]
	}()
	if !ok {
		return nil, fmt.Errorf("unknown container %q", containerName)
	}

	return self.containerDataToContainerInfo(cont, query)
}

func (self *manager) containerDataToContainerInfo(cont *containerData, query *info.ContainerInfoRequest) (*info.ContainerInfo, error) {
	// Get the info from the container.
	cinfo, err := cont.GetInfo()
	if err != nil {
		return nil, err
	}

	stats, err := self.storageDriver.RecentStats(cinfo.Name, query.NumStats)
	if err != nil {
		return nil, err
	}

	// Make a copy of the info for the user.
	ret := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{
			Name:    cinfo.Name,
			Aliases: cinfo.Aliases,
		},
		Subcontainers: cinfo.Subcontainers,
		Spec:          cinfo.Spec,
		Stats:         stats,
	}

	// Set default value to an actual value
	if ret.Spec.HasMemory {
		// Memory.Limit is 0 means there's no limit
		if ret.Spec.Memory.Limit == 0 {
			ret.Spec.Memory.Limit = uint64(self.machineInfo.MemoryCapacity)
		}
	}
	return ret, nil
}

func (self *manager) SubcontainersInfo(containerName string, query *info.ContainerInfoRequest) ([]*info.ContainerInfo, error) {
	var containers []*containerData
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()
		containers = make([]*containerData, 0, len(self.containers))

		// Get all the subcontainers of the specified container
		matchedName := path.Join(containerName, "/")
		for i := range self.containers {
			name := self.containers[i].info.Name
			if name == containerName || strings.HasPrefix(name, matchedName) {
				containers = append(containers, self.containers[i])
			}
		}
	}()
	if len(containers) == 0 {
		return nil, fmt.Errorf("unknown container %q", containerName)
	}

	// Get the info for each container.
	output := make([]*info.ContainerInfo, 0, len(containers))
	for i := range containers {
		cinfo, err := self.containerDataToContainerInfo(containers[i], query)
		if err != nil {
			// Skip containers with errors, we try to degrade gracefully.
			continue
		}
		output = append(output, cinfo)
	}

	return output, nil
}

func (m *manager) GetMachineInfo() (*info.MachineInfo, error) {
	// Copy and return the MachineInfo.
	ret := m.machineInfo
	return &ret, nil
}

func (m *manager) GetVersionInfo() (*info.VersionInfo, error) {
	ret := m.versionInfo
	return &ret, nil
}

// Create a container.
func (m *manager) createContainer(containerName string) error {
	cont, err := NewContainerData(containerName, m.storageDriver)
	if err != nil {
		return err
	}

	// Add to the containers map.
	alreadyExists := func() bool {
		m.containersLock.Lock()
		defer m.containersLock.Unlock()

		// Check that the container didn't already exist
		_, ok := m.containers[containerName]
		if ok {
			return true
		}

		// Add the container name and all its aliases.
		m.containers[containerName] = cont
		for _, alias := range cont.info.Aliases {
			m.containers[alias] = cont
		}

		return false
	}()
	if alreadyExists {
		return nil
	}
	glog.Infof("Added container: %q (aliases: %s)", containerName, cont.info.Aliases)

	// Start the container's housekeeping.
	cont.Start()
	return nil
}

func (m *manager) destroyContainer(containerName string) error {
	m.containersLock.Lock()
	defer m.containersLock.Unlock()

	cont, ok := m.containers[containerName]
	if !ok {
		// Already destroyed, done.
		return nil
	}

	// Tell the container to stop.
	err := cont.Stop()
	if err != nil {
		return err
	}

	// Remove the container from our records (and all its aliases).
	delete(m.containers, containerName)
	for _, alias := range cont.info.Aliases {
		delete(m.containers, alias)
	}
	glog.Infof("Destroyed container: %s (aliases: %s)", containerName, cont.info.Aliases)
	return nil
}

// Detect all containers that have been added or deleted from the specified container.
func (m *manager) getContainersDiff(containerName string) (added []info.ContainerReference, removed []info.ContainerReference, err error) {
	m.containersLock.RLock()
	defer m.containersLock.RUnlock()

	// Get all subcontainers recursively.
	cont, ok := m.containers[containerName]
	if !ok {
		return nil, nil, fmt.Errorf("Failed to find container %q while checking for new containers", containerName)
	}
	allContainers, err := cont.handler.ListContainers(container.LIST_RECURSIVE)
	if err != nil {
		return nil, nil, err
	}
	allContainers = append(allContainers, info.ContainerReference{Name: containerName})

	// Determine which were added and which were removed.
	allContainersSet := make(map[string]*containerData)
	for name, d := range m.containers {
		// Only add the canonical name.
		if d.info.Name == name {
			allContainersSet[name] = d
		}
	}

	// Added containers
	for _, c := range allContainers {
		delete(allContainersSet, c.Name)
		_, ok := m.containers[c.Name]
		if !ok {
			added = append(added, c)
		}
	}

	// Removed ones are no longer in the container listing.
	for _, d := range allContainersSet {
		removed = append(removed, d.info.ContainerReference)
	}

	return
}

// Detect the existing subcontainers and reflect the setup here.
func (m *manager) detectSubcontainers(containerName string) error {
	added, removed, err := m.getContainersDiff(containerName)
	if err != nil {
		return err
	}

	// Add the new containers.
	for _, cont := range added {
		err = m.createContainer(cont.Name)
		if err != nil {
			glog.Errorf("Failed to create existing container: %s: %s", cont.Name, err)
		}
	}

	// Remove the old containers.
	for _, cont := range removed {
		err = m.destroyContainer(cont.Name)
		if err != nil {
			glog.Errorf("Failed to destroy existing container: %s: %s", cont.Name, err)
		}
	}

	return nil
}

func (self *manager) processEvent(event container.SubcontainerEvent) error {
	var err error = nil
	return err
}

// Watches for new containers started in the system. Runs forever unless there is a setup error.
func (self *manager) watchForNewContainers() error {
	var root *containerData
	var ok bool
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()
		root, ok = self.containers["/"]
	}()
	if !ok {
		return fmt.Errorf("root container does not exist when watching for new containers")
	}

	// Register for new subcontainers.
	events := make(chan container.SubcontainerEvent, 16)
	err := root.handler.WatchSubcontainers(events)
	if err != nil {
		return err
	}

	// There is a race between starting the watch and new container creation so we do a detection before we read new containers.
	err = self.detectSubcontainers("/")
	if err != nil {
		return err
	}

	// Listen to events from the container handler.
	for event := range events {
		switch {
		case event.EventType == container.SUBCONTAINER_ADD:
			err = self.createContainer(event.Name)
		case event.EventType == container.SUBCONTAINER_DELETE:
			err = self.destroyContainer(event.Name)
		}
		if err != nil {
			glog.Warning("Failed to process watch event: %v", err)
		}
	}

	return nil
}
