// Copyright 2023 Google Inc. All Rights Reserved.
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

package v1

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMachineInfoClone(t *testing.T) {
	machineInfo := getFakeMachineInfo()

	// Make sure all fields are populated
	machineInfoReflection := reflect.ValueOf(machineInfo)
	for i := 0; i < machineInfoReflection.NumField(); i++ {
		assert.Falsef(t, machineInfoReflection.Field(i).IsZero(), "field %s is not populated", machineInfoReflection.Type().Field(i).Name)
	}

	clonedMachineInfo := *machineInfo.Clone()
	assert.Equal(t, machineInfo, clonedMachineInfo, "cloned machine info should be equal to the original")
}

func getFakeMachineInfo() MachineInfo {
	return MachineInfo{
		Timestamp:        time.Now(),
		CPUVendorID:      "fake-id",
		NumCores:         1,
		NumPhysicalCores: 2,
		NumSockets:       3,
		CpuFrequency:     4,
		MemoryCapacity:   5,
		SwapCapacity:     6,
		MemoryByType: map[string]*MemoryInfo{
			"fake": {Capacity: 2, DimmCount: 3},
		},
		NVMInfo: NVMInfo{
			MemoryModeCapacity:    2,
			AppDirectModeCapacity: 6,
			AvgPowerBudget:        3,
		},
		HugePages: []HugePagesInfo{{
			PageSize: 512,
			NumPages: 343,
		}},
		MachineID:  "fake-machine-id",
		SystemUUID: "fake-uuid",
		BootID:     "fake-boot-id",
		Filesystems: []FsInfo{{
			Device:      "dev",
			DeviceMajor: 1,
			DeviceMinor: 2,
			Capacity:    3,
			Type:        "fake-type",
			Inodes:      4,
			HasInodes:   true,
		}},
		DiskMap: map[string]DiskInfo{"fake-disk": {
			Name:      "fake",
			Major:     1,
			Minor:     2,
			Size:      3,
			Scheduler: "sched",
		}},
		NetworkDevices: []NetInfo{{
			Name:       "fake-net-info",
			MacAddress: "123",
			Speed:      2,
			Mtu:        3,
		}},
		Topology: []Node{{
			Id:     1,
			Memory: 2,
		}},
		CloudProvider: "fake-provider",
		InstanceType:  "fake-instance-type",
		InstanceID:    "fake-instance-id",
	}
}
