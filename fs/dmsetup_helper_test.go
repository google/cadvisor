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

// +build linux

package fs

import (
	"testing"
)

type testDmsetup struct {
	data []byte
	err  error
}

func (t *testDmsetup) table(poolName string) ([]byte, error) {
	return t.data, t.err
}

var dmTableTests = []struct {
	dmTable     string
	major       uint
	minor       uint
	dataBlkSize uint
	errExpected bool
}{
	{`0 409534464 thin-pool 253:6 253:7 128 32768 1 skip_block_zeroing`, 253, 7, 128, false},
	{`0 409534464 thin-pool 253:6 258:9 512 32768 1 skip_block_zeroing otherstuff`, 258, 9, 512, false},
	{`Invalid status line`, 0, 0, 0, false},
}

func TestParseDMTable(t *testing.T) {
	for _, tt := range dmTableTests {
		major, minor, dataBlkSize, err := parseDMTable(tt.dmTable)
		if tt.errExpected && err != nil {
			t.Errorf("parseDMTable(%q) expected error", tt.dmTable)
		}
		if major != tt.major {
			t.Errorf("parseDMTable(%q) wrong major value => %q, want %q", tt.dmTable, major, tt.major)
		}
		if minor != tt.minor {
			t.Errorf("parseDMTable(%q) wrong minor value => %q, want %q", tt.dmTable, minor, tt.minor)
		}
		if dataBlkSize != tt.dataBlkSize {
			t.Errorf("parseDMTable(%q) wrong dataBlkSize value => %q, want %q", tt.dmTable, dataBlkSize, tt.dataBlkSize)
		}
	}
}

var dmStatusTests = []struct {
	dmStatus    string
	used        uint64
	total       uint64
	errExpected bool
}{
	{`0 409534464 thin-pool 64085 3705/4161600 88106/3199488 - rw no_discard_passdown queue_if_no_space -`, 88106, 3199488, false},
	{`0 209715200 thin-pool 707 1215/524288 30282/1638400 - rw discard_passdown`, 30282, 1638400, false},
	{`Invalid status line`, 0, 0, false},
}

func TestParseDMStatus(t *testing.T) {
	for _, tt := range dmStatusTests {
		used, total, err := parseDMStatus(tt.dmStatus)
		if tt.errExpected && err != nil {
			t.Errorf("parseDMStatus(%q) expected error", tt.dmStatus)
		}
		if used != tt.used {
			t.Errorf("parseDMStatus(%q) wrong used value => %q, want %q", tt.dmStatus, used, tt.used)
		}
		if total != tt.total {
			t.Errorf("parseDMStatus(%q) wrong total value => %q, want %q", tt.dmStatus, total, tt.total)
		}
	}
}
