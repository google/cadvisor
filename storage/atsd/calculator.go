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
	"strconv"
	"sync"
	"time"

	atsdNet "github.com/axibase/atsd-api-go/net"
	info "github.com/google/cadvisor/info/v1"
)

const (
	containerCpuUsageSystemPct = "cadvisor.cpu.usage.system%"
	containerCpuUsageTotalPct  = "cadvisor.cpu.usage.total%"
	containerCpuUsageUserPct   = "cadvisor.cpu.usage.user%"

	containerCpuHostUsageSystemPct = "cadvisor.cpu.host.usage.system%"
	containerCpuHostUsageTotalPct  = "cadvisor.cpu.host.usage.total%"
	containerCpuHostUsageUserPct   = "cadvisor.cpu.host.usage.user%"

	containerCpuUsagePerCpuPct = "cadvisor.cpu.usage.percpu%"
)

var oldStats = map[string]*info.ContainerStats{}
var m = &sync.Mutex{}

// derived stats which cannot be calculated within atsd for now
func CalculateDerivedSeriesCpuCommands(entity string, curStats *info.ContainerStats) []*atsdNet.SeriesCommand {
	commands := []*atsdNet.SeriesCommand{}
	m.Lock()
	if prevStats, ok := oldStats[entity]; ok {
		if prevStats.Timestamp.Before(curStats.Timestamp) {
			time := uint64(curStats.Timestamp.UnixNano() / time.Millisecond.Nanoseconds())
			deltaT := curStats.Timestamp.UnixNano() - prevStats.Timestamp.UnixNano()
			cpuCount := len(curStats.Cpu.Usage.PerCpu)
			metricsMap := map[string]float64{}
			if curStats.Cpu.Usage.System >= prevStats.Cpu.Usage.System {
				metricsMap[containerCpuUsageSystemPct] = 1e2 * float64(curStats.Cpu.Usage.System-prevStats.Cpu.Usage.System) / float64(deltaT)
				if cpuCount > 0 {
					metricsMap[containerCpuHostUsageSystemPct] = metricsMap[containerCpuUsageSystemPct] / float64(cpuCount)
				}
			}
			if curStats.Cpu.Usage.User >= prevStats.Cpu.Usage.User {
				metricsMap[containerCpuUsageUserPct] = 1e2 * float64(curStats.Cpu.Usage.User-prevStats.Cpu.Usage.User) / float64(deltaT)
				if cpuCount > 0 {
					metricsMap[containerCpuHostUsageUserPct] = metricsMap[containerCpuUsageUserPct] / float64(cpuCount)
				}
			}
			if curStats.Cpu.Usage.Total >= prevStats.Cpu.Usage.Total {
				metricsMap[containerCpuUsageTotalPct] = 1e2 * float64(curStats.Cpu.Usage.Total-prevStats.Cpu.Usage.Total) / float64(deltaT)
				if cpuCount > 0 {
					metricsMap[containerCpuHostUsageTotalPct] = metricsMap[containerCpuUsageTotalPct] / float64(cpuCount)
				}
			}
			var command *atsdNet.SeriesCommand
			for key, val := range metricsMap {
				if command == nil {
					command = atsdNet.NewSeriesCommand(entity, key, atsdNet.Float32(val))
				} else {
					command.SetMetricValue(key, atsdNet.Float32(val))
				}
			}
			if command != nil {
				command.SetTimestamp(atsdNet.Millis(time))
				commands = append(commands, command)
			}
			for i, cpuUsage := range curStats.Cpu.Usage.PerCpu {
				if i < len(prevStats.Cpu.Usage.PerCpu) && cpuUsage >= prevStats.Cpu.Usage.PerCpu[i] {
					commands = append(commands, atsdNet.NewSeriesCommand(entity, containerCpuUsagePerCpuPct, atsdNet.Float32(1e2*float64(cpuUsage-prevStats.Cpu.Usage.PerCpu[i])/float64(deltaT))).
						SetTag(cpu, strconv.FormatInt(int64(i), 10)).
						SetTimestamp(atsdNet.Millis(time)))
				}
			}
			oldStats[entity] = curStats
		}
	} else {
		oldStats[entity] = curStats
	}
	m.Unlock()
	return commands
}
