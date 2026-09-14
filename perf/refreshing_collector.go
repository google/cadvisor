//go:build libpfm && cgo

// Copyright 2026 Google Inc. All Rights Reserved.
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

package perf

import (
	"fmt"
	"os"
	"sync"
	"syscall"

	"k8s.io/klog/v2"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/lib/stats"
)

// cgroupIdentity distinguishes kernel cgroup instances that use the same path.
type cgroupIdentity struct {
	device uint64
	inode  uint64
}

// collectorFactory returns a collector and the identity to which it is bound.
type collectorFactory func(string) (stats.Collector, cgroupIdentity, error)

// refreshingCollector replaces perf file descriptors when a cgroup path is recreated.
type refreshingCollector struct {
	cgroupPath string
	identity   cgroupIdentity
	collector  stats.Collector
	factory    collectorFactory
	lock       sync.Mutex
}

func newRefreshingCollector(cgroupPath string, factory collectorFactory) (stats.Collector, error) {
	// Capture the identity established by the factory rather than statting again,
	// so the recorded identity always matches the cgroup used to open the perf FDs.
	collector, identity, err := factory(cgroupPath)
	if err != nil {
		return &stats.NoopCollector{}, err
	}
	return &refreshingCollector{
		cgroupPath: cgroupPath,
		identity:   identity,
		collector:  collector,
		factory:    factory,
	}, nil
}

func getCgroupIdentity(path string) (cgroupIdentity, error) {
	// os.Stat resolves the path on every call, unlike the long-lived perf FDs.
	fileInfo, err := os.Stat(path)
	if err != nil {
		return cgroupIdentity{}, err
	}
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return cgroupIdentity{}, fmt.Errorf("unable to read device and inode for cgroup %q", path)
	}
	return cgroupIdentity{device: uint64(stat.Dev), inode: stat.Ino}, nil
}

func (c *refreshingCollector) UpdateStats(containerStats *info.ContainerStats) error {
	if containerStats == nil {
		return fmt.Errorf("containerStats cannot be nil")
	}
	// Serialize reads, replacement, and destruction of the underlying perf FDs.
	c.lock.Lock()
	defer c.lock.Unlock()

	identity, err := getCgroupIdentity(c.cgroupPath)
	if err != nil {
		// The cgroup may be between deletion and recreation. Do not publish data
		// from descriptors that can still refer to the deleted cgroup.
		containerStats.PerfStats = nil
		return fmt.Errorf("unable to identify perf cgroup %q: %w", c.cgroupPath, err)
	}
	if identity != c.identity {
		// Build the replacement before destroying the old collector. A setup
		// failure then remains retryable on the next housekeeping cycle.
		collector, collectorIdentity, err := c.factory(c.cgroupPath)
		if err != nil {
			containerStats.PerfStats = nil
			return fmt.Errorf("unable to refresh perf collector for cgroup %q: %w", c.cgroupPath, err)
		}
		oldIdentity := c.identity
		// The factory verifies that collectorIdentity remained stable while it
		// opened the replacement descriptors.
		c.collector.Destroy()
		c.collector = collector
		c.identity = collectorIdentity
		klog.Infof("Refreshed perf collector for cgroup %q after identity changed from device=%d inode=%d to device=%d inode=%d", c.cgroupPath, oldIdentity.device, oldIdentity.inode, collectorIdentity.device, collectorIdentity.inode)
	}

	return c.collector.UpdateStats(containerStats)
}

func (c *refreshingCollector) Destroy() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.collector.Destroy()
}
