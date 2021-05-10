// +build linux

// Copyright 2021 Google Inc. All Rights Reserved.
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
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/klog/v2"

	info "github.com/google/cadvisor/info/v1"
)

const noInterval = 0

type collector struct {
	id                string
	interval          time.Duration
	getContainerPids  func() ([]string, error)
	resctrlPath       string
	running           bool
	numberOfNUMANodes int
	mu                sync.Mutex
}

func newCollector(id string, getContainerPids func() ([]string, error), interval time.Duration, numberOfNUMANodes int) *collector {
	return &collector{id: id, interval: interval, getContainerPids: getContainerPids, numberOfNUMANodes: numberOfNUMANodes,
		mu: sync.Mutex{}}
}

func (c *collector) setup() error {
	var err error
	c.resctrlPath, err = prepareMonitoringGroup(c.id, c.getContainerPids)

	if c.interval != noInterval {
		if err != nil {
			klog.Errorf("Failed to setup container %q resctrl collector: %w \n Trying again in next intervals.", c.id, err)
		} else {
			c.running = true
		}
		go func() {
			for {
				time.Sleep(c.interval)
				c.mu.Lock()
				klog.V(5).Infof("Trying to check %q containers control group.", c.id)
				if c.running {
					err = c.checkMonitoringGroup()
					if err != nil {
						c.running = false
						klog.Errorf("Failed to check %q resctrl collector control group: %w \n Trying again in next intervals.", c.id, err)
					}
				} else {
					c.resctrlPath, err = prepareMonitoringGroup(c.id, c.getContainerPids)
					if err != nil {
						c.running = false
						klog.Errorf("Failed to setup container %q resctrl collector: %w \n Trying again in next intervals.", c.id, err)
					} else {
						c.running = true
					}
				}
				c.mu.Unlock()
			}
		}()
	} else {
		// There is no interval set, if setup fail, stop.
		if err != nil {
			return fmt.Errorf("failed to setup container %q resctrl collector: %w", c.id, err)
		}
		c.running = true
	}

	return nil
}

func (c *collector) checkMonitoringGroup() error {
	newPath, err := prepareMonitoringGroup(c.id, c.getContainerPids)
	if err != nil {
		return fmt.Errorf("couldn't obtain mon_group path: %v", err)
	}

	// Check if container moved between control groups.
	if newPath != c.resctrlPath {
		err = c.clear()
		if err != nil {
			return fmt.Errorf("couldn't clear previous monitoring group: %w", err)
		}
		c.resctrlPath = newPath
	}

	return nil
}

func (c *collector) UpdateStats(stats *info.ContainerStats) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		stats.Resctrl = info.ResctrlStats{}

		resctrlStats, err := getIntelRDTStatsFrom(c.resctrlPath)
		if err != nil {
			return err
		}

		stats.Resctrl.MemoryBandwidth = make([]info.MemoryBandwidthStats, 0, c.numberOfNUMANodes)
		stats.Resctrl.Cache = make([]info.CacheStats, 0, c.numberOfNUMANodes)

		for _, numaNodeStats := range *resctrlStats.MBMStats {
			stats.Resctrl.MemoryBandwidth = append(stats.Resctrl.MemoryBandwidth,
				info.MemoryBandwidthStats{
					TotalBytes: numaNodeStats.MBMTotalBytes,
					LocalBytes: numaNodeStats.MBMLocalBytes,
				})
		}

		for _, numaNodeStats := range *resctrlStats.CMTStats {
			stats.Resctrl.Cache = append(stats.Resctrl.Cache,
				info.CacheStats{LLCOccupancy: numaNodeStats.LLCOccupancy})
		}
	}

	return nil
}

func (c *collector) Destroy() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	err := c.clear()
	if err != nil {
		klog.Errorf("trying to destroy %q resctrl collector but: %v", c.id, err)
	}
}

func (c *collector) clear() error {
	// Not allowed to remove root or undefined resctrl directory.
	if c.id != rootContainer && c.resctrlPath != "" {
		err := os.RemoveAll(c.resctrlPath)
		if err != nil {
			return fmt.Errorf("couldn't clear mon_group: %v", err)
		}
	}
	return nil
}
