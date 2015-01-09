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
	"github.com/google/cadvisor/info"
)

// A circular buffer for ContainerStats.
type StatsBuffer struct {
	buffer []info.ContainerStats
	size   int
	index  int
}

// Returns a new thread-compatible StatsBuffer.
func NewStatsBuffer(size int) *StatsBuffer {
	return &StatsBuffer{
		buffer: make([]info.ContainerStats, size),
		size:   0,
		index:  size - 1,
	}
}

// Adds an element to the start of the buffer (removing one from the end if necessary).
func (self *StatsBuffer) Add(item *info.ContainerStats) {
	if self.size < len(self.buffer) {
		self.size++
	}
	self.index = (self.index + 1) % len(self.buffer)
	self.buffer[self.index] = *item
}

// Returns the first N elements in the buffer. If N > size of buffer, size of buffer elements are returned.
func (self *StatsBuffer) FirstN(n int) []*info.ContainerStats {
	// Cap n at the number of elements we have.
	if n > self.size {
		n = self.size
	}

	// index points to the latest element, get n before that one (keeping in mind we may have gone through 0).
	start := self.index - (n - 1)
	if start < 0 {
		start += len(self.buffer)
	}

	// Copy the elements.
	res := make([]*info.ContainerStats, n)
	for i := 0; i < n; i++ {
		index := (start + i) % len(self.buffer)
		res[i] = &self.buffer[index]
	}
	return res
}

func (self *StatsBuffer) Size() int {
	return self.size
}
