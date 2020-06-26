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

package http

import (
	"fmt"
	"net/http"

	"github.com/google/cadvisor/cmd/internal/api"
	"github.com/google/cadvisor/cmd/internal/healthz"
	httpmux "github.com/google/cadvisor/cmd/internal/http/mux"
	"github.com/google/cadvisor/cmd/internal/pages"
	"github.com/google/cadvisor/cmd/internal/pages/static"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/metrics"
	"github.com/google/cadvisor/validate"

	auth "github.com/abbot/go-http-auth"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
)

func RegisterHandlers(mux httpmux.Mux, containerManager manager.Manager, httpAuthFile, httpAuthRealm, httpDigestFile, httpDigestRealm string, urlBasePrefix string) error {
	// Basic health handler.
	if err := healthz.RegisterHandler(mux); err != nil {
		return fmt.Errorf("failed to register healthz handler: %s", err)
	}

	// Validation/Debug handler.
	mux.HandleFunc(validate.ValidatePage, func(w http.ResponseWriter, r *http.Request) {
		err := validate.HandleRequest(w, containerManager)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Register API handler.
	if err := api.RegisterHandlers(mux, containerManager); err != nil {
		return fmt.Errorf("failed to register API handlers: %s", err)
	}

	// Redirect / to containers page.
	mux.Handle("/", http.RedirectHandler(urlBasePrefix+pages.ContainersPage, http.StatusTemporaryRedirect))

	var authenticated bool

	// Setup the authenticator object
	if httpAuthFile != "" {
		klog.V(1).Infof("Using auth file %s", httpAuthFile)
		secrets := auth.HtpasswdFileProvider(httpAuthFile)
		authenticator := auth.NewBasicAuthenticator(httpAuthRealm, secrets)
		mux.HandleFunc(static.StaticResource, authenticator.Wrap(staticHandler))
		if err := pages.RegisterHandlersBasic(mux, containerManager, authenticator, urlBasePrefix); err != nil {
			return fmt.Errorf("failed to register pages auth handlers: %s", err)
		}
		authenticated = true
	}
	if httpAuthFile == "" && httpDigestFile != "" {
		klog.V(1).Infof("Using digest file %s", httpDigestFile)
		secrets := auth.HtdigestFileProvider(httpDigestFile)
		authenticator := auth.NewDigestAuthenticator(httpDigestRealm, secrets)
		mux.HandleFunc(static.StaticResource, authenticator.Wrap(staticHandler))
		if err := pages.RegisterHandlersDigest(mux, containerManager, authenticator, urlBasePrefix); err != nil {
			return fmt.Errorf("failed to register pages digest handlers: %s", err)
		}
		authenticated = true
	}

	// Change handler based on authenticator initalization
	if !authenticated {
		mux.HandleFunc(static.StaticResource, staticHandlerNoAuth)
		if err := pages.RegisterHandlersBasic(mux, containerManager, nil, urlBasePrefix); err != nil {
			return fmt.Errorf("failed to register pages handlers: %s", err)
		}
	}

	return nil
}

// RegisterPrometheusHandler creates a new PrometheusCollector and configures
// the provided HTTP mux to handle the given Prometheus endpoint.
func RegisterPrometheusHandler(mux httpmux.Mux, resourceManager manager.Manager, prometheusEndpoint string,
	f metrics.ContainerLabelsFunc, includedMetrics container.MetricSet) {
	goCollector := prometheus.NewGoCollector()
	processCollector := prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{})
	machineCollector := metrics.NewPrometheusMachineCollector(resourceManager, includedMetrics)

	mux.Handle(prometheusEndpoint, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		opts, err := api.GetRequestOptions(req)
		if err != nil {
			http.Error(w, "No metrics gathered, last error:\n\n"+err.Error(), http.StatusInternalServerError)
			return
		}
		opts.Count = 1        // we only want the latest datapoint
		opts.Recursive = true // get all child containers

		r := prometheus.NewRegistry()
		r.MustRegister(
			metrics.NewPrometheusCollector(resourceManager, f, includedMetrics, clock.RealClock{}, opts),
			machineCollector,
			goCollector,
			processCollector,
		)
		promhttp.HandlerFor(r, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, req)
	}))
}

func staticHandlerNoAuth(w http.ResponseWriter, r *http.Request) {
	static.HandleRequest(w, r.URL)
}

func staticHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	static.HandleRequest(w, r.URL)
}
