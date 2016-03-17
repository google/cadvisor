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
	"strconv"
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

// derived stats which cannot be calculated within atsd for now
func CalculateDerivedSeriesCpuCommands(entity string, stats *info.ContainerStats) []*atsdNet.SeriesCommand {
	if ostats, ok := oldStats[entity]; ok {
		if ostats.Timestamp.Before(stats.Timestamp) {
			time := uint64(stats.Timestamp.UnixNano() / 1e6)
			deltaT := float64(stats.Timestamp.UnixNano() - ostats.Timestamp.UnixNano())
			cpuCount := float64(len(stats.Cpu.Usage.PerCpu))
			metricsMap := map[string]float64{}
			if stats.Cpu.Usage.System >= ostats.Cpu.Usage.System {
				metricsMap[containerCpuUsageSystemPct] = 1e2 * float64(stats.Cpu.Usage.System-ostats.Cpu.Usage.System) / deltaT
			}
			if stats.Cpu.Usage.Total >= ostats.Cpu.Usage.Total {
				metricsMap[containerCpuUsageTotalPct] = 1e2 * float64(stats.Cpu.Usage.Total-ostats.Cpu.Usage.Total) / deltaT
			}
			if stats.Cpu.Usage.User >= ostats.Cpu.Usage.User {
				metricsMap[containerCpuUsageUserPct] = 1e2 * float64(stats.Cpu.Usage.User-ostats.Cpu.Usage.User) / deltaT
			}
			if cpuCount > 0 {
				if stats.Cpu.Usage.System >= ostats.Cpu.Usage.System {
					metricsMap[containerCpuHostUsageSystemPct] = 1e2 * float64(stats.Cpu.Usage.System-ostats.Cpu.Usage.System) / (cpuCount * deltaT)
				}
				if stats.Cpu.Usage.Total >= ostats.Cpu.Usage.Total {
					metricsMap[containerCpuHostUsageTotalPct] = 1e2 * float64(stats.Cpu.Usage.Total-ostats.Cpu.Usage.Total) / (cpuCount * deltaT)
				}
				if stats.Cpu.Usage.User >= ostats.Cpu.Usage.User {
					metricsMap[containerCpuHostUsageUserPct] = 1e2 * float64(stats.Cpu.Usage.User-ostats.Cpu.Usage.User) / (cpuCount * deltaT)
				}
			}
			commands := []*atsdNet.SeriesCommand{}
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
			for i, cpuUsage := range stats.Cpu.Usage.PerCpu {
				if i < len(ostats.Cpu.Usage.PerCpu) && cpuUsage >= ostats.Cpu.Usage.PerCpu[i] {
					commands = append(commands, atsdNet.NewSeriesCommand(entity, containerCpuUsagePerCpuPct, atsdNet.Float32(1e2*float64(cpuUsage-ostats.Cpu.Usage.PerCpu[i])/deltaT)).
						SetTag(cpu, strconv.FormatInt(int64(i), 10)).
						SetTimestamp(atsdNet.Millis(time)))
				}
			}
			oldStats[entity] = stats
			return commands
		} else {
			return []*atsdNet.SeriesCommand{}
		}
	} else {
		oldStats[entity] = stats
		return []*atsdNet.SeriesCommand{}
	}
}
