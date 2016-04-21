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

	"github.com/coreos/go-semver/semver"
	rktapi "github.com/coreos/rkt/api/v1alpha"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	minimumRktBinVersion     = "1.4.0"
	defaultRktAPIServiceAddr = "localhost:15441"
	timeout                  = 2 * time.Second
)

var (
	rktClient    rktapi.PublicAPIClient
	rktClientErr error
	once         sync.Once
)

type rktVersion struct {
	*semver.Version
}

func (r rktVersion) Compare(other string) (int, error) {
	v, err := semver.NewVersion(other)
	if err != nil {
		return -1, err
	}

	if r.LessThan(*v) {
		return -1, nil
	}
	if v.LessThan(*r.Version) {
		return 1, nil
	}
	return 0, nil
}

func newRktVersion(version string) (rktVersion, error) {
	sem, err := semver.NewVersion(version)
	if err != nil {
		return rktVersion{}, err
	}
	return rktVersion{sem}, nil
}

func Client() (rktapi.PublicAPIClient, error) {
	once.Do(func() {
		//gRPC doesn't fail fast if service isn't available so test with raw tcp connecction first
		conn, err := net.DialTimeout("tcp", defaultRktAPIServiceAddr, timeout)
		if err != nil {
			rktClient = nil
			rktClientErr = fmt.Errorf("rkt: cannot tcp Dial rkt api service: %v", err)
			return
		}
		conn.Close()

		rktClient = nil
		apisvcConn, err := grpc.Dial(defaultRktAPIServiceAddr, grpc.WithInsecure(), grpc.WithTimeout(timeout))
		if err != nil {
			rktClientErr = fmt.Errorf("rkt: cannot grpc Dial rkt api service: %v", err)
			return
		}

		apisvc := rktapi.NewPublicAPIClient(apisvcConn)

		resp, err := apisvc.GetInfo(context.Background(), &rktapi.GetInfoRequest{})
		if err != nil {
			rktClientErr = fmt.Errorf("rkt: GetInfo() failed: %v", err)
			return
		}

		binVersion, err := newRktVersion(resp.Info.RktVersion)
		if err != nil {
			rktClientErr = fmt.Errorf("rkt: couldn't parse RtVersion: %v", err)
			return
		}
		result, err := binVersion.Compare(minimumRktBinVersion)
		if err != nil {
			rktClientErr = fmt.Errorf("rkt: couldn't compare RktVersion against minimum: %v", err)
			return
		}
		if result < 0 {
			rktClientErr = fmt.Errorf("rkt: binary version is too old(%v), requires at least %v", resp.Info.RktVersion, minimumRktBinVersion)
			return
		}

		//        	result, err = binVersion.Compare(recommendedRktBinVersion)
		//	        if err != nil {
		//        	        return err
		//	        }
		//        	if result != 0 {
		//                	glog.Warningf("rkt: current binary version %q is not recommended (recommended version %q)", resp.Info.RktVersion, recommendedRktBinVersion)
		//	        }

		rktClient = apisvc
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
