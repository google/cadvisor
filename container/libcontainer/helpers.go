package libcontainer

import (
	"time"

	"github.com/docker/libcontainer"
	"github.com/google/cadvisor/info"
)

// Get stats of the specified cgroup
func GetStats(config *libcontainer.Config, state *libcontainer.State) (*info.ContainerStats, error) {
	// TODO(vmarmol): Use libcontainer's Stats() in the new API when that is ready.
	libcontainerStats, err := libcontainer.GetStats(config, state)
	if err != nil {
		return nil, err
	}
	return toContainerStats(libcontainerStats), nil
}

// Convert libcontainer stats to info.ContainerStats.
func toContainerStats(libcontainerStats *libcontainer.ContainerStats) *info.ContainerStats {
	s := libcontainerStats.CgroupStats
	ret := new(info.ContainerStats)
	ret.Timestamp = time.Now()
	ret.Cpu = new(info.CpuStats)
	ret.Cpu.Usage.User = s.CpuStats.CpuUsage.UsageInUsermode
	ret.Cpu.Usage.System = s.CpuStats.CpuUsage.UsageInKernelmode
	n := len(s.CpuStats.CpuUsage.PercpuUsage)
	ret.Cpu.Usage.PerCpu = make([]uint64, n)

	ret.Cpu.Usage.Total = 0
	for i := 0; i < n; i++ {
		ret.Cpu.Usage.PerCpu[i] = s.CpuStats.CpuUsage.PercpuUsage[i]
		ret.Cpu.Usage.Total += s.CpuStats.CpuUsage.PercpuUsage[i]
	}
	ret.Memory = new(info.MemoryStats)
	ret.Memory.Usage = s.MemoryStats.Usage
	if v, ok := s.MemoryStats.Stats["pgfault"]; ok {
		ret.Memory.ContainerData.Pgfault = v
		ret.Memory.HierarchicalData.Pgfault = v
	}
	if v, ok := s.MemoryStats.Stats["pgmajfault"]; ok {
		ret.Memory.ContainerData.Pgmajfault = v
		ret.Memory.HierarchicalData.Pgmajfault = v
	}
	if v, ok := s.MemoryStats.Stats["total_inactive_anon"]; ok {
		ret.Memory.WorkingSet = ret.Memory.Usage - v
		if v, ok := s.MemoryStats.Stats["total_active_file"]; ok {
			ret.Memory.WorkingSet -= v
		}
	}
	ret.Network = (*info.NetworkStats)(&libcontainerStats.NetworkStats)
	return ret
}
