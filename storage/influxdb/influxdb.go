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
	"strings"
	"sync"
	"time"

	"github.com/google/cadvisor/info"
	influxdb "github.com/influxdb/influxdb/client"
)

type influxdbStorage struct {
	client         *influxdb.Client
	prevStats      *info.ContainerStats
	machineName    string
	tableName      string
	windowLen      time.Duration
	bufferDuration time.Duration
	lastWrite      time.Time
	series         []*influxdb.Series
	lock           sync.Mutex
	readyToFlush   func() bool
}

const (
	colTimestamp          string = "time"
	colMachineName        string = "machine"
	colContainerName      string = "container_name"
	colCpuCumulativeUsage string = "cpu_cumulative_usage"
	// Memory Usage
	colMemoryUsage string = "memory_usage"
	// Working set size
	colMemoryWorkingSet string = "memory_working_set"
	// Optional: sample duration. Unit: Nanosecond.
	colSampleDuration string = "sample_duration"
	// Optional: Instant cpu usage
	colCpuInstantUsage string = "cpu_instant_usage"
	// Cumulative count of bytes received.
	colRxBytes string = "rx_bytes"
	// Cumulative count of receive errors encountered.
	colRxErrors string = "rx_errors"
	// Cumulative count of bytes transmitted.
	colTxBytes string = "tx_bytes"
	// Cumulative count of transmit errors encountered.
	colTxErrors string = "tx_errors"
)

func (self *influxdbStorage) containerStatsToValues(
	ref info.ContainerReference,
	stats *info.ContainerStats,
) (columns []string, values []interface{}) {

	// Timestamp
	columns = append(columns, colTimestamp)
	values = append(values, stats.Timestamp.Unix())

	// Machine name
	columns = append(columns, colMachineName)
	values = append(values, self.machineName)

	// Container name
	columns = append(columns, colContainerName)
	if len(ref.Aliases) > 0 {
		values = append(values, ref.Aliases[0])
	} else {
		values = append(values, ref.Name)
	}

	// Cumulative Cpu Usage
	columns = append(columns, colCpuCumulativeUsage)
	values = append(values, stats.Cpu.Usage.Total)

	// Memory Usage
	columns = append(columns, colMemoryUsage)
	values = append(values, stats.Memory.Usage)

	// Working set size
	columns = append(columns, colMemoryWorkingSet)
	values = append(values, stats.Memory.WorkingSet)

	// Optional: Network stats.
	if stats.Network != nil {
		columns = append(columns, colRxBytes)
		values = append(values, stats.Network.RxBytes)

		columns = append(columns, colRxErrors)
		values = append(values, stats.Network.RxErrors)

		columns = append(columns, colTxBytes)
		values = append(values, stats.Network.TxBytes)

		columns = append(columns, colTxErrors)
		values = append(values, stats.Network.TxErrors)
	}

	sample, err := info.NewSample(self.prevStats, stats)
	if err != nil || sample == nil {
		return columns, values
	}
	// DO NOT ADD ANY STATS BELOW THAT ARE NOT PART OF SAMPLING

	// Optional: sample duration. Unit: Nanosecond.
	columns = append(columns, colSampleDuration)
	values = append(values, sample.Duration.String())

	// Optional: Instant cpu usage
	columns = append(columns, colCpuInstantUsage)
	values = append(values, sample.Cpu.Usage)

	return columns, values
}

func convertToUint64(v interface{}) (uint64, error) {
	if v == nil {
		return 0, nil
	}
	switch x := v.(type) {
	case uint64:
		return x, nil
	case int:
		if x < 0 {
			return 0, fmt.Errorf("negative value: %v", x)
		}
		return uint64(x), nil
	case int32:
		if x < 0 {
			return 0, fmt.Errorf("negative value: %v", x)
		}
		return uint64(x), nil
	case int64:
		if x < 0 {
			return 0, fmt.Errorf("negative value: %v", x)
		}
		return uint64(x), nil
	case float64:
		if x < 0 {
			return 0, fmt.Errorf("negative value: %v", x)
		}
		return uint64(x), nil
	case uint32:
		return uint64(x), nil
	}
	return 0, fmt.Errorf("Unknown type")
}

func (self *influxdbStorage) valuesToContainerStats(columns []string, values []interface{}) (*info.ContainerStats, error) {
	stats := &info.ContainerStats{
		Cpu:     &info.CpuStats{},
		Memory:  &info.MemoryStats{},
		Network: &info.NetworkStats{},
	}
	var err error
	for i, col := range columns {
		v := values[i]
		switch {
		case col == colTimestamp:
			if str, ok := v.(string); ok {
				stats.Timestamp, err = time.Parse(time.RFC3339Nano, str)
			}
		case col == colMachineName:
			if m, ok := v.(string); ok {
				if m != self.machineName {
					return nil, fmt.Errorf("different machine")
				}
			} else {
				return nil, fmt.Errorf("machine name field is not a string: %v", v)
			}
		// Cumulative Cpu Usage
		case col == colCpuCumulativeUsage:
			stats.Cpu.Usage.Total, err = convertToUint64(v)
		// Memory Usage
		case col == colMemoryUsage:
			stats.Memory.Usage, err = convertToUint64(v)
		// Working set size
		case col == colMemoryWorkingSet:
			stats.Memory.WorkingSet, err = convertToUint64(v)
		case col == colRxBytes:
			stats.Network.RxBytes, err = convertToUint64(v)
		case col == colRxErrors:
			stats.Network.RxErrors, err = convertToUint64(v)
		case col == colTxBytes:
			stats.Network.TxBytes, err = convertToUint64(v)
		case col == colTxErrors:
			stats.Network.TxErrors, err = convertToUint64(v)
		}
		if err != nil {
			return nil, fmt.Errorf("column %v has invalid value %v: %v", col, v, err)
		}
	}
	return stats, nil
}

func (self *influxdbStorage) valuesToContainerSample(columns []string, values []interface{}) (*info.ContainerStatsSample, error) {
	sample := &info.ContainerStatsSample{}
	var err error
	for i, col := range columns {
		v := values[i]
		switch {
		case col == colTimestamp:
			if str, ok := v.(string); ok {
				sample.Timestamp, err = time.Parse(time.RFC3339Nano, str)
			}
		case col == colMachineName:
			if m, ok := v.(string); ok {
				if m != self.machineName {
					return nil, fmt.Errorf("different machine")
				}
			} else {
				return nil, fmt.Errorf("machine name field is not a string: %v", v)
			}
		// Memory Usage
		case col == colMemoryUsage:
			sample.Memory.Usage, err = convertToUint64(v)
		// sample duration. Unit: Nanosecond.
		case col == colSampleDuration:
			if v == nil {
				// this record does not have sample_duration, so it's the first stats.
				return nil, nil
			}
			sample.Duration, err = time.ParseDuration(v.(string))
		// Instant cpu usage
		case col == colCpuInstantUsage:
			sample.Cpu.Usage, err = convertToUint64(v)
		}
		if err != nil {
			return nil, fmt.Errorf("column %v has invalid value %v: %v", col, v, err)
		}
	}
	if sample.Duration.Nanoseconds() == 0 {
		return nil, nil
	}
	return sample, nil
}

func (self *influxdbStorage) OverrideReadyToFlush(readyToFlush func() bool) {
	self.readyToFlush = readyToFlush
}

func (self *influxdbStorage) defaultReadyToFlush() bool {
	return time.Since(self.lastWrite) >= self.bufferDuration
}

func (self *influxdbStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil || stats.Cpu == nil || stats.Memory == nil {
		return nil
	}
	// AddStats will be invoked simultaneously from multiple threads and only one of them will perform a write.
	var seriesToFlush []*influxdb.Series
	func() {
		self.lock.Lock()
		defer self.lock.Unlock()
		series := self.newSeries(self.containerStatsToValues(ref, stats))
		self.series = append(self.series, series)
		self.prevStats = stats.Copy(self.prevStats)
		if self.readyToFlush() {
			seriesToFlush = self.series
			self.series = make([]*influxdb.Series, 0)
			self.lastWrite = time.Now()
		}
	}()
	if len(seriesToFlush) > 0 {
		err := self.client.WriteSeriesWithTimePrecision(seriesToFlush, influxdb.Second)
		if err != nil {
			return fmt.Errorf("failed to write stats to influxDb - %s", err)
		}
	}

	return nil
}

func (self *influxdbStorage) RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error) {
	if numStats == 0 {
		return nil, nil
	}
	// TODO(dengnan): select only columns that we need
	// TODO(dengnan): escape names
	query := fmt.Sprintf("select * from %v where %v='%v' and %v='%v'", self.tableName, colContainerName, containerName, colMachineName, self.machineName)
	if numStats > 0 {
		query = fmt.Sprintf("%v limit %v", query, numStats)
	}
	series, err := self.client.Query(query)
	if err != nil {
		return nil, err
	}
	statsList := make([]*info.ContainerStats, 0, len(series))
	// By default, influxDB returns data in time descending order.
	// RecentStats() requires stats in time increasing order,
	// so we need to go through from the last one to the first one.
	for i := len(series) - 1; i >= 0; i-- {
		s := series[i]
		for j := len(s.Points) - 1; j >= 0; j-- {
			values := s.Points[j]
			stats, err := self.valuesToContainerStats(s.Columns, values)
			if err != nil {
				return nil, err
			}
			if stats == nil {
				continue
			}
			statsList = append(statsList, stats)
		}
	}
	return statsList, nil
}

func (self *influxdbStorage) Samples(containerName string, numSamples int) ([]*info.ContainerStatsSample, error) {
	if numSamples == 0 {
		return nil, nil
	}
	// TODO(dengnan): select only columns that we need
	// TODO(dengnan): escape names
	query := fmt.Sprintf("select * from %v where %v='%v' and %v='%v'", self.tableName, colContainerName, containerName, colMachineName, self.machineName)
	if numSamples > 0 {
		query = fmt.Sprintf("%v limit %v", query, numSamples)
	}
	series, err := self.client.Query(query)
	if err != nil {
		return nil, err
	}
	sampleList := make([]*info.ContainerStatsSample, 0, len(series))
	for i := len(series) - 1; i >= 0; i-- {
		s := series[i]
		for j := len(s.Points) - 1; j >= 0; j-- {
			values := s.Points[j]
			sample, err := self.valuesToContainerSample(s.Columns, values)
			if err != nil {
				return nil, err
			}
			if sample == nil {
				continue
			}
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
	selectedCol := make([]string, 0, len(cpuUsagePercentiles)+len(memUsagePercentiles)+1)

	selectedCol = append(selectedCol, fmt.Sprintf("max(%v)", colMemoryUsage))
	for _, p := range cpuUsagePercentiles {
		selectedCol = append(selectedCol, fmt.Sprintf("percentile(%v, %v)", colCpuInstantUsage, p))
	}
	for _, p := range memUsagePercentiles {
		selectedCol = append(selectedCol, fmt.Sprintf("percentile(%v, %v)", colMemoryUsage, p))
	}

	query := fmt.Sprintf("select %v from %v where %v='%v' and %v='%v' and time > now() - %v",
		strings.Join(selectedCol, ","),
		self.tableName,
		colContainerName,
		containerName,
		colMachineName,
		self.machineName,
		fmt.Sprintf("%vs", self.windowLen.Seconds()),
	)
	series, err := self.client.Query(query)
	if err != nil {
		return nil, err
	}
	if len(series) != 1 {
		return nil, nil
	}
	if len(series[0].Points) == 0 {
		return nil, nil
	}

	point := series[0].Points[0]

	ret := new(info.ContainerStatsPercentiles)
	ret.MaxMemoryUsage, err = convertToUint64(point[1])
	if err != nil {
		return nil, fmt.Errorf("invalid max memory usage: %v", err)
	}
	retrievedCpuPercentiles := point[2 : 2+len(cpuUsagePercentiles)]
	for i, p := range cpuUsagePercentiles {
		v, err := convertToUint64(retrievedCpuPercentiles[i])
		if err != nil {
			return nil, fmt.Errorf("invalid cpu usage: %v", err)
		}
		ret.CpuUsagePercentiles = append(
			ret.CpuUsagePercentiles,
			info.Percentile{
				Percentage: p,
				Value:      v,
			},
		)
	}
	retrievedMemoryPercentiles := point[2+len(cpuUsagePercentiles):]
	for i, p := range memUsagePercentiles {
		v, err := convertToUint64(retrievedMemoryPercentiles[i])
		if err != nil {
			return nil, fmt.Errorf("invalid memory usage: %v", err)
		}
		ret.MemoryUsagePercentiles = append(
			ret.MemoryUsagePercentiles,
			info.Percentile{
				Percentage: p,
				Value:      v,
			},
		)
	}
	return ret, nil
}

// Returns a new influxdb series.
func (self *influxdbStorage) newSeries(columns []string, points []interface{}) *influxdb.Series {
	out := &influxdb.Series{
		Name:    self.tableName,
		Columns: columns,
		// There's only one point for each stats
		Points: make([][]interface{}, 1),
	}
	out.Points[0] = points
	return out
}

// machineName: A unique identifier to identify the host that current cAdvisor
// instance is running on.
// influxdbHost: The host which runs influxdb.
// percentilesDuration: Time window which will be considered when calls Percentiles()
func New(machineName,
	tablename,
	database,
	username,
	password,
	influxdbHost string,
	isSecure bool,
	bufferDuration time.Duration,
	percentilesDuration time.Duration,
) (*influxdbStorage, error) {
	config := &influxdb.ClientConfig{
		Host:     influxdbHost,
		Username: username,
		Password: password,
		Database: database,
		IsSecure: isSecure,
	}
	client, err := influxdb.NewClient(config)
	if err != nil {
		return nil, err
	}
	// TODO(monnand): With go 1.3, we cannot compress data now.
	client.DisableCompression()
	if percentilesDuration.Seconds() < 1.0 {
		percentilesDuration = 5 * time.Minute
	}

	ret := &influxdbStorage{
		client:         client,
		windowLen:      percentilesDuration,
		machineName:    machineName,
		tableName:      tablename,
		bufferDuration: bufferDuration,
		lastWrite:      time.Now(),
		series:         make([]*influxdb.Series, 0),
	}
	ret.readyToFlush = ret.defaultReadyToFlush
	return ret, nil
}
