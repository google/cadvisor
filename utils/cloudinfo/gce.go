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

package cloudinfo

import (
	"io"
	"net/http"
	"strings"

	info "github.com/google/cadvisor/info/v1"
)

func inGCE() bool {
	_, err := http.Get("http://metadata.google.internal/computeMetadata/v1/instance/machine-type")
	return err == nil
}

func getGceInstanceType() info.InstanceType {
	// Query the metadata server.
	req, err := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/machine-type", nil)
	if err != nil {
		return info.UNKNOWN_INSTANCE
	}
	req.Header.Set("Metadata-Flavor", "Google")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return info.UNKNOWN_INSTANCE
	}
	body := make([]byte, 1000)
	numRead, err := resp.Body.Read(body)
	if err != io.EOF {
		return info.UNKNOWN_INSTANCE
	}

	// Extract the instance name from the response.
	responseString := string(body[:numRead])
	responseParts := strings.Split(responseString, "/")
	return info.InstanceType(responseParts[len(responseParts)-1])
}
