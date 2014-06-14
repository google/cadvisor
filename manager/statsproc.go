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

package manager

import (
	"log"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
)

type ContainerStats struct {
	ContainerReference info.ContainerReference
	Stats              *info.ContainerStats
	ResChan            chan<- error
}

type ContainerStatsProcessor interface {
	StartStatsProcessors(numProcs int) (chan<- *ContainerStats, error)
	StopAllProcessors()
}

type containerStatsWriter struct {
	config *storage.Config
}

func (self *containerStatsWriter) StartStatsProcessors(numProcs int) (chan<- *ContainerStats, error) {
	ch := make(chan *ContainerStats)
	statsWriter, err := storage.NewContainerStatsWriter(self.config)
	if err != nil {
		return nil, err
	}
	for i := 0; i < numProcs; i++ {
		go self.process(statsWriter, ch)
	}
	return ch, nil
}

func (self *containerStatsWriter) process(statsWriter storage.ContainerStatsWriter, ch <-chan *ContainerStats) {
	for stats := range ch {
		err := statsWriter.Write(stats.ContainerReference, stats.Stats)
		if err != nil {
			log.Printf("unable to write stats: %v", err)
		}
		if stats.ResChan != nil {
			stats.ResChan <- err
		}
	}
}
