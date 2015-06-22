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
)

type Collector struct {
	name               string
	configFile         Config
	nextCollectionTime time.Time
	err                error
}

type Config struct {
	Endpoint      string         `json:"endpoint"`
	MetricsConfig []metricConfig `json:"metricsConfig"`
}

type metricConfig struct {
	Name             string `json:"name"`
	MetricType       string `json:"metricType"`
	Units            string `json:"units"`
	PollingFrequency string `json:"pollingFrequency"`
	Regex            string `json:"regex"`
}

func SetCollectorConfig(collector *Collector, file string) error {
	configFile, err := ioutil.ReadFile(file)
	if err != nil {
		collector.err = err
	} else {
		var configInJSON Config

		err1 := json.Unmarshal(configFile, &configInJSON)
		if err1 != nil {
			collector.err = err1
		} else {
			collector.err = nil
			collector.configFile = configInJSON
		}
	}

	return collector.err
}
