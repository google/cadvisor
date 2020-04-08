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

package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/google/cadvisor/cache/memory"
	_ "github.com/google/cadvisor/cmd/internal/storage/bigquery"
	_ "github.com/google/cadvisor/cmd/internal/storage/elasticsearch"
	_ "github.com/google/cadvisor/cmd/internal/storage/influxdb"
	_ "github.com/google/cadvisor/cmd/internal/storage/kafka"
	_ "github.com/google/cadvisor/cmd/internal/storage/redis"
	_ "github.com/google/cadvisor/cmd/internal/storage/statsd"
	_ "github.com/google/cadvisor/cmd/internal/storage/stdout"
	"github.com/google/cadvisor/storage"

	"k8s.io/klog/v2"
)

var (
	storageDriver   = flag.String("storage_driver", "", fmt.Sprintf("Storage `driver` to use. Data is always cached shortly in memory, this controls where data is pushed besides the local cache. Empty means none, multiple separated by commas. Options are: <empty>, %s", strings.Join(storage.ListDrivers(), ", ")))
	storageDuration = flag.Duration("storage_duration", 2*time.Minute, "How long to keep data stored (Default: 2min).")
)

// NewMemoryStorage creates a memory storage with an optional backend storage option.
func NewMemoryStorage() (*memory.InMemoryCache, error) {
	backendStorages := []storage.StorageDriver{}
	for _, driver := range strings.Split(*storageDriver, ",") {
		if driver == "" {
			continue
		}
		storage, err := storage.New(driver)
		if err != nil {
			return nil, err
		}
		backendStorages = append(backendStorages, storage)
		klog.V(1).Infof("Using backend storage type %q", driver)
	}
	klog.V(1).Infof("Caching stats in memory for %v", *storageDuration)
	return memory.New(*storageDuration, backendStorages), nil
}
