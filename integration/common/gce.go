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

	"github.com/GoogleCloudPlatform/gcloud-golang/compute/metadata"
)

var zone = flag.String("zone", "us-central1-f", "Zone the instances are running in")

var gceInternalIpRegexp = regexp.MustCompile(" +networkIP: +([0-9.:]+)\n")
var gceExternalIpRegexp = regexp.MustCompile(" +natIP: +([0-9.:]+)\n")

// Gets the IP of the specified GCE instance.
func GetGceIp(hostname string) (string, error) {
	if hostname == "localhost" {
		return "127.0.0.1", nil
	}

	out, err := exec.Command("gcloud", "compute", "instances", "describe", GetZoneFlag(), hostname).CombinedOutput()
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

func GetZoneFlag() string {
	return fmt.Sprintf("--zone=%s", *zone)
}
