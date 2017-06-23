// Copyright 2016 Google Inc. All Rights Reserved.
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

package server

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage"
)

type StoragePlugin struct {
	driver storage.StorageDriver
}

type server struct {
	*StoragePlugin

	conn io.Closer
}

var (
	// The default socket to listen on.
	DefaultSocket = "/var/run/cadvisor-storage.sock"

	// Signals to listen on to shut down the server.
	TerminationSignals = []os.Signal{syscall.SIGINT, syscall.SIGHUP}

	// Whether to silence log messages.
	DisableLogging = false
)

// Add flags used by the plugin.
func AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&DefaultSocket, "socket", DefaultSocket, "The unix domain socket to listen on.")
}

// Run starts the server using the given driver and default socket. It waits for a termination
// signal and then cleans up the connection. A simple plugin implementation only needs to call this
// method.
func Run(driver storage.StorageDriver) {
	s, err := Start(DefaultSocket, driver)
	if err != nil {
		logf("Error: failed to start server: %v", err)
		return
	}
	defer s.Close()

	logf("Started server listening on %s", DefaultSocket)

	WaitForTermination()
}

// Start exposes finer controls over running the plugin server. It starts the server with the given
// driver and socket. It is the caller's responsibility to run as long as needed
// (e.g. WaitForTermination) and close the server when done.
func Start(socket string, driver storage.StorageDriver) (io.Closer, error) {
	plugin := &StoragePlugin{driver}
	rpc.Register(plugin)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("could not listen on %s: %v", socket, err)
	}
	go rpc.Accept(listener)

	return &server{plugin, listener}, nil
}

// WaitForTermination blocks until a termination signal is received.
func WaitForTermination() {
	stop := make(chan os.Signal)
	signal.Notify(stop, TerminationSignals...)
	<-stop
}

// AddStats implements the main RPC call that receives stats pushes from cAdvisor.
func (s *StoragePlugin) AddStats(info *v1.ContainerInfo, _ *bool) error {
	if info == nil {
		return errors.New("nil request received")
	}
	if len(info.Stats) == 0 {
		return fmt.Errorf("no stats in request %+v", *info)
	}

	for _, stats := range info.Stats {
		err := s.driver.AddStats(info.ContainerReference, stats)
		if err != nil {
			return fmt.Errorf("error adding stats: %v", err)
		}
	}
	return nil
}

// Version reports the version of the plugin API implemented by this server.
func (s *StoragePlugin) Version(_ bool, version *string) error {
	*version = "1.0.0"
	return nil
}

// Close the server and cleanup its resources.
func (s *server) Close() error {
	err := s.driver.Close()
	cerr := s.conn.Close()
	if cerr != nil {
		if err != nil {
			err = fmt.Errorf("%v; %v", cerr, err)
		} else {
			err = cerr
		}
	}
	return err
}

func logf(format string, v ...interface{}) {
	if !DisableLogging {
		log.Printf(format, v...)
	}
}
