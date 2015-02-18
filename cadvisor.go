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

	auth "github.com/abbot/go-http-auth"
	"github.com/golang/glog"
	"github.com/google/cadvisor/api"
	"github.com/google/cadvisor/container/docker"
	"github.com/google/cadvisor/container/raw"
	"github.com/google/cadvisor/healthz"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
	"github.com/google/cadvisor/pages"
	"github.com/google/cadvisor/pages/static"
	"github.com/google/cadvisor/utils/sysfs"
	"github.com/google/cadvisor/validate"
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

func main() {
	defer glog.Flush()
	flag.Parse()

	if *versionFlag {
		fmt.Printf("cAdvisor version %s\n", info.VERSION)
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

	containerManager, err := manager.New(memoryStorage, sysFs)
	if err != nil {
		glog.Fatalf("Failed to create a Container Manager: %s", err)
	}

	// Register Docker.
	if err := docker.Register(containerManager); err != nil {
		glog.Errorf("Docker registration failed: %v.", err)
	}

	// Register the raw driver.
	if err := raw.Register(containerManager); err != nil {
		glog.Fatalf("Raw registration failed: %v.", err)
	}

	// Basic health handler.
	if err := healthz.RegisterHandler(); err != nil {
		glog.Fatalf("Failed to register healthz handler: %s", err)
	}

	// Validation/Debug handler.
	http.HandleFunc(validate.ValidatePage, func(w http.ResponseWriter, r *http.Request) {
		err := validate.HandleRequest(w, containerManager)
		if err != nil {
			fmt.Fprintf(w, "%s", err)
		}
	})

	// Register API handler.
	if err := api.RegisterHandlers(containerManager); err != nil {
		glog.Fatalf("Failed to register API handlers: %s", err)
	}

	// Redirect / to containers page.
	http.Handle("/", http.RedirectHandler(pages.ContainersPage, http.StatusTemporaryRedirect))

	var authenticated bool = false

	// Setup the authenticator object
	if *httpAuthFile != "" {
		glog.Infof("Using auth file %s", *httpAuthFile)
		secrets := auth.HtpasswdFileProvider(*httpAuthFile)
		authenticator := auth.NewBasicAuthenticator(*httpAuthRealm, secrets)
		http.HandleFunc(static.StaticResource, authenticator.Wrap(staticHandler))
		if err := pages.RegisterHandlersBasic(containerManager, authenticator); err != nil {
			glog.Fatalf("Failed to register pages auth handlers: %s", err)
		}
		authenticated = true
	}
	if *httpAuthFile == "" && *httpDigestFile != "" {
		glog.Infof("Using digest file %s", *httpDigestFile)
		secrets := auth.HtdigestFileProvider(*httpDigestFile)
		authenticator := auth.NewDigestAuthenticator(*httpDigestRealm, secrets)
		http.HandleFunc(static.StaticResource, authenticator.Wrap(staticHandler))
		if err := pages.RegisterHandlersDigest(containerManager, authenticator); err != nil {
			glog.Fatalf("Failed to register pages digest handlers: %s", err)
		}
		authenticated = true
	}

	// Change handler based on authenticator initalization
	if !authenticated {
		http.HandleFunc(static.StaticResource, staticHandlerNoAuth)
		if err := pages.RegisterHandlersBasic(containerManager, nil); err != nil {
			glog.Fatalf("Failed to register pages handlers: %s", err)
		}
	}

	// Start the manager.
	if err := containerManager.Start(); err != nil {
		glog.Fatalf("Failed to start container manager: %v", err)
	}

	// Install signal handler.
	installSignalHandler(containerManager)

	glog.Infof("Starting cAdvisor version: %q on port %d", info.VERSION, *argPort)

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

func staticHandlerNoAuth(w http.ResponseWriter, r *http.Request) {
	err := static.HandleRequest(w, r.URL)
	if err != nil {
		fmt.Fprintf(w, "%s", err)
	}
}

func staticHandler(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	err := static.HandleRequest(w, r.URL)
	if err != nil {
		fmt.Fprintf(w, "%s", err)
	}
}
