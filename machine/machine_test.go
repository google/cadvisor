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

package machine

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetVMStatsGetNuma(t *testing.T) {
	vmstatFile = "testdata/vmstat_data"
	re := "numa.*"
	result, err := GetVMStats(&re)
	expected := map[string]int{
		"numa_foreign":           0,
		"numa_hint_faults":       0,
		"numa_hint_faults_local": 0,
		"numa_hit":               120883140,
		"numa_huge_pte_updates":  0,
		"numa_interleave":        23061,
		"numa_local":             120883140,
		"numa_miss":              0,
		"numa_other":             0,
		"numa_pages_migrated":    0,
		"numa_pte_updates":       0,
	}

	assert.Nil(t, err)
	assert.Equal(t, expected, result)

}
func TestGetVMStatsGetFaults(t *testing.T) {
	vmstatFile = "testdata/vmstat_data"
	re := ".*(faults|failed).*"
	result, err := GetVMStats(&re)
	expected := map[string]int{
		"numa_hint_faults":           0,
		"numa_hint_faults_local":     0,
		"thp_collapse_alloc_failed":  0,
		"thp_split_page_failed":      0,
		"thp_zero_page_alloc_failed": 0,
		"zone_reclaim_failed":        0,
	}

	assert.Nil(t, err)
	assert.Equal(t, expected, result)

}
func TestGetVMStatsGetAll(t *testing.T) {
	vmstatFile = "testdata/vmstat_data"
	re := ".*"
	result, err := GetVMStats(&re)
	expected := map[string]int{"allocstall_dma": 0,
		"allocstall_dma32":               0,
		"allocstall_movable":             0,
		"allocstall_normal":              0,
		"balloon_deflate":                0,
		"balloon_inflate":                0,
		"balloon_migrate":                0,
		"compact_daemon_free_scanned":    0,
		"compact_daemon_migrate_scanned": 0,
		"compact_daemon_wake":            0,
		"compact_fail":                   0,
		"compact_free_scanned":           0,
		"compact_isolated":               0,
		"compact_migrate_scanned":        0,
		"compact_stall":                  0,
		"compact_success":                0,
		"drop_pagecache":                 0,
		"drop_slab":                      0,
		"htlb_buddy_alloc_fail":          0,
		"htlb_buddy_alloc_success":       0,
		"kswapd_high_wmark_hit_quickly":  0,
		"kswapd_inodesteal":              0,
		"kswapd_low_wmark_hit_quickly":   0,
		"nr_active_anon":                 1164948,
		"nr_active_file":                 567385,
		"nr_anon_pages":                  1262075,
		"nr_anon_transparent_hugepages":  19,
		"nr_bounce":                      0,
		"nr_dirtied":                     1329549,
		"nr_dirty":                       1145,
		"nr_dirty_background_threshold":  644547,
		"nr_dirty_threshold":             1290670,
		"nr_file_hugepages":              0,
		"nr_file_pages":                  1721185,
		"nr_file_pmdmapped":              0,
		"nr_foll_pin_acquired":           0,
		"nr_foll_pin_released":           0,
		"nr_free_cma":                    0,
		"nr_free_pages":                  4981979,
		"nr_inactive_anon":               56987,
		"nr_inactive_file":               972664,
		"nr_isolated_anon":               0,
		"nr_isolated_file":               0,
		"nr_kernel_misc_reclaimable":     0,
		"nr_kernel_stack":                23952,
		"nr_mapped":                      517738,
		"nr_mlock":                       12,
		"nr_page_table_pages":            14081,
		"nr_shmem":                       278886,
		"nr_shmem_hugepages":             0,
		"nr_shmem_pmdmapped":             0,
		"nr_slab_reclaimable":            111629,
		"nr_slab_unreclaimable":          69328,
		"nr_unevictable":                 221278,
		"nr_unstable":                    0,
		"nr_vmscan_immediate_reclaim":    0,
		"nr_vmscan_write":                0,
		"nr_writeback":                   0,
		"nr_writeback_temp":              0,
		"nr_written":                     1325274,
		"nr_zone_active_anon":            1164948,
		"nr_zone_active_file":            567385,
		"nr_zone_inactive_anon":          56987,
		"nr_zone_inactive_file":          972664,
		"nr_zone_unevictable":            221278,
		"nr_zone_write_pending":          1139,
		"nr_zspages":                     0,
		"numa_foreign":                   0,
		"numa_hint_faults":               0,
		"numa_hint_faults_local":         0,
		"numa_hit":                       120883140,
		"numa_huge_pte_updates":          0,
		"numa_interleave":                23061,
		"numa_local":                     120883140,
		"numa_miss":                      0,
		"numa_other":                     0,
		"numa_pages_migrated":            0,
		"numa_pte_updates":               0,
		"oom_kill":                       0,
		"pageoutrun":                     0,
		"pgactivate":                     5149662,
		"pgalloc_dma":                    0,
		"pgalloc_dma32":                  1,
		"pgalloc_movable":                0,
		"pgalloc_normal":                 126730966,
		"pgdeactivate":                   0,
		"pgfault":                        108683933,
		"pgfree":                         131715713,
		"pginodesteal":                   0,
		"pglazyfree":                     828926,
		"pglazyfreed":                    0,
		"pgmajfault":                     20994,
		"pgmigrate_fail":                 0,
		"pgmigrate_success":              0,
		"pgpgin":                         4079261,
		"pgpgout":                        6255350,
		"pgrefill":                       0,
		"pgrotated":                      11,
		"pgscan_direct":                  0,
		"pgscan_direct_throttle":         0,
		"pgscan_kswapd":                  0,
		"pgskip_dma":                     0,
		"pgskip_dma32":                   0,
		"pgskip_movable":                 0,
		"pgskip_normal":                  0,
		"pgsteal_direct":                 0,
		"pgsteal_kswapd":                 0,
		"pswpin":                         0,
		"pswpout":                        0,
		"slabs_scanned":                  0,
		"swap_ra":                        0,
		"swap_ra_hit":                    0,
		"thp_collapse_alloc":             918,
		"thp_collapse_alloc_failed":      0,
		"thp_deferred_split_page":        891,
		"thp_fault_alloc":                450,
		"thp_fault_fallback":             0,
		"thp_fault_fallback_charge":      0,
		"thp_file_alloc":                 0,
		"thp_file_fallback":              0,
		"thp_file_fallback_charge":       0,
		"thp_file_mapped":                0,
		"thp_split_page":                 891,
		"thp_split_page_failed":          0,
		"thp_split_pmd":                  891,
		"thp_split_pud":                  0,
		"thp_swpout":                     0,
		"thp_swpout_fallback":            0,
		"thp_zero_page_alloc":            1,
		"thp_zero_page_alloc_failed":     0,
		"unevictable_pgs_cleared":        0,
		"unevictable_pgs_culled":         9542080,
		"unevictable_pgs_mlocked":        270870,
		"unevictable_pgs_munlocked":      270858,
		"unevictable_pgs_rescued":        9300587,
		"unevictable_pgs_scanned":        11752725,
		"unevictable_pgs_stranded":       0,
		"workingset_activate":            0,
		"workingset_nodereclaim":         0,
		"workingset_nodes":               0,
		"workingset_refault":             0,
		"workingset_restore":             0,
		"zone_reclaim_failed":            0,
	}

	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}
