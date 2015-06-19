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
	"time"
)

type Config struct {
	//the endpoint to hit to scrape metrics
	Endpoint string `json:"endpoint"`

	//holds information about different metrics that can be collected
	MetricsConfig []MetricConfig `json:"metrics_config"`
}

// metricConfig holds information extracted from the config file about a metric
type MetricConfig struct {
	//the name of the metric
	Name string `json:"name"`

	//enum type for the metric type
	MetricType MetricType `json:"metric_type"`

	//data type of the metric (eg: integer, string)
	Units string `json:"units"`

	//the frequency at which the metric should be collected
	PollingFrequency time.Duration `json:"polling_frequency"`

	//the regular expression that can be used to extract the metric
	Regex string `json:"regex"`
}

// MetricType is an enum type that lists the possible type of the metric
type MetricType string

const (
	Counter MetricType = "counter"
	Gauge   MetricType = "gauge"
)
