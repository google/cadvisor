// Copyright 2021 Google Inc. All Rights Reserved.
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

package podman

import (
	"net/http"
	"sync"

	dclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
)

var (
	podmanClient     *dclient.Client
	podmanClientErr  error
	podmanClientOnce sync.Once
)

// Client creates a Docker API client based on the given Podman flags
//
// At this time, we're using the podmans docker compatibility API layer
// for podman containers.
func Client() (*dclient.Client, error) {
	podmanClientOnce.Do(func() {
		var client *http.Client
		if *ArgPodmanTLS {
			client = &http.Client{}
			options := tlsconfig.Options{
				CAFile:             *ArgPodmanCA,
				CertFile:           *ArgPodmanCert,
				KeyFile:            *ArgPodmanKey,
				InsecureSkipVerify: false,
			}
			tlsc, err := tlsconfig.Client(options)
			if err != nil {
				podmanClientErr = err
				return
			}
			client.Transport = &http.Transport{
				TLSClientConfig: tlsc,
			}
		}
		podmanClient, podmanClientErr = dclient.NewClientWithOpts(
			dclient.WithHost(*ArgPodmanEndpoint),
			dclient.WithHTTPClient(client),
			dclient.WithAPIVersionNegotiation())
	})
	return podmanClient, podmanClientErr
}
