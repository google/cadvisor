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

package splunk

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	info "github.com/google/cadvisor/info/v1"
	storage "github.com/google/cadvisor/storage"
)

var (
	argSplunkURL        = flag.String("storage_driver_splunk_url", "https://localhost:8088", "Splunk protocol://host:port.")
	argSplunkToken      = flag.String("storage_driver_splunk_token", "00000000-0000-0000-0000-000000000000", "Splunk token for authorization.")
	argSplunkSource     = flag.String("storage_driver_splunk_source", "", "Splunk event source.")
	argSplunkSourceType = flag.String("storage_driver_splunk_source_type", "", "Splunk event source type.")
	argSplunkIndex      = flag.String("storage_driver_splunk_index", "", "Splunk event index.")
	argSplunkCertPath   = flag.String("storage_driver_splunk_capath", "", "Path to root certificate.")
	argSplunkCertRoot   = flag.String("storage_driver_splunk_caname", "", "Name to use for validating server certificate; by default the hostname of the `splunk-url` will be used.")
	argSplunkInsecure   = flag.Bool("storage_driver_splunk_insecureskipverify", false, "Ignore server certificate validation.")
	argSplunkVerify     = flag.Bool("storage_driver_splunk_verifyconnection", true, "Verify on start, that cAdvisor can connect to Splunk server.")
	argSplunkGzip       = flag.Bool("storage_driver_splunk_gzip", false, "Enable gzip compression.")
	argSplunkGzipLevel  = flag.Int("storage_driver_splunk_gzip_level", gzip.DefaultCompression, "Set gzip compression level.")
)

const (
	defaultPostMessagesFrequency = 5 * time.Second
	defaultPostMessagesBatchSize = 1000
	defaultBufferMaximum         = 10 * defaultPostMessagesBatchSize
	defaultStreamChannelSize     = 4 * defaultPostMessagesBatchSize
)

const (
	envVarPostMessagesFrequency = "SPLUNK_STORAGE_POST_MESSAGES_FREQUENCY"
	envVarPostMessagesBatchSize = "SPLUNK_STORAGE_POST_MESSAGES_BATCH_SIZE"
	envVarBufferMaximum         = "SPLUNK_STORAGE_BUFFER_MAX"
	envVarStreamChannelSize     = "SPLUNK_STORAGE_CHANNEL_SIZE"
)

func init() {
	storage.RegisterStorageDriver("splunk", new)
}

type splunkStorage struct {
	client      *http.Client
	transport   *http.Transport
	url         string
	auth        string
	nullMessage *splunkMessage

	// http compression
	gzipCompression      bool
	gzipCompressionLevel int

	// Advanced options
	postMessagesFrequency time.Duration
	postMessagesBatchSize int
	bufferMaximum         int

	// For synchronization between background worker and logger
	stream     chan *splunkMessage
	lock       sync.RWMutex
	closed     bool
	closedCond *sync.Cond
}

type splunkMessage struct {
	Event      *splunkMessageEvent `json:"event"`
	Time       string              `json:"time"`
	Host       string              `json:"host"`
	Source     string              `json:"source,omitempty"`
	SourceType string              `json:"sourcetype,omitempty"`
	Index      string              `json:"index,omitempty"`
}

type splunkMessageEvent struct {
	Container *info.ContainerReference `json:"container"`
	Stats     *info.ContainerStats     `json:"stats"`
}

func new() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(
		hostname,
		*argSplunkURL,
		*argSplunkToken,
		*argSplunkSource,
		*argSplunkSourceType,
		*argSplunkIndex,
		*argSplunkCertPath,
		*argSplunkCertRoot,
		*argSplunkInsecure,
		*argSplunkVerify,
		*argSplunkGzip,
		*argSplunkGzipLevel,
	)
}

func (storage *splunkStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	message := *storage.nullMessage
	message.Time = fmt.Sprintf("%f", float64(stats.Timestamp.UnixNano())/1000000000)
	message.Event = &splunkMessageEvent{&ref, stats}

	storage.lock.RLock()
	defer storage.lock.RUnlock()
	if storage.closedCond != nil {
		return fmt.Errorf("Cannot send stats, Splunk storage is closed")
	}
	storage.stream <- &message

	return nil
}

func (storage *splunkStorage) Close() error {
	storage.lock.Lock()
	defer storage.lock.Unlock()
	if storage.closedCond == nil {
		storage.closedCond = sync.NewCond(&storage.lock)
		close(storage.stream)
		for !storage.closed {
			storage.closedCond.Wait()
		}
	}
	return nil
}

func newStorage(
	hostname,
	splunkURLStr,
	splunkToken,
	source,
	sourceType,
	index,
	caPath,
	caName string,
	insecureSkipVerify,
	verifyConnection bool,
	gzipCompression bool,
	gzipCompressionLevel int,
) (storage.StorageDriver, error) {
	// Parse and validate Splunk URL
	splunkURL, err := parseURL(splunkURLStr)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		// Splunk is using autogenerated certificates by default,
		// allow users to trust them with skipping verification
		InsecureSkipVerify: insecureSkipVerify,
	}

	// If path to the root certificate is provided - load it
	if caPath != "" {
		caCert, err := ioutil.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caPool
	}

	if caName != "" {
		tlsConfig.ServerName = caName
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{
		Transport: transport,
	}

	var (
		postMessagesFrequency = getAdvancedOptionDuration(envVarPostMessagesFrequency, defaultPostMessagesFrequency)
		postMessagesBatchSize = getAdvancedOptionInt(envVarPostMessagesBatchSize, defaultPostMessagesBatchSize)
		bufferMaximum         = getAdvancedOptionInt(envVarBufferMaximum, defaultBufferMaximum)
		streamChannelSize     = getAdvancedOptionInt(envVarStreamChannelSize, defaultStreamChannelSize)
	)

	var nullMessage = &splunkMessage{
		Host:       hostname,
		Source:     source,
		SourceType: sourceType,
		Index:      index,
	}

	storage := &splunkStorage{
		client:                client,
		transport:             transport,
		url:                   splunkURL.String(),
		auth:                  "Splunk " + splunkToken,
		nullMessage:           nullMessage,
		gzipCompression:       gzipCompression,
		gzipCompressionLevel:  gzipCompressionLevel,
		stream:                make(chan *splunkMessage, streamChannelSize),
		postMessagesFrequency: postMessagesFrequency,
		postMessagesBatchSize: postMessagesBatchSize,
		bufferMaximum:         bufferMaximum,
	}

	// By default we verify connection, but we allow use to skip that
	if verifyConnection {
		err = verifySplunkConnection(storage)
		if err != nil {
			return nil, err
		}
	}

	go storage.worker()

	return storage, nil
}

func (storage *splunkStorage) worker() {
	timer := time.NewTicker(storage.postMessagesFrequency)
	var messages []*splunkMessage
	for {
		select {
		case message, open := <-storage.stream:
			if !open {
				storage.postMessages(messages, true)
				storage.lock.Lock()
				defer storage.lock.Unlock()
				storage.transport.CloseIdleConnections()
				storage.closed = true
				storage.closedCond.Signal()
				return
			}
			messages = append(messages, message)
			// Only sending when we get exactly to the batch size,
			// This also helps not to fire postMessages on every new message,
			// when previous try failed.
			if len(messages)%storage.postMessagesBatchSize == 0 {
				messages = storage.postMessages(messages, false)
			}
		case <-timer.C:
			messages = storage.postMessages(messages, false)
		}
	}
}

func (storage *splunkStorage) postMessages(messages []*splunkMessage, lastChance bool) []*splunkMessage {
	messagesLen := len(messages)
	for i := 0; i < messagesLen; i += storage.postMessagesBatchSize {
		upperBound := i + storage.postMessagesBatchSize
		if upperBound > messagesLen {
			upperBound = messagesLen
		}
		if err := storage.tryPostMessages(messages[i:upperBound]); err != nil {
			glog.Error(err)
			if messagesLen-i >= storage.bufferMaximum || lastChance {
				if lastChance {
					upperBound = messagesLen
				}
				// Not all sent, but buffer has got to its maximum, let's log all messages
				// we could not send and return buffer minus one batch size
				for j := i; j < upperBound; j++ {
					if jsonEvent, err := json.Marshal(messages[j]); err != nil {
						glog.Error(err)
					} else {
						glog.Error(fmt.Errorf("Failed to send a message '%s'", string(jsonEvent)))
					}
				}
				return messages[upperBound:messagesLen]
			}
			// Not all sent, returning buffer from where we have not sent messages
			return messages[i:messagesLen]
		}
	}
	// All sent, return empty buffer
	return messages[:0]
}

func (storage *splunkStorage) tryPostMessages(messages []*splunkMessage) error {
	if len(messages) == 0 {
		return nil
	}
	var buffer bytes.Buffer
	var writer io.Writer
	var gzipWriter *gzip.Writer
	var err error
	// If gzip compression is enabled - create gzip writer with specified compression
	// level. If gzip compression is disabled, use standard buffer as a writer
	if storage.gzipCompression {
		gzipWriter, err = gzip.NewWriterLevel(&buffer, storage.gzipCompressionLevel)
		if err != nil {
			return err
		}
		writer = gzipWriter
	} else {
		writer = &buffer
	}
	for _, message := range messages {
		jsonEvent, err := json.Marshal(message)
		if err != nil {
			return err
		}
		if _, err := writer.Write(jsonEvent); err != nil {
			return err
		}
	}
	// If gzip compression is enabled, tell it, that we are done
	if storage.gzipCompression {
		err = gzipWriter.Close()
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest("POST", storage.url, bytes.NewBuffer(buffer.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", storage.auth)
	// Tell if we are sending gzip compressed body
	if storage.gzipCompression {
		req.Header.Set("Content-Encoding", "gzip")
	}
	res, err := storage.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var body []byte
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Splunk Storage: failed to send event - %s - %s", res.Status, body)
	}
	io.Copy(ioutil.Discard, res.Body)
	return nil
}

func parseURL(splunkURLStr string) (*url.URL, error) {
	splunkURL, err := url.Parse(splunkURLStr)
	if err != nil {
		return nil, err
	}

	if !splunkURL.IsAbs() ||
		(splunkURL.Path != "" && splunkURL.Path != "/") ||
		splunkURL.RawQuery != "" ||
		splunkURL.Fragment != "" {
		return nil, fmt.Errorf("Expected format scheme://dns_name_or_ip:port for storage_driver_splunk_url, got '%s'.", splunkURLStr)
	}

	splunkURL.Path = "/services/collector/event/1.0"

	return splunkURL, nil
}

func verifySplunkConnection(storage *splunkStorage) error {
	req, err := http.NewRequest("OPTIONS", storage.url, nil)
	if err != nil {
		return err
	}
	res, err := storage.client.Do(req)
	if err != nil {
		return err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	if res.StatusCode != http.StatusOK {
		var body []byte
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Failed to connect to Splunk - %s - %s", res.Status, body)
	}
	return nil
}

func getAdvancedOptionDuration(envName string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(envName)
	if valueStr == "" {
		return defaultValue
	}
	parsedValue, err := time.ParseDuration(valueStr)
	if err != nil {
		glog.Warningf("Failed to parse value of %s as duration. Using default %v. %v", envName, defaultValue, err)
		return defaultValue
	}
	return parsedValue
}

func getAdvancedOptionInt(envName string, defaultValue int) int {
	valueStr := os.Getenv(envName)
	if valueStr == "" {
		return defaultValue
	}
	parsedValue, err := strconv.ParseInt(valueStr, 10, 32)
	if err != nil {
		glog.Warningf("Failed to parse value of %s as integer. Using default %d. %v", envName, defaultValue, err)
		return defaultValue
	}
	return int(parsedValue)
}
