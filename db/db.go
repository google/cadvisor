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

package db

import (
	"fmt"
	"sync"

	"github.com/google/cadvisor/info"
)

type ContainerStatsWriter interface {
	Write(ref info.ContainerReference, stats *info.ContainerStats) error
}

// Database config which should contain all information used to connect to
// all/most databases
type DatabaseConfig struct {
	Engine   string            `json:"engine,omitempty"`
	Host     string            `json:"host,omitempty"`
	Port     int               `json:"port,omitempty"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Database string            `json:"database,omitempty"`
	Params   map[string]string `json:"parameters,omitempty"`
}

type ContainerStatsWriterFactory interface {
	String() string
	New(config *DatabaseConfig) (ContainerStatsWriter, error)
}

type containerStatsWriterFactoryManager struct {
	lock      sync.RWMutex
	factories map[string]ContainerStatsWriterFactory
}

func (self *containerStatsWriterFactoryManager) Register(factory ContainerStatsWriterFactory) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.factories == nil {
		self.factories = make(map[string]ContainerStatsWriterFactory, 8)
	}

	self.factories[factory.String()] = factory
}

func (self *containerStatsWriterFactoryManager) New(config *DatabaseConfig) (ContainerStatsWriter, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()

	if factory, ok := self.factories[config.Engine]; ok {
		return factory.New(config)
	}
	return nil, fmt.Errorf("unknown database %v", config.Engine)
}

var globalContainerStatsWriterFactoryManager containerStatsWriterFactoryManager

func RegisterContainerStatsWriterFactory(factory ContainerStatsWriterFactory) {
	globalContainerStatsWriterFactoryManager.Register(factory)
}

func NewContainerStatsWriter(config *DatabaseConfig) (ContainerStatsWriter, error) {
	return globalContainerStatsWriterFactoryManager.New(config)
}
