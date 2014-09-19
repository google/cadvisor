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
	"runtime"

	"github.com/golang/glog"
	"github.com/google/cadvisor/api"
	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/container/raw"
	"github.com/google/cadvisor/healthz"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/pages"
	"github.com/google/cadvisor/pages/static"
)

var argPort = flag.Int("port", 8080, "port to listen")
var maxProcs = flag.Int("max_procs", 0, "max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).")

var argDbDriver = flag.String("storage_driver", "", "storage driver to use. Empty means none. Options are: <empty> (default), bigquery, and influxdb")

func main() {
	flag.Parse()

	setMaxProcs()

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

	// Basic health handler.
	if err := healthz.RegisterHandler(); err != nil {
		glog.Fatalf("failed to register healthz handler: %s", err)
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

	errChan := make(chan error)
	go func() {
		errChan <- containerManager.Start()
	}()

	glog.Infof("Starting cAdvisor version: %q", info.VERSION)
	glog.Infof("About to serve on port %d", *argPort)

	addr := fmt.Sprintf(":%v", *argPort)

	go func() {
		errChan <- http.ListenAndServe(addr, nil)
	}()
	select {
	case err := <-errChan:
		glog.Fatal(err)
	}
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
