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

package collector

import (
	"github.com/google/cadvisor/info/v2"
	"time"
)

// TODO(vmarmol): Export to a custom metrics type when that is available.

// Metric collector.
type CollectorInterface interface {
	// Collect metrics from this collector.
	// Returns the next time this collector should be collected from.
	// Next collection time is always returned, even when an error occurs.
	// A collection time of zero means no more collection.
	Collect() (time.Time, []v2.Metric, error)

	// Name of this collector.
	Name() string
}

// Manages and runs collectors.
type CollectorManager interface {
	// Register a collector.
	RegisterCollector(collector Collector) error

	// Collect from collectors that are ready and return the next time
	// at which a collector will be ready to collect from.
	// Next collection time is always returned, even when an error occurs.
	// A collection time of zero means no more collection.
	Collect() (time.Time, []v2.Metric, error)
}
