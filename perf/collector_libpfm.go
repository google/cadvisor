// +build libpfm,cgo

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

// Collector of perf events for a container.
package perf

// #cgo CFLAGS: -I/usr/include
// #cgo LDFLAGS: -lpfm
// #include <perfmon/pfmlib.h>
// #include <stdlib.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
	"unsafe"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/stats"

	"golang.org/x/sys/unix"
	"k8s.io/klog"
)

const (
	perfFile = iota
	cgroupFile
)

type collector struct {
	cgroupPath    string
	events        Events
	perfFiles     map[Metadata][]ReaderCloser
	perfFilesLock sync.Mutex
	numCores      int
}

var (
	isLibpfmInitialized = false
	libpmfMutex         = sync.Mutex{}
)

func init() {
	libpmfMutex.Lock()
	defer libpmfMutex.Unlock()
	pErr := C.pfm_initialize()
	if pErr != C.PFM_SUCCESS {
		fmt.Printf("unable to initialize libpfm: %d", int(pErr))
		return
	}
	isLibpfmInitialized = true
}

func NewCollector(cgroupPath string, events Events, numCores int) stats.Collector {
	return &collector{cgroupPath: cgroupPath, events: events, perfFiles: map[Metadata][]ReaderCloser{}, numCores: numCores}
}

func (c *collector) UpdateStats(stats *info.ContainerStats) error {
	c.perfFilesLock.Lock()
	defer c.perfFilesLock.Unlock()

	stats.PerfStats = []info.PerfStat{}
	klog.V(4).Infof("Attempting to update perf_event stats from cgroup %q", c.cgroupPath)
	for metadata, files := range c.perfFiles {
		buf := make([]byte, 32)
		_, err := files[perfFile].Read(buf)
		if err != nil {
			klog.Warningf("Unable to read from perf_event file (event: %q, CPU: %d) for %q", metadata.Name, metadata.Cpu, c.cgroupPath)
			continue
		}
		perfData := &ReadFormat{}
		reader := bytes.NewReader(buf)
		err = binary.Read(reader, binary.LittleEndian, perfData)
		now := time.Now()
		if err != nil {
			klog.Warningf("Unable to decode from binary format read from perf_event file (event: %q, CPU: %d) for %q", metadata.Name, metadata.Cpu, c.cgroupPath)
		}
		klog.Infof("Read metric for event %q for cpu %d from cgroup %q: %d", metadata.Name, metadata.Cpu, c.cgroupPath, perfData.Value)
		scalingRatio := float64(perfData.TimeEnabled) / float64(perfData.TimeRunning)
		stat := info.PerfStat{
			Value:        uint64(float64(perfData.Value) * scalingRatio),
			Name:         metadata.Name,
			Time:         now,
			ScalingRatio: scalingRatio,
			Cpu:          metadata.Cpu,
		}
		stats.PerfStats = append(stats.PerfStats, stat)
	}
	return nil
}

func (c *collector) Setup() {
	c.setupRawNonGrouped()
	c.setupNonGrouped()
}

func (c *collector) setupRawNonGrouped() {
	for _, event := range c.events.Raw.NonGrouped {
		klog.V(4).Infof("Setting up non-grouped raw perf event %#v", event)
		config := createPerfEventAttr(event)
		c.registerEvent(config, event.Name)
	}
}

func (c *collector) registerEvent(config *unix.PerfEventAttr, name string) {
	cgroup, err := os.Open(c.cgroupPath)
	if err != nil {
		klog.Errorf("Cannot open cgroup directory %q: %q", c.cgroupPath, err)
		return
	}
	var cpu int
	for cpu = 0; cpu < c.numCores; cpu++ {
		pid, groupFd, flags := int(cgroup.Fd()), -1, unix.PERF_FLAG_FD_CLOEXEC|unix.PERF_FLAG_PID_CGROUP
		fd, err := unix.PerfEventOpen(config, pid, cpu, groupFd, flags)
		if err != nil {
			klog.Errorf("Setting up perf event %#v failed: %q", config, err)
			return
		}
		perfFile := os.NewFile(uintptr(fd), name)
		if perfFile == nil {
			klog.Warningf("Unable to create os.File from file descriptor %#v", fd)
		}

		c.addEventFiles(Metadata{Name: name, Cpu: cpu}, perfFile, cgroup)
	}
}

func (c *collector) addEventFiles(metadata Metadata, perfFile *os.File, cgroup *os.File) {
	c.perfFilesLock.Lock()
	c.perfFiles[metadata] = []ReaderCloser{perfFile, cgroup}
	c.perfFilesLock.Unlock()
}

func (c *collector) setupNonGrouped() error {
	libpmfMutex.Lock()
	defer libpmfMutex.Unlock()
	if !isLibpfmInitialized {
		klog.Warning("libpfm4 is not initialized, cannot proceed with setting perf events up")
		return errors.New("libpfm4 is not initialized, cannot proceed with setting perf events up")
	}

	for _, name := range c.events.NonGrouped {
		klog.V(4).Infof("Setting up non-grouped perf event %s", name)

		perfEventAttrMemory := C.malloc(C.ulong(unsafe.Sizeof(unix.PerfEventAttr{})))
		event := pfmPerfEncodeArgT{}

		perfEventAttr := (*unix.PerfEventAttr)(perfEventAttrMemory)
		fstr := C.CString("")
		event.fstr = unsafe.Pointer(fstr)
		event.attr = perfEventAttrMemory
		event.size = C.ulong(unsafe.Sizeof(event))

		cSafeName := C.CString(name)
		pErr := C.pfm_get_os_event_encoding(cSafeName, C.PFM_PLM0|C.PFM_PLM3, C.PFM_OS_PERF_EVENT, unsafe.Pointer(&event))
		if pErr != C.PFM_SUCCESS {
			klog.Warningf("Unable to transform event name %s to perf_event_attr: %d", name, int(pErr))
			C.free(perfEventAttrMemory)
			continue
		}

		klog.V(1).Infof("perf_event_attr: %#v", perfEventAttr)
		setAttributes(perfEventAttr)
		c.registerEvent(perfEventAttr, name)
		C.free(perfEventAttrMemory)
	}

	return nil
}

func createPerfEventAttr(event RawEvent) *unix.PerfEventAttr {
	length := len(event.Config)

	config := &unix.PerfEventAttr{
		Type:   event.Type,
		Config: event.Config[0],
	}
	if length >= 2 {
		config.Ext1 = event.Config[1]
	}
	if length == 3 {
		config.Ext2 = event.Config[2]
	}

	setAttributes(config)
	klog.V(4).Infof("perf_event_attr struct prepared: %#v", config)
	return config
}

func setAttributes(config *unix.PerfEventAttr) {
	config.Sample_type = 1 << 16
	config.Read_format = unix.PERF_FORMAT_TOTAL_TIME_ENABLED | unix.PERF_FORMAT_TOTAL_TIME_RUNNING | unix.PERF_FORMAT_ID
	config.Bits = 1<<20 | 1<<1
	config.Size = uint32(unsafe.Sizeof(unix.PerfEventAttr{}))
}

func (c *collector) Destroy() {
	c.perfFilesLock.Lock()
	defer c.perfFilesLock.Unlock()

	for event, files := range c.perfFiles {
		klog.Infof("Closing perf_event file descriptor for cgroup %q", c.cgroupPath)
		err := files[perfFile].Close()
		if err != nil {
			klog.Warningf("Unable to close perf_event file descriptor for cgroup %q", c.cgroupPath)
		}
		klog.Infof("Closing cgroup file descriptor for %q", c.cgroupPath)
		err = files[cgroupFile].Close()
		if err != nil {
			klog.Warningf("Unable to close cgroup file descriptor for %q", c.cgroupPath)
		}
		delete(c.perfFiles, event)
	}
}

func Finalize() {
	libpmfMutex.Lock()
	defer libpmfMutex.Unlock()

	klog.V(1).Info("Attempting to terminate libpfm4")
	if !isLibpfmInitialized {
		klog.V(1).Info("libpfm4 has not been initialized; not terminating.")
		return
	}

	C.pfm_terminate()
	isLibpfmInitialized = false
}
