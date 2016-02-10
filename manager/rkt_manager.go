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
	"time"

	"github.com/google/cadvisor/cache/memory"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/events"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/info/v2"
	"github.com/google/cadvisor/utils/cpuload"
	"github.com/google/cadvisor/utils/sysfs"

	"github.com/golang/glog"
)

type rktManager struct {
	memoryCache             *memory.InMemoryCache
	fsInfo                  fs.FsInfo
	machineInfo             info.MachineInfo
	loadReader              cpuload.CpuLoadReader
	eventHandler            events.EventManager
	startupTime             time.Time
	maxHousekeepingInterval time.Duration
	inHostNamespace         bool
}

func NewRktManager(memoryCache *memory.InMemoryCache, sysfs sysfs.SysFs, maxHousekeepingInterval time.Duration, allowDynamicHousekeeping bool, rktPath string) (Manager, error) {
	if memoryCache == nil {
		return nil, fmt.Errorf("manager requires memory storage")
	}

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
		memoryCache:     memoryCache,
		fsInfo:          fsInfo,
		inHostNamespace: inHostNamespace,
		startupTime:     time.Now(),
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
	return nil
}

func (self *rktManager) Stop() error {
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
