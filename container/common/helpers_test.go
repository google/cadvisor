// Copyright 2018 Google Inc. All Rights Reserved.
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

package common

import (
	"testing"
)

func BenchmarkListDirectories(b *testing.B) {
	for i := 0; i < b.N; i++ {
		output := make(map[string]struct{})
		if err := ListDirectories("/sys/fs/cgroup", "", true, output); err != nil {
			b.Fatal(err)
		}
	}
}

func TestConvertCpuWeightToCpuLimit(t *testing.T) {
	limit, err := convertCPUWeightToCPULimit(1)
	if err != nil {
		t.Fatalf("Error in convertCPUWeightToCPULimit: %s", err)
	}
	if limit != 2 {
		t.Fatalf("convertCPUWeightToCPULimit(1) != 2")
	}
	limit, err = convertCPUWeightToCPULimit(10000)
	if err != nil {
		t.Fatalf("Error in convertCPUWeightToCPULimit: %s", err)
	}
	if limit != 262144 {
		t.Fatalf("convertCPUWeightToCPULimit(10000) != 262144")
	}
	_, err = convertCPUWeightToCPULimit(0)
	if err == nil {
		t.Fatalf("convertCPUWeightToCPULimit(0) must raise an error")
	}
	_, err = convertCPUWeightToCPULimit(10001)
	if err == nil {
		t.Fatalf("convertCPUWeightToCPULimit(10001) must raise an error")
	}
}

func TestParseUint64String(t *testing.T) {
	if parseUint64String("1000") != 1000 {
		t.Fatalf("parseUint64String(\"1000\") != 1000")
	}
	if parseUint64String("-1") != 0 {
		t.Fatalf("parseUint64String(\"-1\") != 0")
	}
	if parseUint64String("0") != 0 {
		t.Fatalf("parseUint64String(\"0\") != 0")
	}
	if parseUint64String("not-a-number") != 0 {
		t.Fatalf("parseUint64String(\"not-a-number\") != 0")
	}
	if parseUint64String(" 1000 ") != 0 {
		t.Fatalf("parseUint64String(\" 1000 \") != 0")
	}
	if parseUint64String("18446744073709551615") != 18446744073709551615 {
		t.Fatalf("parseUint64String(\"18446744073709551615\") != 18446744073709551615")
	}
}
