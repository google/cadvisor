// +build linux

// Copyright 2020 Google Inc. All Rights Reserved.
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

// Collector of resctrl for a container.
package resctrl

import (
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/stats"

	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/intelrdt"
)

type collector struct {
	resctrl intelrdt.IntelRdtManager
	stats.NoopDestroy
}

func newCollector(id string, resctrlPath string) *collector {
	collector := &collector{
		resctrl: intelrdt.IntelRdtManager{
			Config: &configs.Config{
				IntelRdt: &configs.IntelRdt{},
			},
			Id:   id,
			Path: resctrlPath,
		},
	}

	return collector
}

func (c *collector) UpdateStats(stats *info.ContainerStats) error {
	stats.Resctrl = info.ResctrlStats{}

	resctrlStats, err := c.resctrl.GetStats()
	if err != nil {
		return err
	}

	numberOfNUMANodes := len(*resctrlStats.MBMStats)

	stats.Resctrl.MemoryBandwidthMonitoring = make([]info.MemoryBandwidthMonitoringStats, 0, numberOfNUMANodes)
	stats.Resctrl.CacheMonitoringTechnology = make([]info.CacheMonitoringTechnologyStats, 0, numberOfNUMANodes)

	for _, numaNodeStats := range *resctrlStats.MBMStats {
		stats.Resctrl.MemoryBandwidthMonitoring = append(stats.Resctrl.MemoryBandwidthMonitoring,
			info.MemoryBandwidthMonitoringStats{
				TotalBytes: numaNodeStats.MBMTotalBytes,
				LocalBytes: numaNodeStats.MBMLocalBytes,
			})
	}

	for _, numaNodeStats := range *resctrlStats.CMTStats {
		stats.Resctrl.CacheMonitoringTechnology = append(stats.Resctrl.CacheMonitoringTechnology,
			info.CacheMonitoringTechnologyStats{LLCOccupancy: numaNodeStats.LLCOccupancy})
	}

	return nil
}
