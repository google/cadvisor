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
	"time"

	"k8s.io/klog/v2"

	info "github.com/google/cadvisor/info/v1"
)

type collector struct {
	id               string
	interval         time.Duration
	getContainerPids func() ([]string, error)
	resctrlPath      string
	running          bool
}

func newCollector(id string, getContainerPids func() ([]string, error), interval time.Duration) *collector {
	return &collector{id: id, interval: interval, getContainerPids: getContainerPids}
}

func (c *collector) setup() error {
	if c.id != rootContainer {
		// There is no need to prepare or update "/" container.
		err := c.prepareMonGroup()

		if c.interval != 0 {
			if err != nil {
				klog.Errorf("Failed to setup container %q resctrl collector: %w \n Trying again in next intervals!", c.id, err)
			}
			go func() {
				for {
					time.Sleep(c.interval)
					if c.running {
						err = c.prepareMonGroup()
						if err != nil {
							klog.Errorf("checking %q resctrl collector but: %w", c.id, err)
						}
					} else {
						err = c.clear()
						if err != nil {
							klog.Errorf("trying to end %q resctrl collector interval but: %w", c.id, err)
						}
						break
					}
				}
			}()
		} else {
			// There is no interval set, if setup fail, stop.
			if err != nil {
				c.running = false
				return err
			}
		}
	}

	return nil
}

func (c *collector) prepareMonGroup() error {
	newPath, err := getResctrlPath(c.id, c.getContainerPids)
	if err != nil {
		return fmt.Errorf("couldn't obtain mon_group path: %v", err)
	}

	// Check if container moved between control groups.
	if newPath != c.resctrlPath {
		err = c.clear()
		if err != nil {
			c.running = false
			return fmt.Errorf("couldn't clear previous mon group: %v", err)
		}
		c.resctrlPath = newPath
	}

	// Mon group prepared, the collector is running correctly.
	c.running = true
	return nil
}

func (c *collector) UpdateStats(stats *info.ContainerStats) error {
	if c.running {
		stats.Resctrl = info.ResctrlStats{}

		resctrlStats, err := getStats(c.resctrlPath)
		if err != nil {
			return err
		}
		numberOfNUMANodes := len(*resctrlStats.MBMStats)

		stats.Resctrl.MemoryBandwidth = make([]info.MemoryBandwidthStats, 0, numberOfNUMANodes)
		stats.Resctrl.Cache = make([]info.CacheStats, 0, numberOfNUMANodes)

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
	c.running = false
	err := c.clear()
	if err != nil {
		klog.Errorf("trying to destroy %q resctrl collector but: %v", c.id, err)
	}
}

func (c *collector) clear() error {
	// Couldn't remove root resctrl directory.
	if c.id != rootContainer && c.resctrlPath != "" {
		err := os.RemoveAll(c.resctrlPath)
		if err != nil {
			return fmt.Errorf("couldn't clear mon_group: %v", err)
		}
	}
	return nil
}
