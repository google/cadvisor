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
	"fmt"
	"log"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
)

type ContainerStats struct {
	ContainerReference info.ContainerReference
	Stats              *info.ContainerStats

	// ResChan is nil if the sender wants to ignore the result
	ResChan chan<- error
}

// Create numWorkers goroutines to write container stats info into the
// specified storage.  Returns a write-only channel for the caller to write
// container stats. Closing this channel will stop all workers.
func StartContainerStatsWriters(
	numWorkers int,
	queueLength int,
	statsWriter storage.ContainerStatsWriter,
) (chan<- *ContainerStats, error) {
	if statsWriter == nil {
		return nil, fmt.Errorf("invalid stats writer")
	}
	var ch chan *ContainerStats
	if queueLength > 0 {
		ch = make(chan *ContainerStats, queueLength)
	} else {
		ch = make(chan *ContainerStats)
	}
	for i := 0; i < numWorkers; i++ {
		go writeContainerStats(statsWriter, ch)
	}
	return ch, nil
}

func writeContainerStats(statsWriter storage.ContainerStatsWriter, ch <-chan *ContainerStats) {
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
