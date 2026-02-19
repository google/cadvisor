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

package collector

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/google/cadvisor/container"
)

func (ec *EndpointConfig) configure(containerHandler container.ContainerHandler) error {
	containerIPAddress := containerHandler.GetContainerIPAddress()

	// If the exact URL was not specified, generate it based on the ip address of the container.
	if ec.URL == "" {
		protocol := strings.ToLower(ec.URLConfig.Protocol)
		if protocol != "http" && protocol != "https" {
			return fmt.Errorf("unsupported endpoint protocol %q", ec.URLConfig.Protocol)
		}
		if containerIPAddress == "" {
			return fmt.Errorf("container ip address is empty")
		}
		ec.URL = protocol + "://" + net.JoinHostPort(containerIPAddress, ec.URLConfig.Port.String()) + ec.URLConfig.Path
		return nil
	}

	parsed, err := url.Parse(ec.URL)
	if err != nil {
		return fmt.Errorf("invalid endpoint url %q: %w", ec.URL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported endpoint url scheme %q", parsed.Scheme)
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("endpoint url must include a host: %q", ec.URL)
	}
	if containerIPAddress == "" {
		return fmt.Errorf("container ip address is empty")
	}

	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		if port := parsed.Port(); port != "" {
			parsed.Host = net.JoinHostPort(containerIPAddress, port)
		} else {
			parsed.Host = containerIPAddress
		}
		ec.URL = parsed.String()
		return nil
	}

	// Only allow explicit container ip address as a destination, otherwise an
	// attacker-controlled config can turn cAdvisor into a cross-network fetcher.
	if host == containerIPAddress {
		return nil
	}
	hostIP := net.ParseIP(host)
	containerIP := net.ParseIP(containerIPAddress)
	if hostIP != nil && containerIP != nil && hostIP.Equal(containerIP) {
		return nil
	}

	return fmt.Errorf("endpoint url host %q is not allowed; only localhost or the container ip address is supported", parsed.Hostname())
}
