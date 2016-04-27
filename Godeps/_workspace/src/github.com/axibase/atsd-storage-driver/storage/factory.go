/*
* Copyright 2015 Axibase Corporation or its affiliates. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License").
* You may not use this file except in compliance with the License.
* A copy of the License is located at
*
* https://www.axibase.com/atsd/axibase-apache-2.0.pdf
*
* or in the "license" file accompanying this file. This file is distributed
* on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
* express or implied. See the License for the specific language governing
* permissions and limitations under the License.
 */

package storage

import (
	"net/url"
	"time"

	"github.com/axibase/atsd-api-go/http"
)

type StorageFactory interface {
	Create() (*Storage, error)
}

type NetworkStorageFactory struct {
	selfMetricsEntity    string
	url                  *url.URL
	metricPrefix         string
	memstoreLimit        uint
	senderGoroutineLimit int
	updateInterval       time.Duration
	groupParams          map[string]DeduplicationParams
}

func NewNetworkStorageFactory(
	selfMetricsEntity string,
	url *url.URL,
	memstoreLimit uint,
	senderGoroutineLimit int,
	updateInterval time.Duration,
	metricPrefix string,
	groupParams map[string]DeduplicationParams,
) *NetworkStorageFactory {
	return &NetworkStorageFactory{
		selfMetricsEntity:    selfMetricsEntity,
		memstoreLimit:        memstoreLimit,
		url:                  url,
		senderGoroutineLimit: senderGoroutineLimit,
		updateInterval:       updateInterval,
		metricPrefix:         metricPrefix,
		groupParams:          groupParams,
	}
}

func (self *NetworkStorageFactory) Create() (*Storage, error) {
	memstore, err := NewMemStore(self.memstoreLimit)
	if err != nil {
		return nil, err
	}
	writeCommunicator, err := NewNetworkCommunicator(self.senderGoroutineLimit, self.url)
	if err != nil {
		return nil, err
	}
	storage := &Storage{
		selfMetricsEntity:      self.selfMetricsEntity,
		memstore:               memstore,
		dataCompacter:          NewDataCompacter(self.groupParams),
		writeCommunicator:      writeCommunicator,
		updateInterval:         self.updateInterval,
		selfMetricSendInterval: 15 * time.Second,
		isUpdating:             false,
		metricPrefix:           self.metricPrefix,
	}

	return storage, nil
}

func NewHttpStorageFactory(
	selfMetricsEntity string,
	url *url.URL,
	insecureSkipVerify bool,
	memstoreLimit uint,
	updateInterval time.Duration,
	metricPrefix string,
	groupParams map[string]DeduplicationParams,
) *HttpStorageFactory {
	return &HttpStorageFactory{
		selfMetricsEntity:  selfMetricsEntity,
		memstoreLimit:      memstoreLimit,
		url:                url,
		insecureSkipVerify: insecureSkipVerify,
		updateInterval:     updateInterval,
		metricPrefix:       metricPrefix,
		groupParams:        groupParams,
	}
}

type HttpStorageFactory struct {
	selfMetricsEntity string
	memstoreLimit     uint

	url                *url.URL
	insecureSkipVerify bool
	updateInterval     time.Duration
	metricPrefix       string
	groupParams        map[string]DeduplicationParams
}

func (self *HttpStorageFactory) Create() (*Storage, error) {
	memstore, err := NewMemStore(self.memstoreLimit)
	if err != nil {
		return nil, err
	}
	client := http.New(*self.url, self.insecureSkipVerify)
	writeCommunicator := NewHttpCommunicator(client)
	storage := &Storage{
		selfMetricsEntity:      self.selfMetricsEntity,
		memstore:               memstore,
		dataCompacter:          NewDataCompacter(self.groupParams),
		writeCommunicator:      writeCommunicator,
		updateInterval:         self.updateInterval,
		selfMetricSendInterval: 15 * time.Second,
		isUpdating:             false,
		metricPrefix:           self.metricPrefix,
	}
	return storage, nil
}

func NewFactoryFromConfig(config Config) StorageFactory {
	switch config.Url.Scheme {
	case "udp", "tcp":
		return NewNetworkStorageFactory(
			config.SelfMetricEntity,
			config.Url,
			config.MemstoreLimit,
			config.SenderGoroutineLimit,
			config.UpdateInterval,
			config.MetricPrefix,
			config.GroupParams,
		)
	case "http", "https":
		return NewHttpStorageFactory(
			config.SelfMetricEntity,
			config.Url,
			config.InsecureSkipVerify,
			config.MemstoreLimit,
			config.UpdateInterval,
			config.MetricPrefix,
			config.GroupParams,
		)
	default:
		return NewHttpStorageFactory(
			config.SelfMetricEntity,
			config.Url,
			config.InsecureSkipVerify,
			config.MemstoreLimit,
			config.UpdateInterval,
			config.MetricPrefix,
			config.GroupParams,
		)
	}
}
