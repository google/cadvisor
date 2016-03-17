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
	"github.com/axibase/atsd-api-go/http"
	"net/url"
	"time"
)

type StorageFactory interface {
	Create() *Storage
}

type NetworkStorageFactory struct {
	selfMetricsEntity string
	metricPrefix      string
	memstoreLimit     uint64
	protocol          string
	receiverHostport  string
	connectionLimit   uint
	updateInterval    time.Duration
	url               *url.URL
	username          string
	password          string
	groupParams       map[string]DeduplicationParams
}

func NewNetworkStorageFactory(
	selfMetricsEntity,
	protocol,
	receiverHostport string,
	url url.URL,
	username,
	password string,
	memstoreLimit uint64,
	connectionLimit uint,
	updateInterval time.Duration,
	metricPrefix string,
	groupParams map[string]DeduplicationParams,
) *NetworkStorageFactory {
	return &NetworkStorageFactory{
		selfMetricsEntity: selfMetricsEntity,
		memstoreLimit:     memstoreLimit,
		protocol:          protocol,
		receiverHostport:  receiverHostport,
		connectionLimit:   connectionLimit,
		updateInterval:    updateInterval,
		metricPrefix:      metricPrefix,
		url:               &url,
		username:          username,
		password:          password,
		groupParams:       groupParams,
	}
}

func (self *NetworkStorageFactory) Create() *Storage {
	memstore := NewMemStore(self.memstoreLimit)
	writeCommunicator := NewNetworkCommunicator(self.connectionLimit, self.protocol, self.receiverHostport)
	compactorBuffer := map[string]map[string]sample{}
	for group := range self.groupParams {
		compactorBuffer[group] = map[string]sample{}
	}
	storage := &Storage{
		selfMetricsEntity:      self.selfMetricsEntity,
		memstore:               memstore,
		dataCompacter:          &DataCompacter{GroupParams: self.groupParams, Buffer: compactorBuffer},
		writeCommunicator:      writeCommunicator,
		updateInterval:         self.updateInterval,
		selfMetricSendInterval: 15 * time.Second,
		isUpdating:             false,
		metricPrefix:           self.metricPrefix,
		atsdHttpClient:         http.New(*self.url, self.username, self.password),
	}

	return storage
}

func NewHttpStorageFactory(
	selfMetricsEntity string,
	url url.URL,
	username,
	password string,
	memstoreLimit uint64,
	updateInterval time.Duration,
	metricPrefix string,
	groupParams map[string]DeduplicationParams,
) *HttpStorageFactory {
	return &HttpStorageFactory{
		selfMetricsEntity: selfMetricsEntity,
		memstoreLimit:     memstoreLimit,
		url:               &url,
		username:          username,
		password:          password,
		updateInterval:    updateInterval,
		metricPrefix:      metricPrefix,
		groupParams:       groupParams,
	}
}

type HttpStorageFactory struct {
	selfMetricsEntity string
	memstoreLimit     uint64

	url      *url.URL
	username string
	password string

	updateInterval time.Duration
	metricPrefix   string
	groupParams    map[string]DeduplicationParams
}

func (self *HttpStorageFactory) Create() *Storage {
	memstore := NewMemStore(self.memstoreLimit)
	client := http.New(*self.url, self.username, self.password)
	writeCommunicator := NewHttpCommunicator(client)
	compactorBuffer := map[string]map[string]sample{}
	for group := range self.groupParams {
		compactorBuffer[group] = map[string]sample{}
	}
	storage := &Storage{
		selfMetricsEntity:      self.selfMetricsEntity,
		memstore:               memstore,
		dataCompacter:          &DataCompacter{GroupParams: self.groupParams, Buffer: compactorBuffer},
		writeCommunicator:      writeCommunicator,
		updateInterval:         self.updateInterval,
		selfMetricSendInterval: 15 * time.Second,
		isUpdating:             false,
		atsdHttpClient:         client,
		metricPrefix:           self.metricPrefix,
	}
	return storage
}

func NewFactoryFromConfig(config Config) StorageFactory {
	switch config.Protocol {
	case "udp", "tcp":
		return NewNetworkStorageFactory(
			config.SelfMetricEntity,
			config.Protocol,
			config.DataReceiverHostport,
			*config.Url,
			config.Username,
			config.Password,
			config.MemstoreLimit,
			config.ConnectionLimit,
			config.UpdateInterval,
			config.MetricPrefix,
			config.GroupParams,
		)
	case "http/https":
		return NewHttpStorageFactory(
			config.SelfMetricEntity,
			*config.Url,
			config.Username,
			config.Password,
			config.MemstoreLimit,
			config.UpdateInterval,
			config.MetricPrefix,
			config.GroupParams,
		)
	default:
		return NewHttpStorageFactory(
			config.SelfMetricEntity,
			*config.Url,
			config.Username,
			config.Password,
			config.MemstoreLimit,
			config.UpdateInterval,
			config.MetricPrefix,
			config.GroupParams,
		)
	}
}
