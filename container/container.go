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

// Package container defines types for sub-container events and also
// defines an interface for container operation handlers.
package container

import info "github.com/google/cadvisor/info/v1"

// ListType describes whether listing should be just for a
// specific container or performed recursively.
type ListType int

const (
	ListSelf ListType = iota
	ListRecursive
)

// SubcontainerEventType indicates an addition or deletion event.
type SubcontainerEventType int

const (
	SubcontainerAdd SubcontainerEventType = iota
	SubcontainerDelete
)

// SubcontainerEvent represents a
type SubcontainerEvent struct {
	// The type of event that occurred.
	EventType SubcontainerEventType

	// The full container name of the container where the event occurred.
	Name string
}

// Interface for container operation handlers.
type ContainerHandler interface {
	// Returns the ContainerReference
	ContainerReference() (info.ContainerReference, error)

	// Returns container's isolation spec.
	GetSpec() (info.ContainerSpec, error)

	// Returns the current stats values of the container.
	GetStats() (*info.ContainerStats, error)

	// Returns the subcontainers of this container.
	ListContainers(listType ListType) ([]info.ContainerReference, error)

	// Returns the processes inside this container.
	ListProcesses(listType ListType) ([]int, error)

	// Registers a channel to listen for events affecting subcontainers (recursively).
	WatchSubcontainers(events chan SubcontainerEvent) error

	// Stops watching for subcontainer changes.
	StopWatchingSubcontainers() error

	// Returns absolute cgroup path for the requested resource.
	GetCgroupPath(resource string) (string, error)

	// Returns container labels, if available.
	GetContainerLabels() map[string]string

	// Returns whether the container still exists.
	Exists() bool

	// Cleanup frees up any resources being held like fds or go routines, etc.
	Cleanup()

	// Start starts any necessary background goroutines - must be cleaned up in Cleanup().
	// It is expected that most implementations will be a no-op.
	Start()
}

func NewIgnoreHandler(name string) ContainerHandler {
	return &ignoreHandler{
		cGroupPath: name,
	}
}

type ignoreHandler struct {
	cGroupPath string
}

func (handler *ignoreHandler) ContainerReference() (info.ContainerReference, error) {
	cr := info.ContainerReference{}

	return cr, nil
}

func (handler *ignoreHandler) GetSpec() (info.ContainerSpec, error) {
	cs := info.ContainerSpec{}

	return cs, nil
}

func (handler *ignoreHandler) GetStats() (*info.ContainerStats, error) {
	return nil, nil
}

func (handler *ignoreHandler) ListContainers(listType ListType) ([]info.ContainerReference, error) {
	return nil, nil
}

func (handler *ignoreHandler) ListThreads(listType ListType) ([]int, error) {
	return nil, nil
}

func (handler *ignoreHandler) ListProcesses(listType ListType) ([]int, error) {
	return nil, nil
}

func (handler *ignoreHandler) WatchSubcontainers(events chan SubcontainerEvent) error {
	return nil
}

func (handler *ignoreHandler) StopWatchingSubcontainers() error {
	return nil
}

func (handler *ignoreHandler) GetCgroupPath(resource string) (string, error) {
	return handler.cGroupPath, nil
}

func (handler *ignoreHandler) GetContainerLabels() map[string]string {
	return nil
}

func (handler *ignoreHandler) Exists() bool {
	return false
}

func (handler *ignoreHandler) Cleanup() {}

func (handler *ignoreHandler) Start() {}
