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
	"io/ioutil"
	"path/filepath"
	"strings"

	rktapi "github.com/coreos/rkt/api/v1alpha"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

type parsedName struct {
	Pod       string
	Container string
}

func verifyPod(name string) (bool, bool, error) {
	splits := strings.Split(name, "/")
	_, err := convertCgroup(name)

	return err == nil, len(splits) != 4, err
}

func convertCgroup(name string) (*parsedName, error) {
	glog.V(4).Infof("convertCgroup: name = %v", name)

	uuid, err := getRktUUID(name)
	if err != nil {
		glog.V(4).Infof("convertCgroup: getRktUUID failed: %v", err)
		return nil, fmt.Errorf("%s not handled by rkt handler", name)
	}

	splits := strings.Split(name, "/")
	if len(splits) == 3 || len(splits) == 5 {
		parsed := &parsedName{}
		parsed.Pod = uuid

		if len(splits) == 3 {
			return parsed, nil
		} else {
			parsed.Container = strings.Replace(splits[4], ".service", "", -1)
			return parsed, nil
		}
	}

	glog.V(4).Infof("convertCgroup: why did we get here: len(splits) = %v, splits = %v", len(splits), splits)

	return nil, fmt.Errorf("%s not handled by rkt handler", name)
}

func getRktUUID(name string) (string, error) {
	glog.V(4).Infof("getRktUUID: name = %v", name)
	splits := strings.Split(name, "/")

	test_path := name

	if len(splits) >= 4 && splits[3] == "system.slice" {
		//cgroup path correspond to the first 3 elements (really 2, but first is blank)
		test_path = strings.Join(splits[:3], "/")
	} else if len(splits) == 3 {
		test_path = name
	} else {
		return "", fmt.Errorf("%v not supported by rkt handler path", name)
	}

	rktClient, err := Client()
	if err != nil {
		return "", fmt.Errorf("couldn't get rkt api service: %v", err)
	}

	resp, err := rktClient.ListPods(context.Background(), &rktapi.ListPodsRequest{
		Filters: []*rktapi.PodFilter{
			{
				Cgroups: []string{test_path},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("%s not handled by rkt handler", name)
	}

	for _, pod := range resp.Pods {
		if pod.State == rktapi.PodState_POD_STATE_RUNNING {
			return pod.Id, nil
		}
	}

	return "", fmt.Errorf("%s didn't match any running pods out of %v pods", name, len(resp.Pods))
}

// Gets a Rkt container's overlay upper dir
func getRootFs(root string, parsed *parsedName) string {
	/* Example of where it stores the upper dir key
	for container
		/var/lib/rkt/pods/run/bc793ec6-c48f-4480-99b5-6bec16d52210/appsinfo/alpine-sh/treeStoreID
	for pod
		/var/lib/rkt/pods/run/f556b64a-17a7-47d7-93ec-ef2275c3d67e/stage1TreeStoreID

	*/

	var tree string
	if parsed.Container == "" {
		tree = filepath.Join(root, "pods/run", parsed.Pod, "stage1TreeStoreID")
	} else {
		tree = filepath.Join(root, "pods/run", parsed.Pod, "appsinfo", parsed.Container, "treeStoreID")
	}

	bytes, err := ioutil.ReadFile(tree)
	if err != nil {
		glog.Infof("ReadFile failed, couldn't read %v to get upper dir: %v", tree, err)
		return ""
	}

	s := string(bytes)

	/* Example of where the upper dir is stored via key read above
	   /var/lib/rkt/pods/run/bc793ec6-c48f-4480-99b5-6bec16d52210/overlay/deps-sha512-82a099e560a596662b15dec835e9adabab539cad1f41776a30195a01a8f2f22b/
	*/
	return filepath.Join(root, "pods/run", parsed.Pod, "overlay", s)
}
