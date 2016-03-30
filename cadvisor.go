// Copyright 2014 Google Inc. All Rights Reserved.
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

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/google/cadvisor/container"
	cadvisorhttp "github.com/google/cadvisor/http"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/version"

	"github.com/golang/glog"
)

var argPath = flag.String("listen_path", "", "Path to listen on (UNIX socket), defaults to empty (use TCP instead)")
var argIp = flag.String("listen_ip", "", "IP to listen on, defaults to all IPs")
var argPort = flag.Int("port", 8080, "port to listen")
var maxProcs = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).")

var argDbDriver = flag.String("storage_driver", "", "storage driver to use. Data is always cached shortly in memory, this controls where data is pushed besides the local cache. Empty means none. Options are: <empty> (default), bigquery, influxdb, and kafka")
var versionFlag = flag.Bool("version", false, "print cAdvisor version and exit")

var httpAuthFile = flag.String("http_auth_file", "", "HTTP auth file for the web UI")
var httpAuthRealm = flag.String("http_auth_realm", "localhost", "HTTP auth realm for the web UI")
var httpDigestFile = flag.String("http_digest_file", "", "HTTP digest file for the web UI")
var httpDigestRealm = flag.String("http_digest_realm", "localhost", "HTTP digest file for the web UI")

var prometheusEndpoint = flag.String("prometheus_endpoint", "/metrics", "Endpoint to expose Prometheus metrics on")

var maxHousekeepingInterval = flag.Duration("max_housekeeping_interval", 60*time.Second, "Largest interval to allow between container housekeepings")
var allowDynamicHousekeeping = flag.Bool("allow_dynamic_housekeeping", true, "Whether to allow the housekeeping interval to be dynamic")

var enableProfiling = flag.Bool("profiling", false, "Enable profiling via web interface host:port/debug/pprof/")

var (
	// Metrics to be ignored.
	// Tcp metrics are ignored by default.
	ignoreMetrics metricSetValue = metricSetValue{container.MetricSet{container.NetworkTcpUsageMetrics: struct{}{}}}

	// List of metrics that can be ignored.
	ignoreWhitelist = container.MetricSet{
		container.DiskUsageMetrics:       struct{}{},
		container.NetworkUsageMetrics:    struct{}{},
		container.NetworkTcpUsageMetrics: struct{}{},
	}
)

type metricSetValue struct {
	container.MetricSet
}

func (ml *metricSetValue) String() string {
	return fmt.Sprint(*ml)
}

func (ml *metricSetValue) Set(value string) error {
	ignoreMetrics = metricSetValue{}
	if value == "" {
		return nil
	}
	for _, metric := range strings.Split(value, ",") {
		if ignoreWhitelist.Has(container.MetricKind(metric)) {
			(*ml).Add(container.MetricKind(metric))
		} else {
			return fmt.Errorf("unsupported metric %q specified in disable_metrics", metric)
		}
	}
	return nil
}

func init() {
	flag.Var(&ignoreMetrics, "disable_metrics", "comma-separated list of metrics to be disabled. Options are `disk`, `network`, `tcp`. Note: tcp is disabled by default due to high CPU usage.")
}

func main() {
	defer glog.Flush()
	flag.Parse()

	if *versionFlag {
		fmt.Printf("cAdvisor version %s (%s)\n", version.Info["version"], version.Info["revision"])
		os.Exit(0)
	}

	setMaxProcs()

	memoryStorage, err := NewMemoryStorage(*argDbDriver)
	if err != nil {
		glog.Fatalf("Failed to connect to database: %s", err)
	}

	sysFs, err := sysfs.NewRealSysFs()
	if err != nil {
		glog.Fatalf("Failed to create a system interface: %s", err)
	}

	containerManager, err := manager.New(memoryStorage, sysFs, *maxHousekeepingInterval, *allowDynamicHousekeeping, ignoreMetrics.MetricSet)
	if err != nil {
		glog.Fatalf("Failed to create a Container Manager: %s", err)
	}

	mux := http.NewServeMux()

	if *enableProfiling {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}

	// Register all HTTP handlers.
	err = cadvisorhttp.RegisterHandlers(mux, containerManager, *httpAuthFile, *httpAuthRealm, *httpDigestFile, *httpDigestRealm)
	if err != nil {
		glog.Fatalf("Failed to register HTTP handlers: %v", err)
	}

	cadvisorhttp.RegisterPrometheusHandler(mux, containerManager, *prometheusEndpoint, nil)

	// Start the manager.
	if err := containerManager.Start(); err != nil {
		glog.Fatalf("Failed to start container manager: %v", err)
	}

	var listener net.Listener

	if *argPath != "" {
		if _, err := os.Stat(*argPath); err == nil {
			glog.Infof("Deleting existing socket at %s", *argPath)
			os.Remove(*argPath)
		}

		var err error
		listener, err = net.Listen("unix", *argPath)
		if err != nil {
			glog.Fatalf("Failed to start listening on UNIX socket at %s: %v", *argPath, err)
		}
		if err := os.Chmod(*argPath, 0660); err != nil {
			glog.Fatalf("Failed to change permissions on UNIX socket at %s: %v", *argPath, err)
		}
	} else {
		var err error
		listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", *argIp, *argPort))
		if err != nil {
			glog.Fatalf("Failed to start listening on TCP socket at %s:%d: %v", *argIp, *argPort, err)
		}
	}

	glog.Infof("Starting cAdvisor version: %s-%s on %s", version.Info["version"], version.Info["revision"], listener.Addr())

	// Install signal handler.
	installSignalHandler(containerManager, listener)

	// Start serving requests
	glog.Fatal(http.Serve(listener, mux))
}

func setMaxProcs() {
	// TODO(vmarmol): Consider limiting if we have a CPU mask in effect.
	// Allow as many threads as we have cores unless the user specified a value.
	var numProcs int
	if *maxProcs < 1 {
		numProcs = runtime.NumCPU()
	} else {
		numProcs = *maxProcs
	}
	runtime.GOMAXPROCS(numProcs)

	// Check if the setting was successful.
	actualNumProcs := runtime.GOMAXPROCS(0)
	if actualNumProcs != numProcs {
		glog.Warningf("Specified max procs of %v but using %v", numProcs, actualNumProcs)
	}
}

func installSignalHandler(containerManager manager.Manager, listener net.Listener) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Block until a signal is received.
	go func() {
		sig := <-c
		glog.Infof("Exiting containerManager")
		if err := containerManager.Stop(); err != nil {
			glog.Errorf("Failed to stop container manager: %v", err)
		}
		glog.Infof("Exiting listener")
		listener.Close()
		glog.Infof("Exiting given signal: %v", sig)
		os.Exit(0)
	}()
}
