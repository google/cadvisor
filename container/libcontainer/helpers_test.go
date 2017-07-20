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

package libcontainer

import (
	"os"
	"testing"

	info "github.com/google/cadvisor/info/v1"
)

func TestScanInterfaceStats(t *testing.T) {
	stats, err := scanInterfaceStats("testdata/procnetdev")
	if err != nil {
		t.Error(err)
	}

	var netdevstats = []info.InterfaceStats{
		{
			Name:      "wlp4s0",
			RxBytes:   1,
			RxPackets: 2,
			RxErrors:  3,
			RxDropped: 4,
			TxBytes:   9,
			TxPackets: 10,
			TxErrors:  11,
			TxDropped: 12,
		},
		{
			Name:      "em1",
			RxBytes:   315849,
			RxPackets: 1172,
			RxErrors:  0,
			RxDropped: 0,
			TxBytes:   315850,
			TxPackets: 1173,
			TxErrors:  0,
			TxDropped: 0,
		},
	}

	if len(stats) != len(netdevstats) {
		t.Errorf("Expected 2 net stats, got %d", len(stats))
	}

	for i, v := range netdevstats {
		if v != stats[i] {
			t.Errorf("Expected %#v, got %#v", v, stats[i])
		}
	}
}

func TestScanUDPStats(t *testing.T) {
	udpStatsFile := "testdata/procnetudp"
	r, err := os.Open(udpStatsFile)
	if err != nil {
		t.Errorf("failure opening %s: %v", udpStatsFile, err)
	}

	stats, err := scanUdpStats(r)
	if err != nil {
		t.Error(err)
	}

	var udpstats = info.UdpStat{
		Listen:   2,
		Dropped:  4,
		RxQueued: 10,
		TxQueued: 11,
	}

	if stats != udpstats {
		t.Errorf("Expected %#v, got %#v", udpstats, stats)
	}
}
