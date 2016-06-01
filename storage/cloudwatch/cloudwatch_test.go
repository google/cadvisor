package cloudwatch

import (
	"testing"
	info "github.com/google/cadvisor/info/v1"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"time"
)

func TestBatchCreation(t *testing.T) {
	storage := &cloudWatchStorage{machineName: "localtest"}
	metrics := storage.collectMetrics(ref, &stats)

	test := func(limit int, expectedSize int) {
		currentMetrics := metrics[:limit]

		batches := storage.createBatches(currentMetrics)
		if len(batches) != expectedSize {
			t.Errorf("Batches number is not %v but %v", expectedSize, len(batches))
		}
	}
	test(22, 2)
	test(20, 1)
	test(2, 1)
}

func TestValidMetrics(t *testing.T) {
	storage := &cloudWatchStorage{machineName: "localtest"}
	metrics := storage.collectMetrics(ref, &stats)
	for _, m := range metrics {
		if !standardMetric(*m.Unit) {
			t.Errorf("%v is not standard metric", *m.Unit)
		}
		for _, d := range m.Dimensions {
			if *d.Value == "" {
				t.Errorf("%s dimenstion has empty value", *d.Name)
			}
		}
	}
}

func standardMetric(unit string) bool {
	return func(element string, array []string) bool {
		for _, s := range array {
			if element == s {
				return true
			}
		}
		return false
	}(unit, standardMetrics)
}

var (
	ref = info.ContainerReference{Id:"", Name:"/", Aliases:[]string(nil), Namespace:"testspace", Labels:map[string]string(nil)}

	stats = info.ContainerStats{
		Timestamp: time.Now(),
		Cpu: info.CpuStats{
			Usage: info.CpuUsage{
				Total: 0x2fd4864ef,
				PerCpu:[]uint64{0x2fd4864ef,
				},
				User: 0x9df3ca80,
				System: 0x1aac4ee00,
			},
			LoadAverage:0,
		}, DiskIo: info.DiskIoStats{
			IoServiceBytes: []info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats: map[string]uint64{
						"Async":0x6358000,
						"Total":0x64d6400,
						"Read":0x62b3000,
						"Write":0x223400,
						"Sync":0x17e400,
					},
				},
			}, IoServiced: []info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats: map[string]uint64{
						"Write":0xcc,
						"Sync":0x4d,
						"Async":0xdf1,
						"Total":0xe3e,
						"Read":0xd72,
					},
				},
			}, IoQueued: []info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats: map[string]uint64{
						"Read":0x0,
						"Write":0x0,
						"Sync":0x0,
						"Async":0x0,
						"Total":0x0,
					},
				},
			}, Sectors: []info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats:map[string]uint64{
						"Count":0x326b2,
					},
				},
			}, IoServiceTime:[]info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats:map[string]uint64{
						"Sync":0x1c9f3f1,
						"Async":0x5e453043,
						"Total":0x600f2434,
						"Read":0x5b23c740,
						"Write":0x4eb5cf4,
					},
				},
			}, IoWaitTime:[]info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats:map[string]uint64{
						"Total":0x137b1cb92,
						"Read":0x11f236200,
						"Write":0x188e6992,
						"Sync":0x23c6aaa,
						"Async":0x1357560e8,
					},
				},
			}, IoMerged:[]info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats:map[string]uint64{
						"Read":0x670,
						"Write":0x3,
						"Sync":0x0,
						"Async":0x673,
						"Total":0x673,
					},
				},
			}, IoTime:[]info.PerDiskStats{
				info.PerDiskStats{
					Major:0x8,
					Minor:0x0,
					Stats:map[string]uint64{
						"Count":0x1411,
					},
				},
			},
		}, Memory:info.MemoryStats{
			Usage:0x9957000,
			Cache:0x61e2000,
			RSS:0x3775000,
			WorkingSet:0x4fda000,
			Failcnt:0x0,
			ContainerData:info.MemoryStatsMemoryData{
				Pgfault:0xf56b2,
				Pgmajfault:0x232,
			}, HierarchicalData:info.MemoryStatsMemoryData{
				Pgfault:0xf56b2,
				Pgmajfault:0x232,
			},
		}, Network:info.NetworkStats{
			InterfaceStats:info.InterfaceStats{
				Name:"eth0", RxBytes:0x5fa57,
				RxPackets:0xfad,
				RxErrors:0x0,
				RxDropped:0x0,
				TxBytes:0x69920,
				TxPackets:0x131d,
				TxErrors:0x0,
				TxDropped:0x0,
			}, Interfaces:[]info.InterfaceStats{
				info.InterfaceStats{
					Name:"eth0",
					RxBytes:0x5fa57,
					RxPackets:0xfad,
					RxErrors:0x0,
					RxDropped:0x0,
					TxBytes:0x69920,
					TxPackets:0x131d,
					TxErrors:0x0,
					TxDropped:0x0,
				},
			}, Tcp:info.TcpStat{
				Established:0x0,
				SynSent:0x0,
				SynRecv:0x0,
				FinWait1:0x0,
				FinWait2:0x0,
				TimeWait:0x0,
				Close:0x0,
				CloseWait:0x0,
				LastAck:0x0,
				Listen:0x0,
				Closing:0x0,
			}, Tcp6:info.TcpStat{
				Established:0x0,
				SynSent:0x0,
				SynRecv:0x0,
				FinWait1:0x0,
				FinWait2:0x0,
				TimeWait:0x0,
				Close:0x0,
				CloseWait:0x0,
				LastAck:0x0,
				Listen:0x0,
				Closing:0x0,
			},
		}, Filesystem:[]info.FsStats{
			info.FsStats{
				Device:"/dev/mapper/VolGroup00-LogVol00",
				Type:"vfs",
				Limit:0x94fd58000,
				Usage:0x58df9000,
				BaseUsage:0x0,
				Available:0x87c75f000,
				InodesFree:0x253c27,
				ReadsCompleted:0x0,
				ReadsMerged:0x0,
				SectorsRead:0x0,
				ReadTime:0x0,
				WritesCompleted:0x0,
				WritesMerged:0x0,
				SectorsWritten:0x0,
				WriteTime:0x0,
				IoInProgress:0x0,
				IoTime:0x0,
				WeightedIoTime:0x0,
			}, info.FsStats{
				Device:"/dev/sda2",
				Type:"vfs",
				Limit:0xbdaec00,
				Usage:0x9f70800,
				BaseUsage:0x0,
				Available:0x103e400,
				InodesFree:0xc6a6,
				ReadsCompleted:0xcd,
				ReadsMerged:0x25,
				SectorsRead:0xefe,
				ReadTime:0x64,
				WritesCompleted:0x3,
				WritesMerged:0x0,
				SectorsWritten:0x12,
				WriteTime:0x3,
				IoInProgress:0x0,
				IoTime:0x63,
				WeightedIoTime:0x67,
			},
		}, TaskStats:info.LoadStats{
			NrSleeping:0x0,
			NrRunning:0x0,
			NrStopped:0x0,
			NrUninterruptible:0x0,
			NrIoWait:0x0,
		}, CustomMetrics:map[string][]info.MetricVal(nil)}
	standardMetrics = []string{
		cloudwatch.StandardUnitSeconds,
		cloudwatch.StandardUnitMicroseconds,
		cloudwatch.StandardUnitMilliseconds,
		cloudwatch.StandardUnitBytes,
		cloudwatch.StandardUnitKilobytes,
		cloudwatch.StandardUnitMegabytes,
		cloudwatch.StandardUnitGigabytes,
		cloudwatch.StandardUnitTerabytes,
		cloudwatch.StandardUnitBits,
		cloudwatch.StandardUnitKilobits,
		cloudwatch.StandardUnitMegabits,
		cloudwatch.StandardUnitGigabits,
		cloudwatch.StandardUnitTerabits,
		cloudwatch.StandardUnitPercent,
		cloudwatch.StandardUnitCount,
		cloudwatch.StandardUnitBytesSecond,
		cloudwatch.StandardUnitKilobytesSecond,
		cloudwatch.StandardUnitMegabytesSecond,
		cloudwatch.StandardUnitGigabytesSecond,
		cloudwatch.StandardUnitTerabytesSecond,
		cloudwatch.StandardUnitBitsSecond,
		cloudwatch.StandardUnitKilobitsSecond,
		cloudwatch.StandardUnitMegabitsSecond,
		cloudwatch.StandardUnitGigabitsSecond,
		cloudwatch.StandardUnitTerabitsSecond,
		cloudwatch.StandardUnitCountSecond,
		cloudwatch.StandardUnitNone,
	}
)