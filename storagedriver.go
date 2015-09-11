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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/cache/memory"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/storage/bigquery"
	"github.com/google/cadvisor/storage/elasticsearch"
	"github.com/google/cadvisor/storage/influxdb"
	"github.com/google/cadvisor/storage/redis"
	"github.com/google/cadvisor/storage/statsd"
	"github.com/google/cadvisor/storage/stdout"
)

var argDbUsername = flag.String("storage_driver_user", "root", "database username")
var argDbPassword = flag.String("storage_driver_password", "root", "database password")
var argDbHost = flag.String("storage_driver_host", "localhost:8086", "database host:port")
var argDbName = flag.String("storage_driver_db", "cadvisor", "database name")
var argDbTable = flag.String("storage_driver_table", "stats", "table name")
var argDbIsSecure = flag.Bool("storage_driver_secure", false, "use secure connection with database")
var argDbBufferDuration = flag.Duration("storage_driver_buffer_duration", 60*time.Second, "Writes in the storage driver will be buffered for this duration, and committed to the non memory backends as a single transaction")
var storageDuration = flag.Duration("storage_duration", 2*time.Minute, "How long to keep data stored (Default: 2min).")
var argElasticHost = flag.String("storage_driver_es_host", "http://localhost:9200", "ElasticSearch host:port")
var argIndexName = flag.String("storage_driver_es_index", "cadvisor", "ElasticSearch index name")
var argTypeName = flag.String("storage_driver_es_type", "stats", "ElasticSearch type name")
var argEnableSniffer = flag.Bool("storage_driver_es_enable_sniffer", false, "ElasticSearch uses a sniffing process to find all nodes of your cluster by default, automatically")

// Creates a memory storage with an optional backend storage option.
func NewMemoryStorage(backendStorageName string) (*memory.InMemoryCache, error) {
	var storageDriver *memory.InMemoryCache
	var backendStorage storage.StorageDriver
	var err error
	switch backendStorageName {
	case "":
		backendStorage = nil
	case "influxdb":
		var hostname string
		hostname, err = os.Hostname()
		if err != nil {
			return nil, err
		}

		backendStorage, err = influxdb.New(
			hostname,
			*argDbTable,
			*argDbName,
			*argDbUsername,
			*argDbPassword,
			*argDbHost,
			*argDbIsSecure,
			*argDbBufferDuration,
		)
	case "bigquery":
		var hostname string
		hostname, err = os.Hostname()
		if err != nil {
			return nil, err
		}
		backendStorage, err = bigquery.New(
			hostname,
			*argDbTable,
			*argDbName,
		)
	case "redis":
		//machineName: We use os.Hostname as the machineName (A unique identifier to identify the host that runs the current cAdvisor)
		//argDbName: the key for redis's data
		//argDbHost: the redis's server host
		var machineName string
		machineName, err = os.Hostname()
		if err != nil {
			return nil, err
		}
		backendStorage, err = redis.New(
			machineName,
			*argDbName,
			*argDbHost,
			*argDbBufferDuration,
		)
	case "elasticsearch":
		//argIndexName: the index for elasticsearch
		//argTypeName: the type for index
		//argElasticHost: the elasticsearch's server host
		var machineName string
		machineName, err = os.Hostname()
		if err != nil {
			return nil, err
		}
		backendStorage, err = elasticsearch.New(
			machineName,
			*argIndexName,
			*argTypeName,
			*argElasticHost,
			*argEnableSniffer,
		)
	case "statsd":
		backendStorage, err = statsd.New(
			*argDbName,
			*argDbHost,
		)
	case "stdout":
		backendStorage, err = stdout.New(
			*argDbHost,
		)
	default:
		err = fmt.Errorf("unknown backend storage driver: %v", *argDbDriver)
	}
	if err != nil {
		return nil, err
	}
	if backendStorageName != "" {
		glog.Infof("Using backend storage type %q", backendStorageName)
	} else {
		glog.Infof("No backend storage selected")
	}
	glog.Infof("Caching stats in memory for %v", *storageDuration)
	storageDriver = memory.New(*storageDuration, backendStorage)
	return storageDriver, nil
}
