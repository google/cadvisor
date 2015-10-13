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

	auth "github.com/abbot/go-http-auth"
	"github.com/golang/glog"
	"github.com/google/cadvisor/api"
	"github.com/google/cadvisor/healthz"
	httpMux "github.com/google/cadvisor/http/mux"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/metrics"
	"github.com/google/cadvisor/pages"
	"github.com/google/cadvisor/pages/static"
	"github.com/google/cadvisor/validate"
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterHandlers(mux httpMux.Mux, containerManager manager.Manager, httpAuthFile, httpAuthRealm, httpDigestFile, httpDigestRealm string) error {
	// Basic health handler.
	if err := healthz.RegisterHandler(mux); err != nil {
		return fmt.Errorf("failed to register healthz handler: %s", err)
	}

	// Validation/Debug handler.
	mux.HandleFunc(validate.ValidatePage, func(w http.ResponseWriter, r *http.Request) {
		err := validate.HandleRequest(w, containerManager)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	// Register API handler.
	if err := api.RegisterHandlers(mux, containerManager); err != nil {
		return fmt.Errorf("failed to register API handlers: %s", err)
	}

	// Redirect / to containers page.
	mux.Handle("/", http.RedirectHandler(pages.ContainersPage, http.StatusTemporaryRedirect))

	var authenticator auth.AuthenticatorInterface

	staticHandler := static.FileServer()

	// Setup the authenticator object
	if httpAuthFile != "" {
		glog.Infof("Using auth file %s", httpAuthFile)
		secrets := auth.HtpasswdFileProvider(httpAuthFile)
		authenticator = auth.NewBasicAuthenticator(httpAuthRealm, secrets)
	}
	if httpAuthFile == "" && httpDigestFile != "" {
		glog.Infof("Using digest file %s", httpDigestFile)
		secrets := auth.HtdigestFileProvider(httpDigestFile)
		authenticator = auth.NewDigestAuthenticator(httpDigestRealm, secrets)
	}

	if err := pages.RegisterHandlers(mux, containerManager, authenticator); err != nil {
		return fmt.Errorf("failed to register pages auth handlers: %s", err)
	}

	if authenticator != nil {
		mux.Handle(static.StaticResource, authenticator.Wrap(func(w http.ResponseWriter, ar *auth.AuthenticatedRequest) {
			staticHandler.ServeHTTP(w, &ar.Request)
		}))
	} else {
		mux.Handle(static.StaticResource, staticHandler)
	}

	return nil
}

func RegisterPrometheusHandler(mux httpMux.Mux, containerManager manager.Manager, prometheusEndpoint string, containerNameToLabelsFunc metrics.ContainerNameToLabelsFunc) {
	collector := metrics.NewPrometheusCollector(containerManager, containerNameToLabelsFunc)
	prometheus.MustRegister(collector)
	mux.Handle(prometheusEndpoint, prometheus.Handler())
}
