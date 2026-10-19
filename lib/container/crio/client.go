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
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"
)

// ErrContainerNotFound indicates that CRI-O does not yet know about the
// requested container. This is expected during a short window after the
// container cgroup is created but before CRI-O finishes registration.
var ErrContainerNotFound = errors.New("container not found")

var crioClientTimeout = flag.Duration("crio_client_timeout", time.Duration(0), "CRI-O client timeout. Default is no timeout.")

const (
	CrioSocket            = "/var/run/crio/crio.sock"
	maxUnixSocketPathSize = len(syscall.RawSockaddrUnix{}.Path)
)

var (
	theClient      CrioClient
	clientErr      error
	crioClientOnce sync.Once
)

// Info represents CRI-O information as sent by the CRI-O server
type Info struct {
	StorageDriver string `json:"storage_driver"`
	StorageRoot   string `json:"storage_root"`
	StorageImage  string `json:"storage_image"`
	CgroupDriver  string `json:"cgroup_driver"`
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
	IPs         []string          `json:"ip_addresses"`
}

type CrioClient interface {
	Info() (Info, error)
	ContainerInfo(string) (*ContainerInfo, error)
}

type crioClientImpl struct {
	client *http.Client
}

// crioRegistrationTimeout bounds the CRI-O calls made while registering the
// factory and building container handlers. Those run during housekeeping, so
// they need a deadline even when --crio_client_timeout is 0. It is a variable
// so tests can lower it.
var crioRegistrationTimeout = 10 * time.Second

// contextClient is implemented by crioClientImpl. CrioClient has no context
// parameters, and other implementations (such as test mocks) fall back to its
// methods.
type contextClient interface {
	infoContext(ctx context.Context) (Info, error)
	containerInfoContext(ctx context.Context, id string) (*ContainerInfo, error)
}

// boundedInfo calls Info with a crioRegistrationTimeout deadline.
func boundedInfo(c CrioClient) (Info, error) {
	cc, ok := c.(contextClient)
	if !ok {
		return c.Info()
	}
	ctx, cancel := context.WithTimeout(context.Background(), crioRegistrationTimeout)
	defer cancel()
	return cc.infoContext(ctx)
}

// boundedContainerInfo calls ContainerInfo with a crioRegistrationTimeout
// deadline.
func boundedContainerInfo(c CrioClient, id string) (*ContainerInfo, error) {
	cc, ok := c.(contextClient)
	if !ok {
		return c.ContainerInfo(id)
	}
	ctx, cancel := context.WithTimeout(context.Background(), crioRegistrationTimeout)
	defer cancel()
	return cc.containerInfoContext(ctx, id)
}

func configureUnixTransport(tr *http.Transport, proto, addr string) error {
	if len(addr) > maxUnixSocketPathSize {
		return fmt.Errorf("unix socket path %q is too long", addr)
	}
	// No need for compression in local communications.
	tr.DisableCompression = true
	tr.DialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
		return net.DialTimeout(proto, addr, 32*time.Second)
	}
	return nil
}

// Client returns a new configured CRI-O client
func Client() (CrioClient, error) {
	crioClientOnce.Do(func() {
		tr := new(http.Transport)
		theClient = nil
		if clientErr = configureUnixTransport(tr, "unix", CrioSocket); clientErr != nil {
			return
		}
		theClient = &crioClientImpl{
			client: &http.Client{
				Transport: tr,
				Timeout:   *crioClientTimeout,
			},
		}
	})
	return theClient, clientErr
}

func getRequest(ctx context.Context, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
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
	return c.infoContext(context.Background())
}

func (c *crioClientImpl) infoContext(ctx context.Context) (Info, error) {
	info := Info{}
	req, err := getRequest(ctx, "/info")
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
	return c.containerInfoContext(context.Background(), id)
}

func (c *crioClientImpl) containerInfoContext(ctx context.Context, id string) (*ContainerInfo, error) {
	req, err := getRequest(ctx, "/containers/"+id)
	if err != nil {
		return nil, err
	}
	cInfo := ContainerInfo{}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// golang's http.Do doesn't return an error if non 200 response code is returned
	// handle this case here, rather than failing to decode the body
	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error finding container %s: status %d", id, resp.StatusCode)
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("error finding container %s: %s: %w", id, string(respBody), ErrContainerNotFound)
		}
		return nil, fmt.Errorf("error finding container %s: status %d returned error %s", id, resp.StatusCode, string(respBody))
	}

	if err := json.NewDecoder(resp.Body).Decode(&cInfo); err != nil {
		return nil, err
	}
	if len(cInfo.IP) > 0 {
		return &cInfo, nil
	}
	if len(cInfo.IPs) > 0 {
		cInfo.IP = cInfo.IPs[0]
	}
	return &cInfo, nil
}
