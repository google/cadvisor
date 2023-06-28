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
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	cadvisorhttp "github.com/google/cadvisor/cmd/internal/http"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/metrics"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/version"

	// Register container providers
	_ "github.com/google/cadvisor/cmd/internal/container/install"

	// Register CloudProviders
	_ "github.com/google/cadvisor/utils/cloudinfo/aws"
	_ "github.com/google/cadvisor/utils/cloudinfo/azure"
	_ "github.com/google/cadvisor/utils/cloudinfo/gce"

	"k8s.io/klog/v2"
)

var argIP = flag.String("listen_ip", "", "IP to listen on, defaults to all IPs")
var argPort = flag.Int("port", 8080, "port to listen")
var maxProcs = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).")

var versionFlag = flag.Bool("version", false, "print cAdvisor version and exit")

var httpAuthFile = flag.String("http_auth_file", "", "HTTP auth file for the web UI")
var httpAuthRealm = flag.String("http_auth_realm", "localhost", "HTTP auth realm for the web UI")
var httpDigestFile = flag.String("http_digest_file", "", "HTTP digest file for the web UI")
var httpDigestRealm = flag.String("http_digest_realm", "localhost", "HTTP digest file for the web UI")

var prometheusEndpoint = flag.String("prometheus_endpoint", "/metrics", "Endpoint to expose Prometheus metrics on")

var enableProfiling = flag.Bool("profiling", false, "Enable profiling via web interface host:port/debug/pprof/")

var collectorCert = flag.String("collector_cert", "", "Collector's certificate, exposed to endpoints for certificate based authentication.")
var collectorKey = flag.String("collector_key", "", "Key for the collector's certificate")

var storeContainerLabels = flag.Bool("store_container_labels", true, "convert container labels and environment variables into labels on prometheus metrics for each container. If flag set to false, then only metrics exported are container name, first alias, and image name")
var whitelistedContainerLabels = flag.String("whitelisted_container_labels", "", "comma separated list of container labels to be converted to labels on prometheus metrics for each container. store_container_labels must be set to false for this to take effect.")

var envMetadataWhiteList = flag.String("env_metadata_whitelist", "", "a comma-separated list of environment variable keys matched with specified prefix that needs to be collected for containers, only support containerd and docker runtime for now.")

var urlBasePrefix = flag.String("url_base_prefix", "", "prefix path that will be prepended to all paths to support some reverse proxies")

var rawCgroupPrefixWhiteList = flag.String("raw_cgroup_prefix_whitelist", "", "A comma-separated list of cgroup path prefix that needs to be collected even when -docker_only is specified")

var perfEvents = flag.String("perf_events_config", "", "Path to a JSON file containing configuration of perf events to measure. Empty value disabled perf events measuring.")

var resctrlInterval = flag.Duration("resctrl_interval", 0, "Resctrl mon groups updating interval. Zero value disables updating mon groups.")

var (
	// Metrics to be ignored.
	// Tcp metrics are ignored by default.
	ignoreMetrics = container.MetricSet{
		container.MemoryNumaMetrics:              struct{}{},
		container.NetworkTcpUsageMetrics:         struct{}{},
		container.NetworkUdpUsageMetrics:         struct{}{},
		container.NetworkAdvancedTcpUsageMetrics: struct{}{},
		container.ProcessSchedulerMetrics:        struct{}{},
		container.ProcessMetrics:                 struct{}{},
		container.HugetlbUsageMetrics:            struct{}{},
		container.ReferencedMemoryMetrics:        struct{}{},
		container.CPUTopologyMetrics:             struct{}{},
		container.ResctrlMetrics:                 struct{}{},
		container.CPUSetMetrics:                  struct{}{},
	}

	// Metrics to be enabled.  Used only if non-empty.
	enableMetrics = container.MetricSet{}
)

func init() {
	optstr := container.AllMetrics.String()
	flag.Var(&ignoreMetrics, "disable_metrics", fmt.Sprintf("comma-separated list of `metrics` to be disabled. Options are %s.", optstr))
	flag.Var(&enableMetrics, "enable_metrics", fmt.Sprintf("comma-separated list of `metrics` to be enabled. If set, overrides 'disable_metrics'. Options are %s.", optstr))

	// Default logging verbosity to V(2)
	_ = flag.Set("v", "2")
}

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()
	flag.Parse()

	if *versionFlag {
		fmt.Printf("cAdvisor version %s (%s)\n", version.Info["version"], version.Info["revision"])
		os.Exit(0)
	}

	var includedMetrics container.MetricSet
	if len(enableMetrics) > 0 {
		includedMetrics = enableMetrics
	} else {
		includedMetrics = container.AllMetrics.Difference(ignoreMetrics)
	}
	klog.V(1).Infof("enabled metrics: %s", includedMetrics.String())
	setMaxProcs()

	memoryStorage, err := NewMemoryStorage()
	if err != nil {
		klog.Fatalf("Failed to initialize storage driver: %s", err)
	}

	sysFs := sysfs.NewRealSysFs()

	collectorHTTPClient := createCollectorHTTPClient(*collectorCert, *collectorKey)

	resourceManager, err := manager.New(memoryStorage, sysFs, manager.HousekeepingConfigFlags, includedMetrics, &collectorHTTPClient, strings.Split(*rawCgroupPrefixWhiteList, ","), strings.Split(*envMetadataWhiteList, ","), *perfEvents, *resctrlInterval)
	if err != nil {
		klog.Fatalf("Failed to create a manager: %s", err)
	}

	mux := http.NewServeMux()

	if *enableProfiling {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}

	// Register all HTTP handlers.
	err = cadvisorhttp.RegisterHandlers(mux, resourceManager, *httpAuthFile, *httpAuthRealm, *httpDigestFile, *httpDigestRealm, *urlBasePrefix)
	if err != nil {
		klog.Fatalf("Failed to register HTTP handlers: %v", err)
	}

	containerLabelFunc := metrics.DefaultContainerLabels
	if !*storeContainerLabels {
		whitelistedLabels := strings.Split(*whitelistedContainerLabels, ",")
		// Trim spacing in labels
		for i := range whitelistedLabels {
			whitelistedLabels[i] = strings.TrimSpace(whitelistedLabels[i])
		}
		containerLabelFunc = metrics.BaseContainerLabels(whitelistedLabels)
	}

	// Register Prometheus collector to gather information about containers, Go runtime, processes, and machine
	cadvisorhttp.RegisterPrometheusHandler(mux, resourceManager, *prometheusEndpoint, containerLabelFunc, includedMetrics)

	// Start the manager.
	if err := resourceManager.Start(); err != nil {
		klog.Fatalf("Failed to start manager: %v", err)
	}

	// Install signal handler.
	installSignalHandler(resourceManager)

	klog.V(1).Infof("Starting cAdvisor version: %s-%s on port %d", version.Info["version"], version.Info["revision"], *argPort)

	rootMux := http.NewServeMux()
	rootMux.Handle(*urlBasePrefix+"/", http.StripPrefix(*urlBasePrefix, mux))

	addr := fmt.Sprintf("%s:%d", *argIP, *argPort)
	klog.Fatal(http.ListenAndServe(addr, rootMux))
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
		klog.Warningf("Specified max procs of %v but using %v", numProcs, actualNumProcs)
	}
}

func installSignalHandler(containerManager manager.Manager) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received.
	go func() {
		sig := <-c
		if err := containerManager.Stop(); err != nil {
			klog.Errorf("Failed to stop container manager: %v", err)
		}
		klog.Infof("Exiting given signal: %v", sig)
		os.Exit(0)
	}()
}

func createCollectorHTTPClient(collectorCert, collectorKey string) http.Client {
	//Enable accessing insecure endpoints. We should be able to access metrics from any endpoint
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	if collectorCert != "" {
		if collectorKey == "" {
			klog.Fatal("The collector_key value must be specified if the collector_cert value is set.")
		}
		cert, err := tls.LoadX509KeyPair(collectorCert, collectorKey)
		if err != nil {
			klog.Fatalf("Failed to use the collector certificate and key: %s", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate() //nolint: staticcheck
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return http.Client{Transport: transport}
}
