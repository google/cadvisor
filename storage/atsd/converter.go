// Copyright 2016 Google Inc. All Rights Reserved.
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

package atsd

import (
	"strconv"
	"strings"
	"time"

	atsdNet "github.com/axibase/atsd-api-go/net"
	info "github.com/google/cadvisor/info/v1"
)

const (
	containerCpuUsageUser   = "cadvisor.cpu.usage.user"
	containerCpuUsageTotal  = "cadvisor.cpu.usage.total"
	containerCpuUsageSystem = "cadvisor.cpu.usage.system"
	containerCpuLoadAverage = "cadvisor.cpu.loadaverage"
	containerCpuUsagePerCpu = "cadvisor.cpu.usage.percpu"

	containerMemoryWorkingSet                 = "cadvisor.memory.workingset"
	containerMemoryUsage                      = "cadvisor.memory.usage"
	containerMemoryCache                      = "cadvisor.memory.cache"
	containerMemoryRSS                        = "cadvisor.memory.rss"
	containerMemoryHierarchicalDataPgfault    = "cadvisor.memory.hierarchicaldata.pgfault"
	containerMemoryHierarchicalDataPgmajfault = "cadvisor.memory.hierarchicaldata.pgmajfault"
	containerMemoryContainerDataPgfault       = "cadvisor.memory.containerdata.pgfault"
	containerMemoryContainerDataPgmajfault    = "cadvisor.memory.containerdata.pgmajfault"
	containerMemoryFailcnt                    = "cadvisor.memory.failcnt"

	containerNetworkRxBytes   = "cadvisor.network.rxbytes"
	containerNetworkRxDropped = "cadvisor.network.rxdropped"
	containerNetworkRxErrors  = "cadvisor.network.rxerrors"
	containerNetworkRxPackets = "cadvisor.network.rxpackets"
	containerNetworkTxBytes   = "cadvisor.network.txbytes"
	containerNetworkTxDropped = "cadvisor.network.txdropped"
	containerNetworkTxErrors  = "cadvisor.network.txerrors"
	containerNetworkTxPackets = "cadvisor.network.txpackets"

	containerNetworkTcpStatEstablished = "cadvisor.network.tcp.established"
	containerNetworkTcpStatSynSent     = "cadvisor.network.tcp.synsent"
	containerNetworkTcpStatSynRecv     = "cadvisor.network.tcp.synrecv"
	containerNetworkTcpStatFinWait1    = "cadvisor.network.tcp.finwait1"
	containerNetworkTcpStatFinWait2    = "cadvisor.network.tcp.finwait2"
	containerNetworkTcpStatTimeWait    = "cadvisor.network.tcp.timewait"
	containerNetworkTcpStatClose       = "cadvisor.network.tcp.close"
	containerNetworkTcpStatCloseWait   = "cadvisor.network.tcp.closewait"
	containerNetworkTcpStatLastAck     = "cadvisor.network.tcp.lastack"
	containerNetworkTcpStatListen      = "cadvisor.network.tcp.listen"
	containerNetworkTcpStatClosing     = "cadvisor.network.tcp.closing"

	containerNetworkTcp6StatEstablished = "cadvisor.network.tcp6.established"
	containerNetworkTcp6StatSynSent     = "cadvisor.network.tcp6.synsent"
	containerNetworkTcp6StatSynRecv     = "cadvisor.network.tcp6.synrecv"
	containerNetworkTcp6StatFinWait1    = "cadvisor.network.tcp6.finwait1"
	containerNetworkTcp6StatFinWait2    = "cadvisor.network.tcp6.finwait2"
	containerNetworkTcp6StatTimeWait    = "cadvisor.network.tcp6.timewait"
	containerNetworkTcp6StatClose       = "cadvisor.network.tcp6.close"
	containerNetworkTcp6StatCloseWait   = "cadvisor.network.tcp6.closewait"
	containerNetworkTcp6StatLastAck     = "cadvisor.network.tcp6.lastack"
	containerNetworkTcp6StatListen      = "cadvisor.network.tcp6.listen"
	containerNetworkTcp6StatClosing     = "cadvisor.network.tcp6.closing"

	containerTaskStatsNrIoWait          = "cadvisor.taskstats.nriowait"
	containerTaskStatsNrRunning         = "cadvisor.taskstats.nrrunning"
	containerTaskStatsNrSleeping        = "cadvisor.taskstats.nrsleeping"
	containerTaskStatsNrStopped         = "cadvisor.taskstats.nrstopped"
	containerTaskStatsNrUninterruptible = "cadvisor.taskstats.nruninterruptible"

	containerDiskIoIoMerged       = "cadvisor.diskio.iomerged"
	containerDiskIoIoQueued       = "cadvisor.diskio.ioqueued"
	containerDiskIoIoServiceBytes = "cadvisor.diskio.ioservicebytes"
	containerDiskIoIoServiced     = "cadvisor.diskio.ioserviced"
	containerDiskIoIoServiceTime  = "cadvisor.diskio.ioservicetime"
	containerDiskIoIoTime         = "cadvisor.diskio.iotime"
	containerDiskIoIoWaitTime     = "cadvisor.diskio.iowaittime"
	containerDiskIoSectors        = "cadvisor.diskio.sectors"

	containerFilesystemIoInProgress    = "cadvisor.filesystem.ioinprogress"
	containerFilesystemIoTime          = "cadvisor.filesystem.iotime"
	containerFilesystemLimit           = "cadvisor.filesystem.limit"
	containerFilesystemReadsCompleted  = "cadvisor.filesystem.readscompleted"
	containerFilesystemReadsMerged     = "cadvisor.filesystem.readsmerged"
	containerFilesystemReadTime        = "cadvisor.filesystem.readtime"
	containerFilesystemSectorsRead     = "cadvisor.filesystem.sectorsread"
	containerFilesystemSectorsWritten  = "cadvisor.filesystem.sectorswritten"
	containerFilesystemUsage           = "cadvisor.filesystem.usage"
	containerFilesystemBaseUsage       = "cadvisor.filesystem.baseusage"
	containerFilesystemAvailable       = "cadvisor.filesystem.available"
	containerFilesystemInodesFree      = "cadvisor.filesystem.inodesfree"
	containerFilesystemWeightedIoTime  = "cadvisor.filesystem.weightediotime"
	containerFilesystemWritesCompleted = "cadvisor.filesystem.writescompleted"
	containerFilesystemWritesMerged    = "cadvisor.filesystem.writesmerged"
	containerFilesystemWriteTime       = "cadvisor.filesystem.writetime"
)

const (
	propertyType = "cadvisor"

	containerIdTag            = "container_id"
	containerAliasPropertyTag = "container_alias"
	containerAliasEntityTag   = "alias"
	containerAliasTagPrefix   = "container_alias."
	containerNamespaceTag     = "container_namespace"
	containerHostTag          = "container_host"
)

const (
	containerLabelTagPrefix = "container_label."
)

// Tags
const (
	device        = "device"
	fsType        = "type"
	minor         = "minor"
	major         = "major"
	disk          = "disk"
	cpu           = "cpu"
	interfaceName = "name"
)

func CpuSeriesCommandsFromStats(machineName string, ref info.ContainerReference, stats *info.ContainerStats) []*atsdNet.SeriesCommand {
	entity := machineName + ref.Name
	seriesCommands := []*atsdNet.SeriesCommand{
		atsdNet.NewSeriesCommand(entity, containerCpuUsageUser, atsdNet.Uint64(stats.Cpu.Usage.User)).
			SetMetricValue(containerCpuUsageTotal, atsdNet.Uint64(stats.Cpu.Usage.Total)).
			SetMetricValue(containerCpuUsageSystem, atsdNet.Uint64(stats.Cpu.Usage.System)).
			SetMetricValue(containerCpuLoadAverage, atsdNet.Uint64(stats.Cpu.LoadAverage/1e3)), // taking into account load average representation
	}
	for index, perCpu := range stats.Cpu.Usage.PerCpu {
		seriesCommands = append(seriesCommands, atsdNet.NewSeriesCommand(entity, containerCpuUsagePerCpu, atsdNet.Uint64(perCpu)).
			SetTag(cpu, strconv.FormatInt(int64(index), 10)))
	}

	setSeriesTimestamp(seriesCommands, stats.Timestamp)

	return seriesCommands
}
func IOSeriesCommandsFromStats(machineName string, ref info.ContainerReference, includeAllMajorNumbers bool, stats *info.ContainerStats) []*atsdNet.SeriesCommand {
	entity := machineName + ref.Name
	seriesCommands := []*atsdNet.SeriesCommand{}
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoIoMerged, includeAllMajorNumbers, &stats.DiskIo.IoMerged)...)
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoIoQueued, includeAllMajorNumbers, &stats.DiskIo.IoQueued)...)
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoIoServiceBytes, includeAllMajorNumbers, &stats.DiskIo.IoServiceBytes)...)
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoIoServiced, includeAllMajorNumbers, &stats.DiskIo.IoServiced)...)
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoIoServiceTime, includeAllMajorNumbers, &stats.DiskIo.IoServiceTime)...)
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoIoTime, includeAllMajorNumbers, &stats.DiskIo.IoTime)...)
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoIoWaitTime, includeAllMajorNumbers, &stats.DiskIo.IoWaitTime)...)
	seriesCommands = append(seriesCommands, diskIoStatsToSeriesCommands(entity, containerDiskIoSectors, includeAllMajorNumbers, &stats.DiskIo.Sectors)...)

	setSeriesTimestamp(seriesCommands, stats.Timestamp)

	return seriesCommands
}
func MemorySeriesCommandsFromStats(machineName string, ref info.ContainerReference, stats *info.ContainerStats) []*atsdNet.SeriesCommand {
	entity := machineName + ref.Name

	seriesCommands := []*atsdNet.SeriesCommand{
		atsdNet.NewSeriesCommand(entity, containerMemoryWorkingSet, atsdNet.Uint64(stats.Memory.WorkingSet)).
			SetMetricValue(containerMemoryUsage, atsdNet.Uint64(stats.Memory.Usage)).
			SetMetricValue(containerMemoryCache, atsdNet.Uint64(stats.Memory.Cache)).
			SetMetricValue(containerMemoryRSS, atsdNet.Uint64(stats.Memory.RSS)).
			SetMetricValue(containerMemoryHierarchicalDataPgfault, atsdNet.Uint64(stats.Memory.HierarchicalData.Pgfault)).
			SetMetricValue(containerMemoryHierarchicalDataPgmajfault, atsdNet.Uint64(stats.Memory.HierarchicalData.Pgmajfault)).
			SetMetricValue(containerMemoryContainerDataPgfault, atsdNet.Uint64(stats.Memory.ContainerData.Pgfault)).
			SetMetricValue(containerMemoryContainerDataPgmajfault, atsdNet.Uint64(stats.Memory.ContainerData.Pgmajfault)).
			SetMetricValue(containerMemoryFailcnt, atsdNet.Uint64(stats.Memory.Failcnt)),
	}

	setSeriesTimestamp(seriesCommands, stats.Timestamp)

	return seriesCommands
}
func NetworkSeriesCommandsFromStats(machineName string, ref info.ContainerReference, stats *info.ContainerStats) []*atsdNet.SeriesCommand {
	entity := machineName + ref.Name
	seriesCommands := []*atsdNet.SeriesCommand{
		atsdNet.NewSeriesCommand(entity, containerNetworkTcpStatEstablished, atsdNet.Uint64(stats.Network.Tcp.Established)).
			SetMetricValue(containerNetworkTcpStatSynSent, atsdNet.Uint64(stats.Network.Tcp.SynSent)).
			SetMetricValue(containerNetworkTcpStatSynRecv, atsdNet.Uint64(stats.Network.Tcp.SynRecv)).
			SetMetricValue(containerNetworkTcpStatFinWait1, atsdNet.Uint64(stats.Network.Tcp.FinWait1)).
			SetMetricValue(containerNetworkTcpStatFinWait2, atsdNet.Uint64(stats.Network.Tcp.FinWait2)).
			SetMetricValue(containerNetworkTcpStatTimeWait, atsdNet.Uint64(stats.Network.Tcp.TimeWait)).
			SetMetricValue(containerNetworkTcpStatClose, atsdNet.Uint64(stats.Network.Tcp.Close)).
			SetMetricValue(containerNetworkTcpStatCloseWait, atsdNet.Uint64(stats.Network.Tcp.CloseWait)).
			SetMetricValue(containerNetworkTcpStatLastAck, atsdNet.Uint64(stats.Network.Tcp.LastAck)).
			SetMetricValue(containerNetworkTcpStatListen, atsdNet.Uint64(stats.Network.Tcp.Listen)).
			SetMetricValue(containerNetworkTcpStatClosing, atsdNet.Uint64(stats.Network.Tcp.Closing)).
			SetMetricValue(containerNetworkTcp6StatEstablished, atsdNet.Uint64(stats.Network.Tcp6.Established)).
			SetMetricValue(containerNetworkTcp6StatSynSent, atsdNet.Uint64(stats.Network.Tcp6.SynSent)).
			SetMetricValue(containerNetworkTcp6StatSynRecv, atsdNet.Uint64(stats.Network.Tcp6.SynRecv)).
			SetMetricValue(containerNetworkTcp6StatFinWait1, atsdNet.Uint64(stats.Network.Tcp6.FinWait1)).
			SetMetricValue(containerNetworkTcp6StatFinWait2, atsdNet.Uint64(stats.Network.Tcp6.FinWait2)).
			SetMetricValue(containerNetworkTcp6StatTimeWait, atsdNet.Uint64(stats.Network.Tcp6.TimeWait)).
			SetMetricValue(containerNetworkTcp6StatClose, atsdNet.Uint64(stats.Network.Tcp6.Close)).
			SetMetricValue(containerNetworkTcp6StatCloseWait, atsdNet.Uint64(stats.Network.Tcp6.CloseWait)).
			SetMetricValue(containerNetworkTcp6StatLastAck, atsdNet.Uint64(stats.Network.Tcp6.LastAck)).
			SetMetricValue(containerNetworkTcp6StatListen, atsdNet.Uint64(stats.Network.Tcp6.Listen)).
			SetMetricValue(containerNetworkTcp6StatClosing, atsdNet.Uint64(stats.Network.Tcp6.Closing)),
	}

	for _, networkInterface := range stats.Network.Interfaces {
		seriesCommands = append(seriesCommands,
			atsdNet.NewSeriesCommand(entity, containerNetworkRxBytes, atsdNet.Uint64(networkInterface.RxBytes)).
				SetMetricValue(containerNetworkRxDropped, atsdNet.Uint64(networkInterface.RxDropped)).
				SetMetricValue(containerNetworkRxErrors, atsdNet.Uint64(networkInterface.RxErrors)).
				SetMetricValue(containerNetworkRxPackets, atsdNet.Uint64(networkInterface.RxPackets)).
				SetMetricValue(containerNetworkTxBytes, atsdNet.Uint64(networkInterface.TxBytes)).
				SetMetricValue(containerNetworkTxDropped, atsdNet.Uint64(networkInterface.TxDropped)).
				SetMetricValue(containerNetworkTxErrors, atsdNet.Uint64(networkInterface.TxErrors)).
				SetMetricValue(containerNetworkTxPackets, atsdNet.Uint64(networkInterface.TxPackets)).
				SetTag(interfaceName, networkInterface.Name))
	}

	setSeriesTimestamp(seriesCommands, stats.Timestamp)

	return seriesCommands
}
func TaskSeriesCommandsFromStats(machineName string, ref info.ContainerReference, stats *info.ContainerStats) []*atsdNet.SeriesCommand {
	entity := machineName + ref.Name

	seriesCommands := []*atsdNet.SeriesCommand{
		atsdNet.NewSeriesCommand(entity, containerTaskStatsNrIoWait, atsdNet.Uint64(stats.TaskStats.NrIoWait)).
			SetMetricValue(containerTaskStatsNrRunning, atsdNet.Uint64(stats.TaskStats.NrRunning)).
			SetMetricValue(containerTaskStatsNrSleeping, atsdNet.Uint64(stats.TaskStats.NrSleeping)).
			SetMetricValue(containerTaskStatsNrStopped, atsdNet.Uint64(stats.TaskStats.NrStopped)).
			SetMetricValue(containerTaskStatsNrUninterruptible, atsdNet.Uint64(stats.TaskStats.NrUninterruptible)),
	}

	setSeriesTimestamp(seriesCommands, stats.Timestamp)

	return seriesCommands
}
func FileSystemSeriesCommandsFromStats(machineName string, ref info.ContainerReference, stats *info.ContainerStats) []*atsdNet.SeriesCommand {
	entity := machineName + ref.Name

	seriesCommands := []*atsdNet.SeriesCommand{}

	for _, fsStats := range stats.Filesystem {
		seriesCommands = append(seriesCommands, atsdNet.NewSeriesCommand(entity, containerFilesystemIoInProgress, atsdNet.Uint64(fsStats.IoInProgress)).
			SetMetricValue(containerFilesystemIoTime, atsdNet.Uint64(fsStats.IoTime)).
			SetMetricValue(containerFilesystemLimit, atsdNet.Uint64(fsStats.Limit)).
			SetMetricValue(containerFilesystemReadsCompleted, atsdNet.Uint64(fsStats.ReadsCompleted)).
			SetMetricValue(containerFilesystemReadsMerged, atsdNet.Uint64(fsStats.ReadsMerged)).
			SetMetricValue(containerFilesystemReadTime, atsdNet.Uint64(fsStats.ReadTime)).
			SetMetricValue(containerFilesystemSectorsRead, atsdNet.Uint64(fsStats.SectorsRead)).
			SetMetricValue(containerFilesystemSectorsWritten, atsdNet.Uint64(fsStats.SectorsWritten)).
			SetMetricValue(containerFilesystemUsage, atsdNet.Uint64(fsStats.Usage)).
			SetMetricValue(containerFilesystemBaseUsage, atsdNet.Uint64(fsStats.BaseUsage)).
			SetMetricValue(containerFilesystemWeightedIoTime, atsdNet.Uint64(fsStats.WeightedIoTime)).
			SetMetricValue(containerFilesystemWritesCompleted, atsdNet.Uint64(fsStats.WritesCompleted)).
			SetMetricValue(containerFilesystemWritesMerged, atsdNet.Uint64(fsStats.WritesMerged)).
			SetMetricValue(containerFilesystemWriteTime, atsdNet.Uint64(fsStats.WriteTime)).
			SetMetricValue(containerFilesystemAvailable, atsdNet.Uint64(fsStats.Available)).
			SetMetricValue(containerFilesystemInodesFree, atsdNet.Uint64(fsStats.InodesFree)).
			SetTag(device, fsStats.Device).
			SetTag(fsType, fsStats.Type))
	}

	setSeriesTimestamp(seriesCommands, stats.Timestamp)

	return seriesCommands
}

func setSeriesTimestamp(seriesCommands []*atsdNet.SeriesCommand, timestamp time.Time) {
	for _, c := range seriesCommands {
		time := uint64(timestamp.UnixNano() / time.Millisecond.Nanoseconds())
		c.SetTimestamp(atsdNet.Millis(time))
	}
}

func RefToPropertyCommands(machineName string, ref info.ContainerReference, timestamp time.Time) []*atsdNet.PropertyCommand {
	entity := machineName + ref.Name

	var propertyCommand *atsdNet.PropertyCommand

	aliases := make([]string, len(ref.Aliases))
	copy(aliases, ref.Aliases)

	if ref.Namespace == "" {
		if strings.HasPrefix(ref.Name, "/user") {
			propertyCommand = atsdNet.NewPropertyCommand(propertyType, entity, containerNamespaceTag, "user")
			newAlias := strings.Replace(ref.Name, "/user", "", 1)
			if newAlias = strings.Replace(newAlias, "/", "", 1); newAlias != "" {
				aliases = append(aliases, newAlias)
			}
		} else if strings.HasPrefix(ref.Name, "/docker") {
			propertyCommand = atsdNet.NewPropertyCommand(propertyType, entity, containerNamespaceTag, "docker")
			newAlias := strings.Replace(ref.Name, "/docker", "", 1)
			if newAlias = strings.Replace(newAlias, "/", "", 1); newAlias != "" {
				aliases = append(aliases, newAlias)
			} else {
				aliases = append(aliases, "/")
			}
		} else if strings.HasPrefix(ref.Name, "/lxc") {
			propertyCommand = atsdNet.NewPropertyCommand(propertyType, entity, containerNamespaceTag, "lxc")
			newAlias := strings.Replace(ref.Name, "/lxc", "", 1)
			if newAlias = strings.Replace(newAlias, "/", "", 1); newAlias != "" {
				aliases = append(aliases, newAlias)
			} else {
				aliases = append(aliases, "/")
			}
		} else if ref.Name == "/" {
			propertyCommand = atsdNet.NewPropertyCommand(propertyType, entity, containerHostTag, "true")
			aliases = append(aliases, machineName)
			propertyCommand.SetTag(containerNamespaceTag, "default")
		} else {
			aliases = append(aliases, ref.Name)
			propertyCommand = atsdNet.NewPropertyCommand(propertyType, entity, containerNamespaceTag, "default")
		}
	} else {
		propertyCommand = atsdNet.NewPropertyCommand(propertyType, entity, containerNamespaceTag, ref.Namespace)
	}
	if len(aliases) > 0 {
		count := 0
		for i := range aliases {
			if "/"+ref.Namespace+"/"+aliases[i] == ref.Name || len(aliases) == 1 {
				propertyCommand.SetTag(containerIdTag, aliases[i])
			} else {
				if count == 0 {
					propertyCommand.SetTag(containerAliasPropertyTag, aliases[i])
				} else {
					propertyCommand.SetTag(containerAliasTagPrefix+strconv.FormatInt(int64(count), 10), aliases[i])
				}
				count++
			}
		}
	}

	for key, val := range ref.Labels {
		propertyCommand.SetTag(containerLabelTagPrefix+key, val)
	}
	propertyCommand.SetTimestamp(atsdNet.Millis(timestamp.UnixNano() / time.Millisecond.Nanoseconds()))
	return []*atsdNet.PropertyCommand{propertyCommand}
}
func RefToEntityCommands(machineName string, ref info.ContainerReference) []*atsdNet.EntityTagCommand {
	tags := map[string]string{}
	if ref.Id != "" {
		tags[containerIdTag] = ref.Id
	}

	if ref.Name == "/" {
		tags[containerHostTag] = "true"
	}

	if ref.Namespace != "" {
		tags[containerNamespaceTag] = ref.Namespace
	}
	for _, a := range ref.Aliases {
		// is container alias
		if a != ref.Id {
			tags[containerAliasEntityTag] = a
			break
		}
	}
	for labelName, labelValue := range ref.Labels {
		tags[containerLabelTagPrefix+labelName] = labelValue
	}

	var entity *atsdNet.EntityTagCommand
	for key, val := range tags {
		if entity == nil {
			entity = atsdNet.NewEntityTagCommand(machineName+ref.Name, key, val)
		} else {
			entity.SetTag(key, val)
		}
	}

	if entity != nil {
		return []*atsdNet.EntityTagCommand{entity}
	} else {
		return []*atsdNet.EntityTagCommand{}
	}
}

func diskIoStatsToSeriesCommands(entity, metric string, includeAllMajorNumbers bool, stats *[]info.PerDiskStats) []*atsdNet.SeriesCommand {
	seriesCommands := []*atsdNet.SeriesCommand{}
	for _, perDiskStats := range *stats {
		if includeAllMajorNumbers || (perDiskStats.Major != 1) && (perDiskStats.Major != 2) && (perDiskStats.Major != 7) {
			for diskName, stat := range perDiskStats.Stats {
				seriesCommands = append(seriesCommands, atsdNet.NewSeriesCommand(entity, metric+"."+diskName, atsdNet.Uint64(stat)).
					SetTag(major, strconv.FormatUint(perDiskStats.Major, 10)).
					SetTag(minor, strconv.FormatUint(perDiskStats.Minor, 10)))
			}
		}
	}
	return seriesCommands
}
