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
	"path"
	"strings"

	"github.com/golang/glog"
)

type parsedName struct {
	Pod       string
	Container string
}

func verifyName(name string) (bool, error) {
	_, err := parseName(name)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Parse cgroup name into a pod/container name struct
func parseName(name string) (*parsedName, error) {
	splits := strings.Split(name, "/")
	if len(splits) == 3 || len(splits) == 5 {
		parsed := &parsedName{}

		if splits[1] == "machine.slice" {
			replacer := strings.NewReplacer("machine-rkt\\x2d", "", ".scope", "", "\\x2d", "-")
			parsed.Pod = replacer.Replace(splits[2])
			if len(splits) == 3 {
				return parsed, nil
			}
			if splits[3] == "system.slice" {
				parsed.Container = strings.Replace(splits[4], ".service", "", -1)
				return parsed, nil
			}
		}
	}

	return nil, fmt.Errorf("%s not handled by rkt handler", name)
}

// Gets a Rkt container's overlay upper dir
func GetRootFs(root string, parsed *parsedName) string {
	/* Example of where it stores the upper dir key
	/var/lib/rkt/pods/run/bc793ec6-c48f-4480-99b5-6bec16d52210/appsinfo/alpine-sh/treeStoreID
	*/
	if parsed.Container == "" {
		return ""
	}

	tree := path.Join(root, "pods/run", parsed.Pod, "appsinfo", parsed.Container, "treeStoreID")
	glog.Infof("tree = %q", tree)
	bytes, err := ioutil.ReadFile(tree)
	if err != nil {
		glog.Infof("ReadFile failed: %v", err)
		return ""
	}

	s := string(bytes)

	/* Example of where the upper dir is stored via key read above
	   /var/lib/rkt/pods/run/bc793ec6-c48f-4480-99b5-6bec16d52210/overlay/deps-sha512-82a099e560a596662b15dec835e9adabab539cad1f41776a30195a01a8f2f22b/
	*/
	return path.Join(root, "pods/run", parsed.Pod, "overlay", s)
}
