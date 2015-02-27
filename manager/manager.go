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

// Manager of cAdvisor-monitored containers.
package manager

import (
	"flag"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/libcontainer/cgroups"
	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/events"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage/memory"
	"github.com/google/cadvisor/utils/cpuload"
	"github.com/google/cadvisor/utils/oomparser"
	"github.com/google/cadvisor/utils/sysfs"
)

var globalHousekeepingInterval = flag.Duration("global_housekeeping_interval", 1*time.Minute, "Interval between global housekeepings")
var logCadvisorUsage = flag.Bool("log_cadvisor_usage", false, "Whether to log the usage of the cAdvisor container")

// The Manager interface defines operations for starting a manager and getting
// container and machine information.
type Manager interface {
	// Start the manager.
	Start() error

	// Stops the manager.
	Stop() error

	// Get information about a container.
	GetContainerInfo(containerName string, query *info.ContainerInfoRequest) (*info.ContainerInfo, error)

	// Get information about all subcontainers of the specified container (includes self).
	SubcontainersInfo(containerName string, query *info.ContainerInfoRequest) ([]*info.ContainerInfo, error)

	// Gets all the Docker containers. Return is a map from full container name to ContainerInfo.
	AllDockerContainers(query *info.ContainerInfoRequest) (map[string]info.ContainerInfo, error)

	// Gets information about a specific Docker container. The specified name is within the Docker namespace.
	DockerContainer(dockerName string, query *info.ContainerInfoRequest) (info.ContainerInfo, error)

	// Gets spec for a container.
	GetContainerSpec(containerName string) (info.ContainerSpec, error)

	// Get derived stats for a container.
	GetContainerDerivedStats(containerName string) (info.DerivedStats, error)

	// Get information about the machine.
	GetMachineInfo() (*info.MachineInfo, error)

	// Get version information about different components we depend on.
	GetVersionInfo() (*info.VersionInfo, error)

	// Get events streamed through passedChannel that fit the request.
	WatchForEvents(request *events.Request, passedChannel chan *events.Event) error

	// Get past events that have been detected and that fit the request.
	GetPastEvents(request *events.Request) (events.EventSlice, error)
}

// New takes a memory storage and returns a new manager.
func New(memoryStorage *memory.InMemoryStorage, sysfs sysfs.SysFs) (Manager, error) {
	if memoryStorage == nil {
		return nil, fmt.Errorf("manager requires memory storage")
	}

	// Detect the container we are running on.
	selfContainer, err := cgroups.GetThisCgroupDir("cpu")
	if err != nil {
		return nil, err
	}
	glog.Infof("cAdvisor running in container: %q", selfContainer)

	newManager := &manager{
		containers:        make(map[namespacedContainerName]*containerData),
		quitChannels:      make([]chan error, 0, 2),
		memoryStorage:     memoryStorage,
		cadvisorContainer: selfContainer,
	}

	machineInfo, err := getMachineInfo(sysfs)
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

	newManager.eventHandler = events.NewEventManager()

	return newManager, nil
}

// A namespaced container name.
type namespacedContainerName struct {
	// The namespace of the container. Can be empty for the root namespace.
	Namespace string

	// The name of the container in this namespace.
	Name string
}

type manager struct {
	containers             map[namespacedContainerName]*containerData
	containersLock         sync.RWMutex
	memoryStorage          *memory.InMemoryStorage
	machineInfo            info.MachineInfo
	versionInfo            info.VersionInfo
	quitChannels           []chan error
	cadvisorContainer      string
	dockerContainersRegexp *regexp.Regexp
	loadReader             cpuload.CpuLoadReader
	eventHandler           events.EventManager
}

// Start the container manager.
func (self *manager) Start() error {
	// TODO(rjnagal): Skip creating cpu load reader while we improve resource usage and accuracy.
	if false {
		// Create cpu load reader.
		cpuLoadReader, err := cpuload.New()
		if err != nil {
			// TODO(rjnagal): Promote to warning once we support cpu load inside namespaces.
			glog.Infof("Could not initialize cpu load reader: %s", err)
		} else {
			err = cpuLoadReader.Start()
			if err != nil {
				glog.Warning("Could not start cpu load stat collector: %s", err)
			} else {
				self.loadReader = cpuLoadReader
			}
		}
	}

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

	// Watch for new container.
	quitWatcher := make(chan error)
	err = self.watchForNewContainers(quitWatcher)
	if err != nil {
		return err
	}
	self.quitChannels = append(self.quitChannels, quitWatcher)
	// err = self.watchForNewOoms()
	// if err != nil {
	// 	glog.Errorf("Failed to start OOM watcher, will not get OOM events: %v", err)
	// }

	// Look for new containers in the main housekeeping thread.
	quitGlobalHousekeeping := make(chan error)
	self.quitChannels = append(self.quitChannels, quitGlobalHousekeeping)
	go self.globalHousekeeping(quitGlobalHousekeeping)

	return nil
}

func (self *manager) Stop() error {
	// Stop and wait on all quit channels.
	for i, c := range self.quitChannels {
		// Send the exit signal and wait on the thread to exit (by closing the channel).
		c <- nil
		err := <-c
		if err != nil {
			// Remove the channels that quit successfully.
			self.quitChannels = self.quitChannels[i:]
			return err
		}
	}
	self.quitChannels = make([]chan error, 0, 2)
	if self.loadReader != nil {
		self.loadReader.Stop()
		self.loadReader = nil
	}
	return nil
}

func (self *manager) globalHousekeeping(quit chan error) {
	// Long housekeeping is either 100ms or half of the housekeeping interval.
	longHousekeeping := 100 * time.Millisecond
	if *globalHousekeepingInterval/2 < longHousekeeping {
		longHousekeeping = *globalHousekeepingInterval / 2
	}

	ticker := time.Tick(*globalHousekeepingInterval)
	for {
		select {
		case t := <-ticker:
			start := time.Now()

			// Check for new containers.
			err := self.detectSubcontainers("/")
			if err != nil {
				glog.Errorf("Failed to detect containers: %s", err)
			}

			// Log if housekeeping took too long.
			duration := time.Since(start)
			if duration >= longHousekeeping {
				glog.V(1).Infof("Global Housekeeping(%d) took %s", t.Unix(), duration)
			}
		case <-quit:
			// Quit if asked to do so.
			quit <- nil
			glog.Infof("Exiting global housekeeping thread")
			return
		}
	}
}

func (self *manager) getContainerData(containerName string) (*containerData, error) {
	var cont *containerData
	var ok bool
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()

		// Ensure we have the container.
		cont, ok = self.containers[namespacedContainerName{
			Name: containerName,
		}]
	}()
	if !ok {
		return nil, fmt.Errorf("unknown container %q", containerName)
	}
	return cont, nil
}

func (self *manager) GetContainerSpec(containerName string) (info.ContainerSpec, error) {
	cont, err := self.getContainerData(containerName)
	if err != nil {
		return info.ContainerSpec{}, err
	}
	cinfo, err := cont.GetInfo()
	if err != nil {
		return info.ContainerSpec{}, err
	}
	return self.getAdjustedSpec(cinfo), nil
}

func (self *manager) getAdjustedSpec(cinfo *containerInfo) info.ContainerSpec {
	spec := cinfo.Spec

	// Set default value to an actual value
	if spec.HasMemory {
		// Memory.Limit is 0 means there's no limit
		if spec.Memory.Limit == 0 {
			spec.Memory.Limit = uint64(self.machineInfo.MemoryCapacity)
		}
	}
	return spec
}

// Get a container by name.
func (self *manager) GetContainerInfo(containerName string, query *info.ContainerInfoRequest) (*info.ContainerInfo, error) {
	cont, err := self.getContainerData(containerName)
	if err != nil {
		return nil, err
	}
	return self.containerDataToContainerInfo(cont, query)
}

func (self *manager) containerDataToContainerInfo(cont *containerData, query *info.ContainerInfoRequest) (*info.ContainerInfo, error) {
	// Get the info from the container.
	cinfo, err := cont.GetInfo()
	if err != nil {
		return nil, err
	}

	stats, err := self.memoryStorage.RecentStats(cinfo.Name, query.Start, query.End, query.NumStats)
	if err != nil {
		return nil, err
	}

	// Make a copy of the info for the user.
	ret := &info.ContainerInfo{
		ContainerReference: cinfo.ContainerReference,
		Subcontainers:      cinfo.Subcontainers,
		Spec:               self.getAdjustedSpec(cinfo),
		Stats:              stats,
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

	return self.containerDataSliceToContainerInfoSlice(containers, query)
}

func (self *manager) AllDockerContainers(query *info.ContainerInfoRequest) (map[string]info.ContainerInfo, error) {
	var containers map[string]*containerData
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()
		containers = make(map[string]*containerData, len(self.containers))

		// Get containers in the Docker namespace.
		for name, cont := range self.containers {
			if name.Namespace == docker.DockerNamespace {
				containers[cont.info.Name] = cont
			}
		}
	}()

	output := make(map[string]info.ContainerInfo, len(containers))
	for name, cont := range containers {
		inf, err := self.containerDataToContainerInfo(cont, query)
		if err != nil {
			return nil, err
		}
		output[name] = *inf
	}
	return output, nil
}

func (self *manager) DockerContainer(containerName string, query *info.ContainerInfoRequest) (info.ContainerInfo, error) {
	var container *containerData = nil
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()

		// Check for the container in the Docker container namespace.
		cont, ok := self.containers[namespacedContainerName{
			Namespace: docker.DockerNamespace,
			Name:      containerName,
		}]
		if ok {
			container = cont
		}
	}()
	if container == nil {
		return info.ContainerInfo{}, fmt.Errorf("unable to find Docker container %q", containerName)
	}

	inf, err := self.containerDataToContainerInfo(container, query)
	if err != nil {
		return info.ContainerInfo{}, err
	}
	return *inf, nil
}

func (self *manager) containerDataSliceToContainerInfoSlice(containers []*containerData, query *info.ContainerInfoRequest) ([]*info.ContainerInfo, error) {
	if len(containers) == 0 {
		return nil, fmt.Errorf("no containers found")
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

func (self *manager) GetContainerDerivedStats(containerName string) (info.DerivedStats, error) {
	var ok bool
	var cont *containerData
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()
		cont, ok = self.containers[namespacedContainerName{Name: containerName}]
	}()
	if !ok {
		return info.DerivedStats{}, fmt.Errorf("unknown container %q", containerName)
	}
	return cont.DerivedStats()
}

func (m *manager) GetMachineInfo() (*info.MachineInfo, error) {
	// Copy and return the MachineInfo.
	return &m.machineInfo, nil
}

func (m *manager) GetVersionInfo() (*info.VersionInfo, error) {
	return &m.versionInfo, nil
}

// Create a container.
func (m *manager) createContainer(containerName string) error {
	handler, err := container.NewContainerHandler(containerName)
	if err != nil {
		return err
	}
	logUsage := *logCadvisorUsage && containerName == m.cadvisorContainer
	cont, err := newContainerData(containerName, m.memoryStorage, handler, m.loadReader, logUsage)
	if err != nil {
		return err
	}

	// Add to the containers map.
	alreadyExists := func() bool {
		m.containersLock.Lock()
		defer m.containersLock.Unlock()

		namespacedName := namespacedContainerName{
			Name: containerName,
		}

		// Check that the container didn't already exist.
		_, ok := m.containers[namespacedName]
		if ok {
			return true
		}

		// Add the container name and all its aliases. The aliases must be within the namespace of the factory.
		m.containers[namespacedName] = cont
		for _, alias := range cont.info.Aliases {
			m.containers[namespacedContainerName{
				Namespace: cont.info.Namespace,
				Name:      alias,
			}] = cont
		}

		return false
	}()
	if alreadyExists {
		return nil
	}
	glog.Infof("Added container: %q (aliases: %v, namespace: %q)", containerName, cont.info.Aliases, cont.info.Namespace)

	// Start the container's housekeeping.
	cont.Start()

	return nil
}

func (m *manager) destroyContainer(containerName string) error {
	m.containersLock.Lock()
	defer m.containersLock.Unlock()

	namespacedName := namespacedContainerName{
		Name: containerName,
	}
	cont, ok := m.containers[namespacedName]
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
	delete(m.containers, namespacedName)
	for _, alias := range cont.info.Aliases {
		delete(m.containers, namespacedContainerName{
			Namespace: cont.info.Namespace,
			Name:      alias,
		})
	}
	glog.Infof("Destroyed container: %q (aliases: %v, namespace: %q)", containerName, cont.info.Aliases, cont.info.Namespace)
	return nil
}

// Detect all containers that have been added or deleted from the specified container.
func (m *manager) getContainersDiff(containerName string) (added []info.ContainerReference, removed []info.ContainerReference, err error) {
	m.containersLock.RLock()
	defer m.containersLock.RUnlock()

	// Get all subcontainers recursively.
	cont, ok := m.containers[namespacedContainerName{
		Name: containerName,
	}]
	if !ok {
		return nil, nil, fmt.Errorf("failed to find container %q while checking for new containers", containerName)
	}
	allContainers, err := cont.handler.ListContainers(container.ListRecursive)
	if err != nil {
		return nil, nil, err
	}
	allContainers = append(allContainers, info.ContainerReference{Name: containerName})

	// Determine which were added and which were removed.
	allContainersSet := make(map[string]*containerData)
	for name, d := range m.containers {
		// Only add the canonical name.
		if d.info.Name == name.Name {
			allContainersSet[name.Name] = d
		}
	}

	// Added containers
	for _, c := range allContainers {
		delete(allContainersSet, c.Name)
		_, ok := m.containers[namespacedContainerName{
			Name: c.Name,
		}]
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

// Watches for new containers started in the system. Runs forever unless there is a setup error.
func (self *manager) watchForNewContainers(quit chan error) error {
	var root *containerData
	var ok bool
	func() {
		self.containersLock.RLock()
		defer self.containersLock.RUnlock()
		root, ok = self.containers[namespacedContainerName{
			Name: "/",
		}]
	}()
	if !ok {
		return fmt.Errorf("root container does not exist when watching for new containers")
	}

	// Register for new subcontainers.
	eventsChannel := make(chan container.SubcontainerEvent, 16)
	err := root.handler.WatchSubcontainers(eventsChannel)
	if err != nil {
		return err
	}

	// There is a race between starting the watch and new container creation so we do a detection before we read new containers.
	err = self.detectSubcontainers("/")
	if err != nil {
		return err
	}

	// Listen to events from the container handler.
	go func() {
		for {
			select {
			case event := <-eventsChannel:
				switch {
				case event.EventType == container.SubcontainerAdd:
					err = self.createContainer(event.Name)
				case event.EventType == container.SubcontainerDelete:
					err = self.destroyContainer(event.Name)
				}
				if err != nil {
					glog.Warning("Failed to process watch event: %v", err)
				}
			case <-quit:
				// Stop processing events if asked to quit.
				err := root.handler.StopWatchingSubcontainers()
				quit <- err
				if err == nil {
					glog.Infof("Exiting thread watching subcontainers")
					return
				}
			}
		}
	}()
	return nil
}

func (self *manager) watchForNewOoms() error {
	outStream := make(chan *oomparser.OomInstance, 10)
	oomLog, err := oomparser.New()
	if err != nil {
		return err
	}
	err = oomLog.StreamOoms(outStream)
	if err != nil {
		return err
	}
	go func() {
		for oomInstance := range outStream {
			newEvent := &events.Event{
				ContainerName: oomInstance.ContainerName,
				Timestamp:     oomInstance.TimeOfDeath,
				EventType:     events.TypeOom,
				EventData:     oomInstance,
			}
			glog.V(1).Infof("Created an oom event: %v", newEvent)
			err := self.eventHandler.AddEvent(newEvent)
			if err != nil {
				glog.Errorf("Failed to add event %v, got error: %v", newEvent, err)
			}
		}
	}()
	return nil
}

// can be called by the api which will take events returned on the channel
func (self *manager) WatchForEvents(request *events.Request, passedChannel chan *events.Event) error {
	return self.eventHandler.WatchEvents(passedChannel, request)
}

// can be called by the api which will return all events satisfying the request
func (self *manager) GetPastEvents(request *events.Request) (events.EventSlice, error) {
	return self.eventHandler.GetEvents(request)
}
