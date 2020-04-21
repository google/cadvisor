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

package client

import (
	"fmt"
	"net"

	"k8s.io/klog/v2"
)

type Client struct {
	HostPort  string
	Namespace string
	conn      net.Conn
}

func (c *Client) Open() error {
	conn, err := net.Dial("udp", c.HostPort)
	if err != nil {
		klog.Errorf("failed to open udp connection to %q: %v", c.HostPort, err)
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) Close() error {
	c.conn.Close()
	c.conn = nil
	return nil
}

// Simple send to statsd daemon without sampling.
func (c *Client) Send(namespace, containerName, key string, value uint64) error {
	// only send counter value
	formatted := fmt.Sprintf("%s.%s.%s:%d|g", namespace, containerName, key, value)
	_, err := fmt.Fprintf(c.conn, formatted)
	if err != nil {
		return fmt.Errorf("failed to send data %q: %v", formatted, err)
	}
	return nil
}

func New(hostPort string) (*Client, error) {
	Client := Client{HostPort: hostPort}
	if err := Client.Open(); err != nil {
		return nil, err
	}
	return &Client, nil
}
