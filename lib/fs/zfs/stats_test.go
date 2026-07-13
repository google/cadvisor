// Copyright 2024 Google Inc. All Rights Reserved.
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

package zfs

import "testing"

func TestParseZfsListUsage(t *testing.T) {
	// `zfs list -Hp -o used,available,usedbydataset <name>` -> one tab-separated line.
	capacity, free, avail, err := parseZfsListUsage([]byte("123456\t789012\t654321\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := uint64(123456 + 789012 + 654321); capacity != want {
		t.Errorf("capacity = %d, want %d", capacity, want)
	}
	if free != 789012 || avail != 789012 {
		t.Errorf("free/available = %d/%d, want 789012/789012", free, avail)
	}

	// A "-" property value is treated as 0 (matches go-zfs's setUint).
	capacity, _, _, err = parseZfsListUsage([]byte("100\t-\t50\n"))
	if err != nil {
		t.Fatalf("dash value: unexpected error: %v", err)
	}
	if capacity != 150 {
		t.Errorf("capacity with dash = %d, want 150", capacity)
	}

	// Wrong field count must error rather than silently misparse.
	if _, _, _, err := parseZfsListUsage([]byte("100\t200\n")); err == nil {
		t.Error("expected error for 2-field output")
	}
	// Non-numeric values must error.
	if _, _, _, err := parseZfsListUsage([]byte("a\tb\tc\n")); err == nil {
		t.Error("expected error for non-numeric output")
	}
}
