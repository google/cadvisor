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
package raw

import (
	"flag"
	"encoding/json"
	"io/ioutil"
)
var argContainersDesc = flag.String("cDescription", "/etc/docker/cdesc.json", "container description file")
type containersDesc struct {
	All_hosts []containerDesc
}

type containerDesc struct {
	Id                string
	Network_interface *networkInterface
}

type networkInterface struct {
	VethHost  string
	VethChild string
	NsPath    string
}

func Unmarshal(containerDescFile string) (containersDesc, error) {
	dat, err := ioutil.ReadFile(containerDescFile)
	var cDesc containersDesc
	if err == nil {
		err = json.Unmarshal(dat, &cDesc)
	}
	return cDesc, err
}
