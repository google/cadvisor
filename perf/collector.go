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

import (
	"bytes"
	"encoding/binary"
	"os"
	"sync"
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
	perfFiles     map[string][]*os.File
	perfFilesLock sync.Mutex
}

func NewCollector(cgroupPath string, events Events) stats.Collector {
	return &collector{cgroupPath: cgroupPath, events: events, perfFiles: map[string][]*os.File{}}
}

func (c *collector) UpdateStats(stats *info.ContainerStats) error {
	c.perfFilesLock.Lock()
	defer c.perfFilesLock.Unlock()

	klog.V(4).Infof("Attempting to update perf_event stats from cgroup %q", c.cgroupPath)
	for event, file := range c.perfFiles {
		buf := make([]byte, 32)
		_, err := file[perfFile].Read(buf)
		if err != nil {
			klog.Warningf("Unable to read from perf_event file %q for %q", event, c.cgroupPath)
			continue
		}
		metric := &ReadFormat{}
		reader := bytes.NewReader(buf)
		err = binary.Read(reader, binary.LittleEndian, metric)
		if err != nil {
			klog.Warningf("Unable to decode from binary format read from perf_event file %q for %q", event, c.cgroupPath)
		}
		klog.Infof("Read metric for event %q from cgroup %q: %d", event, c.cgroupPath, metric.Value)
	}
	return nil
}

func (c *collector) Setup() {
	for _, event := range c.events.NonGrouped {
		klog.V(4).Infof("Setting up non-grouped perf event %#v", event)
		config := createPerfEventAttr(event)

		cgroup, err := os.Open(c.cgroupPath)
		if err != nil {
			klog.Errorf("Cannot open cgroup directory %q: %q", c.cgroupPath, err)
			return
		}
		pid, cpu, groupFd, flags := int(cgroup.Fd()), -1, -1, unix.PERF_FLAG_FD_CLOEXEC
		fd, err := unix.PerfEventOpen(config, pid, cpu, groupFd, flags)
		if err != nil {
			klog.Errorf("Setting up perf event %#v failed: %q", event, err)
			return
		}
		perfFile := os.NewFile(uintptr(fd), event.Name)
		if perfFile == nil {
			klog.Warningf("Unable to create os.File from file descriptor %#v", fd)
		}

		c.perfFilesLock.Lock()
		c.perfFiles[event.Name] = []*os.File{perfFile, cgroup}
		c.perfFilesLock.Unlock()
	}
}

func createPerfEventAttr(event Event) *unix.PerfEventAttr {
	length := len(event.Config)
	size := uint32(unsafe.Sizeof(unix.PerfEventAttr{}))

	config := &unix.PerfEventAttr{
		Type:        event.Type,
		Config:      event.Config[0],
		Size:        size,
		Sample_type: 1 << 16,
		Read_format: unix.PERF_FORMAT_TOTAL_TIME_ENABLED | unix.PERF_FORMAT_TOTAL_TIME_RUNNING | unix.PERF_FORMAT_ID,
		Bits:        1<<20 | 1<<1,
	}
	if length >= 2 {
		config.Ext1 = event.Config[1]
	}
	if length == 3 {
		config.Ext2 = event.Config[2]
	}
	klog.V(4).Infof("perf_event_attr struct prepared: %#v", config)
	return config
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
