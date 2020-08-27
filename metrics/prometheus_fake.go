// Copyright 2020 Google Inc. All Rights Reserved.
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

package metrics

import (
	"errors"
	"time"

	info "github.com/google/cadvisor/info/v1"
	v2 "github.com/google/cadvisor/info/v2"
)

type testSubcontainersInfoProvider struct{}

func (p testSubcontainersInfoProvider) GetVersionInfo() (*info.VersionInfo, error) {
	return &info.VersionInfo{
		KernelVersion:      "4.1.6-200.fc22.x86_64",
		ContainerOsVersion: "Fedora 22 (Twenty Two)",
		DockerVersion:      "1.8.1",
		CadvisorVersion:    "0.16.0",
		CadvisorRevision:   "abcdef",
	}, nil
}

func (p testSubcontainersInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	return &info.MachineInfo{
		Timestamp:        time.Unix(1395066363, 0),
		NumCores:         4,
		NumPhysicalCores: 1,
		NumSockets:       1,
		MemoryCapacity:   1024,
		MemoryByType: map[string]*info.MemoryInfo{
			"Non-volatile-RAM": {Capacity: 2168421613568, DimmCount: 8},
			"Unbuffered-DDR4":  {Capacity: 412316860416, DimmCount: 12},
		},
		NVMInfo: info.NVMInfo{
			MemoryModeCapacity:    429496729600,
			AppDirectModeCapacity: 1735166787584,
		},
		MachineID:  "machine-id-test",
		SystemUUID: "system-uuid-test",
		BootID:     "boot-id-test",
		Topology: []info.Node{
			{
				Id:     0,
				Memory: 33604804608,
				HugePages: []info.HugePagesInfo{
					{
						PageSize: uint64(1048576),
						NumPages: uint64(0),
					},
					{
						PageSize: uint64(2048),
						NumPages: uint64(0),
					},
				},
				Cores: []info.Core{
					{
						Id:      0,
						Threads: []int{0, 1},
						Caches: []info.Cache{
							{
								Size:  32768,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32768,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262144,
								Type:  "Unified",
								Level: 2,
							},
						},
					},
					{
						Id:      1,
						Threads: []int{2, 3},
						Caches: []info.Cache{
							{
								Size:  32764,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32764,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262148,
								Type:  "Unified",
								Level: 2,
							},
						},
					},

					{
						Id:      2,
						Threads: []int{4, 5},
						Caches: []info.Cache{
							{
								Size:  32768,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32768,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262144,
								Type:  "Unified",
								Level: 2,
							},
						},
					},
					{
						Id:      3,
						Threads: []int{6, 7},
						Caches: []info.Cache{
							{
								Size:  32764,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32764,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262148,
								Type:  "Unified",
								Level: 2,
							},
						},
					},
				},
			},
			{
				Id:     1,
				Memory: 33604804606,
				HugePages: []info.HugePagesInfo{
					{
						PageSize: uint64(1048576),
						NumPages: uint64(2),
					},
					{
						PageSize: uint64(2048),
						NumPages: uint64(4),
					},
				},
				Cores: []info.Core{
					{
						Id:      4,
						Threads: []int{8, 9},
						Caches: []info.Cache{
							{
								Size:  32768,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32768,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262144,
								Type:  "Unified",
								Level: 2,
							},
						},
					},
					{
						Id:      5,
						Threads: []int{10, 11},
						Caches: []info.Cache{
							{
								Size:  32764,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32764,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262148,
								Type:  "Unified",
								Level: 2,
							},
						},
					},
					{
						Id:      6,
						Threads: []int{12, 13},
						Caches: []info.Cache{
							{
								Size:  32768,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32768,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262144,
								Type:  "Unified",
								Level: 2,
							},
						},
					},
					{
						Id:      7,
						Threads: []int{14, 15},
						Caches: []info.Cache{
							{
								Size:  32764,
								Type:  "Data",
								Level: 1,
							},
							{
								Size:  32764,
								Type:  "Instruction",
								Level: 1,
							},
							{
								Size:  262148,
								Type:  "Unified",
								Level: 2,
							},
						},
					},
				},
				Caches: []info.Cache{
					{
						Size:  8388608,
						Type:  "Unified",
						Level: 3,
					},
				},
			},
		},
		VMStats: map[string]int{
			"nr_free_pages":                  4952941,
			"nr_zone_inactive_anon":          125562,
			"nr_zone_active_anon":            1195557,
			"nr_zone_inactive_file":          849631,
			"nr_zone_active_file":            614049,
			"nr_zone_unevictable":            227082,
			"nr_zone_write_pending":          422,
			"nr_mlock":                       12,
			"nr_page_table_pages":            16564,
			"nr_kernel_stack":                27088,
			"nr_bounce":                      0,
			"nr_zspages":                     0,
			"nr_free_cma":                    0,
			"numa_hit":                       69010884,
			"numa_miss":                      0,
			"numa_foreign":                   0,
			"numa_interleave":                23063,
			"numa_local":                     69010884,
			"numa_other":                     0,
			"nr_inactive_anon":               125562,
			"nr_active_anon":                 1195557,
			"nr_inactive_file":               849631,
			"nr_active_file":                 614049,
			"nr_unevictable":                 227082,
			"nr_slab_reclaimable":            110541,
			"nr_slab_unreclaimable":          67078,
			"nr_isolated_anon":               0,
			"nr_isolated_file":               0,
			"workingset_nodes":               0,
			"workingset_refault":             0,
			"workingset_activate":            0,
			"workingset_restore":             0,
			"workingset_nodereclaim":         0,
			"nr_anon_pages":                  1232379,
			"nr_mapped":                      538193,
			"nr_file_pages":                  1779511,
			"nr_dirty":                       422,
			"nr_writeback":                   0,
			"nr_writeback_temp":              0,
			"nr_shmem":                       353284,
			"nr_shmem_hugepages":             0,
			"nr_shmem_pmdmapped":             0,
			"nr_file_hugepages":              0,
			"nr_file_pmdmapped":              0,
			"nr_anon_transparent_hugepages":  19,
			"nr_unstable":                    0,
			"nr_vmscan_write":                0,
			"nr_vmscan_immediate_reclaim":    0,
			"nr_dirtied":                     865602,
			"nr_written":                     862703,
			"nr_kernel_misc_reclaimable":     0,
			"nr_foll_pin_acquired":           2,
			"nr_foll_pin_released":           2,
			"nr_dirty_threshold":             1269594,
			"nr_dirty_background_threshold":  634021,
			"pgpgin":                         4249504,
			"pgpgout":                        4135862,
			"pswpin":                         0,
			"pswpout":                        0,
			"pgalloc_dma":                    0,
			"pgalloc_dma32":                  1,
			"pgalloc_normal":                 72049150,
			"pgalloc_movable":                0,
			"allocstall_dma":                 0,
			"allocstall_dma32":               0,
			"allocstall_normal":              0,
			"allocstall_movable":             0,
			"pgskip_dma":                     0,
			"pgskip_dma32":                   0,
			"pgskip_normal":                  0,
			"pgskip_movable":                 0,
			"pgfree":                         77004776,
			"pgactivate":                     2669465,
			"pgdeactivate":                   0,
			"pglazyfree":                     286137,
			"pgfault":                        63773170,
			"pgmajfault":                     18630,
			"pglazyfreed":                    0,
			"pgrefill":                       0,
			"pgsteal_kswapd":                 0,
			"pgsteal_direct":                 0,
			"pgscan_kswapd":                  0,
			"pgscan_direct":                  0,
			"pgscan_direct_throttle":         0,
			"zone_reclaim_failed":            0,
			"pginodesteal":                   0,
			"slabs_scanned":                  0,
			"kswapd_inodesteal":              0,
			"kswapd_low_wmark_hit_quickly":   0,
			"kswapd_high_wmark_hit_quickly":  0,
			"pageoutrun":                     0,
			"pgrotated":                      107,
			"drop_pagecache":                 0,
			"drop_slab":                      0,
			"oom_kill":                       0,
			"numa_pte_updates":               0,
			"numa_huge_pte_updates":          0,
			"numa_hint_faults":               0,
			"numa_hint_faults_local":         0,
			"numa_pages_migrated":            0,
			"pgmigrate_success":              0,
			"pgmigrate_fail":                 0,
			"compact_migrate_scanned":        0,
			"compact_free_scanned":           0,
			"compact_isolated":               0,
			"compact_stall":                  0,
			"compact_fail":                   0,
			"compact_success":                0,
			"compact_daemon_wake":            0,
			"compact_daemon_migrate_scanned": 0,
			"compact_daemon_free_scanned":    0,
			"htlb_buddy_alloc_success":       0,
			"htlb_buddy_alloc_fail":          0,
			"unevictable_pgs_culled":         5733159,
			"unevictable_pgs_scanned":        6611168,
			"unevictable_pgs_rescued":        5485862,
			"unevictable_pgs_mlocked":        270726,
			"unevictable_pgs_munlocked":      270714,
			"unevictable_pgs_cleared":        0,
			"unevictable_pgs_stranded":       0,
			"thp_fault_alloc":                541,
			"thp_fault_fallback":             0,
			"thp_fault_fallback_charge":      0,
			"thp_collapse_alloc":             5,
			"thp_collapse_alloc_failed":      0,
			"thp_file_alloc":                 0,
			"thp_file_fallback":              0,
			"thp_file_fallback_charge":       0,
			"thp_file_mapped":                0,
			"thp_split_page":                 14,
			"thp_split_page_failed":          0,
			"thp_deferred_split_page":        14,
			"thp_split_pmd":                  14,
			"thp_split_pud":                  0,
			"thp_zero_page_alloc":            3,
			"thp_zero_page_alloc_failed":     0,
			"thp_swpout":                     0,
			"thp_swpout_fallback":            0,
			"balloon_inflate":                0,
			"balloon_deflate":                0,
			"balloon_migrate":                0,
			"swap_ra":                        0,
			"swap_ra_hit":                    0,
		},
	}, nil
}

func (p testSubcontainersInfoProvider) GetRequestedContainersInfo(string, v2.RequestOptions) (map[string]*info.ContainerInfo, error) {
	return map[string]*info.ContainerInfo{
		"testcontainer": {
			ContainerReference: info.ContainerReference{
				Name:    "testcontainer",
				Aliases: []string{"testcontaineralias"},
			},
			Spec: info.ContainerSpec{
				Image:  "test",
				HasCpu: true,
				Cpu: info.CpuSpec{
					Limit:  1000,
					Period: 100000,
					Quota:  10000,
				},
				Memory: info.MemorySpec{
					Limit:       2048,
					Reservation: 1024,
					SwapLimit:   4096,
				},
				HasHugetlb:   true,
				HasProcesses: true,
				Processes: info.ProcessSpec{
					Limit: 100,
				},
				CreationTime: time.Unix(1257894000, 0),
				Labels: map[string]string{
					"foo.label": "bar",
				},
				Envs: map[string]string{
					"foo+env": "prod",
				},
			},
			Stats: []*info.ContainerStats{
				{
					Timestamp: time.Unix(1395066363, 0),
					Cpu: info.CpuStats{
						Usage: info.CpuUsage{
							Total:  1,
							PerCpu: []uint64{2, 3, 4, 5},
							User:   6,
							System: 7,
						},
						CFS: info.CpuCFS{
							Periods:          723,
							ThrottledPeriods: 18,
							ThrottledTime:    1724314000,
						},
						Schedstat: info.CpuSchedstat{
							RunTime:      53643567,
							RunqueueTime: 479424566378,
							RunPeriods:   984285,
						},
						LoadAverage: 2,
					},
					Memory: info.MemoryStats{
						Usage:      8,
						MaxUsage:   8,
						WorkingSet: 9,
						ContainerData: info.MemoryStatsMemoryData{
							Pgfault:    10,
							Pgmajfault: 11,
							NumaStats: info.MemoryNumaStats{
								File:        map[uint8]uint64{0: 16649, 1: 10000},
								Anon:        map[uint8]uint64{0: 10000, 1: 7109},
								Unevictable: map[uint8]uint64{0: 8900, 1: 10000},
							},
						},
						HierarchicalData: info.MemoryStatsMemoryData{
							Pgfault:    12,
							Pgmajfault: 13,
							NumaStats: info.MemoryNumaStats{
								File:        map[uint8]uint64{0: 36649, 1: 10000},
								Anon:        map[uint8]uint64{0: 20000, 1: 7109},
								Unevictable: map[uint8]uint64{0: 8900, 1: 20000},
							},
						},
						Cache:      14,
						RSS:        15,
						MappedFile: 16,
						Swap:       8192,
					},
					Hugetlb: map[string]info.HugetlbStats{
						"2Mi": {
							Usage:    4,
							MaxUsage: 10,
							Failcnt:  1,
						},
						"1Gi": {
							Usage:    0,
							MaxUsage: 0,
							Failcnt:  0,
						},
					},
					Network: info.NetworkStats{
						InterfaceStats: info.InterfaceStats{
							Name:      "eth0",
							RxBytes:   14,
							RxPackets: 15,
							RxErrors:  16,
							RxDropped: 17,
							TxBytes:   18,
							TxPackets: 19,
							TxErrors:  20,
							TxDropped: 21,
						},
						Interfaces: []info.InterfaceStats{
							{
								Name:      "eth0",
								RxBytes:   14,
								RxPackets: 15,
								RxErrors:  16,
								RxDropped: 17,
								TxBytes:   18,
								TxPackets: 19,
								TxErrors:  20,
								TxDropped: 21,
							},
						},
						Tcp: info.TcpStat{
							Established: 13,
							SynSent:     0,
							SynRecv:     0,
							FinWait1:    0,
							FinWait2:    0,
							TimeWait:    0,
							Close:       0,
							CloseWait:   0,
							LastAck:     0,
							Listen:      3,
							Closing:     0,
						},
						Tcp6: info.TcpStat{
							Established: 11,
							SynSent:     0,
							SynRecv:     0,
							FinWait1:    0,
							FinWait2:    0,
							TimeWait:    0,
							Close:       0,
							CloseWait:   0,
							LastAck:     0,
							Listen:      3,
							Closing:     0,
						},
						TcpAdvanced: info.TcpAdvancedStat{
							TCPFullUndo:               2361,
							TCPMD5NotFound:            0,
							TCPDSACKRecv:              83680,
							TCPSackShifted:            2,
							TCPSackShiftFallback:      298,
							PFMemallocDrop:            0,
							EstabResets:               37,
							InSegs:                    140370590,
							TCPPureAcks:               24251339,
							TCPDSACKOldSent:           15633,
							IPReversePathFilter:       0,
							TCPFastOpenPassiveFail:    0,
							InCsumErrors:              0,
							TCPRenoFailures:           43414,
							TCPMemoryPressuresChrono:  0,
							TCPDeferAcceptDrop:        0,
							TW:                        10436427,
							TCPSpuriousRTOs:           0,
							TCPDSACKIgnoredNoUndo:     71885,
							RtoMax:                    120000,
							ActiveOpens:               11038621,
							EmbryonicRsts:             0,
							RcvPruned:                 0,
							TCPLossProbeRecovery:      401,
							TCPHPHits:                 56096478,
							TCPPartialUndo:            3,
							TCPAbortOnMemory:          0,
							AttemptFails:              48997,
							RetransSegs:               462961,
							SyncookiesFailed:          0,
							OfoPruned:                 0,
							TCPAbortOnLinger:          0,
							TCPAbortFailed:            0,
							TCPRenoReorder:            839,
							TCPRcvCollapsed:           0,
							TCPDSACKIgnoredOld:        0,
							TCPReqQFullDrop:           0,
							OutOfWindowIcmps:          0,
							TWKilled:                  0,
							TCPLossProbes:             88648,
							TCPRenoRecoveryFail:       394,
							TCPFastOpenCookieReqd:     0,
							TCPHPAcks:                 21490641,
							TCPSACKReneging:           0,
							TCPTSReorder:              3,
							TCPSlowStartRetrans:       290832,
							MaxConn:                   -1,
							SyncookiesRecv:            0,
							TCPSackFailures:           60,
							DelayedACKLocked:          90,
							TCPDSACKOfoSent:           1,
							TCPSynRetrans:             988,
							TCPDSACKOfoRecv:           10,
							TCPSACKDiscard:            0,
							TCPMD5Unexpected:          0,
							TCPSackMerged:             6,
							RtoMin:                    200,
							CurrEstab:                 22,
							TCPTimeWaitOverflow:       0,
							ListenOverflows:           0,
							DelayedACKs:               503975,
							TCPLossUndo:               61374,
							TCPOrigDataSent:           130698387,
							TCPBacklogDrop:            0,
							TCPReqQFullDoCookies:      0,
							TCPFastOpenPassive:        0,
							PAWSActive:                0,
							OutRsts:                   91699,
							TCPSackRecoveryFail:       2,
							DelayedACKLost:            18843,
							TCPAbortOnData:            8,
							TCPMinTTLDrop:             0,
							PruneCalled:               0,
							TWRecycled:                0,
							ListenDrops:               0,
							TCPAbortOnTimeout:         0,
							SyncookiesSent:            0,
							TCPSACKReorder:            11,
							TCPDSACKUndo:              33,
							TCPMD5Failure:             0,
							TCPLostRetransmit:         0,
							TCPAbortOnClose:           7,
							TCPFastOpenListenOverflow: 0,
							OutSegs:                   211580512,
							InErrs:                    31,
							TCPTimeouts:               27422,
							TCPLossFailures:           729,
							TCPSackRecovery:           159,
							RtoAlgorithm:              1,
							PassiveOpens:              59,
							LockDroppedIcmps:          0,
							TCPRenoRecovery:           3519,
							TCPFACKReorder:            0,
							TCPFastRetrans:            11794,
							TCPRetransFail:            0,
							TCPMemoryPressures:        0,
							TCPFastOpenActive:         0,
							TCPFastOpenActiveFail:     0,
							PAWSEstab:                 0,
						},
						Udp: info.UdpStat{
							Listen:   0,
							Dropped:  0,
							RxQueued: 0,
							TxQueued: 0,
						},
						Udp6: info.UdpStat{
							Listen:   0,
							Dropped:  0,
							RxQueued: 0,
							TxQueued: 0,
						},
					},
					Filesystem: []info.FsStats{
						{
							Device:          "sda1",
							InodesFree:      524288,
							Inodes:          2097152,
							Limit:           22,
							Usage:           23,
							ReadsCompleted:  24,
							ReadsMerged:     25,
							SectorsRead:     26,
							ReadTime:        27,
							WritesCompleted: 28,
							WritesMerged:    39,
							SectorsWritten:  40,
							WriteTime:       41,
							IoInProgress:    42,
							IoTime:          43,
							WeightedIoTime:  44,
						},
						{
							Device:          "sda2",
							InodesFree:      262144,
							Inodes:          2097152,
							Limit:           37,
							Usage:           38,
							ReadsCompleted:  39,
							ReadsMerged:     40,
							SectorsRead:     41,
							ReadTime:        42,
							WritesCompleted: 43,
							WritesMerged:    44,
							SectorsWritten:  45,
							WriteTime:       46,
							IoInProgress:    47,
							IoTime:          48,
							WeightedIoTime:  49,
						},
					},
					Accelerators: []info.AcceleratorStats{
						{
							Make:        "nvidia",
							Model:       "tesla-p100",
							ID:          "GPU-deadbeef-1234-5678-90ab-feedfacecafe",
							MemoryTotal: 20304050607,
							MemoryUsed:  2030405060,
							DutyCycle:   12,
						},
						{
							Make:        "nvidia",
							Model:       "tesla-k80",
							ID:          "GPU-deadbeef-0123-4567-89ab-feedfacecafe",
							MemoryTotal: 10203040506,
							MemoryUsed:  1020304050,
							DutyCycle:   6,
						},
					},
					Processes: info.ProcessStats{
						ProcessCount:   1,
						FdCount:        5,
						SocketCount:    3,
						ThreadsCurrent: 5,
						ThreadsMax:     100,
						Ulimits: []info.UlimitSpec{
							{
								Name:      "max_open_files",
								SoftLimit: 16384,
								HardLimit: 16384,
							},
						},
					},
					TaskStats: info.LoadStats{
						NrSleeping:        50,
						NrRunning:         51,
						NrStopped:         52,
						NrUninterruptible: 53,
						NrIoWait:          54,
					},
					CustomMetrics: map[string][]info.MetricVal{
						"container_custom_app_metric_1": {
							{
								FloatValue: float64(1.1),
								Timestamp:  time.Now(),
								Label:      "testlabel_1_1_1",
								Labels:     map[string]string{"test_label": "1_1", "test_label_2": "2_1"},
							},
							{
								FloatValue: float64(1.2),
								Timestamp:  time.Now(),
								Label:      "testlabel_1_1_2",
								Labels:     map[string]string{"test_label": "1_2", "test_label_2": "2_2"},
							},
						},
						"container_custom_app_metric_2": {
							{
								FloatValue: float64(2),
								Timestamp:  time.Now(),
								Label:      "testlabel2",
								Labels:     map[string]string{"test_label": "test_value"},
							},
						},
						"container_custom_app_metric_3": {
							{
								FloatValue: float64(3),
								Timestamp:  time.Now(),
								Label:      "testlabel3",
								Labels:     map[string]string{"test_label": "test_value"},
							},
						},
					},
					PerfStats: []info.PerfStat{
						{
							PerfValue: info.PerfValue{
								ScalingRatio: 1.0,
								Value:        123,
								Name:         "instructions",
							},
							Cpu: 0,
						},
						{
							PerfValue: info.PerfValue{
								ScalingRatio: 0.5,
								Value:        456,
								Name:         "instructions",
							},
							Cpu: 1,
						},
						{
							PerfValue: info.PerfValue{
								ScalingRatio: 0.66666666666,
								Value:        321,
								Name:         "instructions_retired",
							},
							Cpu: 0,
						},
						{
							PerfValue: info.PerfValue{
								ScalingRatio: 0.33333333333,
								Value:        789,
								Name:         "instructions_retired",
							},
							Cpu: 1,
						},
					},
					PerfUncoreStats: []info.PerfUncoreStat{
						{
							PerfValue: info.PerfValue{
								ScalingRatio: 1.0,
								Value:        1231231512.0,
								Name:         "cas_count_read",
							},
							Socket: 0,
							PMU:    "uncore_imc_0",
						},
						{
							PerfValue: info.PerfValue{
								ScalingRatio: 1.0,
								Value:        1111231331.0,
								Name:         "cas_count_read",
							},
							Socket: 1,
							PMU:    "uncore_imc_0",
						},
					},
					ReferencedMemory: 1234,
					Resctrl: info.ResctrlStats{
						MemoryBandwidth: []info.MemoryBandwidthStats{
							{
								TotalBytes: 4512312,
								LocalBytes: 2390393,
							},
							{
								TotalBytes: 2173713,
								LocalBytes: 1231233,
							},
						},
						Cache: []info.CacheStats{
							{
								LLCOccupancy: 162626,
							},
							{
								LLCOccupancy: 213777,
							},
						},
					},
				},
			},
		},
	}, nil
}

type erroringSubcontainersInfoProvider struct {
	successfulProvider testSubcontainersInfoProvider
	shouldFail         bool
}

func (p *erroringSubcontainersInfoProvider) GetVersionInfo() (*info.VersionInfo, error) {
	if p.shouldFail {
		return nil, errors.New("Oops 1")
	}
	return p.successfulProvider.GetVersionInfo()
}

func (p *erroringSubcontainersInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	if p.shouldFail {
		return nil, errors.New("Oops 2")
	}
	return p.successfulProvider.GetMachineInfo()
}

func (p *erroringSubcontainersInfoProvider) GetRequestedContainersInfo(
	a string, opt v2.RequestOptions) (map[string]*info.ContainerInfo, error) {
	if p.shouldFail {
		return map[string]*info.ContainerInfo{}, errors.New("Oops 3")
	}
	return p.successfulProvider.GetRequestedContainersInfo(a, opt)
}
