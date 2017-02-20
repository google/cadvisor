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
	"strings"
	"sync"
	"time"

	info "github.com/google/cadvisor/info/v1"
	storage "github.com/google/cadvisor/storage"

	"gopkg.in/olivere/elastic.v5"

	"golang.org/x/net/context"
)

func init() {
	storage.RegisterStorageDriver("elasticsearch.v5", newV5)
}

type elasticStorageV5 struct {
	client      *elastic.Client
	machineName string
	indexName   string
	typeName    string
	lock        sync.Mutex
	ctx         context.Context
}

type detailSpecV5 struct {
	Timestamp      int64                `json:"timestamp"`
	MachineName    string               `json:"machine_name,omitempty"`
	ContainerName  string               `json:"container_Name,omitempty"`
	ContainerStats *info.ContainerStats `json:"container_stats,omitempty"`
}

var (
	argBasicAuth        = flag.String("storage_driver_es_basic_auth", "", "ElasticSearch basic auth: user:password")
	argSnifferTimeout   = flag.Int("storage_driver_es_sniffer_timeout", 2, "The time before Elastic times out on sniffing nodes in seconds")
	argSnifferTimeoutSt = flag.Int("storage_driver_es_sniffer_timeout_startup", 5, "The sniffing timeout used while creating a new client")
	argSnifferInterval  = flag.Int("storage_driver_es_sniffer_interval", 15*60, "The interval between two sniffer processes")

	argEnableHealthCheck    = flag.Bool("storage_driver_es_enable_health_check", true, "Enable health check")
	argHealthCheckTimeout   = flag.Int("storage_driver_es_health_check_timeout", 1, "The timeout for health checks")
	argHealthCheckTimeoutSt = flag.Int("storage_driver_es_health_check_timeout_startup", 5, "The health check timeout used while creating a new client")
	argHealthCheckInterval  = flag.Int("storage_driver_es_health_check_interval", 60, "The interval between two health checks")
)

func newV5() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return newStorageV5(
		hostname,
		*argIndexName,
		*argTypeName,
		*argElasticHost,
	)
}

func (self *elasticStorageV5) containerStatsAndDefaultValues(
	ref info.ContainerReference, stats *info.ContainerStats) *detailSpecV5 {
	timestamp := stats.Timestamp.UnixNano() / 1E3
	var containerName string
	if len(ref.Aliases) > 0 {
		containerName = ref.Aliases[0]
	} else {
		containerName = ref.Name
	}
	detail := &detailSpecV5{
		Timestamp:      timestamp,
		MachineName:    self.machineName,
		ContainerName:  containerName,
		ContainerStats: stats,
	}
	return detail
}

func (self *elasticStorageV5) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
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
		_, err := self.client.Index().
			Index(self.indexName).
			Type(self.typeName).
			BodyJson(detail).
			Do(self.ctx)
		if err != nil {
			// Handle error
			fmt.Printf("failed to write stats to ElasticSearch - %s", err)
			return
		}
	}()
	return nil
}

func (self *elasticStorageV5) Close() error {
	self.client = nil
	return nil
}

func newStorageV5(
	machineName,
	indexName,
	typeName,
	elasticHost string,
) (storage.StorageDriver, error) {
	ctx := context.Background()

	client, err := createClient(&ctx)
	if err != nil {
		// Handle error
		return nil, fmt.Errorf("failed to create the elasticsearch client - %s", err)
	}

	// Ping the Elasticsearch server to get e.g. the version number
	info, code, err := client.Ping(elasticHost).Do(ctx)
	if err != nil {
		// Handle error
		return nil, fmt.Errorf("failed to ping the elasticsearch - %s", err)

	}
	fmt.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	ret := &elasticStorageV5{
		client:      client,
		machineName: machineName,
		indexName:   indexName,
		typeName:    typeName,
		ctx:         ctx,
	}
	return ret, nil
}

func createClient(ctx *context.Context) (*elastic.Client, error) {
	options := make([]elastic.ClientOptionFunc, 0, 20)

	options = append(options, elastic.SetHealthcheck(*argEnableHealthCheck))
	options = append(options, elastic.SetHealthcheckTimeout(time.Duration(*argHealthCheckTimeout)*time.Second))
	options = append(options, elastic.SetHealthcheckTimeoutStartup(time.Duration(*argHealthCheckTimeoutSt)*time.Second))
	options = append(options, elastic.SetHealthcheckInterval(time.Duration(*argHealthCheckInterval)*time.Second))

	options = append(options, elastic.SetSniff(*argEnableSniffer))
	options = append(options, elastic.SetSnifferTimeout(time.Duration(*argSnifferTimeout)*time.Second))
	options = append(options, elastic.SetSnifferTimeoutStartup(time.Duration(*argSnifferTimeoutSt)*time.Second))
	options = append(options, elastic.SetURL(*argElasticHost))

	options = append(options, elastic.SetSnifferInterval(time.Duration(*argSnifferInterval)*time.Second))

	basicAuth := *argBasicAuth
	pos := strings.Index(basicAuth, ":")
	if -1 < pos {
		options = append(options, elastic.SetBasicAuth(basicAuth[0:pos], basicAuth[(pos+1):]))
	}

	return elastic.NewClient(options...)
}
