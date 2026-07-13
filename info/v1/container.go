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

package v1

import "github.com/google/cadvisor/lib/model"

type CpuSpec = model.CpuSpec

type MemorySpec = model.MemorySpec

type ProcessSpec = model.ProcessSpec

type ContainerSpec = model.ContainerSpec

// Container reference contains enough information to uniquely identify a container
type ContainerReference = model.ContainerReference

// Sorts by container name.
type ContainerReferenceSlice = model.ContainerReferenceSlice

// ContainerInfoRequest is used when users check a container info from the REST API.
// It specifies how much data users want to get about a container
type ContainerInfoRequest = model.ContainerInfoRequest

// Returns a ContainerInfoRequest with all default values specified.
func DefaultContainerInfoRequest() ContainerInfoRequest {
	return ContainerInfoRequest{
		NumStats: 60,
	}
}

type ContainerInfo = model.ContainerInfo

// PSI statistics for an individual resource.
type PSIStats = model.PSIStats

type PSIData = model.PSIData

// This mirrors kernel internal structure.
type LoadStats = model.LoadStats

// CPU usage time statistics.
type CpuUsage = model.CpuUsage

// Cpu Completely Fair Scheduler statistics.
type CpuCFS = model.CpuCFS

// Cpu Aggregated scheduler statistics
type CpuSchedstat = model.CpuSchedstat

// All CPU usage metrics are cumulative from the creation of the container
type CpuStats = model.CpuStats

type PerDiskStats = model.PerDiskStats

type DiskIoStats = model.DiskIoStats

type HugetlbStats = model.HugetlbStats

type MemoryStats = model.MemoryStats

type MemoryEvents = model.MemoryEvents

type CPUSetStats = model.CPUSetStats

type MemoryNumaStats = model.MemoryNumaStats

type MemoryStatsMemoryData = model.MemoryStatsMemoryData

type InterfaceStats = model.InterfaceStats

type NetworkStats = model.NetworkStats

type TcpStat = model.TcpStat

type TcpAdvancedStat = model.TcpAdvancedStat

type UdpStat = model.UdpStat

type FsStats = model.FsStats

type AcceleratorStats = model.AcceleratorStats

// PerfStat represents value of a single monitored perf event.
type PerfStat = model.PerfStat

type PerfValue = model.PerfValue

// MemoryBandwidthStats corresponds to MBM (Memory Bandwidth Monitoring).
// See: https://01.org/cache-monitoring-technology
// See: https://www.kernel.org/doc/Documentation/x86/intel_rdt_ui.txt
type MemoryBandwidthStats = model.MemoryBandwidthStats

// CacheStats corresponds to CMT (Cache Monitoring Technology).
// See: https://01.org/cache-monitoring-technology
// See: https://www.kernel.org/doc/Documentation/x86/intel_rdt_ui.txt
type CacheStats = model.CacheStats

// ResctrlStats corresponds to statistics from Resource Control.
type ResctrlStats = model.ResctrlStats

// PerfUncoreStat represents value of a single monitored perf uncore event.
type PerfUncoreStat = model.PerfUncoreStat

type UlimitSpec = model.UlimitSpec

type ProcessStats = model.ProcessStats

type Health = model.Health

type ContainerStats = model.ContainerStats

// Event contains information general to events such as the time at which they
// occurred, their specific type, and the actual event. Event types are
// differentiated by the EventType field of Event.
type Event = model.Event

// EventType is an enumerated type which lists the categories under which
// events may fall. The Event field EventType is populated by this enum.
type EventType = model.EventType

const (
	EventOom               EventType = "oom"
	EventOomKill           EventType = "oomKill"
	EventContainerCreation EventType = "containerCreation"
	EventContainerDeletion EventType = "containerDeletion"
)

// Extra information about an event. Only one type will be set.
type EventData = model.EventData

// Information related to an OOM kill instance
type OomKillEventData = model.OomKillEventData

// Information related to a container deletion event
type ContainerDeletionEventData = model.ContainerDeletionEventData
