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
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	Iters = 10000
	Limit = 100
)

var datas = dataItems(Iters)

func BenchmarkAddSizeLimit(b *testing.B) {
	benchmarkAdd(b, Iters*10, Limit, false)
}

func BenchmarkAddAgeLimit(b *testing.B) {
	benchmarkAdd(b, Limit, -1, false)
}

func BenchmarkAddRandom(b *testing.B) {
	benchmarkAdd(b, Limit, Limit, true)
}

func BenchmarkAddRandomUnlimited(b *testing.B) {
	benchmarkAdd(b, Iters, -1, true)
}

func benchmarkAdd(b *testing.B, age time.Duration, maxItems int, random bool) {
	b.StopTimer()
	ts := NewTimedStore(age, maxItems)
	t := time.Now()
	var timer func() time.Duration
	if random {
		rand.Seed(1234)
		timer = func() time.Duration { return time.Duration(rand.Intn(Limit) - Limit/2) }
	} else {
		timer = func() time.Duration { return 1 }
	}
	b.StartTimer()
	for iter := 0; iter < b.N; iter++ {
		for i := 0; i < Iters; i++ {
			t = t.Add(timer())
			ts.Add(t, datas[i])
		}
	}
}

type data struct {
	s string
}

func dataItems(count int) []data {

	datas := make([]data, count)
	for i := 0; i < count; i++ {
		datas[i] = data{"foo bar"}
	}
	return datas
}

func TestRingAppend(t *testing.T) {
	ring := newTimedStoreRingBuffer(5, true)
	assert := assert.New(t)
	assert.Equal(0, ring.Len())
	// To full
	for i := 0; i < 5; i++ {
		ring.Append(timedStoreData{data: i})
		assert.Equal(i+1, ring.Len())
		assert.Equal(i, ring.Get(i).data)
	}
	// Full ring append
	for i := 0; i < 5; i++ {
		ring.Append(timedStoreData{data: i})
		assert.Equal(5, ring.Len())
		assert.Equal(i, ring.Get(4).data)
	}
}

func createTime(id int) time.Time {
	var zero time.Time
	return zero.Add(time.Duration(id+1) * time.Second)
}

func expectSize(t *testing.T, sb *TimedStore, expectedSize int) {
	if sb.Size() != expectedSize {
		t.Errorf("Expected size %v, got %v", expectedSize, sb.Size())
	}
}

func expectAllElements(t *testing.T, sb *TimedStore, expected []int) {
	size := sb.Size()
	els := make([]interface{}, size)
	for i := 0; i < size; i++ {
		els[i] = sb.Get(size - i - 1)
	}
	expectElements(t, els, expected)
}

func expectElements(t *testing.T, actual []interface{}, expected []int) {
	if len(actual) != len(expected) {
		t.Errorf("Expected elements %v, got %v", expected, actual)
		return
	}
	for i, el := range actual {
		if el.(int) != expected[i] {
			t.Errorf("Expected elements %v, got %v", expected, actual)
			return
		}
	}
}

func TestAdd(t *testing.T) {
	sb := NewTimedStore(5*time.Second, 100)

	// Add 1.
	sb.Add(createTime(0), 0)
	expectSize(t, sb, 1)
	expectAllElements(t, sb, []int{0})

	// Fill the buffer.
	for i := 1; i <= 5; i++ {
		expectSize(t, sb, i)
		sb.Add(createTime(i), i)
	}
	expectSize(t, sb, 5)
	expectAllElements(t, sb, []int{1, 2, 3, 4, 5})

	// Add more than is available in the buffer
	sb.Add(createTime(6), 6)
	expectSize(t, sb, 5)
	expectAllElements(t, sb, []int{2, 3, 4, 5, 6})

	// Replace all elements.
	for i := 7; i <= 10; i++ {
		sb.Add(createTime(i), i)
	}
	expectSize(t, sb, 5)
	expectAllElements(t, sb, []int{6, 7, 8, 9, 10})
}

func TestGet(t *testing.T) {
	sb := NewTimedStore(5*time.Second, -1)
	sb.Add(createTime(1), 1)
	sb.Add(createTime(2), 2)
	sb.Add(createTime(3), 3)
	expectSize(t, sb, 3)

	assert := assert.New(t)
	assert.Equal(sb.Get(0).(int), 3)
	assert.Equal(sb.Get(1).(int), 2)
	assert.Equal(sb.Get(2).(int), 1)
}

func TestInTimeRange(t *testing.T) {
	sb := NewTimedStore(5*time.Second, -1)
	assert := assert.New(t)

	var empty time.Time

	// No elements.
	assert.Empty(sb.InTimeRange(createTime(0), createTime(5), 10))
	assert.Empty(sb.InTimeRange(createTime(0), empty, 10))
	assert.Empty(sb.InTimeRange(empty, createTime(5), 10))
	assert.Empty(sb.InTimeRange(empty, empty, 10))

	// One element.
	sb.Add(createTime(1), 1)
	expectSize(t, sb, 1)
	expectElements(t, sb.InTimeRange(createTime(0), createTime(5), 10), []int{1})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(5), 10), []int{1})
	expectElements(t, sb.InTimeRange(createTime(0), createTime(1), 10), []int{1})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(1), 10), []int{1})
	assert.Empty(sb.InTimeRange(createTime(2), createTime(5), 10))

	// Two element.
	sb.Add(createTime(2), 2)
	expectSize(t, sb, 2)
	expectElements(t, sb.InTimeRange(createTime(0), createTime(5), 10), []int{1, 2})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(5), 10), []int{1, 2})
	expectElements(t, sb.InTimeRange(createTime(0), createTime(2), 10), []int{1, 2})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(2), 10), []int{1, 2})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(1), 10), []int{1})
	expectElements(t, sb.InTimeRange(createTime(2), createTime(2), 10), []int{2})
	assert.Empty(sb.InTimeRange(createTime(3), createTime(5), 10))

	// Many elements.
	sb.Add(createTime(3), 3)
	sb.Add(createTime(4), 4)
	expectSize(t, sb, 4)
	expectElements(t, sb.InTimeRange(createTime(0), createTime(5), 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(0), createTime(5), 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(5), 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(0), createTime(4), 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(4), 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(0), createTime(2), 10), []int{1, 2})
	expectElements(t, sb.InTimeRange(createTime(1), createTime(2), 10), []int{1, 2})
	expectElements(t, sb.InTimeRange(createTime(2), createTime(3), 10), []int{2, 3})
	expectElements(t, sb.InTimeRange(createTime(3), createTime(4), 10), []int{3, 4})
	expectElements(t, sb.InTimeRange(createTime(3), createTime(5), 10), []int{3, 4})
	assert.Empty(sb.InTimeRange(createTime(5), createTime(5), 10))

	// Start and end time does't ignore maxResults.
	expectElements(t, sb.InTimeRange(createTime(1), createTime(5), 1), []int{4})

	// No start time.
	expectElements(t, sb.InTimeRange(empty, createTime(5), 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(empty, createTime(4), 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(empty, createTime(3), 10), []int{1, 2, 3})
	expectElements(t, sb.InTimeRange(empty, createTime(2), 10), []int{1, 2})
	expectElements(t, sb.InTimeRange(empty, createTime(1), 10), []int{1})

	// No end time.
	expectElements(t, sb.InTimeRange(createTime(0), empty, 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(1), empty, 10), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(2), empty, 10), []int{2, 3, 4})
	expectElements(t, sb.InTimeRange(createTime(3), empty, 10), []int{3, 4})
	expectElements(t, sb.InTimeRange(createTime(4), empty, 10), []int{4})

	// No start or end time.
	expectElements(t, sb.InTimeRange(empty, empty, 10), []int{1, 2, 3, 4})

	// Start after data.
	assert.Empty(sb.InTimeRange(createTime(5), createTime(5), 10))
	assert.Empty(sb.InTimeRange(createTime(5), empty, 10))

	// End before data.
	assert.Empty(sb.InTimeRange(createTime(0), createTime(0), 10))
	assert.Empty(sb.InTimeRange(empty, createTime(0), 10))
}

func TestInTimeRangeWithLimit(t *testing.T) {
	sb := NewTimedStore(5*time.Second, -1)
	sb.Add(createTime(1), 1)
	sb.Add(createTime(2), 2)
	sb.Add(createTime(3), 3)
	sb.Add(createTime(4), 4)
	expectSize(t, sb, 4)

	var empty time.Time

	// Limit cuts off from latest timestamp.
	expectElements(t, sb.InTimeRange(empty, empty, 4), []int{1, 2, 3, 4})
	expectElements(t, sb.InTimeRange(empty, empty, 3), []int{2, 3, 4})
	expectElements(t, sb.InTimeRange(empty, empty, 2), []int{3, 4})
	expectElements(t, sb.InTimeRange(empty, empty, 1), []int{4})
	assert.Empty(t, sb.InTimeRange(empty, empty, 0))
}

func TestLimitedSize(t *testing.T) {
	sb := NewTimedStore(time.Hour, 5)

	// Add 1.
	sb.Add(createTime(0), 0)
	expectSize(t, sb, 1)
	expectAllElements(t, sb, []int{0})

	// Fill the buffer.
	for i := 1; i <= 5; i++ {
		expectSize(t, sb, i)
		sb.Add(createTime(i), i)
	}
	expectSize(t, sb, 5)
	expectAllElements(t, sb, []int{1, 2, 3, 4, 5})

	// Add more than is available in the buffer
	sb.Add(createTime(6), 6)
	expectSize(t, sb, 5)
	expectAllElements(t, sb, []int{2, 3, 4, 5, 6})

	// Replace all elements.
	for i := 7; i <= 10; i++ {
		sb.Add(createTime(i), i)
	}
	expectSize(t, sb, 5)
	expectAllElements(t, sb, []int{6, 7, 8, 9, 10})
}

func verifyOrder(t *testing.T, ts *TimedStore, expectedSize int) {
	require.Equal(t, expectedSize, ts.Size())
	if expectedSize == 1 {
		return
	}
	for i := 1; i < expectedSize; i++ {
		first := ts.buffer.Get(i - 1)
		second := ts.buffer.Get(i)
		require.True(t, !first.timestamp.After(second.timestamp),
			"Elements %d (%v) and %d (%v) out of order", i, second.timestamp, i-1, first.timestamp)
	}
}

const Forever time.Duration = 1 << 30

func TestAddRandom(t *testing.T) {
	ts := NewTimedStore(Forever, Limit)
	start := time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC)
	rand.Seed(1234)

	// Fill buffer
	for i := 0; i < Limit; i++ {
		timestamp := start.Add(time.Duration(rand.Intn(Limit*2) - Limit))
		ts.Add(timestamp, datas[i])
		verifyOrder(t, ts, i+1)
	}

	// Full buffer
	for i := 0; i < Iters; i++ {
		timestamp := start.Add(time.Duration(rand.Intn(Limit*2) - Limit))
		ts.Add(timestamp, datas[i])
		verifyOrder(t, ts, Limit)
	}
}

func TestRandomUnlimited(t *testing.T) {
	ts := NewTimedStore(Forever, -1)
	start := time.Date(2000, 1, 1, 0, 0, 1, 0, time.UTC)
	rand.Seed(1234)

	for i := 0; i < Iters; i++ {
		timestamp := start.Add(time.Duration(rand.Intn(Limit*2) - Limit))
		ts.Add(timestamp, datas[i])
		verifyOrder(t, ts, i+1)
	}
}
