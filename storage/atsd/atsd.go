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
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/storage"

	"github.com/BurntSushi/toml"
	atsdHttp "github.com/axibase/atsd-api-go/http"
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
)

var (
	urlString       = flag.String("storage_driver_atsd_url", "", "ATSD server http/https endpoint")
	writeHost       = flag.String("storage_driver_atsd_write_host", "", "ATSD server TCP/UDP destination, formatted as host:port")
	protocol        = flag.String("storage_driver_atsd_write_protocol", "", "transfer protocol. Possible settings: http/https, udp, tcp")
	connectionLimit = flag.Uint("storage_driver_atsd_connection_limit", 1, "ATSD storage driver TCP connection count")
	memstoreLimit   = flag.Uint("storage_driver_atsd_buffer_limit", 1000000, "maximum number of series commands stored in buffer until flush")

	configFilePath = flag.String("storage_driver_atsd_config_path", "", "path to ATSD storage driver config file")

	dockerHost             = flag.String("storage_driver_atsd_docker_host", "docker-host", "hostname of machine where docker daemon is running")
	includeAllMajorNumbers = flag.Bool("storage_driver_atsd_store_major_numbers", false, "include statistics for devices with all available major numbers")
	userCgroupsEnabled     = flag.Bool("storage_driver_atsd_store_user_cgroups", false, "include statistics for \"user\" cgroups (for example: docker-host/user.*)")
	propertyInterval       = flag.Duration("storage_driver_atsd_property_interval", 1*time.Minute, "container property update interval. Should be >= housekeeping_interval")
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
}

func new() (storage.StorageDriver, error) {
	// these parameters are bounded below by a housekeeping interval
	if *propertyInterval < *manager.HousekeepingInterval {
		*propertyInterval = *manager.HousekeepingInterval
	}
	if *samplingInterval < *manager.HousekeepingInterval {
		*samplingInterval = *manager.HousekeepingInterval
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(
		*storage.ArgDbUsername,
		*storage.ArgDbPassword,
		hostname,
		*storage.ArgDbBufferDuration,
	)
}

type deduplicationParamsList map[string]atsdStorageDriver.DeduplicationParams

func (self deduplicationParamsList) String() string {
	m := map[string]atsdStorageDriver.DeduplicationParams(self)
	return fmt.Sprint(m)
}

func (self deduplicationParamsList) Set(value string) error {
	groupValues := strings.Split(value, ":")
	if len(groupValues) != 3 {
		return errors.New("Unable to parse a deduplication param value. Expected format: \"group:interval:threshold\"")
	}
	groupName := groupValues[0]

	interval, err := time.ParseDuration(groupValues[1])
	if err != nil {
		return err
	}
	var threshold interface{}
	if strings.HasSuffix(groupValues[2], "%") {
		val, err := strconv.ParseFloat(strings.TrimSuffix(groupValues[2], "%"), 64)
		if err != nil {
			return err
		}
		threshold = atsdStorageDriver.Percent(val)
	} else {
		val, err := strconv.ParseFloat(groupValues[2], 64)
		if err != nil {
			return err
		}
		threshold = atsdStorageDriver.Absolute(val)
	}
	self[groupName] = atsdStorageDriver.DeduplicationParams{Interval: interval, Threshold: threshold}
	return nil
}

// need to parse duration in toml configuration file
// and implement UnmarshalText(text []byte) error method
type duration time.Duration

func (d *duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	*d = duration(dur)
	return err
}

type cadvisorParams struct {
	IncludeAllMajorNumbers bool     `toml:"store_major_numbers"`
	UserCgroupsEnabled     bool     `toml:"store_user_cgroups"`
	PropertyInterval       duration `toml:"property_interval"`
	SamplingInterval       duration `toml:"sampling_interval"`
	DockerHost             string   `toml:"docker_host"`
}

func newStorage(username, password, hostname string, bufferDuration time.Duration) (storage.StorageDriver, error) {
	innerStorageConfig := atsdStorageDriver.GetDefaultConfig()

	selfConfig := struct {
		Cadvisor cadvisorParams
	}{}

	if *configFilePath != "" {
		_, err := toml.DecodeFile(*configFilePath, &innerStorageConfig)
		if err != nil {
			return nil, err
		}
		_, err = toml.DecodeFile(*configFilePath, &selfConfig)
		if err != nil {
			return nil, err
		}
	} else {
		url, err := url.Parse(*urlString)
		if err != nil {
			return nil, err
		}
		innerStorageConfig.Protocol = *protocol
		innerStorageConfig.DataReceiverHostport = *writeHost
		innerStorageConfig.Url = url
		innerStorageConfig.MemstoreLimit = *memstoreLimit
		innerStorageConfig.ConnectionLimit = *connectionLimit
		innerStorageConfig.GroupParams = deduplication
		selfConfig.Cadvisor.DockerHost = *dockerHost
		selfConfig.Cadvisor.PropertyInterval = duration(*propertyInterval)
		selfConfig.Cadvisor.SamplingInterval = duration(*samplingInterval)
		selfConfig.Cadvisor.IncludeAllMajorNumbers = *includeAllMajorNumbers
		selfConfig.Cadvisor.UserCgroupsEnabled = *userCgroupsEnabled
	}

	selfMetricsEntity := selfConfig.Cadvisor.DockerHost + "/" + hostname
	innerStorageConfig.Username = username
	innerStorageConfig.Password = password
	innerStorageConfig.SelfMetricEntity = selfMetricsEntity
	innerStorageConfig.MetricPrefix = metricPrefix
	innerStorageConfig.UpdateInterval = bufferDuration

	storageFactory := atsdStorageDriver.NewFactoryFromConfig(innerStorageConfig)
	storageDriver := &Storage{
		cadvisorParams:          selfConfig.Cadvisor,
		innerStorage:            storageFactory.Create(),
		lastTimeSentPropertyMap: make(map[string]time.Time),
		lastTimeSentSeriesMap:   make(map[string]time.Time),
		hasSentEntityTags:       make(map[string]bool),
	}

	storageDriver.RegisterMetrics()
	time.AfterFunc(startDelay, func() {
		storageDriver.innerStorage.StartPeriodicSending()
	})

	return storageDriver, nil

}

type Storage struct {
	cadvisorParams

	innerStorage *atsdStorageDriver.Storage

	lastTimeSentPropertyMap map[string]time.Time
	lastTimeSentSeriesMap   map[string]time.Time
	hasSentEntityTags       map[string]bool
}

func (self *Storage) RegisterMetrics() {
	for metric, dataType := range map[string]atsdHttp.DataType{
		containerCpuUsageUser:   atsdHttp.LONG,
		containerCpuUsageTotal:  atsdHttp.LONG,
		containerCpuUsageSystem: atsdHttp.LONG,
		containerCpuLoadAverage: atsdHttp.INTEGER,
		containerCpuUsagePerCpu: atsdHttp.LONG,

		containerMemoryWorkingSet:                 atsdHttp.LONG,
		containerMemoryUsage:                      atsdHttp.LONG,
		containerMemoryCache:                      atsdHttp.LONG,
		containerMemoryRSS:                        atsdHttp.LONG,
		containerMemoryHierarchicalDataPgfault:    atsdHttp.LONG,
		containerMemoryHierarchicalDataPgmajfault: atsdHttp.LONG,
		containerMemoryContainerDataPgfault:       atsdHttp.LONG,
		containerMemoryContainerDataPgmajfault:    atsdHttp.LONG,
		containerMemoryFailcnt:                    atsdHttp.LONG,

		containerNetworkRxBytes:   atsdHttp.LONG,
		containerNetworkRxDropped: atsdHttp.LONG,
		containerNetworkRxErrors:  atsdHttp.LONG,
		containerNetworkRxPackets: atsdHttp.LONG,
		containerNetworkTxBytes:   atsdHttp.LONG,
		containerNetworkTxDropped: atsdHttp.LONG,
		containerNetworkTxErrors:  atsdHttp.LONG,
		containerNetworkTxPackets: atsdHttp.LONG,

		containerNetworkTcpStatEstablished: atsdHttp.LONG,
		containerNetworkTcpStatSynSent:     atsdHttp.LONG,
		containerNetworkTcpStatSynRecv:     atsdHttp.LONG,
		containerNetworkTcpStatFinWait1:    atsdHttp.LONG,
		containerNetworkTcpStatFinWait2:    atsdHttp.LONG,
		containerNetworkTcpStatTimeWait:    atsdHttp.LONG,
		containerNetworkTcpStatClose:       atsdHttp.LONG,
		containerNetworkTcpStatCloseWait:   atsdHttp.LONG,
		containerNetworkTcpStatLastAck:     atsdHttp.LONG,
		containerNetworkTcpStatListen:      atsdHttp.LONG,
		containerNetworkTcpStatClosing:     atsdHttp.LONG,

		containerNetworkTcp6StatEstablished: atsdHttp.LONG,
		containerNetworkTcp6StatSynSent:     atsdHttp.LONG,
		containerNetworkTcp6StatSynRecv:     atsdHttp.LONG,
		containerNetworkTcp6StatFinWait1:    atsdHttp.LONG,
		containerNetworkTcp6StatFinWait2:    atsdHttp.LONG,
		containerNetworkTcp6StatTimeWait:    atsdHttp.LONG,
		containerNetworkTcp6StatClose:       atsdHttp.LONG,
		containerNetworkTcp6StatCloseWait:   atsdHttp.LONG,
		containerNetworkTcp6StatLastAck:     atsdHttp.LONG,
		containerNetworkTcp6StatListen:      atsdHttp.LONG,
		containerNetworkTcp6StatClosing:     atsdHttp.LONG,

		containerTaskStatsNrIoWait:          atsdHttp.LONG,
		containerTaskStatsNrRunning:         atsdHttp.LONG,
		containerTaskStatsNrSleeping:        atsdHttp.LONG,
		containerTaskStatsNrStopped:         atsdHttp.LONG,
		containerTaskStatsNrUninterruptible: atsdHttp.LONG,

		containerDiskIoIoMerged:       atsdHttp.LONG,
		containerDiskIoIoQueued:       atsdHttp.LONG,
		containerDiskIoIoServiceBytes: atsdHttp.LONG,
		containerDiskIoIoServiced:     atsdHttp.LONG,
		containerDiskIoIoServiceTime:  atsdHttp.LONG,
		containerDiskIoIoTime:         atsdHttp.LONG,
		containerDiskIoIoWaitTime:     atsdHttp.LONG,
		containerDiskIoSectors:        atsdHttp.LONG,

		containerFilesystemIoInProgress:    atsdHttp.LONG,
		containerFilesystemIoTime:          atsdHttp.LONG,
		containerFilesystemLimit:           atsdHttp.LONG,
		containerFilesystemReadsCompleted:  atsdHttp.LONG,
		containerFilesystemReadsMerged:     atsdHttp.LONG,
		containerFilesystemReadTime:        atsdHttp.LONG,
		containerFilesystemSectorsRead:     atsdHttp.LONG,
		containerFilesystemSectorsWritten:  atsdHttp.LONG,
		containerFilesystemUsage:           atsdHttp.LONG,
		containerFilesystemBaseUsage:       atsdHttp.LONG,
		containerFilesystemAvailable:       atsdHttp.LONG,
		containerFilesystemInodesFree:      atsdHttp.LONG,
		containerFilesystemWeightedIoTime:  atsdHttp.LONG,
		containerFilesystemWritesCompleted: atsdHttp.LONG,
		containerFilesystemWritesMerged:    atsdHttp.LONG,
		containerFilesystemWriteTime:       atsdHttp.LONG,

		containerCpuUsageSystemPct: atsdHttp.FLOAT,
		containerCpuUsageTotalPct:  atsdHttp.FLOAT,
		containerCpuUsageUserPct:   atsdHttp.FLOAT,

		containerCpuHostUsageSystemPct: atsdHttp.FLOAT,
		containerCpuHostUsageTotalPct:  atsdHttp.FLOAT,
		containerCpuHostUsageUserPct:   atsdHttp.FLOAT,
		containerCpuUsagePerCpuPct:     atsdHttp.FLOAT,
	} {
		self.innerStorage.RegisterMetric(
			atsdHttp.NewMetric(metric).SetDataType(dataType))
	}
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

			self.innerStorage.SendSeriesCommands(cpuGroup, cpuSeriesCommands)
			self.innerStorage.SendSeriesCommands(cpuGroup, derivedCpuSeries)
			self.innerStorage.SendSeriesCommands(ioGroup, ioSeriesCommands)
			self.innerStorage.SendSeriesCommands(memoryGroup, memorySeriesCommands)
			self.innerStorage.SendSeriesCommands(taskGroup, taskSeriesCommands)
			self.innerStorage.SendSeriesCommands(networkGroup, networkSeriesCommands)
			self.innerStorage.SendSeriesCommands(filesytemGroup, fileSystemSeriesCommands)
			self.lastTimeSentSeriesMap[ref.Name] = stats.Timestamp
		}

		if self.needToSendProperties(ref.Name, stats.Timestamp) {
			properties := RefToPropertyCommands(self.DockerHost, ref, stats.Timestamp)
			self.innerStorage.SendPropertyCommands(properties)
			self.lastTimeSentPropertyMap[ref.Name] = stats.Timestamp
		}

		if self.needToSendEntityTags(ref.Name) {
			entities := RefToEntityCommands(self.DockerHost, ref)
			self.innerStorage.SendEntityTagCommands(entities)
			self.hasSentEntityTags[ref.Name] = true
		}
	}
	return nil
}

func (self *Storage) Close() error {
	self.innerStorage.StopPeriodicSending()
	self.innerStorage.ForceSend()
	return nil
}

func (self *Storage) needToSendEntityTags(containerRefName string) bool {
	return !self.hasSentEntityTags[containerRefName]
}

func (self *Storage) needToSendProperties(containerRefName string, timestamp time.Time) bool {
	lastTime, ok := self.lastTimeSentPropertyMap[containerRefName]
	return !(ok && timestamp.Sub(lastTime) < time.Duration(self.PropertyInterval))
}
func (self *Storage) needToSendSeries(containerRefName string, timestamp time.Time) bool {
	lastTime, ok := self.lastTimeSentSeriesMap[containerRefName]
	return !(ok && timestamp.Sub(lastTime) < time.Duration(self.SamplingInterval)-timestampPeriodError)
}

func isEnabledToStore(ref info.ContainerReference, userCgroupsEnabled bool) bool {
	return (userCgroupsEnabled || !strings.HasPrefix(ref.Name, "/user"))
}
