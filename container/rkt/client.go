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

package rkt

import (
	"fmt"
	"net"
	"sync"
	"time"

	rktapi "github.com/coreos/rkt/api/v1alpha"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	defaultRktAPIServiceAddr = "localhost:15441"
)

var (
	rktClient    rktapi.PublicAPIClient
	rktClientErr error
	once         sync.Once
)

func Client() (rktapi.PublicAPIClient, error) {
	once.Do(func() {
		_, err := net.Dial("tcp", "localhost:15441")
		if err != nil {
			rktClient = nil
			rktClientErr = fmt.Errorf("rkt: cannot connect to rkt api service: %v", err)
			return
		}

		apisvcConn, err := grpc.Dial(defaultRktAPIServiceAddr, grpc.WithInsecure(), grpc.WithTimeout(5*time.Second))
		if err != nil {
			rktClient = nil
			rktClientErr = fmt.Errorf("rkt: cannot connect to rkt api service: %v", err)
			return
		}

		rktClient = rktapi.NewPublicAPIClient(apisvcConn)
	})

	return rktClient, rktClientErr
}

func RktPath() (string, error) {
	client, err := Client()
	if err != nil {
		return "", err
	}

	resp, err := client.GetInfo(context.Background(), &rktapi.GetInfoRequest{})
	if err != nil {
		return "", fmt.Errorf("couldn't GetInfo from rkt api service: %v", err)
	}

	return resp.Info.GlobalFlags.Dir, nil
}
