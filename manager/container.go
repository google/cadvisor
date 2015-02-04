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

package manager

import (
	"flag"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/docker/docker/pkg/units"
	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/utils/cpuload"
)

// Housekeeping interval.
var HousekeepingInterval = flag.Duration("housekeeping_interval", 1*time.Second, "Interval between container housekeepings")
var maxHousekeepingInterval = flag.Duration("max_housekeeping_interval", 60*time.Second, "Largest interval to allow between container housekeepings")
var allowDynamicHousekeeping = flag.Bool("allow_dynamic_housekeeping", true, "Whether to allow the housekeeping interval to be dynamic")

// Decay value used for load average smoothing. Interval length of 10 seconds is used.
var loadDecay = math.Exp(float64(-1 * (*HousekeepingInterval).Seconds() / 10))

type containerInfo struct {
	info.ContainerReference
	Subcontainers []info.ContainerReference
	Spec          info.ContainerSpec
}

type containerData struct {
	handler              container.ContainerHandler
	info                 containerInfo
	storageDriver        storage.StorageDriver
	lock                 sync.Mutex
	loadReader           cpuload.CpuLoadReader
	loadAvg              float64 // smoothed load average seen so far.
	housekeepingInterval time.Duration
	lastUpdatedTime      time.Time
	lastErrorTime        time.Time

	// Whether to log the usage of this container when it is updated.
	logUsage bool

	// Tells the container to stop.
	stop chan bool
}

func (c *containerData) Start() error {
	go c.housekeeping()
	return nil
}

func (c *containerData) Stop() error {
	c.stop <- true
	return nil
}

func (c *containerData) allowErrorLogging() bool {
	if time.Since(c.lastErrorTime) > time.Minute {
		c.lastErrorTime = time.Now()
		return true
	}
	return false
}

func (c *containerData) GetInfo() (*containerInfo, error) {
	// Get spec and subcontainers.
	if time.Since(c.lastUpdatedTime) > 5*time.Second {
		err := c.updateSpec()
		if err != nil {
			return nil, err
		}
		err = c.updateSubcontainers()
		if err != nil {
			return nil, err
		}
		c.lastUpdatedTime = time.Now()
	}
	// Make a copy of the info for the user.
	c.lock.Lock()
	defer c.lock.Unlock()
	return &c.info, nil
}

func newContainerData(containerName string, driver storage.StorageDriver, handler container.ContainerHandler, loadReader cpuload.CpuLoadReader, logUsage bool) (*containerData, error) {
	if driver == nil {
		return nil, fmt.Errorf("nil storage driver")
	}
	if handler == nil {
		return nil, fmt.Errorf("nil container handler")
	}
	ref, err := handler.ContainerReference()
	if err != nil {
		return nil, err
	}

	cont := &containerData{
		handler:              handler,
		storageDriver:        driver,
		housekeepingInterval: *HousekeepingInterval,
		loadReader:           loadReader,
		logUsage:             logUsage,
		loadAvg:              -1.0, // negative value indicates uninitialized.
		stop:                 make(chan bool, 1),
	}
	cont.info.ContainerReference = ref

	return cont, nil
}

// Determine when the next housekeeping should occur.
func (self *containerData) nextHousekeeping(lastHousekeeping time.Time) time.Time {
	if *allowDynamicHousekeeping {
		stats, err := self.storageDriver.RecentStats(self.info.Name, 2)
		if err != nil {
			if self.allowErrorLogging() {
				glog.Warningf("Failed to get RecentStats(%q) while determining the next housekeeping: %v", self.info.Name, err)
			}
		} else if len(stats) == 2 {
			// TODO(vishnuk): Use no processes as a signal.
			// Raise the interval if usage hasn't changed in the last housekeeping.
			if stats[0].StatsEq(stats[1]) && (self.housekeepingInterval < *maxHousekeepingInterval) {
				self.housekeepingInterval *= 2
				if self.housekeepingInterval > *maxHousekeepingInterval {
					self.housekeepingInterval = *maxHousekeepingInterval
				}
				glog.V(3).Infof("Raising housekeeping interval for %q to %v", self.info.Name, self.housekeepingInterval)
			} else if self.housekeepingInterval != *HousekeepingInterval {
				// Lower interval back to the baseline.
				self.housekeepingInterval = *HousekeepingInterval
				glog.V(3).Infof("Lowering housekeeping interval for %q to %v", self.info.Name, self.housekeepingInterval)
			}
		}
	}

	return lastHousekeeping.Add(self.housekeepingInterval)
}

func (c *containerData) housekeeping() {
	// Long housekeeping is either 100ms or half of the housekeeping interval.
	longHousekeeping := 100 * time.Millisecond
	if *HousekeepingInterval/2 < longHousekeeping {
		longHousekeeping = *HousekeepingInterval / 2
	}

	// Housekeep every second.
	glog.Infof("Start housekeeping for container %q\n", c.info.Name)
	lastHousekeeping := time.Now()
	for {
		select {
		case <-c.stop:
			// Stop housekeeping when signaled.
			return
		default:
			// Perform housekeeping.
			start := time.Now()
			c.housekeepingTick()

			// Log if housekeeping took too long.
			duration := time.Since(start)
			if duration >= longHousekeeping {
				glog.V(2).Infof("[%s] Housekeeping took %s", c.info.Name, duration)
			}
		}

		// Log usage if asked to do so.
		if c.logUsage {
			stats, err := c.storageDriver.RecentStats(c.info.Name, 2)
			if err != nil {
				if c.allowErrorLogging() {
					glog.Infof("[%s] Failed to get recent stats for logging usage: %v", c.info.Name, err)
				}
			} else if len(stats) < 2 {
				// Ignore, not enough stats yet.
			} else {
				usageCpuNs := stats[1].Cpu.Usage.Total - stats[0].Cpu.Usage.Total
				usageMemory := stats[1].Memory.Usage

				usageInCores := float64(usageCpuNs) / float64(stats[1].Timestamp.Sub(stats[0].Timestamp).Nanoseconds())
				usageInHuman := units.HumanSize(int64(usageMemory))
				glog.Infof("[%s] %.3f cores, %s of memory", c.info.Name, usageInCores, usageInHuman)
			}
		}

		// Schedule the next housekeeping. Sleep until that time.
		nextHousekeeping := c.nextHousekeeping(lastHousekeeping)
		if time.Now().Before(nextHousekeeping) {
			time.Sleep(nextHousekeeping.Sub(time.Now()))
		}
		lastHousekeeping = nextHousekeeping
	}
}

func (c *containerData) housekeepingTick() {
	err := c.updateStats()
	if err != nil {
		if c.allowErrorLogging() {
			glog.Infof("Failed to update stats for container \"%s\": %s", c.info.Name, err)
		}
	}
}

func (c *containerData) updateSpec() error {
	spec, err := c.handler.GetSpec()
	if err != nil {
		// Ignore errors if the container is dead.
		if !c.handler.Exists() {
			return nil
		}
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.info.Spec = spec
	return nil
}

// Calculate new smoothed load average using the new sample of runnable threads.
// The decay used ensures that the load will stabilize on a new constant value within
// 10 seconds.
func (c *containerData) updateLoad(newLoad uint64) {
	if c.loadAvg < 0 {
		c.loadAvg = float64(newLoad) // initialize to the first seen sample for faster stabilization.
	} else {
		c.loadAvg = c.loadAvg*loadDecay + float64(newLoad)*(1.0-loadDecay)
	}
	glog.V(3).Infof("New load for %q: %v. latest sample: %d", c.info.Name, c.loadAvg, newLoad)
}

func (c *containerData) updateStats() error {
	stats, statsErr := c.handler.GetStats()
	if statsErr != nil {
		// Ignore errors if the container is dead.
		if !c.handler.Exists() {
			return nil
		}

		// Stats may be partially populated, push those before we return an error.
		statsErr = fmt.Errorf("%v, continuing to push stats", statsErr)
	}
	if stats == nil {
		return statsErr
	}
	if c.loadReader != nil {
		// TODO(vmarmol): Cache this path.
		path, err := c.handler.GetCgroupPath("cpu")
		if err == nil {
			loadStats, err := c.loadReader.GetCpuLoad(c.info.Name, path)
			if err != nil {
				return fmt.Errorf("failed to get load stat for %q - path %q, error %s", c.info.Name, path, err)
			}
			stats.TaskStats = loadStats
			c.updateLoad(loadStats.NrRunning)
			// convert to 'milliLoad' to avoid floats and preserve precision.
			stats.Cpu.LoadAverage = int32(c.loadAvg * 1000)
		}
	}
	ref, err := c.handler.ContainerReference()
	if err != nil {
		// Ignore errors if the container is dead.
		if !c.handler.Exists() {
			return nil
		}
		return err
	}
	err = c.storageDriver.AddStats(ref, stats)
	if err != nil {
		return err
	}
	return statsErr
}

func (c *containerData) updateSubcontainers() error {
	subcontainers, err := c.handler.ListContainers(container.ListSelf)
	if err != nil {
		// Ignore errors if the container is dead.
		if !c.handler.Exists() {
			return nil
		}
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.info.Subcontainers = subcontainers
	return nil
}
