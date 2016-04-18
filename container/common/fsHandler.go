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

// Handler for Docker containers.
package common

import (
	"flag"
	"fmt"
	info "github.com/google/cadvisor/info/v1"
	"regexp"
	"sync"
	"time"

	"github.com/google/cadvisor/fs"
	"github.com/google/cadvisor/utils"

	"github.com/golang/glog"
)

type FsHandler interface {
	Start()
	Usage() ([]*info.FsStats, error)
	Stop()
	update() error // Exposed privately for tests
}

type skippedDevices struct {
	r *regexp.Regexp
}

var (
	defaultSkippedDevicesRegexp = regexp.MustCompile("$^") // Regexp that matches nothing
	skipDevicesFlag             = skippedDevices{
		r: defaultSkippedDevicesRegexp,
	}
)

func (self *skippedDevices) String() string {
	return self.r.String()
}

func (self *skippedDevices) Set(value string) error {
	r, err := regexp.Compile(value)

	if err != nil {
		return err
	}

	self.r = r
	return nil
}

func (self *skippedDevices) Has(device string) bool {
	return self.r.MatchString(device)
}

var skipDuFlag = flag.Bool("disk_skip_du", false, "do not use du for disk metrics (use raw FS stats instead)")

func init() {
	flag.Var(&skipDevicesFlag, "disk_skip_devices", "Regex representing devices to ignore when reporting disk metrics")
}

type realFsHandler struct {
	sync.RWMutex
	lastUpdate  time.Time
	fsStats     []*info.FsStats
	period      time.Duration
	minPeriod   time.Duration
	skipDu      bool
	skipDevices skippedDevices
	fsInfo      fs.FsInfo
	baseDirs    map[string]struct{}
	allDirs     map[string]struct{}
	// Tells the container to stop.
	stopChan chan struct{}
}

const (
	longDu             = time.Second
	duTimeout          = time.Minute
	maxDuBackoffFactor = 20
)

// Enforce that realFsHandler conforms to fsHandler interface
var _ FsHandler = &realFsHandler{}

func NewFsHandler(period time.Duration, baseDirs []string, extraDirs []string, fsInfo fs.FsInfo) FsHandler {
	allDirsSet := make(map[string]struct{})
	baseDirsSet := make(map[string]struct{})

	for _, dir := range baseDirs {
		allDirsSet[dir] = struct{}{}
		baseDirsSet[dir] = struct{}{}
	}

	for _, dir := range extraDirs {
		allDirsSet[dir] = struct{}{}
	}

	return &realFsHandler{
		lastUpdate:  time.Time{},
		fsStats:     nil,
		period:      period,
		minPeriod:   period,
		skipDu:      *skipDuFlag,
		skipDevices: skipDevicesFlag,
		baseDirs:    baseDirsSet,
		allDirs:     allDirsSet,
		fsInfo:      fsInfo,
		stopChan:    make(chan struct{}, 1),
	}
}

func addOrDefault(m map[string]uint64, key string, add uint64) {
	value, ok := m[key]
	if ok {
		m[key] = value + add
	} else {
		m[key] = add
	}
}

func (fh *realFsHandler) gatherDiskUsage(devices map[string]struct{}) (map[string]uint64, map[string]uint64, error) {
	deviceToBaseUsageBytes := make(map[string]uint64)
	deviceToTotalUsageBytes := make(map[string]uint64)

	if fh.skipDu {
		return deviceToBaseUsageBytes, deviceToTotalUsageBytes, nil
	}

	// Go through all directories and get their usage
	for dir := range fh.allDirs {
		if dir == "" {
			// This should not happen if we're called properly, but it's
			// presumably not worth crashing for.
			glog.Warningf("FS handler received an empty dir: %q", dir)
			continue
		}

		deviceInfo, err := fh.fsInfo.GetDirFsDevice(dir)
		if err != nil {
			return nil, nil, err
		}

		// Check whether this device was ignored prior to running du on it.
		_, collectUsageForDevice := devices[deviceInfo.Device]
		if !collectUsageForDevice {
			continue
		}

		usage, err := fh.fsInfo.GetDirUsage(dir, duTimeout)
		if err != nil {
			return nil, nil, err
		}

		// Only count usage against baseUsage if this directory is a base directory
		var baseUsage uint64 = 0
		_, isBaseDir := fh.baseDirs[dir]
		if isBaseDir {
			baseUsage = usage
		}

		addOrDefault(deviceToTotalUsageBytes, deviceInfo.Device, usage)
		addOrDefault(deviceToBaseUsageBytes, deviceInfo.Device, baseUsage)
	}

	return deviceToBaseUsageBytes, deviceToTotalUsageBytes, nil
}

func (fh *realFsHandler) update() error {
	// Start with figuring out which devices we care about
	deviceSet := make(map[string]struct{})
	for dir := range fh.allDirs {
		fsDevice, err := fh.fsInfo.GetDirFsDevice(dir)
		if err != nil {
			glog.Warningf("Unable to find device for directory %q: %v", dir, err)
			continue
		}

		if fh.skipDevices.Has(fsDevice.Device) {
			continue
		}

		deviceSet[fsDevice.Device] = struct{}{}
	}

	// If we are relying on du for metrics, then gather the usage for each of those devices
	deviceToBaseUsageBytes, deviceToTotalUsageBytes, err := fh.gatherDiskUsage(deviceSet)
	if err != nil {
		return err
	}

	// Then, grab the usage limit for each of those devices, as well as usage if we
	// are relying on df for metrics.
	fsStats := make([]*info.FsStats, 0)

	// Request Fs info for all the device in use, but skip io stats (we won't report them,
	// since per-container data is available via cgroups)
	filesystems, err := fh.fsInfo.GetFsInfoForDevices(deviceSet, false)
	if err != nil {
		return err
	}

	for _, fs := range filesystems {
		stat := info.FsStats{
			Device: fs.Device,
			Type:   string(fs.Type),
			Limit:  fs.Capacity,
		}

		// If we're using du, then use the metrics we collected above.
		// If we're using df, then simply use the value provided by GetGlobalFsInfo.
		if fh.skipDu {
			stat.Usage = fs.Capacity - fs.Available
		} else {
			baseUsage, ok := deviceToBaseUsageBytes[fs.Device]
			if !ok {
				return fmt.Errorf("Base usage for device %q expected but not collected!", fs.Device)
			}

			totalUsage, ok := deviceToTotalUsageBytes[fs.Device]
			if !ok {
				return fmt.Errorf("Total usage for device %q expected but not collected!", fs.Device)
			}
			stat.BaseUsage = baseUsage
			stat.Usage = totalUsage
		}

		fsStats = append(fsStats, &stat)
	}

	if err != nil {
		return err
	}

	fh.Lock()
	defer fh.Unlock()
	fh.lastUpdate = time.Now()
	fh.fsStats = fsStats
	return nil
}

func (fh *realFsHandler) trackUsage() {
	err := fh.update()
	if err != nil {
		glog.Errorf("failed to collect filesystem stats - %v", err)
	}

	for {
		select {
		case <-fh.stopChan:
			return
		case <-time.After(utils.Jitter(fh.period, 0.25)):
			start := time.Now()
			if err := fh.update(); err != nil {
				glog.Errorf("failed to collect filesystem stats - %v", err)
				fh.period = fh.period * 2
				if fh.period > maxDuBackoffFactor*fh.minPeriod {
					fh.period = maxDuBackoffFactor * fh.minPeriod
				}
			} else {
				fh.period = fh.minPeriod
			}
			duration := time.Since(start)
			if duration > longDu {
				glog.V(2).Infof("`du` on following dirs took %v: %v", duration, fh.allDirs)
			}
		}
	}
}

func (fh *realFsHandler) Start() {
	go fh.trackUsage()
}

func (fh *realFsHandler) Stop() {
	close(fh.stopChan)
}

func (fh *realFsHandler) Usage() ([]*info.FsStats, error) {
	fh.RLock()
	defer fh.RUnlock()
	if (fh.lastUpdate == time.Time{}) {
		// Do not report metrics if we don't have any!
		return []*info.FsStats{}, fmt.Errorf("No disk usage metrics available yet")
	}
	return fh.fsStats, nil
}
