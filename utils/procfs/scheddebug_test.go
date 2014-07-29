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

package procfs

import (
	"reflect"
	"testing"

	"code.google.com/p/gomock/gomock"

	"github.com/google/cadvisor/utils/fs"
	"github.com/google/cadvisor/utils/fs/mockfs"
)

var schedDebugToLoadsPerContainerPerCore = []struct {
	SchedDebugContent string
	Loads             map[string][]int
	Error             error
}{
	{
		`
Sched Debug Version: v0.11, 3.13.0-29-generic #53-Ubuntu
runnable tasks:
            task   PID         tree-key  switches  prio     exec-runtime         sum-exec        sum-sleep
----------------------------------------------------------------------------------------------------------
        kthreadd     2 159918906.381680       397   120 159918906.381680        16.755308 4408983057.115372 0 /
    kworker/0:0  9225 159645585.744016         8   120 159645585.744016         0.151033   9178573.870160 0 /
        cadvisor 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/hash
        someproc 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/somecontainer

runnable tasks:
            task   PID         tree-key  switches  prio     exec-runtime         sum-exec        sum-sleep
----------------------------------------------------------------------------------------------------------
    kworker/0:0  9225 159645585.744016         8   120 159645585.744016         0.151033   9178573.870160 0 /
        cadvisor 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/hash
        other 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/another

		`,
		map[string][]int{
			"/":                     {2, 1},
			"/docker/hash":          {1, 1},
			"/docker/another":       {0, 1},
			"/docker/somecontainer": {1, 0},
		},
		nil,
	},
	{
		`
Sched Debug Version: v0.11, 3.13.0-29-generic #53-Ubuntu
ktime                                   : 4409767725.360719
sched_clk                               : 4722208509.060012
cpu_clk                                 : 4409767725.360725
jiffies                                 : 5397334228
sched_clock_stable                      : 0

sysctl_sched
  .sysctl_sched_latency                    : 24.000000
  .sysctl_sched_min_granularity            : 3.000000
  .sysctl_sched_wakeup_granularity         : 4.000000
  .sysctl_sched_child_runs_first           : 0
  .sysctl_sched_features                   : 77435
  .sysctl_sched_tunable_scaling            : 1 (logaritmic)

cpu#0, 2599.998 MHz
  .nr_running                    : 0
  .load                          : 0
  .nr_switches                   : 4015586556
  .nr_load_updates               : 105987951
  .nr_uninterruptible            : -334192
  .next_balance                  : 5397.334205
  .curr->pid                     : 0
  .clock                         : 4409767641.399025
  .cpu_load[0]                   : 0
  .cpu_load[1]                   : 0
  .cpu_load[2]                   : 0
  .cpu_load[3]                   : 0
  .cpu_load[4]                   : 0
  .yld_count                     : 1327594
  .sched_count                   : -278003554
  .sched_goidle                  : 2002340573
  .avg_idle                      : 187229
  .ttwu_count                    : 2078272826
  .ttwu_local                    : 1180909486

cfs_rq[0]:/user/1014.user
  .exec_clock                    : 13335742.850392
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 19737907.734191
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -140209124.399553
  .nr_spread_over                : 0
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 3
  .tg_load_contrib               : 1
  .tg_runnable_contrib           : 5
  .tg_load_avg                   : 12
  .tg->runnable_avg              : 40
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767641.399025
  .se->vruntime                  : 22521033.619085
  .se->sum_exec_runtime          : 13335742.878469
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 1176.006499
  .se->statistics.slice_max      : 382.761845
  .se->statistics.wait_max       : 202.195062
  .se->statistics.wait_sum       : 16916.088365
  .se->statistics.wait_count     : 286435218
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 306
  .se->avg.runnable_avg_period   : 46456
  .se->avg.load_avg_contrib      : 1
  .se->avg.decay_count           : 4205482141

cfs_rq[0]:/user
  .exec_clock                    : 14007754.294731
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 22521033.619085
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -137425998.514659
  .nr_spread_over                : 0
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 1
  .tg_load_contrib               : 5
  .tg_runnable_contrib           : 4
  .tg_load_avg                   : 15
  .tg->runnable_avg              : 43
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767641.399025
  .se->vruntime                  : 159947032.133744
  .se->sum_exec_runtime          : 14007766.619125
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 1176.006499
  .se->statistics.slice_max      : 8.170829
  .se->statistics.wait_max       : 202.195062
  .se->statistics.wait_sum       : 19616.722323
  .se->statistics.wait_count     : 291817014
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 308
  .se->avg.runnable_avg_period   : 46789
  .se->avg.load_avg_contrib      : 7
  .se->avg.decay_count           : 4205482141

cfs_rq[0]:/
  .exec_clock                    : 88845316.977008
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 159947032.133744
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : 0.000000
  .nr_spread_over                : 99
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 7
  .tg_load_contrib               : 15
  .tg_runnable_contrib           : 6
  .tg_load_avg                   : 42
  .tg->runnable_avg              : 63
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .avg->runnable_avg_sum         : 300
  .avg->runnable_avg_period      : 46798

cfs_rq[0]:/user/1014.user/127.session
  .exec_clock                    : 230108.794316
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 220535.926524
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -159726496.207220
  .nr_spread_over                : 38
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 9
  .tg_load_contrib               : 7
  .tg_runnable_contrib           : 5
  .tg_load_avg                   : 33
  .tg->runnable_avg              : 23
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767641.399025
  .se->vruntime                  : 19737907.734191
  .se->sum_exec_runtime          : 230108.797063
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 6.318158
  .se->statistics.slice_max      : 6.060779
  .se->statistics.wait_max       : 3.122343
  .se->statistics.wait_sum       : 1247.416895
  .se->statistics.wait_count     : 2198718
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 305
  .se->avg.runnable_avg_period   : 46769
  .se->avg.load_avg_contrib      : 3
  .se->avg.decay_count           : 4205482141

rt_rq[0]:
  .rt_nr_running                 : 0
  .rt_throttled                  : 0
  .rt_time                       : 0.000000
  .rt_runtime                    : 950.000000


runnable tasks:
            task   PID         tree-key  switches  prio     exec-runtime         sum-exec        sum-sleep
----------------------------------------------------------------------------------------------------------
        kthreadd     2 159918906.381680       397   120 159918906.381680        16.755308 4408983057.115372 0 /
    kworker/0:0  9225 159645585.744016         8   120 159645585.744016         0.151033   9178573.870160 0 /
        cadvisor 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/hash
        someproc 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/somecontainer

cpu#1, 2599.998 MHz
  .nr_running                    : 0
  .load                          : 0
  .nr_switches                   : 3944986693
  .nr_load_updates               : 96182719
  .nr_uninterruptible            : 173423
  .next_balance                  : 5397.334255
  .curr->pid                     : 0
  .clock                         : 4409767724.343465
  .cpu_load[0]                   : 0
  .cpu_load[1]                   : 0
  .cpu_load[2]                   : 0
  .cpu_load[3]                   : 0
  .cpu_load[4]                   : 0
  .yld_count                     : 1321983
  .sched_count                   : -348608544
  .sched_goidle                  : 1968452376
  .avg_idle                      : 1000000
  .ttwu_count                    : 1957923792
  .ttwu_local                    : 1102567359

cfs_rq[1]:/user/1014.user
  .exec_clock                    : 13485925.121019
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 19845478.053952
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -140101554.079792
  .nr_spread_over                : 0
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 0
  .tg_load_contrib               : 0
  .tg_runnable_contrib           : 12
  .tg_load_avg                   : 12
  .tg->runnable_avg              : 40
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767724.340638
  .se->vruntime                  : 22532500.804117
  .se->sum_exec_runtime          : 13485925.213913
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 13.643888
  .se->statistics.slice_max      : 489.490466
  .se->statistics.wait_max       : 550.414020
  .se->statistics.wait_sum       : 17330.937081
  .se->statistics.wait_count     : 295976777
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 560
  .se->avg.runnable_avg_period   : 47228
  .se->avg.load_avg_contrib      : 0
  .se->avg.decay_count           : 4205482221

cfs_rq[1]:/user/1018.user
  .exec_clock                    : 480956.077278
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 607293.984023
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -159339738.149721
  .nr_spread_over                : 0
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 0
  .tg_load_contrib               : 0
  .tg_runnable_contrib           : 0
  .tg_load_avg                   : 0
  .tg->runnable_avg              : 2
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767638.164369
  .se->vruntime                  : 22532512.239259
  .se->sum_exec_runtime          : 480956.091761
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 20.787279
  .se->statistics.slice_max      : 128.534664
  .se->statistics.wait_max       : 16.863694
  .se->statistics.wait_sum       : 2144.675431
  .se->statistics.wait_count     : 3454450
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 13
  .se->avg.runnable_avg_period   : 46463
  .se->avg.load_avg_contrib      : 0
  .se->avg.decay_count           : 4205482138

cfs_rq[1]:/docker
  .exec_clock                    : 33323750.675251
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 56928736.465342
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -103018295.668402
  .nr_spread_over                : 0
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 0
  .tg_load_contrib               : 0
  .tg_runnable_contrib           : 2
  .tg_load_avg                   : 0
  .tg->runnable_avg              : 2
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767693.642881
  .se->vruntime                  : 152694416.015894
  .se->sum_exec_runtime          : 33323774.559318
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 244.042161
  .se->statistics.slice_max      : 751.402989
  .se->statistics.wait_max       : 9.112128
  .se->statistics.wait_sum       : 48336.965928
  .se->statistics.wait_count     : 633354214
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 110
  .se->avg.runnable_avg_period   : 46476
  .se->avg.load_avg_contrib      : 0
  .se->avg.decay_count           : 4205482191

cfs_rq[1]:/user
  .exec_clock                    : 14090901.680875
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 22532512.239259
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -137414519.894485
  .nr_spread_over                : 0
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 0
  .tg_load_contrib               : 0
  .tg_runnable_contrib           : 1
  .tg_load_avg                   : 15
  .tg->runnable_avg              : 43
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767724.340638
  .se->vruntime                  : 152694428.387252
  .se->sum_exec_runtime          : 14090908.503310
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 20.787279
  .se->statistics.slice_max      : 9.088991
  .se->statistics.wait_max       : 550.414020
  .se->statistics.wait_sum       : 19912.196787
  .se->statistics.wait_count     : 301194069
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 608
  .se->avg.runnable_avg_period   : 48047
  .se->avg.load_avg_contrib      : 0
  .se->avg.decay_count           : 4205482221

cfs_rq[1]:/
  .exec_clock                    : 85028436.453750
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 152694428.387252
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -7252603.746492
  .nr_spread_over                : 62
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 0
  .tg_load_contrib               : 0
  .tg_runnable_contrib           : 14
  .tg_load_avg                   : 42
  .tg->runnable_avg              : 63
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .avg->runnable_avg_sum         : 670
  .avg->runnable_avg_period      : 46562

cfs_rq[1]:/docker/d1cebfddcf94a8018cbd73175b1484ab239943b5ee517529509bb7502181fd15
  .exec_clock                    : 209.852282
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 208.680611
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -159946823.453133
  .nr_spread_over                : 0
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 0
  .tg_load_contrib               : 0
  .tg_runnable_contrib           : 2
  .tg_load_avg                   : 0
  .tg->runnable_avg              : 2
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767693.642881
  .se->vruntime                  : 56928736.465342
  .se->sum_exec_runtime          : 209.852282
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 2.409911
  .se->statistics.slice_max      : 2.123931
  .se->statistics.wait_max       : 0.062006
  .se->statistics.wait_sum       : 0.319593
  .se->statistics.wait_count     : 1810
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 113
  .se->avg.runnable_avg_period   : 47401
  .se->avg.load_avg_contrib      : 0
  .se->avg.decay_count           : 4205482191

cfs_rq[1]:/user/1018.user/102.session
  .exec_clock                    : 295188.205699
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 267319.397716
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -159679712.736028
  .nr_spread_over                : 11
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 0
  .tg_load_contrib               : 0
  .tg_runnable_contrib           : 0
  .tg_load_avg                   : 10
  .tg->runnable_avg              : 2
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767638.164369
  .se->vruntime                  : 607293.984023
  .se->sum_exec_runtime          : 295188.217342
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 20.787279
  .se->statistics.slice_max      : 9.711102
  .se->statistics.wait_max       : 8.558728
  .se->statistics.wait_sum       : 1146.954809
  .se->statistics.wait_count     : 2361162
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 13
  .se->avg.runnable_avg_period   : 47447
  .se->avg.load_avg_contrib      : 0
  .se->avg.decay_count           : 4205482138

cfs_rq[1]:/user/1014.user/128.session
  .exec_clock                    : 20.815341
  .MIN_vruntime                  : 0.000001
  .min_vruntime                  : 271.766523
  .max_vruntime                  : 0.000001
  .spread                        : 0.000000
  .spread0                       : -159946760.367221
  .nr_spread_over                : 13
  .nr_running                    : 0
  .load                          : 0
  .runnable_load_avg             : 0
  .blocked_load_avg              : 1
  .tg_load_contrib               : 1
  .tg_runnable_contrib           : 0
  .tg_load_avg                   : 981
  .tg->runnable_avg              : 8
  .tg->cfs_bandwidth.timer_active: 0
  .throttled                     : 0
  .throttle_count                : 0
  .se->exec_start                : 4409767724.340638
  .se->vruntime                  : 19845466.859551
  .se->sum_exec_runtime          : 20.815341
  .se->statistics.wait_start     : 0.000000
  .se->statistics.sleep_start    : 0.000000
  .se->statistics.block_start    : 0.000000
  .se->statistics.sleep_max      : 0.000000
  .se->statistics.block_max      : 0.000000
  .se->statistics.exec_max       : 1.712933
  .se->statistics.slice_max      : 0.542023
  .se->statistics.wait_max       : 0.072066
  .se->statistics.wait_sum       : 0.437682
  .se->statistics.wait_count     : 104
  .se->load.weight               : 2
  .se->avg.runnable_avg_sum      : 570
  .se->avg.runnable_avg_period   : 47945
  .se->avg.load_avg_contrib      : 0
  .se->avg.decay_count           : 4205482221

rt_rq[1]:
  .rt_nr_running                 : 0
  .rt_throttled                  : 0
  .rt_time                       : 0.000000
  .rt_runtime                    : 950.000000

runnable tasks:
            task   PID         tree-key  switches  prio     exec-runtime         sum-exec        sum-sleep
----------------------------------------------------------------------------------------------------------
    kworker/0:0  9225 159645585.744016         8   120 159645585.744016         0.151033   9178573.870160 0 /
        cadvisor 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/hash
        other 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/another

		`,
		map[string][]int{
			"/":                     {2, 1},
			"/docker/hash":          {1, 1},
			"/docker/another":       {0, 1},
			"/docker/somecontainer": {1, 0},
		},
		nil,
	},
	{
		`
Sched Debug Version: v0.11, 3.13.0-29-generic #53-Ubuntu
runnable tasks:
            task   PID         tree-key  switches  prio     exec-runtime         sum-exec        sum-sleep
----------------------------------------------------------------------------------------------------------
        kthreadd     2 159918906.381680       397   120 159918906.381680        16.755308 4408983057.115372 0 /
    kworker/0:0  9225 159645585.744016         8   120 159645585.744016         0.151033   9178573.870160 0 /
        cadvisor 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/hash
        someproc 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/somecontainer

runnable tasks:
            task   PID         tree-key  switches  prio     exec-runtime         sum-exec        sum-sleep
----------------------------------------------------------------------------------------------------------
    kworker/0:0  9225 159645585.744016         8   120 159645585.744016         0.151033   9178573.870160 0 /
        cadvisor 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/hash
        other 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/another

runnable tasks:
            task   PID         tree-key  switches  prio     exec-runtime         sum-exec        sum-sleep
----------------------------------------------------------------------------------------------------------
    kworker/0:0  9225 159645585.744016         8   120 159645585.744016         0.151033   9178573.870160 0 /
        cadvisor 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/hash
        other 15008    220535.926524    191029   120    220535.926524      3256.362378    672644.858661 0 /docker/another

		`,
		map[string][]int{
			"/":                     {2, 1, 1},
			"/docker/hash":          {1, 1, 1},
			"/docker/another":       {0, 1, 1},
			"/docker/somecontainer": {1, 0, 0},
		},
		nil,
	},
}

func TestSchedDebugReader(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	for _, testCase := range schedDebugToLoadsPerContainerPerCore {
		mfs := mockfs.NewMockFileSystem(mockCtrl)
		path := "/proc/sched_debug"
		schedDebugContent := testCase.SchedDebugContent
		mockfs.AddTextFile(mfs, path, schedDebugContent)
		fs.ChangeFileSystem(mfs)
		loads, err := NewSchedulerLoadReader()
		if testCase.Error != nil {
			if err == nil {
				t.Fatal("expected error: %v", testCase.Error)
			}
		}
		if err != nil {
			t.Fatal(err)
		}
		containers, err := loads.AllContainers()
		if err != nil {
			t.Fatal(err)
		}
		if len(containers) != len(testCase.Loads) {
			t.Errorf("expected %v container's information; received %v", len(testCase.Loads), len(containers))
		}
		for _, container := range containers {
			l, err := loads.Load(container)
			if err != nil {
				t.Fatal(err)
			}
			if expected, ok := testCase.Loads[container]; ok {
				if !reflect.DeepEqual(expected, l) {
					t.Errorf("Wrong load for container %v; should be %v, received %v\nsched_debug:\n%v\n",
						container, expected, l, schedDebugContent)
					continue
				}
			} else {
				t.Errorf("unexpected container %v.\nsched_debug:\n%v\n", container, schedDebugContent)
			}
		}
	}
}
