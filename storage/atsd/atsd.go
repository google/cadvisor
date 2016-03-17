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

package atsd

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/storage"

	"github.com/BurntSushi/toml"
	atsdHttp "github.com/axibase/atsd-api-go/http"
	atsdStorageDriver "github.com/axibase/atsd-storage-driver/storage"
	"os"
	"strconv"
)

const (
	startDelay           = 15 * time.Second       //waiting to store enough data for all entities before send
	timestampPeriodError = 250 * time.Millisecond //time error accumulated during housekeeping

	metricPrefix = "cadvisor"

	// metric groups for a deduplication
	cpuGroup       = "cpu"
	ioGroup        = "io"
	memoryGroup    = "memory"
	taskGroup      = "task"
	networkGroup   = "network"
	filesytemGroup = "filesystem"
)

var (
	urlString       = flag.String("storage_driver_atsd_url", "", "ATSD server http/https endpoint")
	writeHost       = flag.String("storage_driver_atsd_write_host", "", "ATSD server TCP/UDP destination, formatted as host:port")
	protocol        = flag.String("storage_driver_atsd_write_protocol", "", "transfer protocol. Possible settings: http/https, udp, tcp")
	connectionLimit = flag.Uint("storage_driver_atsd_connection_limit", 1, "ATSD storage driver TCP connection count")
	memstoreLimit   = flag.Uint64("storage_driver_atsd_buffer_limit", 1000000, "maximum number of series commands stored in buffer until flush")

	configFilePath = flag.String("storage_driver_atsd_config_path", "", "path to ATSD storage driver config file")

	dockerHost             = flag.String("storage_driver_atsd_docker_host", "docker-host", "hostname of machine where docker daemon is running")
	includeAllMajorNumbers = flag.Bool("storage_driver_atsd_store_major_numbers", false, "include statistics for devices with all available major numbers")
	userCgroupsEnabled     = flag.Bool("storage_driver_atsd_store_user_cgroups", false, "include statistics for \"user\" cgroups (for example: docker-host/user.*)")
	propertyInterval       = flag.Duration("storage_driver_atsd_property_interval", 1*time.Minute, "container property update interval. Should be >= housekeeping_interval")
	samplingInterval       = flag.Duration("storage_driver_atsd_sampling_interval", *manager.HousekeepingInterval, "series sampling interval. Should be >= housekeeping_interval")
	deduplication          = deduplicationParamsList(map[string]atsdStorageDriver.DeduplicationParams{})
)

func init() {
	storage.RegisterStorageDriver("atsd", new)

	flag.Var(&deduplication, "storage_driver_atsd_deduplication",
		"Specify optional deduplication settings for a metric group using 'group:interval:threshold' syntax. "+
			"Group - Metric group to which the setting applies. Supported metric groups in cAdvisor: cpu, memory, io, network, task, filesystem. "+
			"Interval - Maximum delay between the current and previously sent samples. If exceeded, the current sample is sent to ATSD regardless of the specified threshold. "+
			"Threshold - Absolute or percentage difference between the current and previously sent sample values. If the absolute difference is within the threshold and elapsed time is within Interval, the value is discarded.")

	//these parameters are bounded below by a housekeeping interval
	if *propertyInterval < *manager.HousekeepingInterval {
		*propertyInterval = *manager.HousekeepingInterval
	}
	if *samplingInterval < *manager.HousekeepingInterval {
		*samplingInterval = *manager.HousekeepingInterval
	}
}

func new() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(
		*storage.ArgDbUsername,
		*storage.ArgDbPassword,
		hostname,
		*storage.ArgDbBufferDuration,
	), nil
}

type deduplicationParamsList map[string]atsdStorageDriver.DeduplicationParams

func (self deduplicationParamsList) String() string {
	m := map[string]atsdStorageDriver.DeduplicationParams(self)
	return fmt.Sprint(m)
}

func (self deduplicationParamsList) Set(value string) error {
	groupValues := strings.Split(value, ":")
	groupName := groupValues[0]

	interval, err := time.ParseDuration(groupValues[1])
	if err != nil {
		panic(err)
	}
	var threshold interface{}
	if strings.HasSuffix(groupValues[2], "%") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(groupValues[2], "%"), 64)
		if err != nil {
			panic(err)
		}
		threshold = atsdStorageDriver.Percent(val)
	} else {
		val, err := strconv.ParseFloat(groupValues[2], 64)
		if err != nil {
			panic(err)
		}
		threshold = atsdStorageDriver.Absolute(val)
	}
	self[groupName] = atsdStorageDriver.DeduplicationParams{Interval: interval, Threshold: threshold}
	return nil
}

// need to parse duration in toml configuration file
type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func newStorage(username, password, hostname string, argDbBufferDuration time.Duration) storage.StorageDriver {

	innerStorageConfig := atsdStorageDriver.GetDefaultConfig()

	selfConfig := struct {
		Cadvisor struct {
			IncludeAllMajorNumbers bool     `toml:"store_major_numbers"`
			UserCgroupsEnabled     bool     `toml:"store_user_cgroups"`
			PropertyInterval       duration `toml:"property_interval"`
			SamplingInterval       duration `toml:"sampling_interval"`
			DockerHost             string   `toml:"docker_host"`
		}
	}{}

	if *configFilePath != "" {
		_, err := toml.DecodeFile(*configFilePath, &innerStorageConfig)
		if err != nil {
			panic(err)
		}
		_, err = toml.DecodeFile(*configFilePath, &selfConfig)
		if err != nil {
			panic(err)
		}
	} else {
		url, err := url.Parse(*urlString)
		if err != nil {
			panic(err)
		}
		innerStorageConfig.Protocol = *protocol
		innerStorageConfig.DataReceiverHostport = *writeHost
		innerStorageConfig.Url = url
		innerStorageConfig.MemstoreLimit = *memstoreLimit
		innerStorageConfig.ConnectionLimit = *connectionLimit
		innerStorageConfig.GroupParams = deduplication
		selfConfig.Cadvisor.DockerHost = *dockerHost
		selfConfig.Cadvisor.PropertyInterval = duration{*propertyInterval}
		selfConfig.Cadvisor.SamplingInterval = duration{*samplingInterval}
		selfConfig.Cadvisor.IncludeAllMajorNumbers = *includeAllMajorNumbers
		selfConfig.Cadvisor.UserCgroupsEnabled = *userCgroupsEnabled
	}

	selfMetricsEntity := selfConfig.Cadvisor.DockerHost + "/" + hostname
	innerStorageConfig.Username = username
	innerStorageConfig.Password = password
	innerStorageConfig.SelfMetricEntity = selfMetricsEntity
	innerStorageConfig.MetricPrefix = metricPrefix
	innerStorageConfig.UpdateInterval = argDbBufferDuration

	storageFactory := atsdStorageDriver.NewFactoryFromConfig(innerStorageConfig)
	storage := &Storage{
		machineName:             selfConfig.Cadvisor.DockerHost,
		propertyInterval:        selfConfig.Cadvisor.PropertyInterval.Duration,
		samplingInterval:        selfConfig.Cadvisor.SamplingInterval.Duration,
		innerStorage:            storageFactory.Create(),
		lastTimeSentPropertyMap: make(map[string]uint64),
		lastTimeSentSeriesMap:   make(map[string]uint64),
		firstTime:               make(map[string]bool),
		includeAllMajorNumbers:  selfConfig.Cadvisor.IncludeAllMajorNumbers,
		userCgroupsEnabled:      selfConfig.Cadvisor.UserCgroupsEnabled,
	}

	storage.RegisterMetrics()
	time.AfterFunc(startDelay, func() {
		storage.innerStorage.StartPeriodicSending()
	})

	return storage

}

type Storage struct {
	machineName string

	propertyInterval time.Duration
	samplingInterval time.Duration

	includeAllMajorNumbers bool
	userCgroupsEnabled     bool

	innerStorage *atsdStorageDriver.Storage

	mutex sync.Mutex

	lastTimeSentPropertyMap map[string]uint64
	lastTimeSentSeriesMap   map[string]uint64
	firstTime               map[string]bool
}

func (self *Storage) RegisterMetrics() {
	metrics := []*atsdHttp.Metric{
		atsdHttp.NewMetric(containerCpuUsageUser).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerCpuUsageTotal).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerCpuUsageSystem).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerCpuLoadAverage).SetDataType(atsdHttp.INTEGER),
		atsdHttp.NewMetric(containerCpuUsagePerCpu).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerMemoryWorkingSet).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryUsage).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryCache).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryRSS).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryHierarchicalDataPgfault).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryHierarchicalDataPgmajfault).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryContainerDataPgfault).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryContainerDataPgmajfault).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerMemoryFailcnt).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerNetworkRxBytes).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkRxDropped).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkRxErrors).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkRxPackets).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTxBytes).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTxDropped).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTxErrors).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTxPackets).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerNetworkTcpStatEstablished).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatSynSent).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatSynRecv).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatFinWait1).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatFinWait2).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatTimeWait).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatClose).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatCloseWait).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatLastAck).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatListen).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcpStatClosing).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerNetworkTcp6StatEstablished).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatSynSent).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatSynRecv).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatFinWait1).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatFinWait2).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatTimeWait).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatClose).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatCloseWait).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatLastAck).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatListen).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerNetworkTcp6StatClosing).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerTaskStatsNrIoWait).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerTaskStatsNrRunning).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerTaskStatsNrSleeping).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerTaskStatsNrStopped).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerTaskStatsNrUninterruptible).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerDiskIoIoMerged).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerDiskIoIoQueued).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerDiskIoIoServiceBytes).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerDiskIoIoServiced).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerDiskIoIoServiceTime).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerDiskIoIoTime).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerDiskIoIoWaitTime).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerDiskIoSectors).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerFilesystemIoInProgress).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemIoTime).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemLimit).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemReadsCompleted).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemReadsMerged).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemReadTime).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemSectorsRead).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemSectorsWritten).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemUsage).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemBaseUsage).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemAvailable).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemInodesFree).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemWeightedIoTime).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemWritesCompleted).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemWritesMerged).SetDataType(atsdHttp.LONG),
		atsdHttp.NewMetric(containerFilesystemWriteTime).SetDataType(atsdHttp.LONG),

		atsdHttp.NewMetric(containerCpuUsageSystemPct).SetDataType(atsdHttp.FLOAT),
		atsdHttp.NewMetric(containerCpuUsageTotalPct).SetDataType(atsdHttp.FLOAT),
		atsdHttp.NewMetric(containerCpuUsageUserPct).SetDataType(atsdHttp.FLOAT),

		atsdHttp.NewMetric(containerCpuHostUsageSystemPct).SetDataType(atsdHttp.FLOAT),
		atsdHttp.NewMetric(containerCpuHostUsageTotalPct).SetDataType(atsdHttp.FLOAT),
		atsdHttp.NewMetric(containerCpuHostUsageUserPct).SetDataType(atsdHttp.FLOAT),
		atsdHttp.NewMetric(containerCpuUsagePerCpuPct).SetDataType(atsdHttp.FLOAT),
	}

	for _, metric := range metrics {
		self.innerStorage.RegisterMetric(metric)
	}
}

func (self *Storage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	timestamp := uint64(stats.Timestamp.UnixNano())

	if isEnabledToStore(self.machineName, ref, self.userCgroupsEnabled, stats) {
		if self.needToSendSeries(ref.Name, timestamp) {

			cpuSeriesCommands := CpuSeriesCommandsFromStats(self.machineName, ref, stats)
			ioSeriesCommands := IOSeriesCommandsFromStats(self.machineName, ref, self.includeAllMajorNumbers, stats)
			memorySeriesCommands := MemorySeriesCommandsFromStats(self.machineName, ref, stats)
			taskSeriesCommands := TaskSeriesCommandsFromStats(self.machineName, ref, stats)
			networkSeriesCommands := NetworkSeriesCommandsFromStats(self.machineName, ref, stats)
			fileSystemSeriesCommands := FileSystemSeriesCommandsFromStats(self.machineName, ref, stats)

			derivedCpuSeries := CalculateDerivedSeriesCpuCommands(self.machineName+ref.Name, stats)

			self.innerStorage.SendSeriesCommands(cpuGroup, cpuSeriesCommands)
			self.innerStorage.SendSeriesCommands(cpuGroup, derivedCpuSeries)
			self.innerStorage.SendSeriesCommands(ioGroup, ioSeriesCommands)
			self.innerStorage.SendSeriesCommands(memoryGroup, memorySeriesCommands)
			self.innerStorage.SendSeriesCommands(taskGroup, taskSeriesCommands)
			self.innerStorage.SendSeriesCommands(networkGroup, networkSeriesCommands)
			self.innerStorage.SendSeriesCommands(filesytemGroup, fileSystemSeriesCommands)
			self.updateSeriesSentTime(ref.Name, timestamp)
		}

		if self.needToSendProperties(ref.Name, timestamp) {
			properties := RefToPropertyCommands(self.machineName, ref, timestamp)
			self.innerStorage.SendPropertyCommands(properties)
			self.updatePropertySentTime(ref.Name, timestamp)
		}

		if self.needToSendEntityTags(ref.Name) {
			entities := RefToEntityCommands(self.machineName, ref)
			self.innerStorage.SendEntityTagCommands(entities)
			self.updateEntitySentStatus(ref.Name)
		}
	}
	return nil
}

func (self *Storage) Close() error {
	return nil
}

func (self *Storage) needToSendEntityTags(containerRefName string) bool {
	isFirst, ok := self.firstTime[containerRefName]
	return !ok || isFirst
}

func (self *Storage) updateEntitySentStatus(containerRefName string) {
	self.firstTime[containerRefName] = false
}

func (self *Storage) needToSendProperties(containerRefName string, timestamp uint64) bool {
	lastTime, ok := self.lastTimeSentPropertyMap[containerRefName]
	return !(ok && timestamp-lastTime < uint64(self.propertyInterval))
}
func (self *Storage) needToSendSeries(containerRefName string, timestamp uint64) bool {
	lastTime, ok := self.lastTimeSentSeriesMap[containerRefName]
	return !(ok && timestamp-lastTime < uint64(self.samplingInterval-timestampPeriodError))
}

func (self *Storage) updatePropertySentTime(containerRefName string, timestamp uint64) {
	self.lastTimeSentPropertyMap[containerRefName] = timestamp
}
func (self *Storage) updateSeriesSentTime(containerRefName string, timestamp uint64) {
	self.lastTimeSentSeriesMap[containerRefName] = timestamp
}

func isEnabledToStore(machineName string, ref info.ContainerReference, userCgroupsEnabled bool, stats *info.ContainerStats) bool {
	return (userCgroupsEnabled || !strings.HasPrefix(ref.Name, "/user"))
}
