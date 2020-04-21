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
	"fmt"
	"time"

	"github.com/google/cadvisor/info/v1"
	"k8s.io/klog/v2"
)

func machineFsStatsFromV1(fsStats []v1.FsStats) []MachineFsStats {
	var result []MachineFsStats
	for i := range fsStats {
		stat := fsStats[i]
		readDuration := time.Millisecond * time.Duration(stat.ReadTime)
		writeDuration := time.Millisecond * time.Duration(stat.WriteTime)
		ioDuration := time.Millisecond * time.Duration(stat.IoTime)
		weightedDuration := time.Millisecond * time.Duration(stat.WeightedIoTime)
		machineFsStat := MachineFsStats{
			Device:    stat.Device,
			Type:      stat.Type,
			Capacity:  &stat.Limit,
			Usage:     &stat.Usage,
			Available: &stat.Available,
			DiskStats: DiskStats{
				ReadsCompleted:     &stat.ReadsCompleted,
				ReadsMerged:        &stat.ReadsMerged,
				SectorsRead:        &stat.SectorsRead,
				ReadDuration:       &readDuration,
				WritesCompleted:    &stat.WritesCompleted,
				WritesMerged:       &stat.WritesMerged,
				SectorsWritten:     &stat.SectorsWritten,
				WriteDuration:      &writeDuration,
				IoInProgress:       &stat.IoInProgress,
				IoDuration:         &ioDuration,
				WeightedIoDuration: &weightedDuration,
			},
		}
		if stat.HasInodes {
			machineFsStat.InodesFree = &stat.InodesFree
		}
		result = append(result, machineFsStat)
	}
	return result
}

func MachineStatsFromV1(cont *v1.ContainerInfo) []MachineStats {
	var stats []MachineStats
	var last *v1.ContainerStats
	for i := range cont.Stats {
		val := cont.Stats[i]
		stat := MachineStats{
			Timestamp: val.Timestamp,
		}
		if cont.Spec.HasCPU {
			stat.CPU = &val.CPU
			cpuInst, err := InstCPUStats(last, val)
			if err != nil {
				klog.Warningf("Could not get instant cpu stats: %v", err)
			} else {
				stat.CPUInst = cpuInst
			}
			last = val
		}
		if cont.Spec.HasMemory {
			stat.Memory = &val.Memory
		}
		if cont.Spec.HasNetwork {
			stat.Network = &NetworkStats{
				// FIXME: Use reflection instead.
				TCP:        TCPStat(val.Network.TCP),
				TCP6:       TCPStat(val.Network.TCP6),
				Interfaces: val.Network.Interfaces,
			}
		}
		if cont.Spec.HasFilesystem {
			stat.Filesystem = machineFsStatsFromV1(val.Filesystem)
		}
		// TODO(rjnagal): Handle load stats.
		stats = append(stats, stat)
	}
	return stats
}

func ContainerStatsFromV1(containerName string, spec *v1.ContainerSpec, stats []*v1.ContainerStats) []*ContainerStats {
	newStats := make([]*ContainerStats, 0, len(stats))
	var last *v1.ContainerStats
	for _, val := range stats {
		stat := &ContainerStats{
			Timestamp: val.Timestamp,
		}
		if spec.HasCPU {
			stat.CPU = &val.CPU
			cpuInst, err := InstCPUStats(last, val)
			if err != nil {
				klog.Warningf("Could not get instant cpu stats: %v", err)
			} else {
				stat.CPUInst = cpuInst
			}
			last = val
		}
		if spec.HasMemory {
			stat.Memory = &val.Memory
		}
		if spec.HasHugetlb {
			stat.Hugetlb = &val.Hugetlb
		}
		if spec.HasNetwork {
			// TODO: Handle TcpStats
			stat.Network = &NetworkStats{
				TCP:        TCPStat(val.Network.TCP),
				TCP6:       TCPStat(val.Network.TCP6),
				Interfaces: val.Network.Interfaces,
			}
		}
		if spec.HasProcesses {
			stat.Processes = &val.Processes
		}
		if spec.HasFilesystem {
			if len(val.Filesystem) == 1 {
				stat.Filesystem = &FilesystemStats{
					TotalUsageBytes: &val.Filesystem[0].Usage,
					BaseUsageBytes:  &val.Filesystem[0].BaseUsage,
					InodeUsage:      &val.Filesystem[0].Inodes,
				}
			} else if len(val.Filesystem) > 1 && containerName != "/" {
				// Cannot handle multiple devices per container.
				klog.V(4).Infof("failed to handle multiple devices for container %s. Skipping Filesystem stats", containerName)
			}
		}
		if spec.HasDiskIo {
			stat.DiskIo = &val.DiskIo
		}
		if spec.HasCustomMetrics {
			stat.CustomMetrics = val.CustomMetrics
		}
		if len(val.Accelerators) > 0 {
			stat.Accelerators = val.Accelerators
		}
		if len(val.PerfStats) > 0 {
			stat.PerfStats = val.PerfStats
		}
		// TODO(rjnagal): Handle load stats.
		newStats = append(newStats, stat)
	}
	return newStats
}

func DeprecatedStatsFromV1(cont *v1.ContainerInfo) []DeprecatedContainerStats {
	stats := make([]DeprecatedContainerStats, 0, len(cont.Stats))
	var last *v1.ContainerStats
	for _, val := range cont.Stats {
		stat := DeprecatedContainerStats{
			Timestamp:        val.Timestamp,
			HasCPU:           cont.Spec.HasCPU,
			HasMemory:        cont.Spec.HasMemory,
			HasNetwork:       cont.Spec.HasNetwork,
			HasFilesystem:    cont.Spec.HasFilesystem,
			HasDiskIo:        cont.Spec.HasDiskIo,
			HasCustomMetrics: cont.Spec.HasCustomMetrics,
		}
		if stat.HasCPU {
			stat.CPU = val.CPU
			cpuInst, err := InstCPUStats(last, val)
			if err != nil {
				klog.Warningf("Could not get instant cpu stats: %v", err)
			} else {
				stat.CPUInst = cpuInst
			}
			last = val
		}
		if stat.HasMemory {
			stat.Memory = val.Memory
		}
		if stat.HasNetwork {
			stat.Network.Interfaces = val.Network.Interfaces
		}
		if stat.HasProcesses {
			stat.Processes = val.Processes
		}
		if stat.HasFilesystem {
			stat.Filesystem = val.Filesystem
		}
		if stat.HasDiskIo {
			stat.DiskIo = val.DiskIo
		}
		if stat.HasCustomMetrics {
			stat.CustomMetrics = val.CustomMetrics
		}
		// TODO(rjnagal): Handle load stats.
		stats = append(stats, stat)
	}
	return stats
}

func InstCPUStats(last, cur *v1.ContainerStats) (*CPUInstStats, error) {
	if last == nil {
		return nil, nil
	}
	if !cur.Timestamp.After(last.Timestamp) {
		return nil, fmt.Errorf("container stats move backwards in time")
	}
	if len(last.CPU.Usage.PerCPU) != len(cur.CPU.Usage.PerCPU) {
		return nil, fmt.Errorf("different number of cpus")
	}
	timeDelta := cur.Timestamp.Sub(last.Timestamp)
	// Nanoseconds to gain precision and avoid having zero seconds if the
	// difference between the timestamps is just under a second
	timeDeltaNs := uint64(timeDelta.Nanoseconds())
	convertToRate := func(lastValue, curValue uint64) (uint64, error) {
		if curValue < lastValue {
			return 0, fmt.Errorf("cumulative stats decrease")
		}
		valueDelta := curValue - lastValue
		// Use float64 to keep precision
		return uint64(float64(valueDelta) / float64(timeDeltaNs) * 1e9), nil
	}
	total, err := convertToRate(last.CPU.Usage.Total, cur.CPU.Usage.Total)
	if err != nil {
		return nil, err
	}
	percpu := make([]uint64, len(last.CPU.Usage.PerCPU))
	for i := range percpu {
		var err error
		percpu[i], err = convertToRate(last.CPU.Usage.PerCPU[i], cur.CPU.Usage.PerCPU[i])
		if err != nil {
			return nil, err
		}
	}
	user, err := convertToRate(last.CPU.Usage.User, cur.CPU.Usage.User)
	if err != nil {
		return nil, err
	}
	system, err := convertToRate(last.CPU.Usage.System, cur.CPU.Usage.System)
	if err != nil {
		return nil, err
	}
	return &CPUInstStats{
		Usage: CPUInstUsage{
			Total:  total,
			PerCPU: percpu,
			User:   user,
			System: system,
		},
	}, nil
}

// Get V2 container spec from v1 container info.
func ContainerSpecFromV1(specV1 *v1.ContainerSpec, aliases []string, namespace string) ContainerSpec {
	specV2 := ContainerSpec{
		CreationTime:     specV1.CreationTime,
		HasCPU:           specV1.HasCPU,
		HasMemory:        specV1.HasMemory,
		HasHugetlb:       specV1.HasHugetlb,
		HasFilesystem:    specV1.HasFilesystem,
		HasNetwork:       specV1.HasNetwork,
		HasProcesses:     specV1.HasProcesses,
		HasDiskIo:        specV1.HasDiskIo,
		HasCustomMetrics: specV1.HasCustomMetrics,
		Image:            specV1.Image,
		Labels:           specV1.Labels,
		Envs:             specV1.Envs,
	}
	if specV1.HasCPU {
		specV2.CPU.Limit = specV1.CPU.Limit
		specV2.CPU.MaxLimit = specV1.CPU.MaxLimit
		specV2.CPU.Mask = specV1.CPU.Mask
	}
	if specV1.HasMemory {
		specV2.Memory.Limit = specV1.Memory.Limit
		specV2.Memory.Reservation = specV1.Memory.Reservation
		specV2.Memory.SwapLimit = specV1.Memory.SwapLimit
	}
	if specV1.HasCustomMetrics {
		specV2.CustomMetrics = specV1.CustomMetrics
	}
	specV2.Aliases = aliases
	specV2.Namespace = namespace
	return specV2
}
