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

package collector

import (
	"fmt"
	containertest "github.com/google/cadvisor/container/testing"
	"github.com/google/cadvisor/info/v1"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJolokiaConfigEndpointConfig(t *testing.T) {
	assert := assert.New(t)

	configFile, err := ioutil.ReadFile("config/sample_config_jolokia_endpoint_config.json")
	assert.NoError(err)

	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	containerHandler.On("GetContainerIPAddress").Return(
		"123.456.789.1011",
	)

	collector, err := NewJolokiaCollector("test-jolokia", configFile, 10, containerHandler, http.DefaultClient)
	assert.NoError(err)
	assert.Equal("10s", collector.pollingFrequency.String())

	assert.Equal("test-jolokia", collector.Name())

	assert.Equal("https://123.456.789.1011:8778/jolokia/", collector.configFile.Endpoint.URL)
}

func TestJolokiaConfigNoEndpoint(t *testing.T) {
	assert := assert.New(t)

	configFile, err := ioutil.ReadFile("config/sample_config_jolokia_no_endpoint.json")
	assert.NoError(err)

	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	containerHandler.On("GetContainerIPAddress").Return(
		"123.456.789.1012",
	)

	collector, err := NewJolokiaCollector("test-jolokia", configFile, 10, containerHandler, http.DefaultClient)
	assert.NoError(err)
	assert.Equal("11s", collector.pollingFrequency.String())

	assert.Equal("test-jolokia", collector.Name())

	assert.Equal("https://123.456.789.1012:8778/jolokia/", collector.configFile.Endpoint.URL)
}

func TestJolokiaMultipleMetricGathering(t *testing.T) {
	assert := assert.New(t)

	configFile, err := ioutil.ReadFile("config/sample_config_jolokia.json")
	assert.NoError(err)

	containerHandler := containertest.NewMockContainerHandler("mockContainer")
	containerHandler.On("GetContainerIPAddress").Return(
		"123.456.789.1011",
	)

	collector, err := NewJolokiaCollector("test-jolokia", configFile, 10, containerHandler, http.DefaultClient)
	assert.NoError(err)

	assert.Equal("test-jolokia", collector.Name())

	assert.Equal("https://123.456.789.1011:8778/jolokia/", collector.configFile.Endpoint.URL)

	tempServer := generateTestServer()
	defer tempServer.Close()

	collector.configFile.Endpoint.URL = tempServer.URL + "/jolokia/"

	metrics := map[string][]v1.MetricVal{}
	//time, metrics, errors
	_, metrics, errMetric := collector.Collect(metrics)
	assert.NotNil(metrics)

	assert.Equal(3, len(metrics))

	heapMetric := metrics["JVMHeap"][0]
	assert.Equal(time.Unix(1414141414, 0), heapMetric.Timestamp)
	assert.Equal(290405160, heapMetric.IntValue)
	assert.Equal(0, heapMetric.FloatValue)

	gcMetric := metrics["GCDuration"][0]
	assert.Equal(time.Unix(1515151515, 0), gcMetric.Timestamp)
	assert.Equal(0, gcMetric.IntValue)
	assert.Equal(191, gcMetric.FloatValue)

	assert.Equal(0, len(metrics["nonExistantMBean"]))
	// The nonExistantMBean should throw an error since it couldn't gather metrics for it
	assert.True(strings.HasPrefix(errMetric.Error(), "Error 0: The value was empty"))

}

func generateTestServer() *httptest.Server {
	tempServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/jolokia/read/java.lang:type=Memory/HeapMemoryUsage/used" {
			result := `
			{
    				"request": {
     					"attribute": "HeapMemoryUsage",
     					"mbean": "java.lang:type=Memory",
        				"path": "used",
        				"type": "read"
    				},
    				"status": 200,
				"value": 290405160,
				"timestamp":1414141414
			}`
			fmt.Fprint(w, result)
		} else if r.URL.Path == "/jolokia/read/java.lang:name=PS MarkSweep,type=GarbageCollector/LastGcInfo/duration" {
			result := `
			{
    				"request": {
        				"attribute": "LastGcInfo",
				        "mbean": "java.lang:name=PS MarkSweep,type=GarbageCollector",
        				"path": "duration",
        				"type": "read"
    				},
    				"status": 200,
    				"timestamp": 1515151515,
    				"value": 191
			}`
			fmt.Fprint(w, result)
		} else {
			result := `
			{
    				"error": "javax.management.AttributeNotFoundException : No such attribute: ...",
    				"error_type": "javax.management.AttributeNotFoundException",
    				"request": {
        				"attribute": "....",
        				"mbean": "....",
        				"type": "read"
    				},
    				"stacktrace": "javax.management.AttributeNotFoundException:...",
    				"status": 404
			}
			`
			fmt.Fprint(w, result)
		}
	}))
	return tempServer
}
