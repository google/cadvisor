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
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/utils"
)

type GenericCollector struct {
	//name of the collector
	name string

	//holds information extracted from the config file for a collector
	configFile Config

	//holds information necessary to extract metrics
	info *collectorInfo

	metrics []v1.Metric
}

type collectorInfo struct {
	//minimum polling frequency among all metrics
	minPollingFrequency time.Duration

	//regular expresssions for all metrics
	regexps []*regexp.Regexp
}

//Returns a new collector using the information extracted from the configfile
func NewCollector(collectorName string, configFile []byte) (*GenericCollector, error) {
	var configInJSON Config
	err := json.Unmarshal(configFile, &configInJSON)
	if err != nil {
		return nil, err
	}

	//TODO : Add checks for validity of config file (eg : Accurate JSON fields)

	if len(configInJSON.MetricsConfig) == 0 {
		return nil, fmt.Errorf("No metrics provided in config")
	}

	minPollFrequency := time.Duration(0)
	regexprs := make([]*regexp.Regexp, len(configInJSON.MetricsConfig))

	var metrics []v1.Metric
	var metricType v1.MetricType

	for ind, metricConfig := range configInJSON.MetricsConfig {
		// Find the minimum specified polling frequency in metric config.
		if metricConfig.PollingFrequency != 0 {
			if minPollFrequency == 0 || metricConfig.PollingFrequency < minPollFrequency {
				minPollFrequency = metricConfig.PollingFrequency
			}
		}

		regexprs[ind], err = regexp.Compile(metricConfig.Regex)
		if err != nil {
			return nil, fmt.Errorf("Invalid regexp %v for metric %v", metricConfig.Regex, metricConfig.Name)
		}

		if metricConfig.MetricType == "gauge" {
			metricType = v1.MetricGauge
		} else if metricConfig.MetricType == "counter" {
			metricType = v1.MetricCumulative
		}

		metrics = append(metrics, v1.Metric{
			Name:       metricConfig.Name,
			Type:       metricType,
			Labels:     make(map[string]string),
			DataPoints: utils.NewTimedStore(2*time.Minute, -1), //TimedStore : age is set to storageDuration, maxItems is set to -1 (no limit)
		})
	}

	// Minimum supported polling frequency is 1s.
	minSupportedFrequency := 1 * time.Second
	if minPollFrequency < minSupportedFrequency {
		minPollFrequency = minSupportedFrequency
	}

	return &GenericCollector{
		name:       collectorName,
		configFile: configInJSON,
		info: &collectorInfo{
			minPollingFrequency: minPollFrequency,
			regexps:             regexprs},
		metrics: metrics,
	}, nil
}

//Returns name of the collector
func (collector *GenericCollector) Name() string {
	return collector.name
}

//Returns collected metrics and the next collection time of the collector
func (collector *GenericCollector) Collect() (time.Time, []v1.Metric, error) {
	currentTime := time.Now()
	nextCollectionTime := currentTime.Add(time.Duration(collector.info.minPollingFrequency))

	uri := collector.configFile.Endpoint
	response, err := http.Get(uri)
	if err != nil {
		return nextCollectionTime, nil, err
	}

	defer response.Body.Close()

	pageContent, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nextCollectionTime, nil, err
	}

	var errorSlice []error

	for ind, metricConfig := range collector.configFile.MetricsConfig {
		matchString := collector.info.regexps[ind].FindStringSubmatch(string(pageContent))
		if matchString != nil {
			collector.metrics[ind].Lock.Lock()
			defer collector.metrics[ind].Lock.Unlock()
			if metricConfig.Units == "float" {
				regVal, err := strconv.ParseFloat(strings.TrimSpace(matchString[1]), 64)
				if err != nil {
					errorSlice = append(errorSlice, err)
				}
				collector.metrics[ind].DataPoints.Add(currentTime, v1.DataPoint{
					Value:     regVal,
					Timestamp: currentTime,
				})
			} else if metricConfig.Units == "integer" || metricConfig.Units == "int" {
				regVal, err := strconv.ParseInt(strings.TrimSpace(matchString[1]), 10, 64)
				if err != nil {
					errorSlice = append(errorSlice, err)
				}
				collector.metrics[ind].DataPoints.Add(currentTime, v1.DataPoint{
					Value:     regVal,
					Timestamp: currentTime,
				})
			} else {
				errorSlice = append(errorSlice, fmt.Errorf("Unexpected value of 'units' for metric '%v' in config ", metricConfig.Name))
			}
		} else {
			errorSlice = append(errorSlice, fmt.Errorf("No match found for regexp: %v for metric '%v' in config", metricConfig.Regex, metricConfig.Name))
		}
	}

	return nextCollectionTime, collector.metrics, compileErrors(errorSlice)
}

func (collector *GenericCollector) GetMetrics() (map[string][]v1.DataPoint, error) {
	metrics := make(map[string][]v1.DataPoint)
	var zeroTime time.Time
	for _, metric := range collector.metrics {
		metric.Lock.Lock()
		//defer metric.Lock.Unlock()
		data := metric.DataPoints.InTimeRange(zeroTime, zeroTime, -1)
		if len(data) != 0 {
			dataPoints := make([]v1.DataPoint, len(data))
			for i, val := range data {
				dataPoints[i] = val.(v1.DataPoint)
			}
			metrics[metric.Name] = dataPoints
		}
	}
	return metrics, nil
}
