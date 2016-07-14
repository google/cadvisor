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
	"compress/gzip"
	"fmt"
	"os"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
)

// Test default settings
func TestDefault(t *testing.T) {
	hec := NewHTTPEventCollectorMock(t)

	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		true,
		false,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	if !hec.connectionVerified {
		t.Fatal("By default connection should be verified")
	}

	splunkStorage, ok := storage.(*splunkStorage)
	if !ok {
		t.Fatal("Unexpected Splunk Logging Driver type")
	}

	if splunkStorage.url != hec.URL()+"/services/collector/event/1.0" ||
		splunkStorage.auth != "Splunk "+hec.token ||
		splunkStorage.nullMessage.Host != hostname ||
		splunkStorage.nullMessage.Source != "" ||
		splunkStorage.nullMessage.SourceType != "" ||
		splunkStorage.nullMessage.Index != "" ||
		splunkStorage.gzipCompression != false ||
		splunkStorage.postMessagesFrequency != defaultPostMessagesFrequency ||
		splunkStorage.postMessagesBatchSize != defaultPostMessagesBatchSize ||
		splunkStorage.bufferMaximum != defaultBufferMaximum ||
		cap(splunkStorage.stream) != defaultStreamChannelSize {
		t.Fatal("Found not default values setup in Splunk Storage.")
	}

	ref := info.ContainerReference{
		Id:   "containerid",
		Name: "containername",
	}

	now := time.Now()

	stats1 := info.ContainerStats{
		Timestamp: now.Add(time.Second),
	}

	stats2 := info.ContainerStats{
		Timestamp: now.Add(2 * time.Second),
	}

	if err := storage.AddStats(ref, &stats1); err != nil {
		t.Fatal(err)
	}
	if err := storage.AddStats(ref, &stats2); err != nil {
		t.Fatal(err)
	}

	err = splunkStorage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(hec.messages) != 2 {
		t.Fatal("Expected two messages")
	}

	if *hec.gzipEnabled {
		t.Fatal("Gzip should not be used")
	}

	message1 := hec.messages[0]
	if message1.Time != fmt.Sprintf("%f", float64(stats1.Timestamp.UnixNano())/float64(time.Second)) ||
		message1.Host != hostname ||
		message1.Source != "" ||
		message1.SourceType != "" ||
		message1.Index != "" {
		t.Fatalf("Unexpected values of message 1 %v", message1)
	}

	message2 := hec.messages[1]
	if message2.Time != fmt.Sprintf("%f", float64(stats2.Timestamp.UnixNano())/float64(time.Second)) ||
		message2.Host != hostname ||
		message2.Source != "" ||
		message2.SourceType != "" ||
		message2.Index != "" {
		t.Fatalf("Unexpected values of message 2 %v", message2)
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// Test not default settings
func TestNonDefaultValues(t *testing.T) {
	hec := NewHTTPEventCollectorMock(t)

	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"mysource",
		"mysourcetype",
		"myindex",
		"",
		"",
		false,
		true,
		true,
		gzip.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}

	if !hec.connectionVerified {
		t.Fatal("By default connection should be verified")
	}

	splunkStorage, ok := storage.(*splunkStorage)
	if !ok {
		t.Fatal("Unexpected Splunk Logging Driver type")
	}

	if splunkStorage.url != hec.URL()+"/services/collector/event/1.0" ||
		splunkStorage.auth != "Splunk "+hec.token ||
		splunkStorage.nullMessage.Host != hostname ||
		splunkStorage.nullMessage.Source != "mysource" ||
		splunkStorage.nullMessage.SourceType != "mysourcetype" ||
		splunkStorage.nullMessage.Index != "myindex" ||
		splunkStorage.gzipCompression != true ||
		splunkStorage.gzipCompressionLevel != gzip.BestSpeed ||
		splunkStorage.postMessagesFrequency != defaultPostMessagesFrequency ||
		splunkStorage.postMessagesBatchSize != defaultPostMessagesBatchSize ||
		splunkStorage.bufferMaximum != defaultBufferMaximum ||
		cap(splunkStorage.stream) != defaultStreamChannelSize {
		t.Fatal("Specified values do not match.")
	}

	ref := info.ContainerReference{
		Id:   "containerid",
		Name: "containername",
	}

	now := time.Now()

	stats := info.ContainerStats{
		Timestamp: now.Add(time.Second),
	}

	if err := storage.AddStats(ref, &stats); err != nil {
		t.Fatal(err)
	}

	err = splunkStorage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if !*hec.gzipEnabled {
		t.Fatal("Gzip should be used")
	}

	if len(hec.messages) != 1 {
		t.Fatal("Expected one message")
	}

	message := hec.messages[0]
	if message.Time != fmt.Sprintf("%f", float64(stats.Timestamp.UnixNano())/float64(time.Second)) ||
		message.Host != hostname ||
		message.Source != "mysource" ||
		message.SourceType != "mysourcetype" ||
		message.Index != "myindex" {
		t.Fatalf("Unexpected values of message %v", message)
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// Verify that we will send messages in batches with default batching parameters,
// but change frequency to be sure that numOfRequests will match expected 17 requests
func TestBatching(t *testing.T) {
	if err := os.Setenv(envVarPostMessagesFrequency, "10h"); err != nil {
		t.Fatal(err)
	}

	hec := NewHTTPEventCollectorMock(t)

	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		true,
		true,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	for i := 0; i < defaultStreamChannelSize*4; i++ {
		if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}

	err = storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(hec.messages) != defaultStreamChannelSize*4 {
		t.Fatal("Not all messages delivered")
	}

	for i, message := range hec.messages {
		if message.Time != fmt.Sprintf("%f", float64(now.Add(time.Duration(i)*time.Second).UnixNano())/float64(time.Second)) {
			t.Fatalf("Unexpected timestamp of message %v", message)
		}
	}

	// 1 to verify connection and 16 batches
	if hec.numOfRequests != 17 {
		t.Fatalf("Unexpected number of requests %d", hec.numOfRequests)
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarPostMessagesFrequency, ""); err != nil {
		t.Fatal(err)
	}
}

// Verify that test is using time to fire events not rare than specified frequency
func TestFrequency(t *testing.T) {
	if err := os.Setenv(envVarPostMessagesFrequency, "5ms"); err != nil {
		t.Fatal(err)
	}

	hec := NewHTTPEventCollectorMock(t)

	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		true,
		true,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	for i := 0; i < 10; i++ {
		if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(15 * time.Millisecond)
	}

	err = storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(hec.messages) != 10 {
		t.Fatal("Not all messages delivered")
	}

	for i, message := range hec.messages {
		if message.Time != fmt.Sprintf("%f", float64(now.Add(time.Duration(i)*time.Second).UnixNano())/float64(time.Second)) {
			t.Fatalf("Unexpected timestamp of message %v", message)
		}
	}

	// 1 to verify connection and 10 to verify that we have sent messages with required frequency,
	// but because frequency is too small (to keep test quick), instead of 11, use 9 if context switches will be slow
	if hec.numOfRequests < 9 {
		t.Fatalf("Unexpected number of requests %d", hec.numOfRequests)
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarPostMessagesFrequency, ""); err != nil {
		t.Fatal(err)
	}
}

// Simulate behavior similar to first version of Splunk Logging Driver, when we were sending one message
// per request
func TestOneMessagePerRequest(t *testing.T) {
	if err := os.Setenv(envVarPostMessagesFrequency, "10h"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarPostMessagesBatchSize, "1"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarBufferMaximum, "1"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarStreamChannelSize, "0"); err != nil {
		t.Fatal(err)
	}

	hec := NewHTTPEventCollectorMock(t)

	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		true,
		true,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	for i := 0; i < 10; i++ {
		if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}

	err = storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(hec.messages) != 10 {
		t.Fatal("Not all messages delivered")
	}

	for i, message := range hec.messages {
		if message.Time != fmt.Sprintf("%f", float64(now.Add(time.Duration(i)*time.Second).UnixNano())/float64(time.Second)) {
			t.Fatalf("Unexpected timestamp of message %v", message)
		}
	}

	// 1 to verify connection and 10 messages
	if hec.numOfRequests != 11 {
		t.Fatalf("Unexpected number of requests %d", hec.numOfRequests)
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarPostMessagesFrequency, ""); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarPostMessagesBatchSize, ""); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarBufferMaximum, ""); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarStreamChannelSize, ""); err != nil {
		t.Fatal(err)
	}
}

// Driver should not be created when HEC is unresponsive
func TestVerify(t *testing.T) {
	hec := NewHTTPEventCollectorMock(t)
	hec.simulateServerError = true
	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	_, err = newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		true,
		true,
		gzip.DefaultCompression)
	if err == nil {
		t.Fatal("Expecting driver to fail, when server is unresponsive")
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// Verify that user can specify to skip verification that Splunk HEC is working.
// Also in this test we verify retry logic.
func TestSkipVerify(t *testing.T) {
	hec := NewHTTPEventCollectorMock(t)
	hec.simulateServerError = true
	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		false,
		true,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	if hec.connectionVerified {
		t.Fatal("Connection should not be verified")
	}

	now := time.Now()

	for i := 0; i < defaultStreamChannelSize*2; i++ {
		if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}

	if len(hec.messages) != 0 {
		t.Fatal("No messages should be accepted at this point")
	}

	hec.simulateServerError = false

	for i := defaultStreamChannelSize * 2; i < defaultStreamChannelSize*4; i++ {
		if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}

	err = storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(hec.messages) != defaultStreamChannelSize*4 {
		t.Fatal("Not all messages delivered")
	}

	for i, message := range hec.messages {
		if message.Time != fmt.Sprintf("%f", float64(now.Add(time.Duration(i)*time.Second).UnixNano())/float64(time.Second)) {
			t.Fatalf("Unexpected timestamp of message %v", message)
		}
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// Verify logic for when we filled whole buffer
func TestBufferMaximum(t *testing.T) {
	if err := os.Setenv(envVarPostMessagesBatchSize, "2"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarBufferMaximum, "10"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarStreamChannelSize, "0"); err != nil {
		t.Fatal(err)
	}

	hec := NewHTTPEventCollectorMock(t)
	hec.simulateServerError = true
	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		false,
		true,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	if hec.connectionVerified {
		t.Fatal("Connection should not be verified")
	}

	now := time.Now()

	for i := 0; i < 11; i++ {
		if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}

	if len(hec.messages) != 0 {
		t.Fatal("No messages should be accepted at this point")
	}

	hec.simulateServerError = false

	err = storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(hec.messages) != 9 {
		t.Fatalf("Expected # of messages %d, got %d", 9, len(hec.messages))
	}

	// First 1000 messages are written to daemon log when buffer was full
	for i, message := range hec.messages {
		if message.Time != fmt.Sprintf("%f", float64(now.Add(time.Duration(i+2)*time.Second).UnixNano())/float64(time.Second)) {
			t.Fatalf("Unexpected timestamp of message %v", message)
		}
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarStreamChannelSize, ""); err != nil {
		t.Fatal(err)
	}
}

// Verify that we do not block close when HEC is down for the whole time
func TestServerAlwaysDown(t *testing.T) {
	if err := os.Setenv(envVarPostMessagesBatchSize, "2"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarBufferMaximum, "4"); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarStreamChannelSize, "0"); err != nil {
		t.Fatal(err)
	}

	hec := NewHTTPEventCollectorMock(t)
	hec.simulateServerError = true
	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		false,
		true,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	if hec.connectionVerified {
		t.Fatal("Connection should not be verified")
	}

	now := time.Now()

	for i := 0; i < 5; i++ {
		if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}

	err = storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(hec.messages) != 0 {
		t.Fatal("No messages should be accepted at this point")
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarStreamChannelSize, ""); err != nil {
		t.Fatal(err)
	}
}

// Verify we do not allow to send stats after we close storage
func TestCannotSendAfterClose(t *testing.T) {
	hec := NewHTTPEventCollectorMock(t)
	go hec.Serve()

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	storage, err := newStorage(
		hostname,
		hec.URL(),
		hec.token,
		"",
		"",
		"",
		"",
		"",
		false,
		false,
		true,
		gzip.DefaultCompression)
	if err != nil {
		t.Fatal(err)
	}

	if hec.connectionVerified {
		t.Fatal("Connection should not be verified")
	}

	now := time.Now()

	if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now}); err != nil {
		t.Fatal(err)
	}

	err = storage.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := storage.AddStats(info.ContainerReference{}, &info.ContainerStats{Timestamp: now.Add(time.Second)}); err == nil {
		t.Fatal("Storage should not allow to send messages after close")
	}

	if len(hec.messages) != 1 {
		t.Fatal("Only one message should be sent")
	}

	message := hec.messages[0]
	if message.Time != fmt.Sprintf("%f", float64(now.UnixNano())/float64(time.Second)) {
		t.Fatalf("Unexpected values of message %v", message)
	}

	err = hec.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv(envVarStreamChannelSize, ""); err != nil {
		t.Fatal(err)
	}
}
