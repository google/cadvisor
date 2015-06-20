package collector

import (
	"time"

	"github.com/google/cadvisor/info/v2"
)

func NewNginxCollector() (*Collector, error) {
	return &Collector{
		name: "nginx", configFile: Config{}, nextCollectionTime: time.Now(), err: nil,
	}, nil
}

//Returns name of the collector
func (collector *Collector) Name() string {
	return collector.name
}

func (collector *Collector) Collect() (time.Time, []v2.Metric, error) {
	//TO BE IMPLEMENTED
	return time.Now(), nil, nil
}
