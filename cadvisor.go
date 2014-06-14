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
	"log"
	"net/http"
	"time"

	"github.com/google/cadvisor/api"
	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/container/lmctfy"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/pages"
	"github.com/google/cadvisor/pages/static"
)

var argPort = flag.Int("port", 8080, "port to listen")
var argSampleSize = flag.Int("samples", 1024, "number of samples we want to keep")
var argResetPeriod = flag.Duration("reset_period", 2*time.Hour, "period to reset the samples")

func main() {
	flag.Parse()

	decorators := make([]container.ContainerHandlerDecorator, 0, 2)
	// XXX(dengnan): Should we allow users to specify which sampler they want to use?
	dec, err := container.NewPercentilesDecorator(&container.StatsParameter{
		Sampler:     "uniform",
		NumSamples:  *argSampleSize,
		ResetPeriod: *argResetPeriod,
	})
	if err != nil {
		log.Fatalf("unalbe to get percentiles decorator: %v", err)
	}
	decorators = append(decorators, dec)

	// TODO(dengnan): Add StatsWriterDecorator

	if len(decorators) > 0 {
		container.RegisterContainerHandlerDecorators(decorators...)
	}

	containerManager, err := manager.New()
	if err != nil {
		log.Fatalf("Failed to create a Container Manager: %s", err)
	}

	if err := lmctfy.Register("/"); err != nil {
		log.Printf("lmctfy registration failed: %v.", err)
		log.Print("Running in docker only mode.")
		if err := docker.Register(containerManager, "/"); err != nil {
			log.Printf("Docker registration failed: %v.", err)
			log.Fatalf("Unable to continue without docker or lmctfy.")
		}
	}

	if err := docker.Register(containerManager, "/docker"); err != nil {
		// Ignore this error because we should work with lmctfy only
		log.Printf("Docker registration failed: %v.", err)
		log.Print("Running in lmctfy only mode.")
	}

	// Handler for static content.
	http.HandleFunc(static.StaticResource, func(w http.ResponseWriter, r *http.Request) {
		err := static.HandleRequest(w, r.URL)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	// Handler for the API.
	http.HandleFunc(api.ApiResource, func(w http.ResponseWriter, r *http.Request) {
		err := api.HandleRequest(containerManager, w, r.URL)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	// Redirect / to containers page.
	http.Handle("/", http.RedirectHandler(pages.ContainersPage, http.StatusTemporaryRedirect))

	// Register the handler for the containers page.
	http.HandleFunc(pages.ContainersPage, func(w http.ResponseWriter, r *http.Request) {
		err := pages.ServerContainersPage(containerManager, w, r.URL)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	go containerManager.Start()

	log.Print("About to serve on port ", *argPort)

	addr := fmt.Sprintf(":%v", *argPort)
	log.Fatal(http.ListenAndServe(addr, nil))
}
