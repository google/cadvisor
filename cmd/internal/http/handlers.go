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
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
)

// authWrappedMux wraps every registered handler with auth enforcement.
type authWrappedMux struct {
	mux      httpmux.Mux
	wrapFunc func(http.Handler) http.Handler
}

func (a *authWrappedMux) Handle(pattern string, h http.Handler) {
	a.mux.Handle(pattern, a.wrapFunc(h))
}

func (a *authWrappedMux) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	a.Handle(pattern, http.HandlerFunc(h))
}

func (a *authWrappedMux) Handler(r *http.Request) (http.Handler, string) {
	return a.mux.Handler(r)
}

// basicAuthMiddleware returns a middleware that requires basic authentication.
func basicAuthMiddleware(authenticator *auth.BasicAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authenticator.CheckAuth(r) == "" {
				authenticator.RequireAuth(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// digestAuthMiddleware returns a middleware that requires digest authentication.
func digestAuthMiddleware(authenticator *auth.DigestAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authenticator.CheckAuth(r) == "" {
				authenticator.RequireAuth(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

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

	// Setup authentication BEFORE registering protected handlers.
	var basicAuthenticator *auth.BasicAuth
	var digestAuthenticator *auth.DigestAuth
	var authWrap func(http.Handler) http.Handler
	var authenticated bool

	if httpAuthFile != "" {
		klog.V(1).Infof("Using auth file %s", httpAuthFile)
		secrets := auth.HtpasswdFileProvider(httpAuthFile)
		basicAuthenticator = auth.NewBasicAuthenticator(httpAuthRealm, secrets)
		authWrap = basicAuthMiddleware(basicAuthenticator)
		authenticated = true
	} else if httpDigestFile != "" {
		klog.V(1).Infof("Using digest file %s", httpDigestFile)
		secrets := auth.HtdigestFileProvider(httpDigestFile)
		digestAuthenticator = auth.NewDigestAuthenticator(httpDigestRealm, secrets)
		authWrap = digestAuthMiddleware(digestAuthenticator)
		authenticated = true
	}

	// Register API handler with auth if configured.
	apiMux := httpmux.Mux(mux)
	if authWrap != nil {
		apiMux = &authWrappedMux{mux: mux, wrapFunc: authWrap}
	}
	if err := api.RegisterHandlers(apiMux, containerManager); err != nil {
		return fmt.Errorf("failed to register API handlers: %s", err)
	}

	// Redirect / to containers page.
	mux.Handle("/", http.RedirectHandler(urlBasePrefix+pages.ContainersPage, http.StatusTemporaryRedirect))

	// Register pages and static resources with auth if configured.
	if basicAuthenticator != nil {
		mux.HandleFunc(static.StaticResource, basicAuthenticator.Wrap(staticHandler))
		if err := pages.RegisterHandlersBasic(mux, containerManager, basicAuthenticator, urlBasePrefix); err != nil {
			return fmt.Errorf("failed to register pages auth handlers: %s", err)
		}
	} else if digestAuthenticator != nil {
		mux.HandleFunc(static.StaticResource, digestAuthenticator.Wrap(staticHandler))
		if err := pages.RegisterHandlersDigest(mux, containerManager, digestAuthenticator, urlBasePrefix); err != nil {
			return fmt.Errorf("failed to register pages digest handlers: %s", err)
		}
	} else {
		mux.HandleFunc(static.StaticResource, staticHandlerNoAuth)
		if err := pages.RegisterHandlersBasic(mux, containerManager, nil, urlBasePrefix); err != nil {
			return fmt.Errorf("failed to register pages handlers: %s", err)
		}
	}

	return nil
}

// RegisterPrometheusHandler creates a new PrometheusCollector and configures
// the provided HTTP mux to handle the given Prometheus endpoint.
// If auth is configured, the Prometheus endpoint requires authentication.
func RegisterPrometheusHandler(mux httpmux.Mux, resourceManager manager.Manager, prometheusEndpoint string,
	f metrics.ContainerLabelsFunc, includedMetrics container.MetricSet,
	httpAuthFile, httpAuthRealm, httpDigestFile, httpDigestRealm string) {

	goCollector := collectors.NewGoCollector()
	processCollector := collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
	machineCollector := metrics.NewPrometheusMachineCollector(resourceManager, includedMetrics)

	prometheusHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
	})

	// Wrap with authentication if configured.
	finalHandler := http.Handler(prometheusHandler)
	if httpAuthFile != "" {
		secrets := auth.HtpasswdFileProvider(httpAuthFile)
		authenticator := auth.NewBasicAuthenticator(httpAuthRealm, secrets)
		finalHandler = basicAuthMiddleware(authenticator)(finalHandler)
	} else if httpDigestFile != "" {
		secrets := auth.HtdigestFileProvider(httpDigestFile)
		authenticator := auth.NewDigestAuthenticator(httpDigestRealm, secrets)
		finalHandler = digestAuthMiddleware(authenticator)(finalHandler)
	}

	mux.Handle(prometheusEndpoint, finalHandler)
}

func staticHandlerNoAuth(w http.ResponseWriter, r *http.Request) {
	static.HandleRequest(w, r.URL)
}

func staticHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	static.HandleRequest(w, r.URL)
}
