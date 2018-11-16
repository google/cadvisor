// Copyright 2017 Google Inc. All Rights Reserved.
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

package crio

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

const (
	CrioSocket            = "/var/run/crio/crio.sock"
	maxUnixSocketPathSize = len(syscall.RawSockaddrUnix{}.Path)
)

// Info represents CRI-O information as sent by the CRI-O server
type Info struct {
	StorageDriver string `json:"storage_driver"`
	StorageRoot   string `json:"storage_root"`
}

// ContainerInfo represents a given container information
type ContainerInfo struct {
	Name        string            `json:"name"`
	Pid         int               `json:"pid"`
	Image       string            `json:"image"`
	CreatedTime int64             `json:"created_time"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	LogPath     string            `json:"log_path"`
	Root        string            `json:"root"`
	IP          string            `json:"ip_address"`
}

type crioClient interface {
	Info() (Info, error)
	ContainerInfo(string) (*ContainerInfo, error)
}

type crioClientImpl struct {
	client *http.Client
}

func configureUnixTransport(tr *http.Transport, proto, addr string) error {
	if len(addr) > maxUnixSocketPathSize {
		return fmt.Errorf("Unix socket path %q is too long", addr)
	}
	// No need for compression in local communications.
	tr.DisableCompression = true
	tr.Dial = func(_, _ string) (net.Conn, error) {
		return net.DialTimeout(proto, addr, 32*time.Second)
	}
	return nil
}

// Client returns a new configured CRI-O client
func Client() (crioClient, error) {
	tr := new(http.Transport)
	configureUnixTransport(tr, "unix", CrioSocket)
	c := &http.Client{
		Transport: tr,
	}
	return &crioClientImpl{
		client: c,
	}, nil
}

func getRequest(path string) (*http.Request, error) {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	// For local communications over a unix socket, it doesn't matter what
	// the host is. We just need a valid and meaningful host name.
	req.Host = "crio"
	req.URL.Host = CrioSocket
	req.URL.Scheme = "http"
	return req, nil
}

// Info returns generic info from the CRI-O server
func (c *crioClientImpl) Info() (Info, error) {
	info := Info{}
	req, err := getRequest("/info")
	if err != nil {
		return info, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return info, err
	}
	return info, nil
}

// ContainerInfo returns information about a given container
func (c *crioClientImpl) ContainerInfo(id string) (*ContainerInfo, error) {
	req, err := getRequest("/containers/" + id)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	cInfo := ContainerInfo{}
	if err := json.NewDecoder(resp.Body).Decode(&cInfo); err != nil {
		return nil, err
	}
	return &cInfo, nil
}
