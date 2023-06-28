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

package healthz

import (
	"io"
	"net/http"
	"testing"

	"github.com/google/cadvisor/integration/framework"
)

func TestHealthzOk(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	// Ensure that /heathz returns "ok"
	resp, err := http.Get(fm.Hostname().FullHostname() + "healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "ok" {
		t.Fatalf("cAdvisor returned unexpected healthz status of %q", body)
	}
}
