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
	"sync"
	"time"

	"github.com/google/cadvisor/fs"

	"github.com/golang/glog"
)

type FsHandler interface {
	Start()
	Usage() (uint64, uint64)
	Stop()
}

type realFsHandler struct {
	sync.RWMutex
	lastUpdate     time.Time
	usageBytes     uint64
	baseUsageBytes uint64
	period         time.Duration
	rootfs         string
	extraDir       string
	fsInfo         fs.FsInfo
	// Tells the container to stop.
	stopChan chan struct{}
}

const longDu = time.Second

var _ FsHandler = &realFsHandler{}

func NewFsHandler(period time.Duration, rootfs, extraDir string, fsInfo fs.FsInfo) FsHandler {
	return &realFsHandler{
		lastUpdate:     time.Time{},
		usageBytes:     0,
		baseUsageBytes: 0,
		period:         period,
		rootfs:         rootfs,
		extraDir:       extraDir,
		fsInfo:         fsInfo,
		stopChan:       make(chan struct{}, 1),
	}
}

func (fh *realFsHandler) needsUpdate() bool {
	return time.Now().After(fh.lastUpdate.Add(fh.period))
}

func (fh *realFsHandler) update() error {
	// TODO(vishh): Add support for external mounts.
	baseUsage, err := fh.fsInfo.GetDirUsage(fh.rootfs)
	if err != nil {
		return err
	}

	extraDirUsage, err := fh.fsInfo.GetDirUsage(fh.extraDir)
	if err != nil {
		return err
	}

	fh.Lock()
	defer fh.Unlock()
	fh.lastUpdate = time.Now()
	fh.usageBytes = baseUsage + extraDirUsage
	fh.baseUsageBytes = baseUsage
	return nil
}

func (fh *realFsHandler) trackUsage() {
	fh.update()
	for {
		select {
		case <-fh.stopChan:
			return
		case <-time.After(fh.period):
			start := time.Now()
			if err := fh.update(); err != nil {
				glog.V(2).Infof("failed to collect filesystem stats - %v", err)
			}
			duration := time.Since(start)
			if duration > longDu {
				glog.V(3).Infof("`du` on following dirs took %v: %v", duration, []string{fh.rootfs, fh.extraDir})
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

func (fh *realFsHandler) Usage() (baseUsageBytes, totalUsageBytes uint64) {
	fh.RLock()
	defer fh.RUnlock()
	return fh.baseUsageBytes, fh.usageBytes
}
