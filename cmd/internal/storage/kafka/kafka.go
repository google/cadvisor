// Copyright 2016 Google Inc. All Rights Reserved.
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

package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/utils/container"

	kafka "github.com/Shopify/sarama"
	"k8s.io/klog/v2"
)

func init() {
	storage.RegisterStorageDriver("kafka", new)
	kafka.Logger = log.New(os.Stderr, "[kafka]", log.LstdFlags)
}

var (
	brokers   = flag.String("storage_driver_kafka_broker_list", "localhost:9092", "kafka broker(s) csv")
	topic     = flag.String("storage_driver_kafka_topic", "stats", "kafka topic")
	certFile  = flag.String("storage_driver_kafka_ssl_cert", "", "optional certificate file for TLS client authentication")
	keyFile   = flag.String("storage_driver_kafka_ssl_key", "", "optional key file for TLS client authentication")
	caFile    = flag.String("storage_driver_kafka_ssl_ca", "", "optional certificate authority file for TLS client authentication")
	verifySSL = flag.Bool("storage_driver_kafka_ssl_verify", true, "verify ssl certificate chain")
)

type kafkaStorage struct {
	producer    kafka.AsyncProducer
	topic       string
	machineName string
}

type detailSpec struct {
	Timestamp       time.Time            `json:"timestamp"`
	MachineName     string               `json:"machine_name,omitempty"`
	ContainerName   string               `json:"container_Name,omitempty"`
	ContainerID     string               `json:"container_Id,omitempty"`
	ContainerLabels map[string]string    `json:"container_labels,omitempty"`
	ContainerStats  *info.ContainerStats `json:"container_stats,omitempty"`
}

func (s *kafkaStorage) infoToDetailSpec(cInfo *info.ContainerInfo, stats *info.ContainerStats) *detailSpec {
	timestamp := time.Now()
	containerID := cInfo.ContainerReference.Id
	containerLabels := cInfo.Spec.Labels
	containerName := container.GetPreferredName(cInfo.ContainerReference)

	detail := &detailSpec{
		Timestamp:       timestamp,
		MachineName:     s.machineName,
		ContainerName:   containerName,
		ContainerID:     containerID,
		ContainerLabels: containerLabels,
		ContainerStats:  stats,
	}
	return detail
}

func (s *kafkaStorage) AddStats(cInfo *info.ContainerInfo, stats *info.ContainerStats) error {
	detail := s.infoToDetailSpec(cInfo, stats)
	b, err := json.Marshal(detail)

	s.producer.Input() <- &kafka.ProducerMessage{
		Topic: s.topic,
		Value: kafka.StringEncoder(b),
	}

	return err
}

func (s *kafkaStorage) Close() error {
	return s.producer.Close()
}

func new() (storage.StorageDriver, error) {
	machineName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(machineName)
}

func generateTLSConfig() (*tls.Config, error) {
	if *certFile != "" && *keyFile != "" && *caFile != "" {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile(*caFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: *verifySSL,
		}, nil
	}

	return nil, nil
}

func newStorage(machineName string) (storage.StorageDriver, error) {
	config := kafka.NewConfig()

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	config.Producer.RequiredAcks = kafka.WaitForAll

	brokerList := strings.Split(*brokers, ",")
	klog.V(4).Infof("Kafka brokers:%q", *brokers)

	producer, err := kafka.NewAsyncProducer(brokerList, config)
	if err != nil {
		return nil, err
	}
	ret := &kafkaStorage{
		producer:    producer,
		topic:       *topic,
		machineName: machineName,
	}
	return ret, nil
}
