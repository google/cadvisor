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

	"github.com/golang/glog"
	"github.com/google/cadvisor/api"
	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/container/raw"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/pages"
	"github.com/google/cadvisor/pages/static"
)

var argPort = flag.Int("port", 8080, "port to listen")

var argDbDriver = flag.String("storage_driver", "memory", "storage driver to use. Options are: memory (default), bigquery, and influxdb")

func main() {
	flag.Parse()

	storageDriver, err := NewStorageDriver(*argDbDriver)
	if err != nil {
		glog.Fatalf("Failed to connect to database: %s", err)
	}

	containerManager, err := manager.New(storageDriver)
	if err != nil {
		glog.Fatalf("Failed to create a Container Manager: %s", err)
	}

	// Register Docker.
	if err := docker.Register(containerManager); err != nil {
		glog.Errorf("Docker registration failed: %v.", err)
	}

	// Register the raw driver.
	if err := raw.Register(containerManager); err != nil {
		glog.Fatalf("raw registration failed: %v.", err)
	}

	// Handler for static content.
	http.HandleFunc(static.StaticResource, func(w http.ResponseWriter, r *http.Request) {
		err := static.HandleRequest(w, r.URL)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	// Register API handler.
	if err := api.RegisterHandlers(containerManager); err != nil {
		glog.Fatalf("failed to register API handlers: %s", err)
	}

	// Redirect / to containers page.
	http.Handle("/", http.RedirectHandler(pages.ContainersPage, http.StatusTemporaryRedirect))

	// Register the handler for the containers page.
	http.HandleFunc(pages.ContainersPage, func(w http.ResponseWriter, r *http.Request) {
		err := pages.ServerContainersPage(containerManager, w, r.URL)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	defer glog.Flush()

	go func() {
		glog.Fatal(containerManager.Start())
	}()

	glog.Infof("Starting cAdvisor version: %q", info.VERSION)
	glog.Infof("About to serve on port ", *argPort)

	addr := fmt.Sprintf(":%v", *argPort)

	glog.Fatal(http.ListenAndServe(addr, nil))
}
