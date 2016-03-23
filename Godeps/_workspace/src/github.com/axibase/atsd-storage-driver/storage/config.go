package storage

import (
	"errors"
	neturl "net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Url                  *neturl.URL
	DataReceiverHostport string
	Protocol             string
	MetricPrefix         string
	SelfMetricEntity     string

	ConnectionLimit uint
	MemstoreLimit   uint

	Username string
	Password string

	UpdateInterval time.Duration

	GroupParams map[string]DeduplicationParams
}

func GetDefaultConfig() Config {
	urlStruct, _ := neturl.ParseRequestURI("http://localhost:8088")
	return Config{
		Url:                  urlStruct,
		DataReceiverHostport: "localhost:8082",
		Protocol:             "tcp",
		MetricPrefix:         "storagedriver",
		SelfMetricEntity:     "hostname",
		ConnectionLimit:      1,
		MemstoreLimit:        1000000,
		Username:             "admin",
		Password:             "admin",
		UpdateInterval:       1 * time.Minute,
		GroupParams:          map[string]DeduplicationParams{},
	}
}

func (self *Config) UnmarshalTOML(data interface{}) error {
	d, _ := data.(map[string]interface{})

	if u, ok := d["url"]; ok {
		urlString, _ := u.(string)
		url, err := neturl.ParseRequestURI(urlString)
		if err != nil {
			return err
		}
		self.Url = url
	}

	if wh, ok := d["write_host"]; ok {
		writeHost, _ := wh.(string)
		self.DataReceiverHostport = writeHost
	}

	if p, ok := d["write_protocol"]; ok {
		protocol, _ := p.(string)
		switch protocol {
		case "tcp", "udp", "http/https":
			self.Protocol = protocol
		default:
			return errors.New("Unknown protocol type")
		}

	}

	if mp, ok := d["metric_prefix"]; ok {
		metricPrefix, _ := mp.(string)
		self.MetricPrefix = metricPrefix
	}

	if sme, ok := d["self_metric_entity"]; ok {
		selfMetricEntity, _ := sme.(string)
		self.SelfMetricEntity = selfMetricEntity
	}

	if cl, ok := d["connection_limit"]; ok {
		connectionLimit, _ := cl.(int64)
		self.ConnectionLimit = uint(connectionLimit)
	}

	if ml, ok := d["memstore_limit"]; ok {
		memstoreLimit, _ := ml.(int64)
		self.MemstoreLimit = uint(memstoreLimit)
	}

	if u, ok := d["username"]; ok {
		username, _ := u.(string)
		self.Username = username
	}
	if p, ok := d["password"]; ok {
		password, _ := p.(string)
		self.Password = password
	}

	if ui, ok := d["update_interval"]; ok {
		updateInterval, _ := ui.(string)
		duration, err := time.ParseDuration(updateInterval)
		if err != nil {
			return errors.New("unknown update_interval format")
		}
		self.UpdateInterval = duration
	}

	if self.GroupParams == nil {
		self.GroupParams = map[string]DeduplicationParams{}
	}
	if g, ok := d["deduplication"]; ok {
		groups, _ := g.(map[string]interface{})
		for key, val := range groups {
			m := val.(map[string]interface{})
			thresholdString, _ := m["threshold"].(string)
			var threshold interface{}
			if strings.HasSuffix(thresholdString, "%") {
				val, err := strconv.ParseFloat(strings.TrimSuffix(thresholdString, "%"), 64)
				if err != nil {
					panic(err)
				}
				threshold = Percent(val)
			} else {
				val, err := strconv.ParseFloat(thresholdString, 64)
				if err != nil {
					panic(err)
				}
				threshold = Absolute(val)
			}

			intervalString, _ := m["interval"].(string)
			interval, err := time.ParseDuration(intervalString)
			if err != nil {
				return err
			}
			self.GroupParams[key] = DeduplicationParams{Threshold: threshold, Interval: interval}
		}
	}

	return nil
}
