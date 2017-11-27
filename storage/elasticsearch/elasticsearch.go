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

	"context"
	elasticLegacy "gopkg.in/olivere/elastic.v2"
	elastic "gopkg.in/olivere/elastic.v5"
	"reflect"
	"strconv"
	"strings"
)

func init() {
	storage.RegisterStorageDriver("elasticsearch", new)
}

type elasticStorage struct {
	client      ElasticClient
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

type ElasticClient interface{}

func findElasticVersion(elasticHost string, enableSniffer bool) (string, error) {
	client, err := createElasticClient(elasticHost, enableSniffer)

	if err != nil {
		return "", err
	}

	pingInfo, code, err := client.Ping(elasticHost).Do(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to ping the elasticsearch - %s", err)
	}

	fmt.Printf("Elasticsearch returned with code %d and version %s", code, pingInfo.Version.Number)

	return pingInfo.Version.Number, nil
}

func createElasticClient(elasticHost string, enableSniffer bool) (*elastic.Client, error) {
	client, err := elastic.NewClient(
		elastic.SetHealthcheck(true),
		elastic.SetSniff(enableSniffer),
		elastic.SetHealthcheckInterval(30*time.Second),
		elastic.SetURL(elasticHost),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create the elasticsearch client - %s", err)
	}

	return client, nil
}

func createLegacyElasticClient(elasticHost string, enableSniffer bool) (*elasticLegacy.Client, error) {
	client, err := elasticLegacy.NewClient(
		elasticLegacy.SetHealthcheck(true),
		elasticLegacy.SetSniff(enableSniffer),
		elasticLegacy.SetHealthcheckInterval(30*time.Second),
		elasticLegacy.SetURL(elasticHost),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create the elasticsearch legacy client - %s", err)
	}

	return client, nil
}

func createElasticClientByVersion(version string, elasticHost string, enableSniffer bool) (ElasticClient, error) {
	majorV, _ := strconv.Atoi(strings.Split(version, ".")[0])

	if majorV >= 5 {
		return createElasticClient(elasticHost, enableSniffer)
	} else {
		return createLegacyElasticClient(elasticHost, enableSniffer)
	}
}

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
	ref info.ContainerReference, stats *info.ContainerStats) *detailSpec {
	timestamp := stats.Timestamp.UnixNano() / 1E3
	var containerName string
	if len(ref.Aliases) > 0 {
		containerName = ref.Aliases[0]
	} else {
		containerName = ref.Name
	}
	detail := &detailSpec{
		Timestamp:      timestamp,
		MachineName:    self.machineName,
		ContainerName:  containerName,
		ContainerStats: stats,
	}
	return detail
}

func (self *elasticStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}
	func() {
		// AddStats will be invoked simultaneously from multiple threads and only one of them will perform a write.
		self.lock.Lock()
		defer self.lock.Unlock()
		// Add some default params based on ContainerStats
		detail := self.containerStatsAndDefaultValues(ref, stats)
		// Index a cadvisor (using JSON serialization)
		err := self.Send(detail)
		if err != nil {
			// Handle error
			fmt.Printf("failed to write stats to ElasticSearch - %s", err)
			return
		}
	}()
	return nil
}

func (self *elasticStorage) Send(detail *detailSpec) error {
	var err error
	switch client := self.client.(type) {
	case *elasticLegacy.Client:
		_, err = client.Index().Index(self.indexName).Type(self.typeName).BodyJson(detail).Do()
	case *elastic.Client:
		_, err = client.Index().Index(self.indexName).Type(self.typeName).BodyJson(detail).Do(context.Background())
	default:
		err = fmt.Errorf("unknow elastic client of type %s", reflect.TypeOf(client))
	}

	return err
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
	version, err := findElasticVersion(elasticHost, enableSniffer)
	if err != nil {
		return nil, err
	}

	client, err := createElasticClientByVersion(version, elasticHost, enableSniffer)
	if err != nil {
		return nil, err
	}

	ret := &elasticStorage{
		client:      client,
		machineName: machineName,
		indexName:   indexName,
		typeName:    typeName,
	}
	return ret, nil
}
