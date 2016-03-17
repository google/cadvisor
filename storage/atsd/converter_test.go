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

package atsd

import (
	atsdNet "github.com/axibase/atsd-api-go/net"
	info "github.com/google/cadvisor/info/v1"
	"reflect"
	"testing"
	"time"
)

type MetaTestCase struct {
	ref               *info.ContainerReference
	props             []*atsdNet.PropertyCommand
	entityTagCommands []*atsdNet.EntityTagCommand
}

func TestRefToPropertyAndEntityTags(t *testing.T) {
	machineName := "hostname"
	timestamp := atsdNet.Millis(123456789)
	tests := []*MetaTestCase{
		{
			ref: &info.ContainerReference{
				Name: "/",
			},
			props: []*atsdNet.PropertyCommand{
				atsdNet.NewPropertyCommand("cadvisor", machineName+"/", containerHostTag, "true").
					SetTimestamp(timestamp).
					SetTag(containerIdTag, machineName).
					SetTag(containerNamespaceTag, "default"),
			},
			entityTagCommands: []*atsdNet.EntityTagCommand{
				atsdNet.NewEntityTagCommand(machineName+"/", containerHostTag, "true"),
			},
		},
		{
			ref: &info.ContainerReference{
				Name: "/docker/",
			},
			props: []*atsdNet.PropertyCommand{
				atsdNet.NewPropertyCommand("cadvisor", machineName+"/docker/", containerIdTag, "/").
					SetTimestamp(timestamp).
					SetTag(containerNamespaceTag, "docker"),
			},
		},
		{
			ref: &info.ContainerReference{
				Id:        "safvjmaw3o4",
				Name:      "/docker/safvjmaw3o4",
				Namespace: "docker",
				Aliases:   []string{"safvjmaw3o4", "my_container"},
			},
			props: []*atsdNet.PropertyCommand{
				atsdNet.NewPropertyCommand("cadvisor", machineName+"/docker/safvjmaw3o4", containerIdTag, "safvjmaw3o4").
					SetTimestamp(timestamp).
					SetTag(containerNamespaceTag, "docker").
					SetTag(containerAliasPropertyTag, "my_container"),
			},
			entityTagCommands: []*atsdNet.EntityTagCommand{
				atsdNet.NewEntityTagCommand(machineName+"/docker/safvjmaw3o4", containerIdTag, "safvjmaw3o4").
					SetTag(containerAliasEntityTag, "my_container").
					SetTag(containerNamespaceTag, "docker"),
			},
		},
		{
			ref: &info.ContainerReference{
				Name: "/lxc/",
			},
			props: []*atsdNet.PropertyCommand{
				atsdNet.NewPropertyCommand("cadvisor", machineName+"/lxc/", containerIdTag, "/").
					SetTimestamp(timestamp).
					SetTag(containerNamespaceTag, "lxc"),
			},
		},
		{
			ref: &info.ContainerReference{
				Name: "/lxc/araicmzs12",
			},
			props: []*atsdNet.PropertyCommand{
				atsdNet.NewPropertyCommand("cadvisor", machineName+"/lxc/araicmzs12", containerIdTag, "araicmzs12").
					SetTimestamp(timestamp).
					SetTag(containerNamespaceTag, "lxc"),
			},
		},
	}

	for i := range tests {
		ref := tests[i].ref
		assertProps := tests[i].props
		assertEntityTagCommands := tests[i].entityTagCommands

		props := RefToPropertyCommands(machineName, *ref, uint64(timestamp)*1e6)
		if !IsPropertiesEqual(props, assertProps) {
			t.Error("IsPropertiesEqual = false result props: ", props, "assert props: ", assertProps)
		}

		entityTagCommands := RefToEntityCommands(machineName, *ref)
		if !IsEntityTagCommandEquals(entityTagCommands, assertEntityTagCommands) {
			t.Error("IsEntityTagCommandEquals = false result entityTagCommands: ", entityTagCommands, "assert entityTagCommands: ", assertEntityTagCommands)
		}

	}
}

func IsPropertiesEqual(props1, props2 []*atsdNet.PropertyCommand) bool {
	return (len(props1) == 0 && len(props2) == 0) || reflect.DeepEqual(props1, props2)
}
func IsEntityTagCommandEquals(en1, en2 []*atsdNet.EntityTagCommand) bool {
	return (len(en1) == 0 && len(en2) == 0) || reflect.DeepEqual(en1, en2)
}

type DataTestCase struct {
	stats          *info.ContainerStats
	seriesCommands []*atsdNet.SeriesCommand
}

func TestStatsToSeries(t *testing.T) {
	machineName := "hostname"
	entityName := machineName + "/test-entity"
	testCases := []*DataTestCase{
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 123456789000000),
				Cpu: info.CpuStats{
					Usage: info.CpuUsage{
						Total:  123,
						PerCpu: []uint64{123, 123},
						User:   123,
						System: 123,
					},
					LoadAverage: 123000,
				},
				DiskIo: info.DiskIoStats{
					IoServiceBytes: []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
					IoServiced:     []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
					IoQueued:       []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
					Sectors:        []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
					IoServiceTime:  []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
					IoWaitTime:     []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
					IoMerged:       []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
					IoTime:         []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1}}},
				},
				Memory: info.MemoryStats{
					Usage:            123,
					Cache:            123,
					RSS:              123,
					WorkingSet:       123,
					ContainerData:    info.MemoryStatsMemoryData{Pgfault: 123, Pgmajfault: 123},
					HierarchicalData: info.MemoryStatsMemoryData{Pgfault: 123, Pgmajfault: 123},
					Failcnt:          123,
				},
				Network: info.NetworkStats{
					InterfaceStats: info.InterfaceStats{
						Name:      "eth",
						RxBytes:   123,
						RxPackets: 123,
						RxErrors:  123,
						RxDropped: 123,
						TxBytes:   123,
						TxPackets: 123,
						TxErrors:  123,
						TxDropped: 123,
					},
					Tcp: info.TcpStat{
						Established: 123,
						SynSent:     123,
						SynRecv:     123,
						FinWait1:    123,
						FinWait2:    123,
						TimeWait:    123,
						Close:       123,
						CloseWait:   123,
						LastAck:     123,
						Listen:      123,
						Closing:     123,
					},
					Tcp6: info.TcpStat{
						Established: 123,
						SynSent:     123,
						SynRecv:     123,
						FinWait1:    123,
						FinWait2:    123,
						TimeWait:    123,
						Close:       123,
						CloseWait:   123,
						LastAck:     123,
						Listen:      123,
						Closing:     123,
					},
					Interfaces: []info.InterfaceStats{},
				},
				Filesystem: []info.FsStats{
					{
						Device:          "sda",
						Type:            "ext4",
						Limit:           123,
						Usage:           123,
						BaseUsage:       123,
						Available:       123,
						InodesFree:      123,
						ReadsCompleted:  123,
						ReadsMerged:     123,
						SectorsRead:     123,
						ReadTime:        123,
						WritesCompleted: 123,
						WritesMerged:    123,
						SectorsWritten:  123,
						WriteTime:       123,
						IoInProgress:    123,
						IoTime:          123,
						WeightedIoTime:  123,
					},
				},
				TaskStats: info.LoadStats{
					NrSleeping:        123,
					NrRunning:         123,
					NrStopped:         123,
					NrUninterruptible: 123,
					NrIoWait:          123,
				},
			},
			seriesCommands: []*atsdNet.SeriesCommand{
				atsdNet.NewSeriesCommand(entityName, containerCpuUsageUser, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuUsageTotal, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuUsageSystem, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuLoadAverage, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuLoadAverage, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerMemoryWorkingSet, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryUsage, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryCache, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryRSS, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryHierarchicalDataPgfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryHierarchicalDataPgmajfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryContainerDataPgfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryContainerDataPgmajfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryFailcnt, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerTaskStatsNrIoWait, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrRunning, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrSleeping, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrStopped, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrUninterruptible, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerNetworkRxBytes, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkRxDropped, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkRxErrors, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkRxPackets, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxBytes, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxDropped, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxErrors, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxPackets, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatEstablished, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatSynSent, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatSynRecv, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatFinWait1, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatFinWait2, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatTimeWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatClose, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatCloseWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatLastAck, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatListen, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatClosing, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatEstablished, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatSynSent, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatSynRecv, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatFinWait1, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatFinWait2, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatTimeWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatClose, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatCloseWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatLastAck, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatListen, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatClosing, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerCpuUsagePerCpu, atsdNet.Uint64(123)).
					SetTag(cpu, "0").
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerCpuUsagePerCpu, atsdNet.Uint64(123)).
					SetTag(cpu, "1").
					SetTimestamp(atsdNet.Millis(123456789)),

				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoMerged+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoMerged+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoMerged+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoMerged+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoMerged+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoQueued+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoQueued+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoQueued+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoQueued+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoQueued+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceTime+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceTime+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceTime+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceTime+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceTime+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoTime+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoTime+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoTime+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoTime+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoTime+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoWaitTime+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoWaitTime+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoWaitTime+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoWaitTime+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoWaitTime+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoSectors+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoSectors+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoSectors+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoSectors+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoSectors+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),

				atsdNet.NewSeriesCommand(entityName, containerFilesystemIoInProgress, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemIoTime, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemLimit, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemReadsCompleted, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemReadsMerged, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemReadTime, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemSectorsRead, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemSectorsWritten, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemUsage, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemBaseUsage, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemInodesFree, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemWeightedIoTime, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemWritesCompleted, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemWritesMerged, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemWriteTime, atsdNet.Uint64(123)).
					SetMetricValue(containerFilesystemAvailable, atsdNet.Uint64(123)).
					SetTag(device, "sda").
					SetTag(fsType, "ext4").
					SetTimestamp(atsdNet.Millis(123456789)),
			},
		},
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 123456789000000),
			},
			seriesCommands: []*atsdNet.SeriesCommand{
				atsdNet.NewSeriesCommand(entityName, containerCpuUsageUser, atsdNet.Uint64(0)).
					SetMetricValue(containerCpuUsageTotal, atsdNet.Uint64(0)).
					SetMetricValue(containerCpuUsageSystem, atsdNet.Uint64(0)).
					SetMetricValue(containerCpuLoadAverage, atsdNet.Uint64(0)).
					SetMetricValue(containerCpuLoadAverage, atsdNet.Uint64(0)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerMemoryWorkingSet, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryUsage, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryCache, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryRSS, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryHierarchicalDataPgfault, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryHierarchicalDataPgmajfault, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryContainerDataPgfault, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryContainerDataPgmajfault, atsdNet.Uint64(0)).
					SetMetricValue(containerMemoryFailcnt, atsdNet.Uint64(0)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerTaskStatsNrIoWait, atsdNet.Uint64(0)).
					SetMetricValue(containerTaskStatsNrRunning, atsdNet.Uint64(0)).
					SetMetricValue(containerTaskStatsNrSleeping, atsdNet.Uint64(0)).
					SetMetricValue(containerTaskStatsNrStopped, atsdNet.Uint64(0)).
					SetMetricValue(containerTaskStatsNrUninterruptible, atsdNet.Uint64(0)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerNetworkRxBytes, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkRxDropped, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkRxErrors, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkRxPackets, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTxBytes, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTxDropped, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTxErrors, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTxPackets, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatEstablished, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatSynSent, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatSynRecv, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatFinWait1, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatFinWait2, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatTimeWait, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatClose, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatCloseWait, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatLastAck, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatListen, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcpStatClosing, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatEstablished, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatSynSent, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatSynRecv, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatFinWait1, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatFinWait2, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatTimeWait, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatClose, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatCloseWait, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatLastAck, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatListen, atsdNet.Uint64(0)).
					SetMetricValue(containerNetworkTcp6StatClosing, atsdNet.Uint64(0)).
					SetTimestamp(atsdNet.Millis(123456789)),
			},
		},
		{
			stats: &info.ContainerStats{
				Timestamp: time.Unix(0, 123456789000000),
				Cpu: info.CpuStats{
					Usage: info.CpuUsage{
						Total:  123,
						PerCpu: []uint64{},
						User:   123,
						System: 123,
					},
					LoadAverage: 123000,
				},
				DiskIo: info.DiskIoStats{
					IoServiceBytes: []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "sync": 1, "async": 1, "total": 1, "newtag": 1}}},
					IoServiced:     []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{"read": 1, "write": 1, "total": 1}}},
					IoQueued:       []info.PerDiskStats{{Major: 1, Minor: 2, Stats: map[string]uint64{}}},
					Sectors:        []info.PerDiskStats{},
					IoServiceTime:  []info.PerDiskStats{},
					IoWaitTime:     []info.PerDiskStats{},
					IoMerged:       []info.PerDiskStats{},
					IoTime:         []info.PerDiskStats{},
				},
				Memory: info.MemoryStats{
					Usage:            123,
					Cache:            123,
					RSS:              123,
					WorkingSet:       123,
					ContainerData:    info.MemoryStatsMemoryData{Pgfault: 123, Pgmajfault: 123},
					HierarchicalData: info.MemoryStatsMemoryData{Pgfault: 123, Pgmajfault: 123},
					Failcnt:          123,
				},
				Network: info.NetworkStats{
					InterfaceStats: info.InterfaceStats{
						Name:      "eth",
						RxBytes:   123,
						RxPackets: 123,
						RxErrors:  123,
						RxDropped: 123,
						TxBytes:   123,
						TxPackets: 123,
						TxErrors:  123,
						TxDropped: 123,
					},
					Tcp: info.TcpStat{
						Established: 123,
						SynSent:     123,
						SynRecv:     123,
						FinWait1:    123,
						FinWait2:    123,
						TimeWait:    123,
						Close:       123,
						CloseWait:   123,
						LastAck:     123,
						Listen:      123,
						Closing:     123,
					},
					Tcp6: info.TcpStat{
						Established: 123,
						SynSent:     123,
						SynRecv:     123,
						FinWait1:    123,
						FinWait2:    123,
						TimeWait:    123,
						Close:       123,
						CloseWait:   123,
						LastAck:     123,
						Listen:      123,
						Closing:     123,
					},
					Interfaces: []info.InterfaceStats{},
				},
				Filesystem: []info.FsStats{},
				TaskStats: info.LoadStats{
					NrSleeping:        123,
					NrRunning:         123,
					NrStopped:         123,
					NrUninterruptible: 123,
					NrIoWait:          123,
				},
			},
			seriesCommands: []*atsdNet.SeriesCommand{
				atsdNet.NewSeriesCommand(entityName, containerCpuUsageUser, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuUsageTotal, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuUsageSystem, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuLoadAverage, atsdNet.Uint64(123)).
					SetMetricValue(containerCpuLoadAverage, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerMemoryWorkingSet, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryUsage, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryCache, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryRSS, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryHierarchicalDataPgfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryHierarchicalDataPgmajfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryContainerDataPgfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryContainerDataPgmajfault, atsdNet.Uint64(123)).
					SetMetricValue(containerMemoryFailcnt, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerTaskStatsNrIoWait, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrRunning, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrSleeping, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrStopped, atsdNet.Uint64(123)).
					SetMetricValue(containerTaskStatsNrUninterruptible, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerNetworkRxBytes, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkRxDropped, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkRxErrors, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkRxPackets, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxBytes, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxDropped, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxErrors, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTxPackets, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatEstablished, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatSynSent, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatSynRecv, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatFinWait1, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatFinWait2, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatTimeWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatClose, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatCloseWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatLastAck, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatListen, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcpStatClosing, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatEstablished, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatSynSent, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatSynRecv, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatFinWait1, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatFinWait2, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatTimeWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatClose, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatCloseWait, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatLastAck, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatListen, atsdNet.Uint64(123)).
					SetMetricValue(containerNetworkTcp6StatClosing, atsdNet.Uint64(123)).
					SetTimestamp(atsdNet.Millis(123456789)),

				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"sync", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"async", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiceBytes+"."+"newtag", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"read", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"write", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
				atsdNet.NewSeriesCommand(entityName, containerDiskIoIoServiced+"."+"total", atsdNet.Uint64(1)).SetTag(major, "1").SetTag(minor, "2").SetTimestamp(atsdNet.Millis(123456789)),
			},
		},
	}

	for _, test := range testCases {
		ref := info.ContainerReference{Name: "/test-entity"}
		stats := test.stats
		assertSeriesCommands := test.seriesCommands

		seriesCommands := []*atsdNet.SeriesCommand{}
		seriesCommands = append(seriesCommands, CpuSeriesCommandsFromStats(machineName, ref, stats)...)
		seriesCommands = append(seriesCommands, MemorySeriesCommandsFromStats(machineName, ref, stats)...)
		seriesCommands = append(seriesCommands, TaskSeriesCommandsFromStats(machineName, ref, stats)...)
		seriesCommands = append(seriesCommands, NetworkSeriesCommandsFromStats(machineName, ref, stats)...)
		seriesCommands = append(seriesCommands, IOSeriesCommandsFromStats(machineName, ref, true, stats)...)
		seriesCommands = append(seriesCommands, FileSystemSeriesCommandsFromStats(machineName, ref, stats)...)

		if len(assertSeriesCommands) != len(seriesCommands) {
			t.Error("Wrong series command count: ", len(seriesCommands), "!=", len(assertSeriesCommands), " series: ", seriesCommands, "assert series: ", assertSeriesCommands)
		}

		for _, comm1 := range assertSeriesCommands {
			found := false
			for _, comm2 := range seriesCommands {
				if reflect.DeepEqual(comm1, comm2) {
					found = true
				}
			}
			if !found {
				t.Error("Series command not found: ", comm1, "series: ", seriesCommands)
			}
		}

	}
}
