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
	"github.com/google/cadvisor/db"
	"github.com/google/cadvisor/info"
)

type containerStatsWriter struct {
	statsWriter db.ContainerStatsWriter
	handler     ContainerHandler
}

func (self *containerStatsWriter) ContainerReference() (info.ContainerReference, error) {
	return self.handler.ContainerReference()
}

func (self *containerStatsWriter) GetStats() (*info.ContainerStats, error) {
	stats, err := self.handler.GetStats()
	if err != nil {
		return nil, err
	}
	if self.statsWriter == nil {
		return stats, nil
	}
	// XXX(dengnan): should we write stats in another goroutine?
	ref, err := self.handler.ContainerReference()
	if err != nil {
		return nil, err
	}
	err = self.statsWriter.Write(ref, stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (self *containerStatsWriter) GetSpec() (*info.ContainerSpec, error) {
	return self.handler.GetSpec()
}

func (self *containerStatsWriter) ListContainers(listType ListType) ([]info.ContainerReference, error) {
	return self.handler.ListContainers(listType)
}

func (self *containerStatsWriter) ListThreads(listType ListType) ([]int, error) {
	return self.handler.ListThreads(listType)
}

func (self *containerStatsWriter) ListProcesses(listType ListType) ([]int, error) {
	return self.handler.ListProcesses(listType)
}

func (self *containerStatsWriter) StatsSummary() (*info.ContainerStatsPercentiles, error) {
	return self.handler.StatsSummary()
}

type containerStatsWriterDecorator struct {
	dbConfig    *db.DatabaseConfig
	statsWriter db.ContainerStatsWriter
}

func (self *containerStatsWriterDecorator) Decorate(container ContainerHandler) (ContainerHandler, error) {
	return &containerStatsWriter{
		statsWriter: self.statsWriter,
		handler:     container,
	}, nil
}

func NewStatsWriterDecorator(config *db.DatabaseConfig) (ContainerHandlerDecorator, error) {
	statsWriter, err := db.NewContainerStatsWriter(config)
	if err != nil {
		return nil, err
	}
	return &containerStatsWriterDecorator{
		dbConfig:    config,
		statsWriter: statsWriter,
	}, nil
}
