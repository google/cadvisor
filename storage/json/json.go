// Copyright 2015 Google Inc. All Rights Reserved.
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

package json

import (
	"encoding/json"
	"flag"
	"net"
	"os"

	info "github.com/google/cadvisor/info/v1"
	storage "github.com/google/cadvisor/storage"
)

func init() {
	storage.RegisterStorageDriver("json", new)
}

type jsonStorage struct {
	infoField   string
	machineName string
	connection  net.Conn
}

func new() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return newStorage(
		hostname,
		*storage.ArgDbHost,
		*argInfoField,
		*argProtocol,
	)
}

type DetailSpec struct {
	InfoField      string               `json:"info,omitempty"`
	MachineName    string               `json:"machine_name,omitempty"`
	ContainerName  string               `json:"container_name,omitempty"`
	ContainerStats *info.ContainerStats `json:"stats,omitempty"`
}

var (
	// network protocol: either udp or tcp
	argProtocol = flag.String("storage_driver_json_protocol", "udp", "Json storage driver protocol")
	// useful if a user wants to pass any extra information in the json, such as an identifying string
	argInfoField = flag.String("storage_driver_json_info_field", "", "Optional additional info")
)

func (self *jsonStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}
	containerName := ref.Name
	if len(ref.Aliases) > 0 {
		containerName = ref.Aliases[0]
	}
	// convert stats into json object
	output, err := json.Marshal(DetailSpec{self.infoField, self.machineName, containerName, stats})
	if err != nil {
		return err
	}
	// write json object to udp connection
	_, err = self.connection.Write(output)
	return err
}

func (driver *jsonStorage) Close() error {
	return nil
}

func newStorage(machineName string, storageHost string, infoField string, protocol string) (*jsonStorage, error) {
	// create udp connection to host
	connection, err := net.Dial(protocol, storageHost)
	if err != nil {
		return nil, err
	}
	jsonStorage := &jsonStorage{
		machineName: machineName,
		infoField:   infoField,
		connection:  connection,
	}
	return jsonStorage, nil
}
