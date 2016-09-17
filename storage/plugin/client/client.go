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

package client

import (
	"flag"
	"fmt"
	"net/rpc"

	"github.com/golang/glog"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage"
)

func init() {
	storage.RegisterStorageDriver("plugin", NewClient)
}

var socket = flag.String("storage_driver_socket", "/var/run/cadvisor-storage.sock", "Unix domain socket to connect to for storage driver plugin.")

type pluginClient struct {
	client  *rpc.Client
	version string
}

func NewClient() (storage.StorageDriver, error) {
	// Connect to server.
	client, err := rpc.Dial("unix", *socket)
	if err != nil {
		return nil, fmt.Errorf("could not connect to server: %v", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			client.Close()
		}
	}()

	// Check the plugin version.
	var version string
	if err := client.Call("StoragePlugin.Version", false, &version); err != nil {
		return nil, fmt.Errorf("failed to get plugin version: %v", err)
	}
	// Currently only version 1.0.0 is supported.
	if version != "1.0.0" {
		return nil, fmt.Errorf("incompatible plugin version: %s", version)
	}

	cleanup = false
	return &pluginClient{client, version}, nil
}

func (c *pluginClient) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}

	req := info.ContainerInfo{
		ContainerReference: ref,
		Stats:              []*info.ContainerStats{stats},
	}

	err := c.client.Call("StoragePlugin.AddStats", req, nil)
	if err == rpc.ErrShutdown {
		// TODO: Figure out a better way of handling this permanent error.
		glog.Exitf("Storage plugin connection terminated.")
	}

	return err
}

func (c *pluginClient) Close() error {
	return c.client.Close()
}
