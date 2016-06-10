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

	"encoding/json"
	"github.com/google/cadvisor/info/v1"
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
	MetricType v1.MetricType `json:"metric_type"`

	// metric units to display on UI and in storage (eg: MB, cores)
	// this is only used for display.
	Units string `json:"units"`

	//data type of the metric (eg: int, float)
	DataType v1.DataType `json:"data_type"`

	//the frequency at which the metric should be collected
	PollingFrequency time.Duration `json:"polling_frequency"`

	//the regular expression that can be used to extract the metric
	Regex string `json:"regex"`
}

type Prometheus struct {
	//the endpoint to hit to scrape metrics
	Endpoint string `json:"endpoint"`

	//the frequency at which metrics should be collected
	PollingFrequency time.Duration `json:"polling_frequency"`

	//holds names of different metrics that can be collected
	MetricsConfig []string `json:"metrics_config"`
}

type Jolokia struct {
	//the endpoint to hit to scrape metrics. Eg https://${HOST}/jolokia
	Endpoint JolokiaEndpointConfig `json:"endpoint"`

	//holds names of different metrics that can be collected
	MetricsConfig []JolokiaMetricConfig `json:"metrics_config"`

	//the frequency at which metrics should be collected
	PollingFrequency string `json:"polling_frequency"`

	// if insecure connections should be allowed or not
	TrustInsecure bool `json:"trust_insecure"`

	// optional value to be able to specify the CA certificate to used to verify an certificate
	CAPath string `json:"ca_cert_path"`
}

type JolokiaMetricConfig struct {
	//the name of the metric Eg "HeapMemoryUsage" or "MyMetricFoo"
	Name string `json:"name"`

	//the name of the mbean. Eg java.lang:type=Memory
	MBean string `json:"mbean"`
	// the attribute from the mbean Eg HeapMemoryUsage
	Attribute string `json:"attribute"`
	//the path contianing the mbean to be used Eg used
	Path string `json:"path"`

	//enum type for the metric type
	MetricType v1.MetricType `json:"metric_type"`

	// metric units to display on UI and in storage (eg: MB, cores)
	// this is only used for display.
	Units string `json:"units"`

	//data type of the metric (eg: int, float)
	DataType v1.DataType `json:"data_type"`
}

type JolokiaEndpointConfig struct {
	//The full URL of the endpoint to reach
	URL string
	// A configuration in which an actual URL is constructed from, using the container's ip address
	URLConfig JolokiaURLConfig
}

type JolokiaURLConfig struct {
	// the protocol to use for connecting to the endpoint. Eg 'http' or 'https'
	Protocol string `json:"protocol"`

	// the port to use for connecting to the endpoint. Eg '8778'
	Port json.Number `json:"port"`

	// the path to use for the endpoint. Eg '/jolokia'
	Path string `json:"path"`
}
