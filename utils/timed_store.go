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

package utils

import (
	"sort"
	"time"
)

type timedStoreRingBuffer struct {
	start   int
	end     int // Last element index + 1
	buffer  []timedStoreData
	limited bool // Whether the buffer should grow or overwrite
	empty   bool // Necessary to distinguish full & empty
}

func newTimedStoreRingBuffer(cap int, limited bool) timedStoreRingBuffer {
	return timedStoreRingBuffer{
		start:   0,
		end:     0,
		buffer:  make([]timedStoreData, cap),
		limited: limited,
		empty:   true,
	}
}

// Translate index to buffer position.
func (t *timedStoreRingBuffer) index(i int) int {
	size := len(t.buffer)
	offset := i + t.start
	if offset < size {
		return offset
	}
	return offset - size
}

// Get the element at position i. Assume buffer is non-empty.
func (t *timedStoreRingBuffer) Get(i int) timedStoreData {
	return t.buffer[t.index(i)]
}

// Get the last element in the buffer. Assume buffer is non-empty.
func (t *timedStoreRingBuffer) Last() timedStoreData {
	if t.end > 0 {
		return t.buffer[t.end-1]
	}
	return t.buffer[len(t.buffer)-1]
}

// timedStoreRingBuffer implements sort.Interface
func (t *timedStoreRingBuffer) Less(i, j int) bool {
	return t.Get(i).timestamp.Before(t.Get(j).timestamp)
}

func (t *timedStoreRingBuffer) Len() int {
	if t.empty {
		return 0
	}
	length := t.end - t.start
	if length <= 0 {
		return length + len(t.buffer)
	}
	return length
}

func (t *timedStoreRingBuffer) Swap(i, j int) {
	iIndex := t.index(i)
	jIndex := t.index(j)
	t.buffer[iIndex], t.buffer[jIndex] = t.buffer[jIndex], t.buffer[iIndex]
}

func (t *timedStoreRingBuffer) grow() int {
	// Double capacity.
	newCap := len(t.buffer) * 2
	buffer := make([]timedStoreData, newCap)
	// Copy data
	if t.start < t.end {
		copy(buffer, t.buffer[t.start:t.end])
	} else {
		// Data wraps around, copy 2 segments.
		size := copy(buffer, t.buffer[t.start:])
		copy(buffer[size:], t.buffer[:t.end])
	}
	t.end = t.Len()
	t.start = 0
	t.buffer = buffer
	return newCap
}

func (t *timedStoreRingBuffer) Append(item timedStoreData) {
	size := len(t.buffer)
	if t.start == t.end && !t.empty {
		// Buffer is full.
		if t.limited {
			// Buffer is limited, evict oldest data.
			t.start++
			if t.start == size {
				t.start = 0
			}
		} else {
			// Buffer is unlimited, add more room.
			size = t.grow()
		}
	}
	t.empty = false
	t.buffer[t.end] = item
	t.end++
	if t.end == size {
		t.end = 0
	}
}

func (t *timedStoreRingBuffer) Insert(i int, item timedStoreData) {
	size := len(t.buffer)
	if t.start == t.end && !t.empty {
		// Buffer is full.
		if t.limited {
			// Buffer is limited, evict oldest data.
			t.start++
			i--
			if t.start == size {
				t.start = 0
			}
		} else {
			// Buffer is unlimited, add more room.
			size = t.grow()
		}
	}
	index := t.index(i)
	t.empty = false
	if i <= 0 { // Prepend data.
		t.start--
		if t.start < 0 {
			t.start = size - 1
		}
		index = t.start
	} else if i < size/2 { // Shift lower half
		index--
		if index < 0 {
			index = size - 1
		}
		bottom := t.start
		if t.start > index {
			copy(t.buffer[t.start-1:], t.buffer[t.start:]) // Shift upper segment.
		}
		if t.start > index || t.start == 0 {
			bottom = 1                     // 0 is wrapped, this is the start of the lower block.
			t.buffer[size-1] = t.buffer[0] // Wrap bottom
		}
		if bottom <= index {
			copy(t.buffer[bottom-1:index], t.buffer[bottom:index+1]) // Shift lower segment.
		}
		t.start--
		if t.start < 0 {
			t.start = size - 1
		}
	} else { // Shift upper half
		top := t.end
		if t.end < index {
			copy(t.buffer[1:t.end+1], t.buffer[0:t.end]) // Shift lower segment.
		}
		if t.end < index || t.end == size {
			top = size - 1                 // end is wrapped, this is the end of the upper block.
			t.buffer[0] = t.buffer[size-1] // Wrap top
		}
		if top > index {
			copy(t.buffer[index+1:top+1], t.buffer[index:top]) // Shift upper segment.
		}
		t.end++
		if t.end == size {
			t.end = 0
		}
	}
	t.buffer[index] = item
}

func (t *timedStoreRingBuffer) RemoveFirstN(n int) {
	if n >= t.Len() {
		t.start = t.end
		t.empty = true
		return
	}
	t.start += n
	size := len(t.buffer)
	if t.start >= size {
		t.start = t.start - size
	}
}

// A time-based buffer for ContainerStats.
// Holds information for a specific time period and/or a max number of items.
type TimedStore struct {
	buffer   timedStoreRingBuffer
	age      time.Duration
	maxItems int
}

type timedStoreData struct {
	timestamp time.Time
	data      interface{}
}

// Returns a new thread-compatible TimedStore.
// A maxItems value of -1 means no limit.
func NewTimedStore(age time.Duration, maxItems int) *TimedStore {
	var buffer timedStoreRingBuffer
	if maxItems < 0 {
		buffer = newTimedStoreRingBuffer(128, false)
	} else {
		buffer = newTimedStoreRingBuffer(maxItems, true)
	}
	return &TimedStore{
		buffer:   buffer,
		age:      age,
		maxItems: maxItems,
	}
}

// Adds an element to the start of the buffer (removing one from the end if necessary).
func (self *TimedStore) Add(timestamp time.Time, item interface{}) {
	data := timedStoreData{
		timestamp: timestamp,
		data:      item,
	}
	// Common case: data is added in order.
	if self.buffer.Len() == 0 || !timestamp.Before(self.buffer.Last().timestamp) {
		self.buffer.Append(data)
	} else {
		// Data is out of order; insert it in the correct position.
		index := sort.Search(self.buffer.Len(), func(index int) bool {
			return self.buffer.Get(index).timestamp.After(timestamp)
		})
		self.buffer.Insert(index, data)
	}

	// Remove any elements before eviction time.
	// TODO(rjnagal): This is assuming that the added entry has timestamp close to now.
	evictTime := timestamp.Add(-self.age)
	index := sort.Search(self.buffer.Len(), func(index int) bool {
		return self.buffer.Get(index).timestamp.After(evictTime)
	})
	if index < self.buffer.Len() {
		self.buffer.RemoveFirstN(index)
	}
}

// Returns up to maxResult elements in the specified time period (inclusive).
// Results are from first to last. maxResults of -1 means no limit.
func (self *TimedStore) InTimeRange(start, end time.Time, maxResults int) []interface{} {
	// No stats, return empty.
	if self.buffer.Len() == 0 {
		return []interface{}{}
	}

	var startIndex int
	if start.IsZero() {
		// None specified, start at the beginning.
		startIndex = self.buffer.Len() - 1
	} else {
		// Start is the index before the elements smaller than it. We do this by
		// finding the first element smaller than start and taking the index
		// before that element
		startIndex = sort.Search(self.buffer.Len(), func(index int) bool {
			// buffer[index] < start
			return self.getData(index).timestamp.Before(start)
		}) - 1
		// Check if start is after all the data we have.
		if startIndex < 0 {
			return []interface{}{}
		}
	}

	var endIndex int
	if end.IsZero() {
		// None specified, end with the latest stats.
		endIndex = 0
	} else {
		// End is the first index smaller than or equal to it (so, not larger).
		endIndex = sort.Search(self.buffer.Len(), func(index int) bool {
			// buffer[index] <= t -> !(buffer[index] > t)
			return !self.getData(index).timestamp.After(end)
		})
		// Check if end is before all the data we have.
		if endIndex == self.buffer.Len() {
			return []interface{}{}
		}
	}

	// Trim to maxResults size.
	numResults := startIndex - endIndex + 1
	if maxResults != -1 && numResults > maxResults {
		startIndex -= numResults - maxResults
		numResults = maxResults
	}

	// Return in sorted timestamp order so from the "back" to "front".
	result := make([]interface{}, numResults)
	for i := 0; i < numResults; i++ {
		result[i] = self.Get(startIndex - i)
	}
	return result
}

// Gets the element at the specified index. Note that elements are output in LIFO order.
func (self *TimedStore) Get(index int) interface{} {
	return self.getData(index).data
}

// Gets the data at the specified index. Note that elements are output in LIFO order.
func (self *TimedStore) getData(index int) timedStoreData {
	return self.buffer.Get(self.buffer.Len() - index - 1)
}

func (self *TimedStore) Size() int {
	return self.buffer.Len()
}
