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

package logstash

import (
	"encoding/json"
	"flag"
	"net"
	"os"

	info "github.com/google/cadvisor/info/v1"
	storage "github.com/google/cadvisor/storage"
)

func init() {
	storage.RegisterStorageDriver("logstash", new)
}

type logstashStorage struct {
	logstashType string
	machineName  string
	connection   net.Conn
}

func new() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return newStorage(
		hostname,
		*storage.ArgDbHost,
		*argLogstashType,
	)
}

type DetailSpec struct {
	LogstashType   string               `json:"type,omitempty"`
	MachineName    string               `json:"machine_name,omitempty"`
	ContainerName  string               `json:"container_name,omitempty"`
	ContainerStats *info.ContainerStats `json:"stats,omitempty"`
}

var (
	// lets users set the type for the stats sent to logstash
	argLogstashType = flag.String("storage_driver_logstash_type", "cadvisor", "Logstash type name")
)

func (self *logstashStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}
	containerName := ref.Name
	if len(ref.Aliases) > 0 {
		containerName = ref.Aliases[0]
	}
	// convert stats into json object
	output, err := json.Marshal(DetailSpec{self.logstashType, self.machineName, containerName, stats})
	if err != nil {
		return err
	}
	// write json object to udp connection
	_, err = self.connection.Write(output)
	return err
}

func (driver *logstashStorage) Close() error {
	return nil
}

func newStorage(machineName string, logstashHost string, logstashType string) (*logstashStorage, error) {
	// create udp connection to logstash host
	connection, err := net.Dial("udp", logstashHost)
	if err != nil {
		return nil, err
	}
	logstashStorage := &logstashStorage{
		machineName:  machineName,
		logstashType: logstashType,
		connection:   connection,
	}
	return logstashStorage, nil
}
