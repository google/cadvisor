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

// Handler for /validate content.
// Validates cadvisor dependencies - kernel, os, docker setup.

package docker

import (
	"net/http"

	dclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
)

// Client creates a Docker API client based on the given Docker flags
func (opts *Options) Client() (*dclient.Client, error) {
	opts.dockerClientOnce.Do(func() {
		var client *http.Client
		if opts.DockerTLS {
			client = &http.Client{}
			options := tlsconfig.Options{
				CAFile:             opts.DockerCA,
				CertFile:           opts.DockerCert,
				KeyFile:            opts.DockerKey,
				InsecureSkipVerify: false,
			}
			tlsc, err := tlsconfig.Client(options)
			if err != nil {
				opts.dockerClientErr = err
				return
			}
			client.Transport = &http.Transport{
				TLSClientConfig: tlsc,
			}
		}
		opts.dockerClient, opts.dockerClientErr = dclient.NewClientWithOpts(
			dclient.WithHost(opts.DockerEndpoint),
			dclient.WithHTTPClient(client),
			dclient.WithAPIVersionNegotiation())
	})
	return opts.dockerClient, opts.dockerClientErr
}
