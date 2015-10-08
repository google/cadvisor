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
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/golang/glog"
	cadvisorHttp "github.com/google/cadvisor/http"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/version"
)

var argIp = flag.String("listen_ip", "", "IP to listen on, defaults to all IPs")
var argPort = flag.Int("port", 8080, "port to listen")
var maxProcs = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).")

var argDbDriver = flag.String("storage_driver", "", "storage driver to use. Data is always cached shortly in memory, this controls where data is pushed besides the local cache. Empty means none. Options are: <empty> (default), bigquery, and influxdb")
var versionFlag = flag.Bool("version", false, "print cAdvisor version and exit")

var httpAuthFile = flag.String("http_auth_file", "", "HTTP auth file for the web UI")
var httpAuthRealm = flag.String("http_auth_realm", "localhost", "HTTP auth realm for the web UI")
var httpDigestFile = flag.String("http_digest_file", "", "HTTP digest file for the web UI")
var httpDigestRealm = flag.String("http_digest_realm", "localhost", "HTTP digest file for the web UI")

var prometheusEndpoint = flag.String("prometheus_endpoint", "/metrics", "Endpoint to expose Prometheus metrics on")

var maxHousekeepingInterval = flag.Duration("max_housekeeping_interval", 60*time.Second, "Largest interval to allow between container housekeepings")
var allowDynamicHousekeeping = flag.Bool("allow_dynamic_housekeeping", true, "Whether to allow the housekeeping interval to be dynamic")

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

	containerManager, err := manager.New(memoryStorage, sysFs, *maxHousekeepingInterval, *allowDynamicHousekeeping)
	if err != nil {
		glog.Fatalf("Failed to create a Container Manager: %s", err)
	}

	mux := http.DefaultServeMux

	// Register all HTTP handlers.
	err = cadvisorHttp.RegisterHandlers(mux, containerManager, *httpAuthFile, *httpAuthRealm, *httpDigestFile, *httpDigestRealm)
	if err != nil {
		glog.Fatalf("Failed to register HTTP handlers: %v", err)
	}

	cadvisorHttp.RegisterPrometheusHandler(mux, containerManager, *prometheusEndpoint, nil)

	// Start the manager.
	if err := containerManager.Start(); err != nil {
		glog.Fatalf("Failed to start container manager: %v", err)
	}

	// Install signal handler.
	installSignalHandler(containerManager)

	glog.Infof("Starting cAdvisor version: %s-%s on port %d", version.Info["version"], version.Info["revision"], *argPort)

	addr := fmt.Sprintf("%s:%d", *argIp, *argPort)
	glog.Fatal(http.ListenAndServe(addr, nil))
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

func installSignalHandler(containerManager manager.Manager) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Block until a signal is received.
	go func() {
		sig := <-c
		if err := containerManager.Stop(); err != nil {
			glog.Errorf("Failed to stop container manager: %v", err)
		}
		glog.Infof("Exiting given signal: %v", sig)
		os.Exit(0)
	}()
}
