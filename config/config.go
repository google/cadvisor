package config

import "time"

var Global = Config{
	CadvisorPort:                 8080,
	HTTPAuthRealm:                "localhost",
	HTTPDigestRealm:              "localhost",
	PrometheusEndpoint:           "/metrics",
	MaxHousekeepingInterval:      60 * time.Second,
	AllowDynamicHousekeeping:     true,
	EnableProfiling:              false,
	ContainerHintsFile:           "/etc/cadvisor/container_hints.json",
	HousekeepingInterval:         1 * time.Second,
	MachineIDFilePath:            "/etc/machine-id,/var/lib/dbus/machine-id",
	BootIDFilePath:               "/proc/sys/kernel/random/boot_id",
	GlobalHousekeepingInterval:   1 * time.Minute,
	LogCadvisorUsage:             false,
	EnableLoadReader:             false,
	EventStorageAgeLimit:         "default=24h",
	EventStorageEventLimit:       "default=100000",
	ApplicationMetricsCountLimit: 100,
	ClientSecret:                 "notasecret",
	Docker: Docker{
		Endpoint: "unix:///var/run/docker.sock",
		RootDir:  "/var/lib/docker",
		RunDir:   "/var/run/docker",
	},
	CacheDuration: 2 * time.Minute,
}

type Config struct {
	// IP to listen on, defaults to all IPs
	CadvisorIP string
	// port to listen
	CadvisorPort int
	// max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).
	MaxProcs int
	// HTTP auth file for the web UI
	HTTPAuthFile string
	// HTTP auth realm for the web UI
	HTTPAuthRealm string
	// HTTP digest file for the web UI
	HTTPDigestFile string
	// HTTP digest file for the web UI
	HTTPDigestRealm string
	// Endpoint to expose Prometheus metrics on
	PrometheusEndpoint string
	// Largest interval to allow between container housekeepings
	MaxHousekeepingInterval time.Duration
	// Whether to allow the housekeeping interval to be dynamic
	AllowDynamicHousekeeping bool
	// Enable profiling via web interface host:port/debug/pprof/
	EnableProfiling bool
	// location of the container hints file
	ContainerHintsFile string
	// Interval between container housekeepings
	HousekeepingInterval time.Duration
	// Comma-separated list of files to check for machine-id. Use the first one that exists.
	MachineIDFilePath string
	// Comma-separated list of files to check for boot-id. Use the first one that exists.
	BootIDFilePath string
	// Interval between global housekeepings
	GlobalHousekeepingInterval time.Duration
	// Whether to log the usage of the cAdvisor container
	LogCadvisorUsage bool
	// Whether to enable cpu load reader
	EnableLoadReader bool
	// Max length of time for which to store events (per type). Value is a comma separated list of key values, where the keys are event types (e.g.: creation, oom) or \"default\" and the value is a duration. Default is applied to all non-specified event types
	EventStorageAgeLimit string
	// Max number of events to store (per type). Value is a comma separated list of key values, where the keys are event types (e.g.: creation, oom) or \"default\" and the value is an integer. Default is applied to all non-specified event types
	EventStorageEventLimit string
	// Max number of application metrics to store (per container)
	ApplicationMetricsCountLimit int
	// Client ID
	ClientID string
	// Client Secret
	ClientSecret string
	// Bigquery project ID
	ProjectID string
	// Service account email
	ServiceAccount string
	// Credential Key file (pem)
	PemFile string
	// Docker specific options.
	Docker Docker
	// FIXME -- collector struct
	// Collector's certificate, exposed to endpoints for certificate based authentication.
	CollectorCert string
	// Key for the collector's certificate
	CollectorKey string
	// How long to keep data cached in memory (Default: 2min).
	CacheDuration time.Duration
}

type Docker struct {
	// docker endpoint
	Endpoint string
	// Absolute path to the Docker state root directory (default: /var/lib/docker)
	RootDir string
	// Absolute path to the Docker run directory (default: /var/run/docker)
	RunDir string
	// a comma-separated list of environment variable keys that needs to be collected for docker containers
	EnvWhitelist string
	// Only report docker containers in addition to root stats
	DockerOnly bool
}
