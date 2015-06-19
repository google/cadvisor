package collector

import (
	"io/ioutil"
	"time"
	"encoding/json"
)

type Collector struct {
	name               string
	configFile         Config
	nextCollectionTime time.Time
	err                error
}

type Config struct {
	Endpoint      string         "json:'endpoint'"
	MetricsConfig []metricConfig "json:'metrics'"
}

type metricConfig struct {
	Name             string "json:'name'"
	MetricType       string "json:'metricType'"
	Units            string "json:'units'"
	PollingFrequency string "json:'pollingFrequency'"
	Regex            string "json:'regex'"
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

