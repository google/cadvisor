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
	atsdNet "github.com/axibase/atsd-api-go/net"
	info "github.com/google/cadvisor/info/v1"
	"reflect"
	"testing"
	"time"
)

func TestCalcDerivedSeries(t *testing.T) {
	machineName := "hostname"
	entityName := machineName + "/test-entity"
	testCases := []*DataTestCase{
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 100000001000000),
				Cpu: info.CpuStats{
					Usage: info.CpuUsage{
						Total:  1000000,
						PerCpu: []uint64{1000000, 1000000},
						User:   1000000,
						System: 1000000,
					},
					LoadAverage: 1000000000,
				},
			},
			seriesCommands: []*atsdNet.SeriesCommand{},
		},
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 100000002000000),
				Cpu: info.CpuStats{
					Usage: info.CpuUsage{
						Total:  2000000,
						PerCpu: []uint64{2000000, 2000000},
						User:   2000000,
						System: 2000000,
					},
					LoadAverage: 2000000000,
				},
			},
			seriesCommands: []*atsdNet.SeriesCommand{
				atsdNet.NewSeriesCommand(entityName, containerCpuUsageSystemPct, atsdNet.Float32(100)).
					SetMetricValue(containerCpuUsageTotalPct, atsdNet.Float32(100)).
					SetMetricValue(containerCpuUsageUserPct, atsdNet.Float32(100)).
					SetMetricValue(containerCpuHostUsageSystemPct, atsdNet.Float32(50)).
					SetMetricValue(containerCpuHostUsageTotalPct, atsdNet.Float32(50)).
					SetMetricValue(containerCpuHostUsageUserPct, atsdNet.Float32(50)).
					SetTimestamp(atsdNet.Millis(100000002)),
				atsdNet.NewSeriesCommand(entityName, containerCpuUsagePerCpuPct, atsdNet.Float32(100)).
					SetTag(cpu, "0").
					SetTimestamp(atsdNet.Millis(100000002)),
				atsdNet.NewSeriesCommand(entityName, containerCpuUsagePerCpuPct, atsdNet.Float32(100)).
					SetTag(cpu, "1").
					SetTimestamp(atsdNet.Millis(100000002)),
			},
		},
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 100000003000000),
				Cpu: info.CpuStats{
					Usage: info.CpuUsage{
						Total:  3000000,
						PerCpu: []uint64{3000000, 1000000},
						User:   1000000,
						System: 2000000,
					},
					LoadAverage: 1000000000,
				},
			},
			seriesCommands: []*atsdNet.SeriesCommand{
				atsdNet.NewSeriesCommand(entityName, containerCpuUsageSystemPct, atsdNet.Float32(0)).
					SetMetricValue(containerCpuUsageTotalPct, atsdNet.Float32(100)).
					SetMetricValue(containerCpuHostUsageSystemPct, atsdNet.Float32(0)).
					SetMetricValue(containerCpuHostUsageTotalPct, atsdNet.Float32(50)).
					SetTimestamp(atsdNet.Millis(100000003)),
				atsdNet.NewSeriesCommand(entityName, containerCpuUsagePerCpuPct, atsdNet.Float32(100)).
					SetTag(cpu, "0").
					SetTimestamp(atsdNet.Millis(100000003)),
			},
		},
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 100000004000000),
				Cpu:       info.CpuStats{},
			},
			seriesCommands: []*atsdNet.SeriesCommand{},
		},
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 100000000000000),
				Cpu:       info.CpuStats{},
			},
			seriesCommands: []*atsdNet.SeriesCommand{},
		},
	}

	for _, test := range testCases {
		stats := test.stats
		assertSeriesCommands := test.seriesCommands

		seriesCommands := CalculateDerivedSeriesCpuCommands(entityName, stats)

		if len(assertSeriesCommands) != len(seriesCommands) {
			t.Error("Wrong series command count: ", len(seriesCommands), "!=", len(assertSeriesCommands), " series: ", seriesCommands, "assert series: ", assertSeriesCommands)
		}

		for _, comm1 := range assertSeriesCommands {
			found := false
			for _, comm2 := range seriesCommands {
				if reflect.DeepEqual(comm1, comm2) {
					found = true
				}
			}
			if !found {
				t.Error("Series command not found: ", comm1, "series: ", seriesCommands)
			}
		}
	}
}
