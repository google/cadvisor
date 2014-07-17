package libcontainer

import (
	"time"

	"github.com/docker/libcontainer/cgroups"
	"github.com/docker/libcontainer/cgroups/fs"
	"github.com/docker/libcontainer/cgroups/systemd"
	"github.com/google/cadvisor/info"
)

// Get stats of the specified cgroup
func GetStats(cgroup *cgroups.Cgroup, useSystemd bool) (*info.ContainerStats, error) {
	// TODO(vmarmol): Use libcontainer's Stats() in the new API when that is ready.
	// Use systemd paths if systemd is being used.
	var (
		s   *cgroups.Stats
		err error
	)
	if useSystemd {
		s, err = systemd.GetStats(cgroup)
	} else {
		s, err = fs.GetStats(cgroup)
	}
	if err != nil {
		return nil, err
	}
	return toContainerStats(s), nil
}

// Convert libcontainer stats to info.ContainerStats.
func toContainerStats(s *cgroups.Stats) *info.ContainerStats {
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
	return ret
}
