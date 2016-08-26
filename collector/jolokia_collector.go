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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/info/v1"
	"strconv"
)

const minpollingfrequency = 1 * time.Second
const defaultPollingFrequency = 15 * time.Second

var defaultEndpointConfig = URLConfig{
	Protocol: "https",
	Port:     "8778",
	Path:     "/jolokia",
}

type JolokiaCollector struct {
	//name of the collector
	name string

	//holds information extracted from the config file for a collector
	configFile Jolokia

	// Limit for the number of scaped metrics. If the count is higher,
	// no metrics will be returned.
	metricCountLimit int

	//rate at which metrics are collected
	pollingFrequency time.Duration

	//the client to use when accessing the endpoints
	httpClient *http.Client
}

type timeEpoch time.Time

type JolokiaResponse struct {
	Request   JolokiaRequest `json:"request"`
	Status    int32          `json:"status"`
	TimeStamp timeEpoch      `json:"timestamp"`
	Value     json.Number    `json:"value"`
}

type JolokiaRequest struct {
	Attribute string `json:"attribute"`
	MBean     string `json:"mbean"`
	Path      string `json:"path"`
}

func (te *timeEpoch) UnmarshalJSON(b []byte) error {
	var seconds int64
	err := json.Unmarshal(b, &seconds)
	if err == nil {
		time := time.Unix(seconds, 0)
		*te = timeEpoch(time)
		return nil
	} else {
		return err
	}
}

//Returns a new collector using the information extracted from the configfile
func NewJolokiaCollector(collectorName string, configFile []byte, metricCountLimit int, containerHandler container.ContainerHandler, httpClient *http.Client) (*JolokiaCollector, error) {
	var configInJSON Jolokia
	err := json.Unmarshal(configFile, &configInJSON)
	if err != nil {
		return nil, err
	}

	configureDefaultURLConfig(&configInJSON.Endpoint.URLConfig, defaultEndpointConfig)
	configInJSON.Endpoint.configure(containerHandler)
	if err != nil {
		return nil, err
	}

	//glog.Fatal("FOOBAR ", configInJSON.PollingFrequency)
	pollingFrequency := configInJSON.PollingFrequency.Duration
	//pollingFrequency, err := time.ParseDuration(configInJSON.PollingFrequency)
	//if err != nil {
	//	glog.Warningf("Failed to properly parse the 'polling_frequency' values as a go time.Duration object. Setting to %vs.", defaultPollingFrequency.Seconds())
	//	pollingFrequency = defaultPollingFrequency
	//}
	if pollingFrequency < minpollingfrequency {
		glog.Warningf("The polling frequency is lower than the minium, the poll frequency for %s has now been to the minium (%vs).", collectorName, minpollingfrequency.Seconds())
		pollingFrequency = minpollingfrequency
	}

	if metricCountLimit < 0 {
		return nil, fmt.Errorf("Metric count limit must be greater than 0")
	}

	if len(configInJSON.MetricsConfig) > metricCountLimit {
		return nil, fmt.Errorf("Too many metrics defined: %d limit %d", len(configInJSON.MetricsConfig), metricCountLimit)
	}

	if !strings.HasSuffix(configInJSON.Endpoint.URL, "/") {
		configInJSON.Endpoint.URL = configInJSON.Endpoint.URL + "/"
	}

	return &JolokiaCollector{
		name:             collectorName,
		configFile:       configInJSON,
		metricCountLimit: metricCountLimit,
		pollingFrequency: pollingFrequency,
		httpClient:       httpClient,
	}, nil
}

func configureDefaultURLConfig(urlConfig *URLConfig, defaultURLConfig URLConfig) {
	if urlConfig.Protocol == "" {
		urlConfig.Protocol = defaultURLConfig.Protocol
	}
	if urlConfig.Port == "" {
		urlConfig.Port = defaultURLConfig.Port
	}
	if urlConfig.Path == "" {
		urlConfig.Path = defaultURLConfig.Path
	}
}

//Returns name of the collector
func (collector *JolokiaCollector) Name() string {
	return collector.name
}

func (collector *JolokiaCollector) configToSpec(config JolokiaMetricConfig) v1.MetricSpec {
	return v1.MetricSpec{
		Name:   config.Name,
		Type:   config.MetricType,
		Format: config.DataType,
		Units:  config.Units,
	}
}

func (collector *JolokiaCollector) GetSpec() []v1.MetricSpec {
	specs := []v1.MetricSpec{}

	for _, metricConfig := range collector.configFile.MetricsConfig {
		spec := collector.configToSpec(metricConfig)
		specs = append(specs, spec)
	}
	return specs
}

//Returns collected metrics and the next collection time of the collector
func (collector *JolokiaCollector) Collect(metrics map[string][]v1.MetricVal) (time.Time, map[string][]v1.MetricVal, error) {
	currentTime := time.Now()
	nextCollectionTime := currentTime.Add(time.Duration(collector.pollingFrequency))

	var errorSlice []error

	for _, metricConfig := range collector.configFile.MetricsConfig {
		metricValues, err := collectSingleJolokiaMetric(collector.configFile.Endpoint.URL, metricConfig, collector.httpClient)
		if err != nil {
			errorSlice = append(errorSlice, err)
		}
		metrics[metricConfig.Name] = metricValues
	}
	return nextCollectionTime, metrics, compileErrors(errorSlice)
}

func collectSingleJolokiaMetric(endpointURL string, metricConfig JolokiaMetricConfig, httpClient *http.Client) ([]v1.MetricVal, error) {
	metricURL := endpointURL + "read/" + metricConfig.MBean.Name + "/" + metricConfig.MBean.Attribute + "/" + metricConfig.MBean.Path

	http.Get(metricURL)
	response, err := httpClient.Get(metricURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	jolokiaResponse := &JolokiaResponse{}
	json.NewDecoder(response.Body).Decode(jolokiaResponse)

	if metricConfig.DataType == v1.FloatType {
		regVal, err := strconv.ParseFloat(string(jolokiaResponse.Value), 64)
		if err != nil {
			return nil, err
		}
		metricsVal := []v1.MetricVal{
			{
				FloatValue: regVal,
				Timestamp:  time.Time(jolokiaResponse.TimeStamp),
			},
		}
		return metricsVal, nil
	} else if metricConfig.DataType == v1.IntType {
		if jolokiaResponse.Value != "" {
			regVal, err := strconv.ParseInt(string(jolokiaResponse.Value), 10, 64)
			if err != nil {
				return nil, err
			}
			metricVal := []v1.MetricVal{
				{
					IntValue:  regVal,
					Timestamp: time.Time(jolokiaResponse.TimeStamp),
				},
			}
			return metricVal, nil
		} else {
			return nil, fmt.Errorf("The value was empty %v", jolokiaResponse)
		}

	} else {
		return nil, fmt.Errorf("Unexpected value of 'data_type' for metric '%v' in config ", metricConfig.Name)
	}
}
