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

package collector

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/google/cadvisor/info/v2"
)

type Collector struct {
	//name of the collector
	name string

	//holds information extracted from the config file for a collector
	configFile Config

	//time at which metrics need to be collected next
	nextCollectionTime time.Time

	//error (if any) during metrics collection
	err error
}

type Config struct {
	//the endpoint to hit to scrape metrics
	Endpoint string `json:"endpoint"`

	//holds information about different metrics that can be collected
	MetricsConfig []metricConfig `json:"metricsConfig"`
}

// metricConfig holds information extracted from the config file about a metric
type metricConfig struct {
	//the name of the metric
	Name string `json:"name"`

	//enum type for the metric type
	MetricType MetricType `json:"metricType"`

	//data type of the metric (eg: integer, string)
	Units string `json:"units"`

	//the frequency at which the metric should be collected (in seconds)
	PollingFrequency string `json:"pollingFrequency"`

	//the regular expression that can be used to extract the metric
	Regex string `json:"regex"`
}

// MetricType is an enum type that lists the possible type of the metric
type MetricType string

const (
	Counter MetricType = "counter"
	Gauge   MetricType = "gauge"
)

//Returns a new collector using the information extracted from the configfile
func NewCollector(collectorName string, configfile string) (*Collector, error) {
	configFile, err := ioutil.ReadFile(configfile)
	if err != nil {
		return nil, err
	}

	var configInJSON Config
	err = json.Unmarshal(configFile, &configInJSON)
	if err != nil {
		return nil, err
	}

	return &Collector{
		name: collectorName, configFile: configInJSON, nextCollectionTime: time.Now(), err: nil,
	}, nil
}

//Returns name of the collector
func (collector *Collector) Name() string {
	return collector.name
}

//Returns the next collection time and collected metrics for the collector; Returns the next collection time and an error message in case of any error during metrics collection
func (collector *Collector) Collect() (time.Time, []v2.Metric, error) {
	//TO BE IMPLEMENTED
	return time.Now(), nil, nil
}
