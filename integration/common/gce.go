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

package common

import (
	"flag"
	"fmt"
	"os/exec"
	"regexp"

	"google.golang.org/cloud/compute/metadata"
)

var zone = flag.String("zone", "us-central1-f", "Zone the instances are running in")
var project = flag.String("project", "", "Project the instances are running in")

var gceInternalIpRegexp = regexp.MustCompile(`\s+networkIP:\s+([0-9.:]+)\n`)
var gceExternalIpRegexp = regexp.MustCompile(`\s+natIP:\s+([0-9.:]+)\n`)

// Gets the IP of the specified GCE instance.
func GetGceIp(hostname string) (string, error) {
	if hostname == "localhost" {
		return "127.0.0.1", nil
	}

	args := []string{"compute"}
	args = append(args, getProjectFlag()...)
	args = append(args, "instances", "describe")
	args = append(args, getZoneFlag()...)
	args = append(args, hostname)
	out, err := exec.Command("gcloud", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get instance information for %q with error %v and output %s", hostname, err, string(out))
	}

	// Use the internal IP within GCE and the external one outside.
	var matches []string
	if metadata.OnGCE() {
		matches = gceInternalIpRegexp.FindStringSubmatch(string(out))
	} else {
		matches = gceExternalIpRegexp.FindStringSubmatch(string(out))
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("failed to find IP from output %q", string(out))
	}
	return matches[1], nil
}

func getZoneFlag() []string {
	if *zone == "" {
		return []string{}
	}
	return []string{"--zone", *zone}
}

func getProjectFlag() []string {
	if *project == "" {
		return []string{}
	}
	return []string{"--project", *project}
}
func GetGCComputeArgs(cmd string, cmdArgs ...string) []string {
	args := []string{"compute"}
	args = append(args, getProjectFlag()...)
	args = append(args, cmd)
	args = append(args, getZoneFlag()...)
	args = append(args, cmdArgs...)
	return args
}
