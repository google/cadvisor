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

// Manager of cAdvisor-monitored rkt containers.

package manager

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/cadvisor/cache/memory"
	"github.com/google/cadvisor/collector"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/raw"
	"github.com/google/cadvisor/events"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/info/v2"
	"github.com/google/cadvisor/utils/cpuload"
	"github.com/google/cadvisor/utils/oomparser"
	"github.com/google/cadvisor/utils/sysfs"

	"github.com/golang/glog"
	"github.com/opencontainers/runc/libcontainer/cgroups"
)

type rktManager struct {
	containers               map[namespacedContainerName]*containerData
	containersLock           sync.RWMutex
	memoryCache              *memory.InMemoryCache
	fsInfo                   fs.FsInfo
	machineInfo              info.MachineInfo
	quitChannels             []chan error
	cadvisorContainer        string
	inHostNamespace          bool
	loadReader               cpuload.CpuLoadReader
	eventHandler             events.EventManager
	startupTime              time.Time
	maxHousekeepingInterval  time.Duration
	allowDynamicHousekeeping bool
}

func NewRktManager(memoryCache *memory.InMemoryCache, sysfs sysfs.SysFs, maxHousekeepingInterval time.Duration, allowDynamicHousekeeping bool, rktPath string) (Manager, error) {
	if memoryCache == nil {
		return nil, fmt.Errorf("manager requires memory storage")
	}

	// Detect the container we are running on.
	selfContainer, err := cgroups.GetThisCgroupDir("cpu")
	if err != nil {
		return nil, err
	}
	glog.Infof("cAdvisor running in container: %q", selfContainer)

	context := fs.Context{RktPath: rktPath}
	fsInfo, err := fs.NewFsInfo(context)
	if err != nil {
		return nil, err
	}

	// If cAdvisor was started with host's rootfs mounted, assume that its running
	// in its own namespaces.
	inHostNamespace := false
	if _, err := os.Stat("/rootfs/proc"); os.IsNotExist(err) {
		inHostNamespace = true
	}
	newManager := &rktManager{
		containers:               make(map[namespacedContainerName]*containerData),
		quitChannels:             make([]chan error, 0, 2),
		memoryCache:              memoryCache,
		fsInfo:                   fsInfo,
		cadvisorContainer:        selfContainer,
		inHostNamespace:          inHostNamespace,
		startupTime:              time.Now(),
		maxHousekeepingInterval:  maxHousekeepingInterval,
		allowDynamicHousekeeping: allowDynamicHousekeeping,
	}

	machineInfo, err := getMachineInfo(sysfs, fsInfo, inHostNamespace)
	newManager.machineInfo = *machineInfo
	glog.Infof("Machine: %+v", newManager.machineInfo)

	versionInfo, err := getVersionInfo()
	if err != nil {
		return nil, err
	}
	glog.Infof("Version: %+v", *versionInfo)

	newManager.eventHandler = events.NewEventManager(parseEventsStoragePolicy())
	return newManager, nil
}

func (self *rktManager) Start() error {
	// Register the raw driver.
	err := raw.Register(self, self.fsInfo)
	if err != nil {
		glog.Errorf("Registration of the raw container factory failed: %v", err)
	}

	if *enableLoadReader {
		// Create cpu load reader.
		cpuLoadReader, err := cpuload.New()
		if err != nil {
			// TODO(rjnagal): Promote to warning once we support cpu load inside namespaces.
			glog.Infof("Could not initialize cpu load reader: %s", err)
		} else {
			err = cpuLoadReader.Start()
			if err != nil {
				glog.Warningf("Could not start cpu load stat collector: %s", err)
			} else {
				self.loadReader = cpuLoadReader
			}
		}
	}

	// Watch for OOMs.
	err = self.watchForNewOoms()
	if err != nil {
		glog.Warningf("Could not configure a source for OOM detection, disabling OOM events: %v", err)
	}

	// Create root and then recover all containers.
	err = self.createContainer("/")
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

	// Look for new containers in the main housekeeping thread.
	quitGlobalHousekeeping := make(chan error)
	self.quitChannels = append(self.quitChannels, quitGlobalHousekeeping)
	go self.globalHousekeeping(quitGlobalHousekeeping)

	return nil
}

func (self *rktManager) Stop() error {
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

func (self *rktManager) GetDerivedStats(containerName string, options v2.RequestOptions) (map[string]v2.DerivedStats, error) {
	return nil, fmt.Errorf("unknown container %q", containerName)
}

func (self *rktManager) GetContainerSpec(containerName string, options v2.RequestOptions) (map[string]v2.ContainerSpec, error) {
	return nil, fmt.Errorf("unknown container %q", containerName)
}

func (self *rktManager) GetContainerInfo(containerName string, query *info.ContainerInfoRequest) (*info.ContainerInfo, error) {
	return nil, fmt.Errorf("unknown container %q", containerName)
}

func (self *rktManager) GetContainerInfoV2(containerName string, options v2.RequestOptions) (map[string]v2.ContainerInfo, error) {
	return nil, fmt.Errorf("unknown container %q", containerName)
}

func (self *rktManager) SubcontainersInfo(containerName string, query *info.ContainerInfoRequest) ([]*info.ContainerInfo, error) {
	return nil, fmt.Errorf("unknown container %q", containerName)
}

func (self *rktManager) AllDockerContainers(query *info.ContainerInfoRequest) (map[string]info.ContainerInfo, error) {
	return nil, fmt.Errorf("Not Docker")
}

func (self *rktManager) DockerContainer(containerName string, query *info.ContainerInfoRequest) (info.ContainerInfo, error) {
	return info.ContainerInfo{}, fmt.Errorf("Not Docker")
}

func (self *rktManager) GetRequestedContainersInfo(containerName string, options v2.RequestOptions) (map[string]*info.ContainerInfo, error) {
	return nil, fmt.Errorf("unknown container %q", containerName)
}

//duplicated from manager.go, probably needs to figure out how to make it cleaner by sharing code
func (self *rktManager) GetFsInfo(label string) ([]v2.FsInfo, error) {
	var empty time.Time
	// Get latest data from filesystems hanging off root container.
	stats, err := self.memoryCache.RecentStats("/", empty, empty, 1)
	if err != nil {
		return nil, err
	}
	dev := ""
	if len(label) != 0 {
		dev, err = self.fsInfo.GetDeviceForLabel(label)
		if err != nil {
			return nil, err
		}
	}
	fsInfo := []v2.FsInfo{}
	for _, fs := range stats[0].Filesystem {
		if len(label) != 0 && fs.Device != dev {
			continue
		}
		mountpoint, err := self.fsInfo.GetMountpointForDevice(fs.Device)
		if err != nil {
			return nil, err
		}
		labels, err := self.fsInfo.GetLabelsForDevice(fs.Device)
		if err != nil {
			return nil, err
		}
		fi := v2.FsInfo{
			Device:     fs.Device,
			Mountpoint: mountpoint,
			Capacity:   fs.Limit,
			Usage:      fs.Usage,
			Available:  fs.Available,
			Labels:     labels,
		}
		fsInfo = append(fsInfo, fi)
	}
	return fsInfo, nil
}

func (m *rktManager) GetMachineInfo() (*info.MachineInfo, error) {
	// Copy and return the MachineInfo.
	return &m.machineInfo, nil
}

func (m *rktManager) GetVersionInfo() (*info.VersionInfo, error) {
	return getVersionInfo()
}

func (m *rktManager) Exists(containerName string) bool {
	return false
}

func (m *rktManager) GetProcessList(containerName string, options v2.RequestOptions) ([]v2.ProcessInfo, error) {
	return nil, fmt.Errorf("unknown container %q", containerName)
}

// can be called by the api which will take events returned on the channel
func (self *rktManager) WatchForEvents(request *events.Request) (*events.EventChannel, error) {
	return self.eventHandler.WatchEvents(request)
}

// can be called by the api which will return all events satisfying the request
func (self *rktManager) GetPastEvents(request *events.Request) ([]*info.Event, error) {
	return self.eventHandler.GetEvents(request)
}

// called by the api when a client is no longer listening to the channel
func (self *rktManager) CloseEventChannel(watch_id int) {
	self.eventHandler.StopWatch(watch_id)
}

func (m *rktManager) DockerImages() ([]DockerImage, error) {
	return nil, fmt.Errorf("Not Docker")
}

func (m *rktManager) DockerInfo() (DockerStatus, error) {
	return DockerStatus{}, fmt.Errorf("Not Docker")
}

func (m *rktManager) DebugInfo() map[string][]string {
	debugInfo := container.DebugInfo()
	return debugInfo
}

// Create a container.
// taken fronm manager.go
func (m *rktManager) createContainer(containerName string) error {
	m.containersLock.Lock()
	defer m.containersLock.Unlock()

	namespacedName := namespacedContainerName{
		Name: containerName,
	}

	// Check that the container didn't already exist.
	if _, ok := m.containers[namespacedName]; ok {
		return nil
	}

	handler, accept, err := container.NewContainerHandler(containerName, m.inHostNamespace)
	if err != nil {
		return err
	}
	if !accept {
		// ignoring this container.
		glog.V(4).Infof("ignoring container %q", containerName)
		return nil
	}
	collectorManager, err := collector.NewCollectorManager()
	if err != nil {
		return err
	}

	logUsage := *logCadvisorUsage && containerName == m.cadvisorContainer
	cont, err := newContainerData(containerName, m.memoryCache, handler, m.loadReader, logUsage, collectorManager, m.maxHousekeepingInterval, m.allowDynamicHousekeeping)
	if err != nil {
		return err
	}

	// Add collectors
	labels := handler.GetContainerLabels()
	collectorConfigs := collector.GetCollectorConfigs(labels)
	err = m.registerCollectors(collectorConfigs, cont)
	if err != nil {
		glog.Infof("failed to register collectors for %q: %v", containerName, err)
		return err
	}

	// Add the container name and all its aliases. The aliases must be within the namespace of the factory.
	m.containers[namespacedName] = cont
	for _, alias := range cont.info.Aliases {
		m.containers[namespacedContainerName{
			Namespace: cont.info.Namespace,
			Name:      alias,
		}] = cont
	}

	glog.V(3).Infof("Added container: %q (aliases: %v, namespace: %q)", containerName, cont.info.Aliases, cont.info.Namespace)

	contSpec, err := cont.handler.GetSpec()
	if err != nil {
		return err
	}

	contRef, err := cont.handler.ContainerReference()
	if err != nil {
		return err
	}

	newEvent := &info.Event{
		ContainerName: contRef.Name,
		Timestamp:     contSpec.CreationTime,
		EventType:     info.EventContainerCreation,
	}
	err = m.eventHandler.AddEvent(newEvent)
	if err != nil {
		return err
	}

	// Start the container's housekeeping.
	return cont.Start()
}

func (self *rktManager) watchForNewOoms() error {
	glog.Infof("Started watching for new ooms in manager")
	outStream := make(chan *oomparser.OomInstance, 10)
	oomLog, err := oomparser.New()
	if err != nil {
		return err
	}
	go oomLog.StreamOoms(outStream)

	go func() {
		for oomInstance := range outStream {
			// Surface OOM and OOM kill events.
			newEvent := &info.Event{
				ContainerName: oomInstance.ContainerName,
				Timestamp:     oomInstance.TimeOfDeath,
				EventType:     info.EventOom,
			}
			err := self.eventHandler.AddEvent(newEvent)
			if err != nil {
				glog.Errorf("failed to add OOM event for %q: %v", oomInstance.ContainerName, err)
			}
			glog.V(3).Infof("Created an OOM event in container %q at %v", oomInstance.ContainerName, oomInstance.TimeOfDeath)

			newEvent = &info.Event{
				ContainerName: oomInstance.VictimContainerName,
				Timestamp:     oomInstance.TimeOfDeath,
				EventType:     info.EventOomKill,
				EventData: info.EventData{
					OomKill: &info.OomKillEventData{
						Pid:         oomInstance.Pid,
						ProcessName: oomInstance.ProcessName,
					},
				},
			}
			err = self.eventHandler.AddEvent(newEvent)
			if err != nil {
				glog.Errorf("failed to add OOM kill event for %q: %v", oomInstance.ContainerName, err)
			}
		}
	}()
	return nil
}

func (self *rktManager) globalHousekeeping(quit chan error) {
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
				glog.V(3).Infof("Global Housekeeping(%d) took %s", t.Unix(), duration)
			}
		case <-quit:
			// Quit if asked to do so.
			quit <- nil
			glog.Infof("Exiting global housekeeping thread")
			return
		}
	}
}

// Detect the existing subcontainers and reflect the setup here.
func (m *rktManager) detectSubcontainers(containerName string) error {
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
func (self *rktManager) watchForNewContainers(quit chan error) error {
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
					glog.Warningf("Failed to process watch event: %v", err)
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

func (m *rktManager) registerCollectors(collectorConfigs map[string]string, cont *containerData) error {
	for k, v := range collectorConfigs {
		configFile, err := cont.ReadFile(v, m.inHostNamespace)
		if err != nil {
			return fmt.Errorf("failed to read config file %q for config %q, container %q: %v", k, v, cont.info.Name, err)
		}
		glog.V(3).Infof("Got config from %q: %q", v, configFile)

		if strings.HasPrefix(k, "prometheus") || strings.HasPrefix(k, "Prometheus") {
			newCollector, err := collector.NewPrometheusCollector(k, configFile)
			if err != nil {
				glog.Infof("failed to create collector for container %q, config %q: %v", cont.info.Name, k, err)
				return err
			}
			err = cont.collectorManager.RegisterCollector(newCollector)
			if err != nil {
				glog.Infof("failed to register collector for container %q, config %q: %v", cont.info.Name, k, err)
				return err
			}
		} else {
			newCollector, err := collector.NewCollector(k, configFile)
			if err != nil {
				glog.Infof("failed to create collector for container %q, config %q: %v", cont.info.Name, k, err)
				return err
			}
			err = cont.collectorManager.RegisterCollector(newCollector)
			if err != nil {
				glog.Infof("failed to register collector for container %q, config %q: %v", cont.info.Name, k, err)
				return err
			}
		}
	}
	return nil
}

// Detect all containers that have been added or deleted from the specified container.
func (m *rktManager) getContainersDiff(containerName string) (added []info.ContainerReference, removed []info.ContainerReference, err error) {
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

func (m *rktManager) destroyContainer(containerName string) error {
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
	glog.V(3).Infof("Destroyed container: %q (aliases: %v, namespace: %q)", containerName, cont.info.Aliases, cont.info.Namespace)

	contRef, err := cont.handler.ContainerReference()
	if err != nil {
		return err
	}

	newEvent := &info.Event{
		ContainerName: contRef.Name,
		Timestamp:     time.Now(),
		EventType:     info.EventContainerDeletion,
	}
	err = m.eventHandler.AddEvent(newEvent)
	if err != nil {
		return err
	}
	return nil
}
