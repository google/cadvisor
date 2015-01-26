// Copyright 2015 Google Inc. All Rights Reserved.
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

package scheddebug

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/utils"
)

const (
	schedDebugPath = "/proc/sched_debug"
)

var (
	// Scans cpu number, task group name, and number of running threads.
	// TODO(rjnagal): cpu number is only used for debug. Remove it later.
	schedRegExp  = regexp.MustCompile(`cfs_rq\[([0-9]+)\]:(.*)\n(?:.*\n)*?.*nr_running.*: ([0-9]+)`)
	pollInterval = 1 * time.Second
)

type SchedReader struct {
	quitChan      chan error // Used to cleanly shutdown housekeeping.
	lastErrorTime time.Time  // Limit errors to one per minute.
	dataLock      sync.RWMutex
	load          map[string]int // Load per container. Guarded by dataLock.
}

func (self *SchedReader) Start() error {
	self.quitChan = make(chan error)
	self.refresh()
	go self.housekeep()
	return nil
}

func (self *SchedReader) Stop() {
	self.quitChan <- nil
	err := <-self.quitChan
	if err != nil {
		glog.Warning("Failed to stop scheddebug load reader: %s", err)
	}
}

// Since load housekeeping and normal container housekeeping runs at the same rate,
// there is a chance of sometimes picking the last cycle's data. We can solve that by
// calling this housekeeping from globalhousekeeping if its an issue.
func (self *SchedReader) housekeep() {
	ticker := time.Tick(pollInterval)
	for {
		select {
		case <-ticker:
			self.refresh()
		case <-self.quitChan:
			self.quitChan <- nil
			glog.Infof("Exiting housekeeping")
			return
		}
	}
}

func (self *SchedReader) refresh() {
	out, err := ioutil.ReadFile(schedDebugPath)
	if err != nil {
		if self.allowErrorLogging() {
			glog.Warningf("Error reading sched debug file %v: %v", schedDebugPath, err)
		}
		return
	}
	load := make(map[string]int)
	matches := schedRegExp.FindAllSubmatch(out, -1)
	for _, matchSlice := range matches {
		if len(matchSlice) != 4 {
			if self.allowErrorLogging() {
				glog.Warningf("Malformed sched debug entry: %v", matchSlice)
			}
			continue
		}
		cpu := string(matchSlice[1])
		cgroup := string(matchSlice[2])
		n := string(matchSlice[3])
		numRunning, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			if self.allowErrorLogging() {
				glog.Warningf("Could not parse running tasks from: %q", n)
			}
			continue
		}
		glog.V(2).Infof("Load for %q on cpu %s: %d", cgroup, cpu, numRunning)
		if strings.HasPrefix(cgroup, "/autogroup") {
			// collapse all autogroups to root. This keeps our internal map compact.
			cgroup = "/"
		}
		// TODO(rjnagal): Walk up the path and add load to all parent containers. That will make
		// it different from netlink approach which is non-hierarchical.
		load[cgroup] += int(numRunning)

		if cgroup != "/" {
			// Return the whole machine load for root. Add all task group's running processes to root.
			load["/"] += int(numRunning)
		}
	}
	glog.V(2).Infof("New loads : %+v", load)
	self.dataLock.Lock()
	defer self.dataLock.Unlock()
	self.load = load
}

func (self *SchedReader) GetCpuLoad(name string, path string) (stats info.LoadStats, err error) {
	self.dataLock.RLock()
	defer self.dataLock.RUnlock()
	stats.NrRunning = uint64(self.load[name])
	return stats, nil
}

func (self *SchedReader) allowErrorLogging() bool {
	if time.Since(self.lastErrorTime) > time.Minute {
		self.lastErrorTime = time.Now()
		return true
	}
	return false
}

func New() (*SchedReader, error) {
	if !utils.FileExists(schedDebugPath) {
		return nil, fmt.Errorf("sched debug file %q not accessible", schedDebugPath)
	}
	return &SchedReader{}, nil
}
