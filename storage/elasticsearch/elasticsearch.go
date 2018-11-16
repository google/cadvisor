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

package elasticsearch

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	info "github.com/google/cadvisor/info/v1"
	storage "github.com/google/cadvisor/storage"

	"gopkg.in/olivere/elastic.v2"
)

func init() {
	storage.RegisterStorageDriver("elasticsearch", new)
}

type elasticStorage struct {
	client      *elastic.Client
	machineName string
	indexName   string
	typeName    string
	lock        sync.Mutex
}

type detailSpec struct {
	Timestamp      int64                `json:"timestamp"`
	MachineName    string               `json:"machine_name,omitempty"`
	ContainerName  string               `json:"container_Name,omitempty"`
	ContainerStats *info.ContainerStats `json:"container_stats,omitempty"`
}

var (
	argElasticHost   = flag.String("storage_driver_es_host", "http://localhost:9200", "ElasticSearch host:port")
	argIndexName     = flag.String("storage_driver_es_index", "cadvisor", "ElasticSearch index name")
	argTypeName      = flag.String("storage_driver_es_type", "stats", "ElasticSearch type name")
	argEnableSniffer = flag.Bool("storage_driver_es_enable_sniffer", false, "ElasticSearch uses a sniffing process to find all nodes of your cluster by default, automatically")
)

func new() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(
		hostname,
		*argIndexName,
		*argTypeName,
		*argElasticHost,
		*argEnableSniffer,
	)
}

func (self *elasticStorage) containerStatsAndDefaultValues(
	cInfo *info.ContainerInfo, stats *info.ContainerStats) *detailSpec {
	timestamp := stats.Timestamp.UnixNano() / 1E3
	var containerName string
	if len(cInfo.ContainerReference.Aliases) > 0 {
		containerName = cInfo.ContainerReference.Aliases[0]
	} else {
		containerName = cInfo.ContainerReference.Name
	}
	detail := &detailSpec{
		Timestamp:      timestamp,
		MachineName:    self.machineName,
		ContainerName:  containerName,
		ContainerStats: stats,
	}
	return detail
}

func (self *elasticStorage) AddStats(cInfo *info.ContainerInfo, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}
	func() {
		// AddStats will be invoked simultaneously from multiple threads and only one of them will perform a write.
		self.lock.Lock()
		defer self.lock.Unlock()
		// Add some default params based on ContainerStats
		detail := self.containerStatsAndDefaultValues(cInfo, stats)
		// Index a cadvisor (using JSON serialization)
		_, err := self.client.Index().
			Index(self.indexName).
			Type(self.typeName).
			BodyJson(detail).
			Do()
		if err != nil {
			// Handle error
			fmt.Printf("failed to write stats to ElasticSearch - %s", err)
			return
		}
	}()
	return nil
}

func (self *elasticStorage) Close() error {
	self.client = nil
	return nil
}

// machineName: A unique identifier to identify the host that current cAdvisor
// instance is running on.
// ElasticHost: The host which runs ElasticSearch.
func newStorage(
	machineName,
	indexName,
	typeName,
	elasticHost string,
	enableSniffer bool,
) (storage.StorageDriver, error) {
	// Obtain a client and connect to the default Elasticsearch installation
	// on 127.0.0.1:9200. Of course you can configure your client to connect
	// to other hosts and configure it in various other ways.
	client, err := elastic.NewClient(
		elastic.SetHealthcheck(true),
		elastic.SetSniff(enableSniffer),
		elastic.SetHealthcheckInterval(30*time.Second),
		elastic.SetURL(elasticHost),
	)
	if err != nil {
		// Handle error
		return nil, fmt.Errorf("failed to create the elasticsearch client - %s", err)
	}

	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping().URL(elasticHost).Do()
	if err != nil {
		// Handle error
		return nil, fmt.Errorf("failed to ping the elasticsearch - %s", err)

	}
	fmt.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	ret := &elasticStorage{
		client:      client,
		machineName: machineName,
		indexName:   indexName,
		typeName:    typeName,
	}
	return ret, nil
}
