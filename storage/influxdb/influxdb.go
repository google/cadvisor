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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
	"github.com/influxdb/influxdb-go"
	"github.com/kr/pretty"
)

type influxdbStorage struct {
	client      *influxdb.Client
	prevStats   *info.ContainerStats
	machineName string
	tableName   string
	windowLen   time.Duration
}

func (self *influxdbStorage) containerStatsToValues(
	ref info.ContainerReference,
	stats *info.ContainerStats,
) (columns []string, values []interface{}) {

	// Machine name
	columns = append(columns, "machine")
	values = append(values, self.machineName)

	// Container path
	columns = append(columns, "container_path")
	values = append(values, ref.Name)

	// Cumulative Cpu Usage
	columns = append(columns, "cpu_cumulative_usage")
	values = append(values, stats.Cpu.Usage.Total)

	// Cumulative Cpu Usage in kernel mode
	columns = append(columns, "cpu_cumulative_usage_kernel")
	values = append(values, stats.Cpu.Usage.System)

	// Cumulative Cpu Usage in user mode
	columns = append(columns, "cpu_cumulative_usage_user")
	values = append(values, stats.Cpu.Usage.User)

	// Memory Usage
	columns = append(columns, "memory_usage")
	values = append(values, stats.Memory.Usage)

	// Working set size
	columns = append(columns, "memory_working_set")
	values = append(values, stats.Memory.WorkingSet)

	// container page fault
	columns = append(columns, "memory_container_pgfault")
	values = append(values, stats.Memory.ContainerData.Pgfault)

	// container major page fault
	columns = append(columns, "memory_container_pgmajfault")
	values = append(values, stats.Memory.ContainerData.Pgmajfault)

	// hierarchical page fault
	columns = append(columns, "memory_hierarchical_pgfault")
	values = append(values, stats.Memory.HierarchicalData.Pgfault)

	// hierarchical major page fault
	columns = append(columns, "memory_hierarchical_pgmajfault")
	values = append(values, stats.Memory.HierarchicalData.Pgmajfault)

	// per cpu cumulative usage
	for i, u := range stats.Cpu.Usage.PerCpu {
		columns = append(columns, fmt.Sprintf("per_core_cumulative_usage_core_%v", i))
		values = append(values, u)
	}

	sample, err := info.NewSample(self.prevStats, stats)
	if err != nil || sample == nil {
		return columns, values
	}

	// Optional: sample duration. Unit: Nanosecond.
	columns = append(columns, "sample_duration")
	values = append(values, sample.Duration.Nanoseconds())

	// Optional: Instant cpu usage
	columns = append(columns, "cpu_instant_usage")
	values = append(values, sample.Cpu.Usage)

	// Optional: Instant per core usage
	for i, u := range sample.Cpu.PerCpuUsage {
		columns = append(columns, fmt.Sprintf("per_core_instant_usage_core_%v", i))
		values = append(values, u)
	}

	return columns, values
}

func (self *influxdbStorage) valuesToContainerStats(columns []string, values []interface{}) *info.ContainerStats {
	stats := &info.ContainerStats{
		Cpu:    &info.CpuStats{},
		Memory: &info.MemoryStats{},
	}
	perCoreUsage := make(map[int]uint64, 32)
	for i, col := range columns {
		v := values[i]
		switch col {
		case "machine":
			if v.(string) != self.machineName {
				return nil
			}
		// Cumulative Cpu Usage
		case "cpu_cumulative_usage":
			stats.Cpu.Usage.Total = v.(uint64)
		// Cumulative Cpu Usage in kernel mode
		case "cpu_cumulative_usage_kernel":
			stats.Cpu.Usage.System = v.(uint64)
		// Cumulative Cpu Usage in user mode
		case "cpu_cumulative_usage_user":
			stats.Cpu.Usage.User = v.(uint64)
		// Memory Usage
		case "memory_usage":
			stats.Memory.Usage = v.(uint64)
		// Working set size
		case "memory_working_set":
			stats.Memory.WorkingSet = v.(uint64)
		// container page fault
		case "memory_container_pgfault":
			stats.Memory.ContainerData.Pgfault = v.(uint64)
		// container major page fault
		case "memory_container_pgmajfault":
			stats.Memory.ContainerData.Pgmajfault = v.(uint64)
		// hierarchical page fault
		case "memory_hierarchical_pgfault":
			stats.Memory.HierarchicalData.Pgfault = v.(uint64)
		// hierarchical major page fault
		case "memory_hierarchical_pgmajfault":
			stats.Memory.HierarchicalData.Pgmajfault = v.(uint64)
		default:
			if !strings.HasPrefix(col, "per_core_cumulative_usage_core_") {
				continue
			}
			idxStr := col[len("per_core_cumulative_usage_core_"):]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				continue
			}
			perCoreUsage[idx] = v.(uint64)
		}
	}
	stats.Cpu.Usage.PerCpu = make([]uint64, len(perCoreUsage))
	for idx, usage := range perCoreUsage {
		stats.Cpu.Usage.PerCpu[idx] = usage
	}
	return stats
}

func (self *influxdbStorage) valuesToContainerSample(columns []string, values []interface{}) *info.ContainerStatsSample {
	sample := &info.ContainerStatsSample{}
	perCoreUsage := make(map[int]uint64, 32)
	for i, col := range columns {
		v := values[i]
		switch col {
		case "machine":
			if v.(string) != self.machineName {
				return nil
			}
		// Memory Usage
		case "memory_usage":
			sample.Memory.Usage = v.(uint64)
		// sample duration. Unit: Nanosecond.
		case "sample_duration":
			sample.Duration = time.Duration(v.(int64))
			// Instant cpu usage
		case "cpu_instant_usage":
			sample.Cpu.Usage = v.(uint64)

		default:
			if !strings.HasPrefix(col, "per_core_instant_usage_core_") {
				continue
			}
			idxStr := col[len("per_core_instant_usage_core_"):]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				continue
			}
			perCoreUsage[idx] = v.(uint64)
		}
	}
	sample.Cpu.PerCpuUsage = make([]uint64, len(perCoreUsage))
	for idx, usage := range perCoreUsage {
		sample.Cpu.PerCpuUsage[idx] = usage
	}
	if sample.Duration.Nanoseconds() == 0 {
		return nil
	}
	return sample
}

func (self *influxdbStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	series := &influxdb.Series{
		Name: self.tableName,
		// There's only one point for each stats
		Points: make([][]interface{}, 1),
	}
	series.Columns, series.Points[0] = self.containerStatsToValues(ref, stats)

	self.prevStats = stats.Copy(self.prevStats)
	pretty.Printf("% #v", series)
	err := self.client.WriteSeries([]*influxdb.Series{series})
	if err != nil {
		return err
	}
	return nil
}

func (self *influxdbStorage) RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error) {
	// TODO(dengnan): select only columns that we need
	// TODO(dengnan): escape containerName
	query := fmt.Sprintf("select * from %v limit %v where container_path=\"%v\"", self.tableName, numStats, containerName)
	series, err := self.client.Query(query)
	if err != nil {
		return nil, err
	}
	statsList := make([]*info.ContainerStats, 0, len(series))
	for _, s := range series {
		for _, values := range s.Points {
			stats := self.valuesToContainerStats(s.Columns, values)
			statsList = append(statsList, stats)
		}
	}
	return statsList, nil
}

func (self *influxdbStorage) Samples(containerName string, numSamples int) ([]*info.ContainerStatsSample, error) {
	query := fmt.Sprintf("select * from %v limit %v where container_path=\"%v\"", self.tableName, numSamples, containerName)
	series, err := self.client.Query(query)
	if err != nil {
		return nil, err
	}
	sampleList := make([]*info.ContainerStatsSample, 0, len(series))
	for _, s := range series {
		for _, values := range s.Points {
			sample := self.valuesToContainerSample(s.Columns, values)
			sampleList = append(sampleList, sample)
		}
	}
	return sampleList, nil
}

func (self *influxdbStorage) Close() error {
	self.client = nil
	return nil
}

func (self *influxdbStorage) Percentiles(
	containerName string,
	cpuUsagePercentiles []int,
	memUsagePercentiles []int,
) (*info.ContainerStatsPercentiles, error) {
	// TODO(dengnan): Implement it
	return nil, nil
}

// machineName: A unique identifier to identify the host that current cAdvisor
// instance is running on.
// hostname: The host which runs influxdb.
// percentilesDuration: Time window which will be considered when calls Percentiles()
func New(machineName,
	tablename,
	database,
	username,
	password,
	hostname string,
	percentilesDuration time.Duration,
) (storage.StorageDriver, error) {
	config := &influxdb.ClientConfig{
		Host:     hostname,
		Username: username,
		Password: password,
		Database: database,
		// IsSecure: true,
	}
	client, err := influxdb.NewClient(config)
	if err != nil {
		return nil, err
	}
	if percentilesDuration.Seconds() < 1.0 {
		percentilesDuration = 5 * time.Minute
	}

	ret := &influxdbStorage{
		client:      client,
		windowLen:   percentilesDuration,
		machineName: machineName,
	}
	return ret, nil
}
