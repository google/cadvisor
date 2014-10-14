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

// Unmarshal's a Containers description json file. The json file contains
// an array of ContainerDesc structs, each with a container's id and network_interface
// This allows collecting stats about network interfaces configured outside docker
// and lxc
package lxc

import (
	"encoding/json"
	"io/ioutil"
)

type ContainersDesc struct {
	All_hosts []ContainerDesc
}

type ContainerDesc struct {
	Id                string
	Network_interface *NetworkInterface
}

type NetworkInterface struct {
	VethHost  string
	VethChild string
	NsPath    string
}

func Unmarshal(containerDescFile string) (ContainersDesc, error) {
	dat, err := ioutil.ReadFile(containerDescFile)
	var cDesc ContainersDesc
	if err == nil {
		err = json.Unmarshal(dat, &cDesc)
	}
	return cDesc, err
}
