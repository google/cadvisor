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

package v2

import (
	"time"

	// TODO(rjnagal): Remove dependency after moving all stats structs from v1.
	// using v1 now for easy conversion.
	v1 "github.com/google/cadvisor/info/v1"
	model "github.com/google/cadvisor/lib/model"
)

const (
	TypeName   = "name"
	TypeDocker = "docker"
	TypePodman = "podman"
)

type CpuSpec struct {
	// Requested cpu shares. Default is 1024.
	Limit uint64 `json:"limit"`
	// Requested cpu hard limit. Default is unlimited (0).
	// Units: milli-cpus.
	MaxLimit uint64 `json:"max_limit"`
	// Cpu affinity mask.
	// TODO(rjnagal): Add a library to convert mask string to set of cpu bitmask.
	Mask string `json:"mask,omitempty"`
	// CPUQuota Default is disabled
	Quota uint64 `json:"quota,omitempty"`
	// Period is the CPU reference time in ns e.g the quota is compared against this.
	Period uint64 `json:"period,omitempty"`
}

type MemorySpec struct {
	// The amount of memory requested. Default is unlimited (-1).
	// Units: bytes.
	Limit uint64 `json:"limit,omitempty"`

	// The amount of guaranteed memory.  Default is 0.
	// Units: bytes.
	Reservation uint64 `json:"reservation,omitempty"`

	// The amount of swap space requested. Default is unlimited (-1).
	// Units: bytes.
	SwapLimit uint64 `json:"swap_limit,omitempty"`
}

type ContainerInfo struct {
	// Describes the container.
	Spec ContainerSpec `json:"spec,omitempty"`

	// Historical statistics gathered from the container.
	Stats []*ContainerStats `json:"stats,omitempty"`
}

type ContainerSpec struct {
	// Time at which the container was created.
	CreationTime time.Time `json:"creation_time,omitempty"`

	// Time at which the container was started.
	// This may be unset if the runtime does not provide it.
	StartTime time.Time `json:"start_time,omitempty"`

	// Other names by which the container is known within a certain namespace.
	// This is unique within that namespace.
	Aliases []string `json:"aliases,omitempty"`

	// Namespace under which the aliases of a container are unique.
	// An example of a namespace is "docker" for Docker containers.
	Namespace string `json:"namespace,omitempty"`

	// Metadata labels associated with this container.
	Labels map[string]string `json:"labels,omitempty"`
	// Metadata envs associated with this container. Only whitelisted envs are added.
	Envs map[string]string `json:"envs,omitempty"`

	HasCpu bool    `json:"has_cpu"`
	Cpu    CpuSpec `json:"cpu,omitempty"`

	HasMemory bool       `json:"has_memory"`
	Memory    MemorySpec `json:"memory,omitempty"`

	HasHugetlb bool `json:"has_hugetlb"`

	HasCustomMetrics bool            `json:"has_custom_metrics"`
	CustomMetrics    []v1.MetricSpec `json:"custom_metrics,omitempty"`

	HasProcesses bool           `json:"has_processes"`
	Processes    v1.ProcessSpec `json:"processes,omitempty"`

	// Following resources have no associated spec, but are being isolated.
	HasNetwork    bool `json:"has_network"`
	HasFilesystem bool `json:"has_filesystem"`
	HasDiskIo     bool `json:"has_diskio"`

	// Image name used for this container.
	Image string `json:"image,omitempty"`
}

type DeprecatedContainerStats struct {
	// The time of this stat point.
	Timestamp time.Time `json:"timestamp"`
	// CPU statistics
	HasCpu bool `json:"has_cpu"`
	// In nanoseconds (aggregated)
	Cpu v1.CpuStats `json:"cpu,omitempty"`
	// In nanocores per second (instantaneous)
	CpuInst *CpuInstStats `json:"cpu_inst,omitempty"`
	// Disk IO statistics
	HasDiskIo bool           `json:"has_diskio"`
	DiskIo    v1.DiskIoStats `json:"diskio,omitempty"`
	// Memory statistics
	HasMemory bool           `json:"has_memory"`
	Memory    v1.MemoryStats `json:"memory,omitempty"`
	// Hugepage statistics
	HasHugetlb bool                       `json:"has_hugetlb"`
	Hugetlb    map[string]v1.HugetlbStats `json:"hugetlb,omitempty"`
	// Network statistics
	HasNetwork bool         `json:"has_network"`
	Network    NetworkStats `json:"network,omitempty"`
	// Processes statistics
	HasProcesses bool            `json:"has_processes"`
	Processes    v1.ProcessStats `json:"processes,omitempty"`
	// Filesystem statistics
	HasFilesystem bool         `json:"has_filesystem"`
	Filesystem    []v1.FsStats `json:"filesystem,omitempty"`
	// Task load statistics
	HasLoad bool         `json:"has_load"`
	Load    v1.LoadStats `json:"load_stats,omitempty"`
	// Custom Metrics
	HasCustomMetrics bool                      `json:"has_custom_metrics"`
	CustomMetrics    map[string][]v1.MetricVal `json:"custom_metrics,omitempty"`
	// Perf events counters
	PerfStats []v1.PerfStat `json:"perf_stats,omitempty"`
	// Statistics originating from perf uncore events.
	// Applies only for root container.
	PerfUncoreStats []v1.PerfUncoreStat `json:"perf_uncore_stats,omitempty"`
	// Referenced memory
	ReferencedMemory uint64 `json:"referenced_memory,omitempty"`
	// Resource Control (resctrl) statistics
	Resctrl v1.ResctrlStats `json:"resctrl,omitempty"`
}

type ContainerStats struct {
	// The time of this stat point.
	Timestamp time.Time `json:"timestamp"`
	// CPU statistics
	// In nanoseconds (aggregated)
	Cpu *v1.CpuStats `json:"cpu,omitempty"`
	// In nanocores per second (instantaneous)
	CpuInst *CpuInstStats `json:"cpu_inst,omitempty"`
	// Disk IO statistics
	DiskIo *v1.DiskIoStats `json:"diskio,omitempty"`
	// Memory statistics
	Memory *v1.MemoryStats `json:"memory,omitempty"`
	// Hugepage statistics
	Hugetlb *map[string]v1.HugetlbStats `json:"hugetlb,omitempty"`
	// Network statistics
	Network *NetworkStats `json:"network,omitempty"`
	// Processes statistics
	Processes *v1.ProcessStats `json:"processes,omitempty"`
	// Filesystem statistics
	Filesystem *FilesystemStats `json:"filesystem,omitempty"`
	// Task load statistics
	Load *v1.LoadStats `json:"load_stats,omitempty"`
	// Metrics for Accelerators. Each Accelerator corresponds to one element in the array.
	Accelerators []v1.AcceleratorStats `json:"accelerators,omitempty"`
	// Custom Metrics
	CustomMetrics map[string][]v1.MetricVal `json:"custom_metrics,omitempty"`
	// Perf events counters
	PerfStats []v1.PerfStat `json:"perf_stats,omitempty"`
	// Statistics originating from perf uncore events.
	// Applies only for root container.
	PerfUncoreStats []v1.PerfUncoreStat `json:"perf_uncore_stats,omitempty"`
	// Referenced memory
	ReferencedMemory uint64 `json:"referenced_memory,omitempty"`
	// Resource Control (resctrl) statistics
	Resctrl v1.ResctrlStats `json:"resctrl,omitempty"`
}

// Percentiles, Usage, InstantUsage and DerivedStats are summary/derived-stats
// types defined in the library so the manager can return them; identical shape.
type Percentiles = model.Percentiles

type Usage = model.Usage

type InstantUsage = model.InstantUsage

type DerivedStats = model.DerivedStats

// FsInfo (runtime filesystem stats) is identical to the lean library's
// model.FsInfo; alias it so manager methods can return the library
// type and REST handlers consume it as v2.FsInfo with no conversion.
type FsInfo = model.FsInfo

// RequestOptions is identical to the library's model.RequestOptions; alias it
// so the library manager's query methods accept the same value the REST
// handlers build.
type RequestOptions = model.RequestOptions

// ProcessInfo is defined in the library (model) so the manager can return it.
type ProcessInfo = model.ProcessInfo

type TcpStat struct {
	Established uint64
	SynSent     uint64
	SynRecv     uint64
	FinWait1    uint64
	FinWait2    uint64
	TimeWait    uint64
	Close       uint64
	CloseWait   uint64
	LastAck     uint64
	Listen      uint64
	Closing     uint64
}

type NetworkStats struct {
	// Network stats by interface.
	Interfaces []v1.InterfaceStats `json:"interfaces,omitempty"`
	// TCP connection stats (Established, Listen...)
	Tcp TcpStat `json:"tcp"`
	// TCP6 connection stats (Established, Listen...)
	Tcp6 TcpStat `json:"tcp6"`
	// UDP connection stats
	Udp v1.UdpStat `json:"udp"`
	// UDP6 connection stats
	Udp6 v1.UdpStat `json:"udp6"`
	// TCP advanced stats
	TcpAdvanced v1.TcpAdvancedStat `json:"tcp_advanced"`
}

// Instantaneous CPU stats
type CpuInstStats struct {
	Usage CpuInstUsage `json:"usage"`
}

// CPU usage time statistics.
type CpuInstUsage struct {
	// Total CPU usage.
	// Units: nanocores per second
	Total uint64 `json:"total"`

	// Per CPU/core usage of the container.
	// Unit: nanocores per second
	PerCpu []uint64 `json:"per_cpu_usage,omitempty"`

	// Time spent in user space.
	// Unit: nanocores per second
	User uint64 `json:"user"`

	// Time spent in kernel space.
	// Unit: nanocores per second
	System uint64 `json:"system"`
}

// Filesystem usage statistics.
type FilesystemStats struct {
	// Total Number of bytes consumed by container.
	TotalUsageBytes *uint64 `json:"totalUsageBytes,omitempty"`
	// Number of bytes consumed by a container through its root filesystem.
	BaseUsageBytes *uint64 `json:"baseUsageBytes,omitempty"`
	// Number of inodes used within the container's root filesystem.
	// This only accounts for inodes that are shared across containers,
	// and does not include inodes used in mounted directories.
	InodeUsage *uint64 `json:"containter_inode_usage,omitempty"`
}
