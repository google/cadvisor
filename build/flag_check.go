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

package main

import (
	"flag"
	"fmt"
	"os"

	// Import everything to verify that flags are not added to command line.

	_ "github.com/google/cadvisor/cache/memory"
	_ "github.com/google/cadvisor/http"
	_ "github.com/google/cadvisor/manager"
	_ "github.com/google/cadvisor/storage"
	_ "github.com/google/cadvisor/storage/bigquery"
	_ "github.com/google/cadvisor/storage/elasticsearch"
	_ "github.com/google/cadvisor/storage/influxdb"
	_ "github.com/google/cadvisor/storage/kafka"
	_ "github.com/google/cadvisor/storage/redis"
	_ "github.com/google/cadvisor/storage/statsd"
	_ "github.com/google/cadvisor/storage/stdout"
	_ "github.com/google/cadvisor/utils/sysfs"
	_ "github.com/google/cadvisor/version"
)

// Flags which are accepted from other packages.
var allowedFlags = []string{
	// Flags added from the glog package
	"logtostderr",
	"alsologtostderr",
	"v",
	"stderrthreshold",
	"vmodule",
	"log_backtrace_at",
	"log_dir",
}

func main() {
	expected := map[string]struct{}{}
	for _, f := range allowedFlags {
		expected[f] = struct{}{}
	}

	hasLeak := false
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if _, ok := expected[f.Name]; !ok {
			fmt.Fprintf(os.Stderr, "Leaking flag %q: %q\n", f.Name, f.Usage)
			hasLeak = true
		}
	})

	if hasLeak {
		os.Exit(1)
	}
}
