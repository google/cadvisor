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

package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	info "github.com/google/cadvisor/info/v1"
	infoV2 "github.com/google/cadvisor/info/v2"
	"github.com/kr/pretty"
)

func testGetJsonData(
	expected interface{},
	f func() (interface{}, error),
) error {
	reply, err := f()
	if err != nil {
		return fmt.Errorf("unable to retrieve data: %v", err)
	}
	if !reflect.DeepEqual(reply, expected) {
		return pretty.Errorf("retrieved wrong data: %# v != %# v", reply, expected)
	}
	return nil
}

func cadvisorTestClient(path string, expectedPostObj *info.ContainerInfoRequest, replyObj interface{}, t *testing.T) (*Client, *httptest.Server, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			if expectedPostObj != nil {
				expectedPostObjEmpty := new(info.ContainerInfoRequest)
				decoder := json.NewDecoder(r.Body)
				if err := decoder.Decode(expectedPostObjEmpty); err != nil {
					t.Errorf("Received invalid object: %v", err)
				}
				if expectedPostObj.NumStats != expectedPostObjEmpty.NumStats ||
					expectedPostObj.Start.Unix() != expectedPostObjEmpty.Start.Unix() ||
					expectedPostObj.End.Unix() != expectedPostObjEmpty.End.Unix() {
					t.Errorf("Received unexpected object: %+v, expected: %+v", expectedPostObjEmpty, expectedPostObj)
				}
			}
			encoder := json.NewEncoder(w)
			encoder.Encode(replyObj)
		} else if r.URL.Path == "/api/v2.0/version" {
			fmt.Fprintf(w, "0.1.2")
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Page not found.")
		}
	}))
	client, err := NewClient(ts.URL)
	if err != nil {
		ts.Close()
		return nil, nil, err
	}
	return client, ts, err
}

// TestGetMachineInfo performs one test to check if MachineInfo()
// in a cAdvisor client returns the correct result.
func TestGetMachineinfo(t *testing.T) {
	minfo := &info.MachineInfo{
		NumCores:       8,
		MemoryCapacity: 31625871360,
		DiskMap: map[string]info.DiskInfo{
			"8:0": {
				Name:  "sda",
				Major: 8,
				Minor: 0,
				Size:  10737418240,
			},
		},
	}
	client, server, err := cadvisorTestClient("/api/v2.0/machine", nil, minfo, t)
	if err != nil {
		t.Fatalf("unable to get a client %v", err)
	}
	defer server.Close()
	returned, err := client.MachineInfo()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(returned, minfo) {
		t.Fatalf("received unexpected machine info")
	}
}

// TestGetVersionInfo performs one test to check if VersionInfo()
// in a cAdvisor client returns the correct result.
func TestGetVersioninfo(t *testing.T) {
	version := "0.1.2"
	client, server, err := cadvisorTestClient("", nil, version, t)
	if err != nil {
		t.Fatalf("unable to get a client %v", err)
	}
	defer server.Close()
	returned, err := client.VersionInfo()
	if err != nil {
		t.Fatal(err)
	}
	if returned != version {
		t.Fatalf("received unexpected version info")
	}
}

// TestAttributes performs one test to check if Attributes()
// in a cAdvisor client returns the correct result.
func TestGetAttributes(t *testing.T) {
	attr := &infoV2.Attributes{
		KernelVersion:      "3.3.0",
		ContainerOsVersion: "Ubuntu 14.4",
		DockerVersion:      "Docker 1.5",
		CadvisorVersion:    "0.1.2",
		NumCores:           8,
		MemoryCapacity:     31625871360,
		DiskMap: map[string]info.DiskInfo{
			"8:0": {
				Name:  "sda",
				Major: 8,
				Minor: 0,
				Size:  10737418240,
			},
		},
	}
	client, server, err := cadvisorTestClient("/api/v2.0/attributes", nil, attr, t)
	if err != nil {
		t.Fatalf("unable to get a client %v", err)
	}
	defer server.Close()
	returned, err := client.Attributes()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(returned, attr) {
		t.Fatalf("received unexpected attributes")
	}
}

func TestRequestFails(t *testing.T) {
	errorText := "there was an error"
	// Setup a server that simply fails.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, errorText, 500)
	}))
	client, err := NewClient(ts.URL)
	if err != nil {
		ts.Close()
		t.Fatal(err)
	}
	defer ts.Close()

	_, err = client.MachineInfo()
	if err == nil {
		t.Fatalf("Expected non-nil error")
	}
	expectedError := fmt.Sprintf("request failed with error: %q", errorText)
	if strings.Contains(err.Error(), expectedError) {
		t.Fatalf("Expected error %q but received %q", expectedError, err)
	}
}
