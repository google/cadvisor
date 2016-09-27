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

package atsd

import (
	"flag"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/storage"

	atsdStorageDriver "github.com/axibase/atsd-storage-driver/storage"
)

const (
	startDelay           = 15 * time.Second       // waiting to store enough data for all entities before send
	timestampPeriodError = 250 * time.Millisecond // time error accumulated during housekeeping

	metricPrefix = "cadvisor"

	// metric groups for a deduplication
	cpuGroup       = "cpu"
	ioGroup        = "io"
	memoryGroup    = "memory"
	taskGroup      = "task"
	networkGroup   = "network"
	filesytemGroup = "filesystem"

	dockerHostDefault = "empty_flag"
)

var (
	protocol             = flag.String("storage_driver_atsd_protocol", "tcp", "transfer protocol. Supported protocols: http, https, udp, tcp")
	skipVerify           = flag.Bool("storage_driver_atsd_skip_verify", false, "controls whether a client verifies the server's certificate chain and host name")
	senderGoroutineLimit = flag.Int("storage_driver_atsd_sender_thread_limit", 4, "maximum thread (goroutine) count sending data to ATSD server via tcp/udp")
	memstoreLimit        = flag.Uint("storage_driver_atsd_buffer_limit", 1000000, "maximum network command count stored in buffer before being sent into ATSD")

	dockerHost             = flag.String("storage_driver_atsd_docker_host", dockerHostDefault, "hostname of the docker host, used as entity prefix")
	includeAllMajorNumbers = flag.Bool("storage_driver_atsd_store_major_numbers", false, "include statistics for devices with all available major numbers")
	userCgroupsEnabled     = flag.Bool("storage_driver_atsd_store_user_cgroups", false, "include statistics for \"user\" cgroups (for example: docker-host/user.*)")
	propertyInterval       = flag.Duration("storage_driver_atsd_property_interval", 1*time.Minute, "container property (host, id, namespace) update interval. Should be >= housekeeping_interval")
	samplingInterval       = flag.Duration("storage_driver_atsd_sampling_interval", *manager.HousekeepingInterval, "series sampling interval. Should be >= housekeeping_interval")

	deduplication = make(deduplicationParamsList)
)

func init() {
	storage.RegisterStorageDriver("atsd", new)

	flag.Var(&deduplication, "storage_driver_atsd_deduplication",
		"Specify optional deduplication settings for a metric group using 'group:interval:threshold' syntax. "+
			"Group - Metric group to which the setting applies. Supported metric groups in cAdvisor: cpu, memory, io, network, task, filesystem. "+
			"Interval - Maximum delay between the current and previously sent samples. If exceeded, the current sample is sent to ATSD regardless of the specified threshold. "+
			"Threshold - Absolute or percentage difference between the current and previously sent sample values. If the absolute difference is within the threshold and elapsed time is within Interval, the value is discarded.")
	if *dockerHost == dockerHostDefault {
		content, err := ioutil.ReadFile("/rootfs/etc/hostname")
		if err != nil {
			*dockerHost = ""
		} else {
			*dockerHost = strings.TrimSpace(string(content))
		}
	}
}

func new() (storage.StorageDriver, error) {
	// these parameters are bounded below by a housekeeping interval
	if *propertyInterval < *manager.HousekeepingInterval {
		*propertyInterval = *manager.HousekeepingInterval
	}
	if *samplingInterval < *manager.HousekeepingInterval {
		*samplingInterval = *manager.HousekeepingInterval
	}

	return newStorage()
}

func newStorage() (storage.StorageDriver, error) {
	cadvisorConfig := cadvisorParams{
		DockerHost:             *dockerHost,
		PropertyInterval:       *propertyInterval,
		SamplingInterval:       *samplingInterval,
		IncludeAllMajorNumbers: *includeAllMajorNumbers,
		UserCgroupsEnabled:     *userCgroupsEnabled,
	}

	innerStorageConfig := atsdStorageDriver.GetDefaultConfig()
	innerStorageConfig.MemstoreLimit = *memstoreLimit
	innerStorageConfig.SenderGoroutineLimit = *senderGoroutineLimit
	innerStorageConfig.GroupParams = deduplication
	innerStorageConfig.InsecureSkipVerify = *skipVerify
	innerStorageConfig.MetricPrefix = metricPrefix
	innerStorageConfig.UpdateInterval = *storage.ArgDbBufferDuration
	innerStorageConfig.Url = &url.URL{
		Scheme: *protocol,
		User:   url.UserPassword(*storage.ArgDbUsername, *storage.ArgDbPassword),
		Host:   *storage.ArgDbHost,
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	innerStorageConfig.SelfMetricEntity = cadvisorConfig.DockerHost + "/" + hostname

	storageFactory := atsdStorageDriver.NewFactoryFromConfig(innerStorageConfig)
	innerStorage, err := storageFactory.Create()
	if err != nil {
		return nil, err
	}
	storageDriver := &Storage{
		cadvisorParams:             cadvisorConfig,
		innerStorage:               innerStorage,
		lastTimeSentPropertyMap:    make(map[string]time.Time),
		lastTimePropertyMapMutex:   &sync.Mutex{},
		lastTimeSentSeriesMap:      make(map[string]time.Time),
		lastTimeSentSeriesMapMutex: &sync.Mutex{},
	}

	time.AfterFunc(startDelay, func() {
		storageDriver.innerStorage.StartPeriodicSending()
	})

	return storageDriver, nil

}

type Storage struct {
	cadvisorParams

	innerStorage *atsdStorageDriver.Storage

	lastTimeSentPropertyMap    map[string]time.Time
	lastTimePropertyMapMutex   *sync.Mutex
	lastTimeSentSeriesMap      map[string]time.Time
	lastTimeSentSeriesMapMutex *sync.Mutex
}

func (self *Storage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if isEnabledToStore(ref, self.UserCgroupsEnabled) {
		if self.needToSendSeries(ref.Name, stats.Timestamp) {
			cpuSeriesCommands := CpuSeriesCommandsFromStats(self.DockerHost, ref, stats)
			derivedCpuSeries := CalculateDerivedSeriesCpuCommands(self.DockerHost+ref.Name, stats)
			ioSeriesCommands := IOSeriesCommandsFromStats(self.DockerHost, ref, self.IncludeAllMajorNumbers, stats)
			memorySeriesCommands := MemorySeriesCommandsFromStats(self.DockerHost, ref, stats)
			taskSeriesCommands := TaskSeriesCommandsFromStats(self.DockerHost, ref, stats)
			networkSeriesCommands := NetworkSeriesCommandsFromStats(self.DockerHost, ref, stats)
			fileSystemSeriesCommands := FileSystemSeriesCommandsFromStats(self.DockerHost, ref, stats)

			self.innerStorage.QueuedSendSeriesCommands(cpuGroup, cpuSeriesCommands)
			self.innerStorage.QueuedSendSeriesCommands(cpuGroup, derivedCpuSeries)
			self.innerStorage.QueuedSendSeriesCommands(ioGroup, ioSeriesCommands)
			self.innerStorage.QueuedSendSeriesCommands(memoryGroup, memorySeriesCommands)
			self.innerStorage.QueuedSendSeriesCommands(taskGroup, taskSeriesCommands)
			self.innerStorage.QueuedSendSeriesCommands(networkGroup, networkSeriesCommands)
			self.innerStorage.QueuedSendSeriesCommands(filesytemGroup, fileSystemSeriesCommands)
			self.lastTimeSentSeriesMapMutex.Lock()
			self.lastTimeSentSeriesMap[ref.Name] = stats.Timestamp
			self.lastTimeSentSeriesMapMutex.Unlock()
		}

		if self.needToSendProperties(ref.Name, stats.Timestamp) {
			properties := RefToPropertyCommands(self.DockerHost, ref, stats.Timestamp)
			self.innerStorage.QueuedSendPropertyCommands(properties)
			entities := RefToEntityCommands(self.DockerHost, ref)
			self.innerStorage.QueuedSendEntityTagCommands(entities)

			self.lastTimePropertyMapMutex.Lock()
			self.lastTimeSentPropertyMap[ref.Name] = stats.Timestamp
			self.lastTimePropertyMapMutex.Unlock()
		}
	}
	return nil
}

func (self *Storage) Close() error {
	self.innerStorage.StopPeriodicSending()
	self.innerStorage.ForceSend()
	return nil
}

func (self *Storage) needToSendProperties(containerRefName string, timestamp time.Time) bool {
	self.lastTimePropertyMapMutex.Lock()
	lastTime, ok := self.lastTimeSentPropertyMap[containerRefName]
	self.lastTimePropertyMapMutex.Unlock()
	return !(ok && timestamp.Sub(lastTime) < time.Duration(self.PropertyInterval))
}
func (self *Storage) needToSendSeries(containerRefName string, timestamp time.Time) bool {
	self.lastTimeSentSeriesMapMutex.Lock()
	lastTime, ok := self.lastTimeSentSeriesMap[containerRefName]
	self.lastTimeSentSeriesMapMutex.Unlock()
	return !(ok && timestamp.Sub(lastTime) < time.Duration(self.SamplingInterval)-timestampPeriodError)
}

func isEnabledToStore(ref info.ContainerReference, userCgroupsEnabled bool) bool {
	return (userCgroupsEnabled || !strings.HasPrefix(ref.Name, "/user"))
}
