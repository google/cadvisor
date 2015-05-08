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
	NewContainerHandler(name string) (c ContainerHandler, err error)

	// Returns whether this factory can handle and accept the specified container.
	CanHandleAndAccept(name string) (handle bool, accept bool, err error)

	// Name of the factory.
	String() string
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
func NewContainerHandler(name string) (ContainerHandler, bool, error) {
	factoriesLock.RLock()
	defer factoriesLock.RUnlock()

	// Create the ContainerHandler with the first factory that supports it.
	for _, factory := range factories {
		canHandle, canAccept, err := factory.CanHandleAndAccept(name)
		if err != nil {
			glog.V(4).Infof("Error trying to work out if we can handle %s: %v", name, err)
		}
		if canHandle {
			if !canAccept {
				glog.V(3).Infof("Factory %q can handle container %q, but ignoring.", factory, name)
				return nil, false, nil
			}
			glog.V(3).Infof("Using factory %q for container %q", factory, name)
			handle, err := factory.NewContainerHandler(name)
			return handle, canAccept, err
		} else {
			glog.V(4).Infof("Factory %q was unable to handle container %q", factory, name)
		}
	}

	return nil, false, fmt.Errorf("no known factory can handle creation of container")
}

// Clear the known factories.
func ClearContainerHandlerFactories() {
	factoriesLock.Lock()
	defer factoriesLock.Unlock()

	factories = make([]ContainerHandlerFactory, 0, 4)
}
