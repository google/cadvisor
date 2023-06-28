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
	"context"
	"fmt"
	"net"
	"net/http"
	urllib "net/url"
)

type clientKey struct{}

func (c clientKey) String() string {
	return "client"
}

type Connection struct {
	URI    *urllib.URL
	Client *http.Client
}

func client(ctx *context.Context) (*Connection, error) {
	url, err := urllib.Parse(*endpointFlag)
	if err != nil {
		return nil, err
	}

	switch url.Scheme {
	case "unix":
		connection := Connection{URI: url}
		connection.Client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", url.Path)
				},
				DisableCompression: true,
			},
		}
		*ctx = context.WithValue(*ctx, clientKey{}, &connection)
		return &connection, nil
	}

	return nil, fmt.Errorf("couldn't get podman client")
}
