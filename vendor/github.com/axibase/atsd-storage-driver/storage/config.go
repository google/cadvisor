package storage

import (
	"os"
	"time"

	neturl "net/url"
)

type Config struct {
	Url              *neturl.URL
	MetricPrefix     string
	SelfMetricEntity string

	SenderGoroutineLimit int
	MemstoreLimit        uint

	InsecureSkipVerify bool

	UpdateInterval time.Duration

	GroupParams map[string]DeduplicationParams
}

func GetDefaultConfig() Config {
	urlStruct := &neturl.URL{
		Scheme: "tcp",
		Host:   "localhost:8081",
	}
	hostname, _ := os.Hostname()
	return Config{
		Url:                  urlStruct,
		MetricPrefix:         "storagedriver",
		SelfMetricEntity:     hostname,
		SenderGoroutineLimit: 1,
		MemstoreLimit:        1000000,
		UpdateInterval:       1 * time.Minute,
		GroupParams:          map[string]DeduplicationParams{},
	}
}
