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

package v1

import "github.com/google/cadvisor/lib/model"

// FsInfo is the machine-level filesystem type (model.FilesystemInfo in the library).
type FsInfo = model.FilesystemInfo

type Node = model.Node

type Core = model.Core

type Cache = model.Cache

type HugePagesInfo = model.HugePagesInfo

type DiskInfo = model.DiskInfo

type NetInfo = model.NetInfo

type CloudProvider = model.CloudProvider

const (
	GCE             CloudProvider = "GCE"
	AWS             CloudProvider = "AWS"
	Azure           CloudProvider = "Azure"
	UnknownProvider CloudProvider = "Unknown"
)

type InstanceType = model.InstanceType

const (
	UnknownInstance = "Unknown"
)

type InstanceID = model.InstanceID

const (
	UnNamedInstance InstanceID = "None"
)

type MachineInfo = model.MachineInfo

type MemoryInfo = model.MemoryInfo

type NVMInfo = model.NVMInfo

type VersionInfo = model.VersionInfo

type MachineInfoFactory = model.MachineInfoFactory
