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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/units"
	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/info/v2"
	"github.com/google/cadvisor/storage/memory"
	"github.com/google/cadvisor/summary"
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
	memoryStorage        *memory.InMemoryStorage
	lock                 sync.RWMutex
	loadReader           cpuload.CpuLoadReader
	summaryReader        *summary.StatsSummary
	loadAvg              float64 // smoothed load average seen so far.
	housekeepingInterval time.Duration
	lastUpdatedTime      time.Time
	lastErrorTime        time.Time
	lastCpuLoadGrab      time.Time
	cgroupPath           string
	contSubcontainers    map[namespacedContainerName]*containerData

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

func (c *containerData) DerivedStats() (v2.DerivedStats, error) {
	if c.summaryReader == nil {
		return v2.DerivedStats{}, fmt.Errorf("derived stats not enabled for container %q", c.info.Name)
	}
	return c.summaryReader.DerivedStats()
}

func newContainerData(containerName string, memoryStorage *memory.InMemoryStorage, handler container.ContainerHandler, loadReader cpuload.CpuLoadReader, logUsage bool) (*containerData, error) {
	if memoryStorage == nil {
		return nil, fmt.Errorf("nil memory storage")
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
		memoryStorage:        memoryStorage,
		housekeepingInterval: *HousekeepingInterval,
		loadReader:           loadReader,
		logUsage:             logUsage,
		loadAvg:              -1.0, // negative value indicates uninitialized.
		stop:                 make(chan bool, 1),
	}
	cont.info.ContainerReference = ref

	err = cont.updateSpec()
	if err != nil {
		return nil, err
	}
	cont.summaryReader, err = summary.New(cont.info.Spec)
	if err != nil {
		cont.summaryReader = nil
		glog.Warningf("Failed to create summary reader for %q: %v", ref.Name, err)
	}

	return cont, nil
}

// Determine when the next housekeeping should occur.
func (self *containerData) nextHousekeeping(lastHousekeeping time.Time) time.Time {
	if *allowDynamicHousekeeping {
		var empty time.Time
		stats, err := self.memoryStorage.RecentStats(self.info.Name, empty, empty, 2)
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
			} else if self.housekeepingInterval != *HousekeepingInterval {
				// Lower interval back to the baseline.
				self.housekeepingInterval = *HousekeepingInterval
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
	glog.V(3).Infof("Start housekeeping for container %q\n", c.info.Name)
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
				glog.V(3).Infof("[%s] Housekeeping took %s", c.info.Name, duration)
			}
		}

		// Log usage if asked to do so.
		if c.logUsage {
			const numSamples = 60
			var empty time.Time
			stats, err := c.memoryStorage.RecentStats(c.info.Name, empty, empty, numSamples)
			if err != nil {
				if c.allowErrorLogging() {
					glog.Infof("[%s] Failed to get recent stats for logging usage: %v", c.info.Name, err)
				}
			} else if len(stats) < numSamples {
				// Ignore, not enough stats yet.
			} else {
				usageCpuNs := uint64(0)
				for i := range stats {
					if i > 0 {
						usageCpuNs += (stats[i].Cpu.Usage.Total - stats[i-1].Cpu.Usage.Total)
					}
				}
				usageMemory := stats[numSamples-1].Memory.Usage

				instantUsageInCores := float64(stats[numSamples-1].Cpu.Usage.Total-stats[numSamples-2].Cpu.Usage.Total) / float64(stats[numSamples-1].Timestamp.Sub(stats[numSamples-2].Timestamp).Nanoseconds())
				usageInCores := float64(usageCpuNs) / float64(stats[numSamples-1].Timestamp.Sub(stats[0].Timestamp).Nanoseconds())
				usageInHuman := units.HumanSize(float64(usageMemory))
				glog.Infof("[%s] %.3f cores (average: %.3f cores), %s of memory", c.info.Name, instantUsageInCores, usageInCores, usageInHuman)
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
}

func (c *containerData) getContainerDataLoadStats() (info.LoadStats, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	var loadstats info.LoadStats
	if time.Since(c.lastCpuLoadGrab) > time.Second {
		loadstats, err := c.loadReader.GetCpuLoad(c.info.Name, c.cgroupPath)
		if err != nil {
			glog.V(3).Infof("got error trying to read cpuload for container %v: %v", c.info.Name, err)
		}
		c.lastCpuLoadGrab = time.Now()
		return loadstats, err
	} else {
		var uninitializedTime time.Time
		recentStats, err := c.memoryStorage.RecentStats(c.info.Name, uninitializedTime, uninitializedTime, 1)
		if err != nil || len(recentStats) < 1 {
			glog.V(3).Infof("Failed to get recent stats for %v. len(recentStats) = %v and err is %v", c.info.Name, len(recentStats), err)
			return loadstats, err
		}
		glog.V(3).Infof("container %v: recentStats[0].TaskStats = %+v", c.info.Name, recentStats[0].TaskStats)
		return recentStats[0].TaskStats, err
	}
}

func (c *containerData) updateSubcontainerStats(stats *info.ContainerStats) (*info.ContainerStats, error) {
	for _, sub := range c.contSubcontainers {
		glog.V(3).Infof("calling updateSubcontainerStats for container %v and sub %v", c.info.Name, sub.info.Name)
		stats, err := sub.updateSubcontainerStats(stats)
		if err != nil {
			glog.Errorf("trying to updateSubcontainerStats on sub %v of container %v got error %v", sub.info.Name, c.info.Name, err)
			continue
		}
		loadStats, err := sub.getContainerDataLoadStats()
		if err != nil {
			glog.Errorf("failed to get load stat for %q - path %q, error %s", c.info.Name, c.cgroupPath, err)
			continue
		}
		stats.UpdateTaskStats(loadStats)
		c.updateLoad(loadStats.NrRunning)
		// convert to 'milliLoad' to avoid floats and preserve precision.
		stats.Cpu.LoadAverage = int32(c.loadAvg * 1000)
	}
	return stats, nil
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
		preloadavg := c.loadAvg
		// calls GetCpuLoad or gets most recent stats
		loadStats, err := c.getContainerDataLoadStats()
		if err != nil {
			return fmt.Errorf("failed to get load stat for %q - path %q, error %s", c.info.Name, c.cgroupPath, err)
		}
		stats.TaskStats = loadStats
		c.updateLoad(loadStats.NrRunning)
		// convert to 'milliLoad' to avoid floats and preserve precision.
		stats.Cpu.LoadAverage = int32(c.loadAvg * 1000)
		midloadavg := c.loadAvg

		stats, err = c.updateSubcontainerStats(stats)
		if err != nil {
			glog.Errorf("failed to updateStats getting subcontainer info with error %v", err)
			return fmt.Errorf("failed to get subcontainer load stat for %q - path %q, error %s", c.info.Name, c.cgroupPath, err)
		}
		postloadavg := c.loadAvg
		glog.V(2).Infof("container: %+v; loadavg pre: %v, mid: %+v, post: %v\n", c.info.Name, preloadavg, midloadavg, postloadavg)

	}
	if c.summaryReader != nil {
		err := c.summaryReader.AddSample(*stats)
		if err != nil {
			// Ignore summary errors for now.
			glog.V(2).Infof("failed to add summary stats for %q: %v", c.info.Name, err)
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
	err = c.memoryStorage.AddStats(ref, stats)
	if err != nil {
		return err
	}
	return statsErr
}

func (c *containerData) updateSubcontainers() error {
	var subcontainers info.ContainerReferenceSlice
	subcontainers, err := c.handler.ListContainers(container.ListSelf)
	if err != nil {
		// Ignore errors if the container is dead.
		if !c.handler.Exists() {
			return nil
		}
		return err
	}
	sort.Sort(subcontainers)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.info.Subcontainers = subcontainers
	return nil
}

func isImmediateSubcontainer(potentialParent string, potentialChild string) bool {
	// if they're the same container return false
	potentialParent = strings.Trim(potentialParent, "/ ")
	potentialChild = strings.Trim(potentialChild, "/ ")
	if potentialParent == potentialChild {
		// glog.Infof("parent: %v, child: %v, %v", potentialParent, potentialChild, false)
		return false
	}
	// if parent is not prefix of child it cannot be parent
	if !strings.HasPrefix(potentialChild, potentialParent) {
		// glog.Infof("parent: %v, child: %v, %v", potentialParent, potentialChild, false)
		return false
	}
	// if the subcontainer is more than one layer down it's not a direct subcontainer
	extraSubcontainers := strings.TrimLeft(potentialChild, potentialParent)
	if len(strings.Split(strings.Trim(extraSubcontainers, "/ "), "/")) > 1 {
		// glog.Infof("parent: %v, child: %v, %v", potentialParent, potentialChild, false)
		return false
	}
	// glog.Infof("parent: %v, child: %v, %v", potentialParent, potentialChild, true)
	return true
}

func (m *manager) getSubcontainerContainerData(containerName string) map[namespacedContainerName]*containerData {
	subcontainers := make(map[namespacedContainerName]*containerData)
	for namespacedNameKey, containerDataValue := range m.containers {
		if isImmediateSubcontainer(containerName, namespacedNameKey.Name) {
			subcontainers[namespacedNameKey] = containerDataValue
		}
	}
	return subcontainers
}

func (m *manager) addAsSubcontainer(cont *containerData, currentNamespacedName namespacedContainerName) {
	containerName := cont.info.Name
	for namespacedNameKey, containerDataValue := range m.containers {
		if isImmediateSubcontainer(namespacedNameKey.Name, containerName) {
			containerDataValue.contSubcontainers[currentNamespacedName] = cont
		}
	}
}

func (m *manager) removeSubcontainer(cont *containerData, currentNamespacedName namespacedContainerName) {
	for namespacedNameKey, containerDataValue := range m.containers {
		delete(containerDataValue.contSubcontainers, namespacedNameKey)
	}
}
