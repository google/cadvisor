// Copyright 2014 Google Inc. All Rights Reserved.
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

package cadvisor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/cadvisor/info"
)

func testGetJsonData(
	strRep string,
	emptyData interface{},
	f func() (interface{}, error),
) error {
	err := json.Unmarshal([]byte(strRep), emptyData)
	if err != nil {
		return fmt.Errorf("invalid json input: %v", err)
	}
	reply, err := f()
	if err != nil {
		return fmt.Errorf("unable to retrieve data: %v", err)
	}
	if !reflect.DeepEqual(reply, emptyData) {
		return fmt.Errorf("retrieved wrong data: %+v != %+v", reply, emptyData)
	}
	return nil
}

func cadvisorTestClient(path, reply string) (*Client, *httptest.Server, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path {
			fmt.Fprint(w, reply)
		} else if r.URL.Path == "/api/v1.0/machine" {
			fmt.Fprint(w, `{"num_cores":8,"memory_capacity":31625871360}`)
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

func TestGetMachineinfo(t *testing.T) {
	respStr := `{"num_cores":8,"memory_capacity":31625871360}`
	client, server, err := cadvisorTestClient("/api/v1.0/machine", respStr)
	if err != nil {
		t.Fatalf("unable to get a client %v", err)
	}
	defer server.Close()
	err = testGetJsonData(respStr, &info.MachineInfo{}, func() (interface{}, error) {
		return client.MachineInfo()
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetContainerInfo(t *testing.T) {
	respStr := `
{
  "name": "%v",
  "subcontainers": [
    "%v/cadvisor",
    "%v/docker"
  ],
  "spec": {
    "cpu": {
      "limit": 1000,
      "max_limit": 18446744073709551000,
      "mask": {
        "data": [
          255
        ]
      }
    },
    "memory": {
      "limit": 9223372036854776000,
      "reservation": 9223372036854776000
    }
  },
  "stats": [
    {
      "timestamp": "2014-06-02T19:04:37.112952602Z",
      "cpu": {
        "usage": {
          "total": 39238585820907,
          "per_cpu": [
            6601830440753,
            5399612248434,
            4705072360704,
            4516301677099,
            4585779804509,
            4556599077540,
            4479432306284,
            4393957905584
          ],
          "user": 14513300000000,
          "system": 14484560000000
        },
        "load": 0
      },
      "memory": {
        "limit": 9223372036854776000,
        "usage": 837984256,
        "container_data": {
          "pgfault": 71601418,
          "pgmajfault": 664
        },
        "hierarchical_data": {
          "pgfault": 100135740,
          "pgmajfault": 1454
        }
      }
    },
    {
      "timestamp": "2014-06-02T19:04:38.117962404Z",
      "cpu": {
        "usage": {
          "total": 39238651066794,
          "per_cpu": [
            6601838963405,
            5399619233761,
            4705084549250,
            4516308300389,
            4585786473026,
            4556607159034,
            4479432477487,
            4393973910442
          ],
          "user": 14513320000000,
          "system": 14484590000000
        },
        "load": 0
      },
      "memory": {
        "limit": 9223372036854776000,
        "usage": 838025216,
        "container_data": {
          "pgfault": 71601418,
          "pgmajfault": 664
        },
        "hierarchical_data": {
          "pgfault": 100140566,
          "pgmajfault": 1454
        }
      }
    },
    {
      "timestamp": "2014-06-02T19:04:39.122826983Z",
      "cpu": {
        "usage": {
          "total": 39238719219625,
          "per_cpu": [
            6601852483847,
            5399630546695,
            4705091917507,
            4516318924052,
            4585791208645,
            4556612305230,
            4479432924370,
            4393988909279
          ],
          "user": 14513340000000,
          "system": 14484630000000
        },
        "load": 0
      },
      "memory": {
        "limit": 9223372036854776000,
        "usage": 838131712,
        "container_data": {
          "pgfault": 71601418,
          "pgmajfault": 664
        },
        "hierarchical_data": {
          "pgfault": 100145700,
          "pgmajfault": 1454
        }
      }
    }
  ]
}
	`
	containerName := "/some/container"
	respStr = fmt.Sprintf(respStr, containerName, containerName, containerName)
	client, server, err := cadvisorTestClient(fmt.Sprintf("/api/v1.0/containers%v", containerName), respStr)
	if err != nil {
		t.Fatalf("unable to get a client %v", err)
	}
	defer server.Close()
	err = testGetJsonData(respStr, &info.ContainerInfo{}, func() (interface{}, error) {
		return client.ContainerInfo(containerName)
	})
	if err != nil {
		t.Fatal(err)
	}
}
