// Copyright 2017 Google Inc. All Rights Reserved.
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
	"github.com/google/cadvisor/storage"
)

func init() {
	storage.RegisterStorageDriver("json", new)
}

type jsonStorage struct {
	description string
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
		*argDescription,
		*argProtocol,
	)
}

type DetailSpec struct {
	Description    string               `json:"description,omitempty"`
	MachineName    string               `json:"machine_name,omitempty"`
	ContainerName  string               `json:"container_name,omitempty"`
	ContainerStats *info.ContainerStats `json:"container_stats,omitempty"`
}

var (
	// network protocol: either udp or tcp
	argProtocol = flag.String("storage_driver_json_protocol", "udp", "Json storage driver protocol")
	// useful if a user wants to pass any extra information in the json, such as an identifying string
	argDescription = flag.String("storage_driver_json_description", "", "Optional description for this connection")
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
	output, err := json.Marshal(DetailSpec{self.description, self.machineName, containerName, stats})
	if err != nil {
		return err
	}
	// write json object to tcp/udp connection
	_, err = self.connection.Write(output)
	return err
}

func (driver *jsonStorage) Close() error {
	return driver.connection.Close()
}

func newStorage(machineName string, storageHost string, description string, protocol string) (*jsonStorage, error) {
	// create tcp/udp connection to host
	connection, err := net.Dial(protocol, storageHost)
	if err != nil {
		return nil, err
	}
	jsonStorage := &jsonStorage{
		machineName: machineName,
		description: description,
		connection:  connection,
	}
	return jsonStorage, nil
}
