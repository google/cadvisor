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

package info

type MachineInfo struct {
	// The number of cores in this machine.
	NumCores int `json:"num_cores"`

	// The amount of memory (in bytes) in this machine
	MemoryCapacity int64 `json:"memory_capacity"`
}

type MachineInfoFactory interface {
	GetMachineInfo() (*MachineInfo, error)
}
