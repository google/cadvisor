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

package influxdb

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/version"

	influxdb "github.com/influxdb/influxdb/client"
)

func init() {
	storage.RegisterStorageDriver("influxdb", new)
}

var argDbRetentionPolicy = flag.String("storage_driver_influxdb_retention_policy", "", "retention policy")

type influxdbStorage struct {
	client          *influxdb.Client
	machineName     string
	database        string
	retentionPolicy string
	bufferDuration  time.Duration
	lastWrite       time.Time
	points          []*influxdb.Point
	lock            sync.Mutex
	readyToFlush    func() bool
}

// Series names
const (
	// Cumulative CPU usage
	serCpuUsageTotal  string = "cpu_usage_total"
	serCpuUsageSystem string = "cpu_usage_system"
	serCpuUsageUser   string = "cpu_usage_user"
	serCpuUsagePerCpu string = "cpu_usage_per_cpu"
	// Smoothed average of number of runnable threads x 1000.
	serLoadAverage string = "load_average"
	// Memory Usage
	serMemoryUsage string = "memory_usage"
	// Maximum memory usage recorded
	serMemoryMaxUsage string = "memory_max_usage"
	// //Number of bytes of page cache memory
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
	// Filesystem limit.
	serFsLimit string = "fs_limit"
	// Filesystem usage.
	serFsUsage string = "fs_usage"
	// Hugetlb stat - current res_counter usage for hugetlb
	setHugetlbUsage = "hugetlb_usage"
	// Hugetlb stat - maximum usage ever recorded
	setHugetlbMaxUsage = "hugetlb_max_usage"
	// Hugetlb stat - number of times hugetlb usage allocation failure
	setHugetlbFailcnt = "hugetlb_failcnt"
	// Perf statistics
	serPerfStat = "perf_stat"
	// Referenced memory
	serReferencedMemory = "referenced_memory"
	// Resctrl - Total memory bandwidth
	serResctrlMemoryBandwidthTotal = "resctrl_memory_bandwidth_total"
	// Resctrl - Local memory bandwidth
	serResctrlMemoryBandwidthLocal = "resctrl_memory_bandwidth_local"
	// Resctrl - Last level cache usage
	serResctrlLLCOccupancy = "resctrl_llc_occupancy"
)

func new() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(
		hostname,
		*storage.ArgDbTable,
		*storage.ArgDbName,
		*argDbRetentionPolicy,
		*storage.ArgDbUsername,
		*storage.ArgDbPassword,
		*storage.ArgDbHost,
		*storage.ArgDbIsSecure,
		*storage.ArgDbBufferDuration,
	)
}

// Field names
const (
	fieldValue  string = "value"
	fieldType   string = "type"
	fieldDevice string = "device"
)

// Tag names
const (
	tagMachineName   string = "machine"
	tagContainerName string = "container_name"
)

func (s *influxdbStorage) containerFilesystemStatsToPoints(
	cInfo *info.ContainerInfo,
	stats *info.ContainerStats) (points []*influxdb.Point) {
	if len(stats.Filesystem) == 0 {
		return points
	}
	for _, fsStat := range stats.Filesystem {
		tagsFsUsage := map[string]string{
			fieldDevice: fsStat.Device,
			fieldType:   "usage",
		}
		fieldsFsUsage := map[string]interface{}{
			fieldValue: int64(fsStat.Usage),
		}
		pointFsUsage := &influxdb.Point{
			Measurement: serFsUsage,
			Tags:        tagsFsUsage,
			Fields:      fieldsFsUsage,
		}

		tagsFsLimit := map[string]string{
			fieldDevice: fsStat.Device,
			fieldType:   "limit",
		}
		fieldsFsLimit := map[string]interface{}{
			fieldValue: int64(fsStat.Limit),
		}
		pointFsLimit := &influxdb.Point{
			Measurement: serFsLimit,
			Tags:        tagsFsLimit,
			Fields:      fieldsFsLimit,
		}

		points = append(points, pointFsUsage, pointFsLimit)
	}

	s.tagPoints(cInfo, stats, points)

	return points
}

// Set tags and timestamp for all points of the batch.
// Points should inherit the tags that are set for BatchPoints, but that does not seem to work.
func (s *influxdbStorage) tagPoints(cInfo *info.ContainerInfo, stats *info.ContainerStats, points []*influxdb.Point) {
	// Use container alias if possible
	var containerName string
	if len(cInfo.ContainerReference.Aliases) > 0 {
		containerName = cInfo.ContainerReference.Aliases[0]
	} else {
		containerName = cInfo.ContainerReference.Name
	}

	commonTags := map[string]string{
		tagMachineName:   s.machineName,
		tagContainerName: containerName,
	}
	for i := 0; i < len(points); i++ {
		// merge with existing tags if any
		addTagsToPoint(points[i], commonTags)
		addTagsToPoint(points[i], cInfo.Spec.Labels)
		points[i].Time = stats.Timestamp
	}
}

func (s *influxdbStorage) containerStatsToPoints(
	cInfo *info.ContainerInfo,
	stats *info.ContainerStats,
) (points []*influxdb.Point) {
	// CPU usage: Total usage in nanoseconds
	points = append(points, makePoint(serCpuUsageTotal, stats.Cpu.Usage.Total))

	// CPU usage: Time spend in system space (in nanoseconds)
	points = append(points, makePoint(serCpuUsageSystem, stats.Cpu.Usage.System))

	// CPU usage: Time spent in user space (in nanoseconds)
	points = append(points, makePoint(serCpuUsageUser, stats.Cpu.Usage.User))

	// CPU usage per CPU
	for i := 0; i < len(stats.Cpu.Usage.PerCpu); i++ {
		point := makePoint(serCpuUsagePerCpu, stats.Cpu.Usage.PerCpu[i])
		tags := map[string]string{"instance": fmt.Sprintf("%v", i)}
		addTagsToPoint(point, tags)

		points = append(points, point)
	}

	// Load Average
	points = append(points, makePoint(serLoadAverage, stats.Cpu.LoadAverage))

	// Network Stats
	points = append(points, makePoint(serRxBytes, stats.Network.RxBytes))
	points = append(points, makePoint(serRxErrors, stats.Network.RxErrors))
	points = append(points, makePoint(serTxBytes, stats.Network.TxBytes))
	points = append(points, makePoint(serTxErrors, stats.Network.TxErrors))

	// Referenced Memory
	points = append(points, makePoint(serReferencedMemory, stats.ReferencedMemory))

	s.tagPoints(cInfo, stats, points)

	return points
}

func (s *influxdbStorage) memoryStatsToPoints(
	cInfo *info.ContainerInfo,
	stats *info.ContainerStats,
) (points []*influxdb.Point) {
	// Memory Usage
	points = append(points, makePoint(serMemoryUsage, stats.Memory.Usage))
	// Maximum memory usage recorded
	points = append(points, makePoint(serMemoryMaxUsage, stats.Memory.MaxUsage))
	//Number of bytes of page cache memory
	points = append(points, makePoint(serMemoryCache, stats.Memory.Cache))
	// Size of RSS
	points = append(points, makePoint(serMemoryRss, stats.Memory.RSS))
	// Container swap usage
	points = append(points, makePoint(serMemorySwap, stats.Memory.Swap))
	// Size of memory mapped files in bytes
	points = append(points, makePoint(serMemoryMappedFile, stats.Memory.MappedFile))
	// Working Set Size
	points = append(points, makePoint(serMemoryWorkingSet, stats.Memory.WorkingSet))
	// Number of memory usage hits limits
	points = append(points, makePoint(serMemoryFailcnt, stats.Memory.Failcnt))

	// Cumulative count of memory allocation failures
	memoryFailuresTags := map[string]string{
		"failure_type": "pgfault",
		"scope":        "container",
	}
	memoryFailurePoint := makePoint(serMemoryFailure, stats.Memory.ContainerData.Pgfault)
	addTagsToPoint(memoryFailurePoint, memoryFailuresTags)
	points = append(points, memoryFailurePoint)

	memoryFailuresTags["failure_type"] = "pgmajfault"
	memoryFailurePoint = makePoint(serMemoryFailure, stats.Memory.ContainerData.Pgmajfault)
	addTagsToPoint(memoryFailurePoint, memoryFailuresTags)
	points = append(points, memoryFailurePoint)

	memoryFailuresTags["failure_type"] = "pgfault"
	memoryFailuresTags["scope"] = "hierarchical"
	memoryFailurePoint = makePoint(serMemoryFailure, stats.Memory.HierarchicalData.Pgfault)
	addTagsToPoint(memoryFailurePoint, memoryFailuresTags)
	points = append(points, memoryFailurePoint)

	memoryFailuresTags["failure_type"] = "pgmajfault"
	memoryFailurePoint = makePoint(serMemoryFailure, stats.Memory.HierarchicalData.Pgmajfault)
	addTagsToPoint(memoryFailurePoint, memoryFailuresTags)
	points = append(points, memoryFailurePoint)

	s.tagPoints(cInfo, stats, points)

	return points
}

func (s *influxdbStorage) hugetlbStatsToPoints(
	cInfo *info.ContainerInfo,
	stats *info.ContainerStats,
) (points []*influxdb.Point) {

	for pageSize, hugetlbStat := range stats.Hugetlb {
		tags := map[string]string{
			"page_size": pageSize,
		}

		// Hugepage usage
		point := makePoint(setHugetlbUsage, hugetlbStat.Usage)
		addTagsToPoint(point, tags)
		points = append(points, point)

		//Maximum hugepage usage recorded
		point = makePoint(setHugetlbMaxUsage, hugetlbStat.MaxUsage)
		addTagsToPoint(point, tags)
		points = append(points, point)

		// Number of hugepage usage hits limits
		point = makePoint(setHugetlbFailcnt, hugetlbStat.Failcnt)
		addTagsToPoint(point, tags)
		points = append(points, point)
	}

	s.tagPoints(cInfo, stats, points)

	return points
}

func (s *influxdbStorage) perfStatsToPoints(
	cInfo *info.ContainerInfo,
	stats *info.ContainerStats,
) (points []*influxdb.Point) {

	for _, perfStat := range stats.PerfStats {
		point := makePoint(serPerfStat, perfStat.Value)
		tags := map[string]string{
			"cpu":           fmt.Sprintf("%v", perfStat.Cpu),
			"name":          perfStat.Name,
			"scaling_ratio": fmt.Sprintf("%v", perfStat.ScalingRatio),
		}
		addTagsToPoint(point, tags)
		points = append(points, point)
	}

	s.tagPoints(cInfo, stats, points)

	return points
}

func (s *influxdbStorage) resctrlStatsToPoints(
	cInfo *info.ContainerInfo,
	stats *info.ContainerStats,
) (points []*influxdb.Point) {

	// Memory bandwidth
	for nodeID, rdtMemoryBandwidth := range stats.Resctrl.MemoryBandwidth {
		tags := map[string]string{
			"node_id": fmt.Sprintf("%v", nodeID),
		}
		point := makePoint(serResctrlMemoryBandwidthTotal, rdtMemoryBandwidth.TotalBytes)
		addTagsToPoint(point, tags)
		points = append(points, point)

		point = makePoint(serResctrlMemoryBandwidthLocal, rdtMemoryBandwidth.LocalBytes)
		addTagsToPoint(point, tags)
		points = append(points, point)
	}

	// Cache
	for nodeID, rdtCache := range stats.Resctrl.Cache {
		tags := map[string]string{
			"node_id": fmt.Sprintf("%v", nodeID),
		}
		point := makePoint(serResctrlLLCOccupancy, rdtCache.LLCOccupancy)
		addTagsToPoint(point, tags)
		points = append(points, point)
	}

	s.tagPoints(cInfo, stats, points)

	return points
}

func (s *influxdbStorage) OverrideReadyToFlush(readyToFlush func() bool) {
	s.readyToFlush = readyToFlush
}

func (s *influxdbStorage) defaultReadyToFlush() bool {
	return time.Since(s.lastWrite) >= s.bufferDuration
}

func (s *influxdbStorage) AddStats(cInfo *info.ContainerInfo, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}
	var pointsToFlush []*influxdb.Point
	func() {
		// AddStats will be invoked simultaneously from multiple threads and only one of them will perform a write.
		s.lock.Lock()
		defer s.lock.Unlock()

		s.points = append(s.points, s.containerStatsToPoints(cInfo, stats)...)
		s.points = append(s.points, s.memoryStatsToPoints(cInfo, stats)...)
		s.points = append(s.points, s.hugetlbStatsToPoints(cInfo, stats)...)
		s.points = append(s.points, s.perfStatsToPoints(cInfo, stats)...)
		s.points = append(s.points, s.resctrlStatsToPoints(cInfo, stats)...)
		s.points = append(s.points, s.containerFilesystemStatsToPoints(cInfo, stats)...)
		if s.readyToFlush() {
			pointsToFlush = s.points
			s.points = make([]*influxdb.Point, 0)
			s.lastWrite = time.Now()
		}
	}()
	if len(pointsToFlush) > 0 {
		points := make([]influxdb.Point, len(pointsToFlush))
		for i, p := range pointsToFlush {
			points[i] = *p
		}

		batchTags := map[string]string{tagMachineName: s.machineName}
		bp := influxdb.BatchPoints{
			Points:          points,
			Database:        s.database,
			RetentionPolicy: s.retentionPolicy,
			Tags:            batchTags,
			Time:            stats.Timestamp,
		}
		response, err := s.client.Write(bp)
		if err != nil || checkResponseForErrors(response) != nil {
			return fmt.Errorf("failed to write stats to influxDb - %s", err)
		}
	}
	return nil
}

func (s *influxdbStorage) Close() error {
	s.client = nil
	return nil
}

// machineName: A unique identifier to identify the host that current cAdvisor
// instance is running on.
// influxdbHost: The host which runs influxdb (host:port)
func newStorage(
	machineName,
	tablename,
	database,
	retentionPolicy,
	username,
	password,
	influxdbHost string,
	isSecure bool,
	bufferDuration time.Duration,
) (*influxdbStorage, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   influxdbHost,
	}
	if isSecure {
		url.Scheme = "https"
	}

	config := &influxdb.Config{
		URL:       *url,
		Username:  username,
		Password:  password,
		UserAgent: fmt.Sprintf("%v/%v", "cAdvisor", version.Info["version"]),
	}
	client, err := influxdb.NewClient(*config)
	if err != nil {
		return nil, err
	}

	ret := &influxdbStorage{
		client:          client,
		machineName:     machineName,
		database:        database,
		retentionPolicy: retentionPolicy,
		bufferDuration:  bufferDuration,
		lastWrite:       time.Now(),
		points:          make([]*influxdb.Point, 0),
	}
	ret.readyToFlush = ret.defaultReadyToFlush
	return ret, nil
}

// Creates a measurement point with a single value field
func makePoint(name string, value interface{}) *influxdb.Point {
	fields := map[string]interface{}{
		fieldValue: toSignedIfUnsigned(value),
	}

	return &influxdb.Point{
		Measurement: name,
		Fields:      fields,
	}
}

// Adds additional tags to the existing tags of a point
func addTagsToPoint(point *influxdb.Point, tags map[string]string) {
	if point.Tags == nil {
		point.Tags = tags
	} else {
		for k, v := range tags {
			point.Tags[k] = v
		}
	}
}

// Checks response for possible errors
func checkResponseForErrors(response *influxdb.Response) error {
	const msg = "failed to write stats to influxDb - %s"

	if response != nil && response.Err != nil {
		return fmt.Errorf(msg, response.Err)
	}
	if response != nil && response.Results != nil {
		for _, result := range response.Results {
			if result.Err != nil {
				return fmt.Errorf(msg, result.Err)
			}
			if result.Series != nil {
				for _, row := range result.Series {
					if row.Err != nil {
						return fmt.Errorf(msg, row.Err)
					}
				}
			}
		}
	}
	return nil
}

// Some stats have type unsigned integer, but the InfluxDB client accepts only signed integers.
func toSignedIfUnsigned(value interface{}) interface{} {
	switch v := value.(type) {
	case uint64:
		return int64(v)
	case uint32:
		return int32(v)
	case uint16:
		return int16(v)
	case uint8:
		return int8(v)
	case uint:
		return int(v)
	}
	return value
}
