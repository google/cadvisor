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

package container

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/watcher"

	"k8s.io/klog/v2"
)

type Factories = map[watcher.ContainerWatchSource][]ContainerHandlerFactory

type ContainerHandlerFactory interface {
	// Create a new ContainerHandler using this factory. CanHandleAndAccept() must have returned true.
	NewContainerHandler(name string, metadataEnvAllowList []string, inHostNamespace bool) (c ContainerHandler, err error)

	// Returns whether this factory can handle and accept the specified container.
	CanHandleAndAccept(name string) (handle bool, accept bool, err error)

	// Name of the factory.
	String() string

	// Returns debugging information. Map of lines per category.
	DebugInfo() map[string][]string
}

// MetricKind represents the kind of metrics that cAdvisor exposes.
type MetricKind string

const (
	CpuUsageMetrics                MetricKind = "cpu"
	ProcessSchedulerMetrics        MetricKind = "sched"
	PerCpuUsageMetrics             MetricKind = "percpu"
	MemoryUsageMetrics             MetricKind = "memory"
	MemoryNumaMetrics              MetricKind = "memory_numa"
	CpuLoadMetrics                 MetricKind = "cpuLoad"
	DiskIOMetrics                  MetricKind = "diskIO"
	DiskUsageMetrics               MetricKind = "disk"
	NetworkUsageMetrics            MetricKind = "network"
	NetworkTcpUsageMetrics         MetricKind = "tcp"
	NetworkAdvancedTcpUsageMetrics MetricKind = "advtcp"
	NetworkUdpUsageMetrics         MetricKind = "udp"
	AppMetrics                     MetricKind = "app"
	ProcessMetrics                 MetricKind = "process"
	HugetlbUsageMetrics            MetricKind = "hugetlb"
	PerfMetrics                    MetricKind = "perf_event"
	ReferencedMemoryMetrics        MetricKind = "referenced_memory"
	CPUTopologyMetrics             MetricKind = "cpu_topology"
	ResctrlMetrics                 MetricKind = "resctrl"
	CPUSetMetrics                  MetricKind = "cpuset"
	OOMMetrics                     MetricKind = "oom_event"
)

// AllMetrics represents all kinds of metrics that cAdvisor supported.
var AllMetrics = MetricSet{
	CpuUsageMetrics:                struct{}{},
	ProcessSchedulerMetrics:        struct{}{},
	PerCpuUsageMetrics:             struct{}{},
	MemoryUsageMetrics:             struct{}{},
	MemoryNumaMetrics:              struct{}{},
	CpuLoadMetrics:                 struct{}{},
	DiskIOMetrics:                  struct{}{},
	DiskUsageMetrics:               struct{}{},
	NetworkUsageMetrics:            struct{}{},
	NetworkTcpUsageMetrics:         struct{}{},
	NetworkAdvancedTcpUsageMetrics: struct{}{},
	NetworkUdpUsageMetrics:         struct{}{},
	ProcessMetrics:                 struct{}{},
	AppMetrics:                     struct{}{},
	HugetlbUsageMetrics:            struct{}{},
	PerfMetrics:                    struct{}{},
	ReferencedMemoryMetrics:        struct{}{},
	CPUTopologyMetrics:             struct{}{},
	ResctrlMetrics:                 struct{}{},
	CPUSetMetrics:                  struct{}{},
	OOMMetrics:                     struct{}{},
}

// AllNetworkMetrics represents all network metrics that cAdvisor supports.
var AllNetworkMetrics = MetricSet{
	NetworkUsageMetrics:            struct{}{},
	NetworkTcpUsageMetrics:         struct{}{},
	NetworkAdvancedTcpUsageMetrics: struct{}{},
	NetworkUdpUsageMetrics:         struct{}{},
}

func (mk MetricKind) String() string {
	return string(mk)
}

type MetricSet map[MetricKind]struct{}

func (ms MetricSet) Has(mk MetricKind) bool {
	_, exists := ms[mk]
	return exists
}

func (ms MetricSet) HasAny(ms1 MetricSet) bool {
	for m := range ms1 {
		if _, ok := ms[m]; ok {
			return true
		}
	}
	return false
}

func (ms MetricSet) add(mk MetricKind) {
	ms[mk] = struct{}{}
}

func (ms MetricSet) String() string {
	values := make([]string, 0, len(ms))
	for metric := range ms {
		values = append(values, string(metric))
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

// Not thread-safe, exported only for https://pkg.go.dev/flag#Value
func (ms *MetricSet) Set(value string) error {
	*ms = MetricSet{}
	if value == "" {
		return nil
	}
	for _, metric := range strings.Split(value, ",") {
		if AllMetrics.Has(MetricKind(metric)) {
			(*ms).add(MetricKind(metric))
		} else {
			return fmt.Errorf("unsupported metric %q specified", metric)
		}
	}
	return nil
}

func (ms MetricSet) Difference(ms1 MetricSet) MetricSet {
	result := MetricSet{}
	for kind := range ms {
		if !ms1.Has(kind) {
			result.add(kind)
		}
	}
	return result
}

func (ms MetricSet) Append(ms1 MetricSet) MetricSet {
	result := ms
	for kind := range ms1 {
		if !ms.Has(kind) {
			result.add(kind)
		}
	}
	return result
}

type plugins map[string]Plugin

type Plugin interface {
	// InitializeFSContext is invoked when populating an fs.Context object for a new manager.
	// A returned error here is fatal.
	InitializeFSContext(context *fs.Context) error

	// Register is invoked when starting a manager. A returned error is logged,
	// but is not fatal.
	Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics MetricSet) (Factories, error)
}

func InitializePlugins(factory info.MachineInfoFactory, plugins plugins, fsInfo fs.FsInfo, includedMetrics MetricSet) Factories {
	allFactories := make(Factories)

	for name, plugin := range plugins {
		factories, err := plugin.Register(factory, fsInfo, includedMetrics)
		if err != nil {
			klog.V(5).Infof("Registration of the %s container factory failed: %v", name, err)
		}

		for watchType, list := range factories {
			allFactories[watchType] = append(allFactories[watchType], list...)
		}
	}
	return allFactories
}

// Create a new ContainerHandler for the specified container.
func NewContainerHandler(factories Factories, name string, watchType watcher.ContainerWatchSource, metadataEnvAllowList []string, inHostNamespace bool) (ContainerHandler, bool, error) {
	// Create the ContainerHandler with the first factory that supports it.
	// Note that since RawContainerHandler can support a wide range of paths,
	// it's evaluated last just to make sure if any other ContainerHandler
	// can support it.
	for _, factory := range factories[watchType] {
		canHandle, canAccept, err := factory.CanHandleAndAccept(name)
		if err != nil {
			klog.V(4).Infof("Error trying to work out if we can handle %s: %v", name, err)
		}
		if canHandle {
			if !canAccept {
				klog.V(3).Infof("Factory %q can handle container %q, but ignoring.", factory, name)
				return nil, false, nil
			}
			klog.V(3).Infof("Using factory %q for container %q", factory, name)
			handle, err := factory.NewContainerHandler(name, metadataEnvAllowList, inHostNamespace)
			return handle, canAccept, err
		}
		klog.V(4).Infof("Factory %q was unable to handle container %q", factory, name)
	}

	return nil, false, fmt.Errorf("no known factory can handle creation of container")
}

func DebugInfo(factories Factories) map[string][]string {
	// Get debug information for all factories.
	out := make(map[string][]string)
	for _, factoriesSlice := range factories {
		for _, factory := range factoriesSlice {
			for k, v := range factory.DebugInfo() {
				out[k] = v
			}
		}
	}
	return out
}
