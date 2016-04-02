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

package container

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
)

type ContainerHandlerFactory interface {
	// Create a new ContainerHandler using this factory. CanHandleAndAccept() must have returned true.
	NewContainerHandler(name string, inHostNamespace bool) (c ContainerHandler, err error)

	// Name of the factory.
	String() string

	// Returns debugging information. Map of lines per category.
	DebugInfo() map[string][]string
}

// MetricKind represents the kind of metrics that cAdvisor exposes.
type MetricKind string

const (
	CpuUsageMetrics        MetricKind = "cpu"
	MemoryUsageMetrics     MetricKind = "memory"
	CpuLoadMetrics         MetricKind = "cpuLoad"
	DiskIOMetrics          MetricKind = "diskIO"
	DiskUsageMetrics       MetricKind = "disk"
	NetworkUsageMetrics    MetricKind = "network"
	NetworkTcpUsageMetrics MetricKind = "tcp"
	AppMetrics             MetricKind = "app"
)

func (mk MetricKind) String() string {
	return string(mk)
}

type MetricSet map[MetricKind]struct{}

func (ms MetricSet) Has(mk MetricKind) bool {
	_, exists := ms[mk]
	return exists
}

func (ms MetricSet) Add(mk MetricKind) {
	ms[mk] = struct{}{}
}

// TODO(vmarmol): Consider not making this global.
// Global list of factories.
var (
	factories     []ContainerHandlerFactory
	factoriesLock sync.RWMutex
)

// Register a ContainerHandlerFactory. These should be registered from least general to most general
// as they will be asked in order whether they can handle a particular container.
func RegisterContainerHandlerFactory(factory ContainerHandlerFactory) {
	factoriesLock.Lock()
	defer factoriesLock.Unlock()

	factories = append(factories, factory)
}

// Returns whether there are any container handler factories registered.
func HasFactories() bool {
	factoriesLock.Lock()
	defer factoriesLock.Unlock()

	return len(factories) != 0
}

// Create a new ContainerHandler for the specified container.
func NewContainerHandler(name string, inHostNamespace bool) (ContainerHandler, error) {
	factoriesLock.RLock()
	defer factoriesLock.RUnlock()

	// Create the ContainerHandler with the first factory that supports it.
	for _, factory := range factories {
		handler, err := factory.NewContainerHandler(name, inHostNamespace)
		if err != nil {
			glog.V(4).Infof("Error trying to work out if we can handle %s: %v", name, err)
		} else if handler != nil {
			if _, ok := handler.(*ignoreHandler); ok {
				glog.V(3).Infof("Factory %q can handle container %q, but ignoring.", factory, name)
				return nil, nil
			}
			return handler, err
		}

		glog.V(4).Infof("Factory %q was unable to handle container %q", factory, name)
	}

	return nil, fmt.Errorf("no known factory can handle creation of container")
}

// Clear the known factories.
func ClearContainerHandlerFactories() {
	factoriesLock.Lock()
	defer factoriesLock.Unlock()

	factories = make([]ContainerHandlerFactory, 0, 4)
}

func DebugInfo() map[string][]string {
	factoriesLock.RLock()
	defer factoriesLock.RUnlock()

	// Get debug information for all factories.
	out := make(map[string][]string)
	for _, factory := range factories {
		for k, v := range factory.DebugInfo() {
			out[k] = v
		}
	}
	return out
}
