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

package memory

import (
	"sort"
	"time"

	info "github.com/google/cadvisor/info/v1"
)

// A time-based buffer for ContainerStats. Holds information for a specific time period.
type StatsBuffer struct {
	buffer []*info.ContainerStats
	age    time.Duration
}

// Returns a new thread-compatible StatsBuffer.
func NewStatsBuffer(age time.Duration) *StatsBuffer {
	return &StatsBuffer{
		buffer: make([]*info.ContainerStats, 0),
		age:    age,
	}
}

// Adds an element to the start of the buffer (removing one from the end if necessary).
func (self *StatsBuffer) Add(item *info.ContainerStats) {
	// Remove any elements before the eviction time.
	evictTime := item.Timestamp.Add(-self.age)
	index := sort.Search(len(self.buffer), func(index int) bool {
		return self.buffer[index].Timestamp.After(evictTime)
	})
	if index < len(self.buffer) {
		self.buffer = self.buffer[index:]
	}

	copied := *item
	self.buffer = append(self.buffer, &copied)
}

// Returns up to maxResult elements in the specified time period (inclusive).
// Results are from first to last. maxResults of -1 means no limit. When first
// and last are specified, maxResults is ignored.
func (self *StatsBuffer) InTimeRange(start, end time.Time, maxResults int) []*info.ContainerStats {
	// No stats, return empty.
	if len(self.buffer) == 0 {
		return []*info.ContainerStats{}
	}

	// Return all results in a time range if specified.
	if !start.IsZero() && !end.IsZero() {
		maxResults = -1
	}

	// NOTE: Since we store the elments in descending timestamp order "start" will
	// be a higher index than "end".

	var startIndex int
	if start.IsZero() {
		// None specified, start at the beginning.
		startIndex = len(self.buffer) - 1
	} else {
		// Start is the index before the elements smaller than it. We do this by
		// finding the first element smaller than start and taking the index
		// before that element
		startIndex = sort.Search(len(self.buffer), func(index int) bool {
			// buffer[index] < start
			return self.Get(index).Timestamp.Before(start)
		}) - 1
		// Check if start is after all the data we have.
		if startIndex < 0 {
			return []*info.ContainerStats{}
		}
	}

	var endIndex int
	if end.IsZero() {
		// None specified, end with the latest stats.
		endIndex = 0
	} else {
		// End is the first index smaller than or equal to it (so, not larger).
		endIndex = sort.Search(len(self.buffer), func(index int) bool {
			// buffer[index] <= t -> !(buffer[index] > t)
			return !self.Get(index).Timestamp.After(end)
		})
		// Check if end is before all the data we have.
		if endIndex == len(self.buffer) {
			return []*info.ContainerStats{}
		}
	}

	// Trim to maxResults size.
	numResults := startIndex - endIndex + 1
	if maxResults != -1 && numResults > maxResults {
		startIndex -= numResults - maxResults
		numResults = maxResults
	}

	// Return in sorted timestamp order so from the "back" to "front".
	result := make([]*info.ContainerStats, numResults)
	for i := 0; i < numResults; i++ {
		result[i] = self.Get(startIndex - i)
	}
	return result
}

// Gets the element at the specified index. Note that elements are output in LIFO order.
func (self *StatsBuffer) Get(index int) *info.ContainerStats {
	return self.buffer[len(self.buffer)-index-1]
}

func (self *StatsBuffer) Size() int {
	return len(self.buffer)
}
