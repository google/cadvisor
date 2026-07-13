// Copyright 2024 Google Inc. All Rights Reserved.
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

// Package appmetrics builds the application-metrics collector manager for the
// full cAdvisor binary. It is injected into the lean library manager via
// manager.CollectorManagerFactory (the kubelet leaves that nil and runs no
// collectors). The collector implementations live in the root collector package
// rather than the library to keep the library lean.
package appmetrics

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/cadvisor/collector"
	"github.com/google/cadvisor/lib/container"

	"k8s.io/klog/v2"
)

// client is the process-wide HTTP client used to scrape collector endpoints. It
// accepts any TLS certificate (collector endpoints are frequently self-signed);
// SetHTTPClient adds client-certificate (mutual TLS) auth when configured.
var client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

// SetHTTPClient (re)configures the collector HTTP client. With a cert/key it
// adds client-certificate (mutual TLS) authentication to collector endpoints.
// Call once at startup (from main, after flag parsing) before any collectors
// are built; an invalid cert/key is fatal, matching upstream's startup check.
func SetHTTPClient(certFile, keyFile string) {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	if certFile != "" {
		if keyFile == "" {
			klog.Fatal("the collector_key value must be specified if the collector_cert value is set")
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			klog.Fatalf("failed to use the collector certificate and key: %s", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	client = &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
}

// NewManager builds a collector manager for a container, registering the
// application-metrics collectors declared via its labels. readFile reads a
// collector config file from inside the container. It satisfies the shape the
// library's manager.CollectorManagerFactory expects.
func NewManager(handler container.ContainerHandler, readFile func(string) ([]byte, error), countLimit int) (collector.CollectorManager, error) {
	cm, err := collector.NewCollectorManager()
	if err != nil {
		return nil, err
	}
	for name, configPath := range collector.GetCollectorConfigs(handler.GetContainerLabels()) {
		configFile, err := readFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config %q for collector %q: %v", configPath, name, err)
		}
		var c collector.Collector
		if strings.HasPrefix(strings.ToLower(name), "prometheus") {
			c, err = collector.NewPrometheusCollector(name, configFile, countLimit, handler, client)
		} else {
			c, err = collector.NewCollector(name, configFile, countLimit, handler, client)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create collector %q: %v", name, err)
		}
		if err := cm.RegisterCollector(c); err != nil {
			return nil, fmt.Errorf("failed to register collector %q: %v", name, err)
		}
	}
	return cm, nil
}
