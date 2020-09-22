// Copyright 2015 Google Inc. All Rights Reserved.
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

package statsd

import (
	"strconv"

	client "github.com/google/cadvisor/cmd/internal/storage/statsd/client"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage"
)

func init() {
	storage.RegisterStorageDriver("statsd", new)
}

type statsdStorage struct {
	client    *client.Client
	Namespace string
}

const (
	// Cumulative CPU usage
	// To be deprecated in 0.39
	// https://github.com/google/cadvisor/issues/2637
	colCpuCumulativeUsage string = "cpu_cumulative_usage"
	// Cumulative CPU usage
	serCpuUsageTotal  string = "cpu_usage_total"
	serCpuUsageSystem string = "cpu_usage_system"
	serCpuUsageUser   string = "cpu_usage_user"
	serCpuUsagePerCpu string = "cpu_usage_per_cpu"
	// Smoothed average of number of runnable threads x 1000.
	serLoadAverage string = "cpu_load_average"
	// Memory Usage
	serMemoryUsage string = "memory_usage"
	// Maximum memory usage recorded
	serMemoryMaxUsage string = "memory_max_usage"
	// Number of bytes of page cache memory
	serMemoryCache string = "memory_cache"
	// Size of RSS
	serMemoryRss string = "memory_rss"
	// Container swap usage
	serMemorySwap string = "memory_swap"
	// Size of memory mapped files in bytes
	serMemoryMappedFile string = "memory_mapped_file"
	// Working set size
	serMemoryWorkingSet string = "memory_working_set"
	// Number of memory usage hits limits
	serMemoryFailcnt string = "memory_failcnt"
	// Cumulative count of memory allocation failures
	serMemoryFailure string = "memory_failure"
	// Cumulative count of bytes received.
	serRxBytes string = "rx_bytes"
	// Cumulative count of receive errors encountered.
	serRxErrors string = "rx_errors"
	// Cumulative count of bytes transmitted.
	serTxBytes string = "tx_bytes"
	// Cumulative count of transmit errors encountered.
	serTxErrors string = "tx_errors"
	// Filesystem summary
	serFsSummary string = "fs_summary"
	// Filesystem limit.
	serFsLimit string = "fs_limit"
	// Filesystem usage.
	serFsUsage string = "fs_usage"
	// Hugetlb stat - current res_counter usage for hugetlb
	setHugetlbUsage string = "hugetlb_usage"
	// Hugetlb stat - maximum usage ever recorded
	setHugetlbMaxUsage string = "hugetlb_max_usage"
	// Hugetlb stat - number of times hugetlb usage allocation failure
	setHugetlbFailcnt string = "hugetlb_failcnt"
	// Perf statistics
	serPerfStat string = "perf_stat"
	// Referenced memory
	serReferencedMemory string = "referenced_memory"
	// Resctrl - Total memory bandwidth
	serResctrlMemoryBandwidthTotal string = "resctrl_memory_bandwidth_total"
	// Resctrl - Local memory bandwidth
	serResctrlMemoryBandwidthLocal string = "resctrl_memory_bandwidth_local"
	// Resctrl - Last level cache usage
	serResctrlLLCOccupancy string = "resctrl_llc_occupancy"
)

func new() (storage.StorageDriver, error) {
	return newStorage(*storage.ArgDbName, *storage.ArgDbHost)
}

func (s *statsdStorage) containerStatsToValues(stats *info.ContainerStats) (series map[string]uint64) {
	series = make(map[string]uint64)

	// Total usage in nanoseconds
	series[serCpuUsageTotal] = stats.Cpu.Usage.Total

	// To be deprecated in 0.39
	series[colCpuCumulativeUsage] = series[serCpuUsageTotal]

	// CPU usage: Time spend in system space (in nanoseconds)
	series[serCpuUsageSystem] = stats.Cpu.Usage.System

	// CPU usage: Time spent in user space (in nanoseconds)
	series[serCpuUsageUser] = stats.Cpu.Usage.User

	// CPU usage per CPU
	for i := 0; i < len(stats.Cpu.Usage.PerCpu); i++ {
		series[serCpuUsagePerCpu+"."+strconv.Itoa(i)] = stats.Cpu.Usage.PerCpu[i]
	}

	// Load Average
	series[serLoadAverage] = uint64(stats.Cpu.LoadAverage)

	// Network stats.
	series[serRxBytes] = stats.Network.RxBytes
	series[serRxErrors] = stats.Network.RxErrors
	series[serTxBytes] = stats.Network.TxBytes
	series[serTxErrors] = stats.Network.TxErrors

	// Referenced Memory
	series[serReferencedMemory] = stats.ReferencedMemory

	return series
}

func (s *statsdStorage) containerFsStatsToValues(series *map[string]uint64, stats *info.ContainerStats) {
	for _, fsStat := range stats.Filesystem {
		// Summary stats.
		(*series)[serFsSummary+"."+serFsLimit] += fsStat.Limit
		(*series)[serFsSummary+"."+serFsUsage] += fsStat.Usage

		// Per device stats.
		(*series)[fsStat.Device+"."+serFsLimit] = fsStat.Limit
		(*series)[fsStat.Device+"."+serFsUsage] = fsStat.Usage
	}
}

func (s *statsdStorage) memoryStatsToValues(series *map[string]uint64, stats *info.ContainerStats) {
	// Memory Usage
	(*series)[serMemoryUsage] = stats.Memory.Usage
	// Maximum memory usage recorded
	(*series)[serMemoryMaxUsage] = stats.Memory.MaxUsage
	//Number of bytes of page cache memory
	(*series)[serMemoryCache] = stats.Memory.Cache
	// Size of RSS
	(*series)[serMemoryRss] = stats.Memory.RSS
	// Container swap usage
	(*series)[serMemorySwap] = stats.Memory.Swap
	// Size of memory mapped files in bytes
	(*series)[serMemoryMappedFile] = stats.Memory.MappedFile
	// Working Set Size
	(*series)[serMemoryWorkingSet] = stats.Memory.WorkingSet
	// Number of memory usage hits limits
	(*series)[serMemoryFailcnt] = stats.Memory.Failcnt

	// Cumulative count of memory allocation failures
	(*series)[serMemoryFailure+".container.pgfault"] = stats.Memory.ContainerData.Pgfault
	(*series)[serMemoryFailure+".container.pgmajfault"] = stats.Memory.ContainerData.Pgmajfault
	(*series)[serMemoryFailure+".hierarchical.pgfault"] = stats.Memory.HierarchicalData.Pgfault
	(*series)[serMemoryFailure+".hierarchical.pgmajfault"] = stats.Memory.HierarchicalData.Pgmajfault
}

func (s *statsdStorage) hugetlbStatsToValues(series *map[string]uint64, stats *info.ContainerStats) {
	for pageSize, hugetlbStat := range stats.Hugetlb {
		(*series)[setHugetlbUsage+"."+pageSize] = hugetlbStat.Usage
		(*series)[setHugetlbMaxUsage+"."+pageSize] = hugetlbStat.MaxUsage
		(*series)[setHugetlbFailcnt+"."+pageSize] = hugetlbStat.Failcnt
	}
}

func (s *statsdStorage) perfStatsToValues(series *map[string]uint64, stats *info.ContainerStats) {
	for _, perfStat := range stats.PerfStats {
		(*series)[serPerfStat+"."+perfStat.Name+"."+strconv.Itoa(perfStat.Cpu)] = perfStat.Value
	}
}

func (s *statsdStorage) resctrlStatsToValues(series *map[string]uint64, stats *info.ContainerStats) {
	for nodeID, rdtMemoryBandwidth := range stats.Resctrl.MemoryBandwidth {
		(*series)[serResctrlMemoryBandwidthTotal+"."+strconv.Itoa(nodeID)] = rdtMemoryBandwidth.TotalBytes
		(*series)[serResctrlMemoryBandwidthLocal+"."+strconv.Itoa(nodeID)] = rdtMemoryBandwidth.LocalBytes
	}
	for nodeID, rdtCache := range stats.Resctrl.Cache {
		(*series)[serResctrlLLCOccupancy+"."+strconv.Itoa(nodeID)] = rdtCache.LLCOccupancy
	}

}

// Push the data into redis
func (s *statsdStorage) AddStats(cInfo *info.ContainerInfo, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}

	series := s.containerStatsToValues(stats)
	s.containerFsStatsToValues(&series, stats)
	s.memoryStatsToValues(&series, stats)
	s.hugetlbStatsToValues(&series, stats)
	s.perfStatsToValues(&series, stats)
	s.resctrlStatsToValues(&series, stats)

	var containerName string
	if len(cInfo.ContainerReference.Aliases) > 0 {
		containerName = cInfo.ContainerReference.Aliases[0]
	} else {
		containerName = cInfo.ContainerReference.Name
	}

	for key, value := range series {
		err := s.client.Send(s.Namespace, containerName, key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *statsdStorage) Close() error {
	s.client.Close()
	s.client = nil
	return nil
}

func newStorage(namespace, hostPort string) (*statsdStorage, error) {
	statsdClient, err := client.New(hostPort)
	if err != nil {
		return nil, err
	}
	statsdStorage := &statsdStorage{
		client:    statsdClient,
		Namespace: namespace,
	}
	return statsdStorage, nil
}
