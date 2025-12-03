// Copyright 2025 Google Inc. All Rights Reserved.
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
	"sync"
	"testing"
)

// Benchmark concurrent reads on sync.Map
func BenchmarkSyncMapConcurrentReads(b *testing.B) {
	var m sync.Map

	for i := 0; i < 1000; i++ {
		key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i)}
		m.Store(key, &containerData{})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i%1000)}
			m.Load(key)
			i++
		}
	})
}

// Benchmark concurrent reads on RWMutex and map
func BenchmarkRWMutexMapConcurrentReads(b *testing.B) {
	var mu sync.RWMutex
	m := make(map[namespacedContainerName]*containerData)

	for i := 0; i < 1000; i++ {
		key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i)}
		m[key] = &containerData{}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i%1000)}
			mu.RLock()
			_ = m[key]
			mu.RUnlock()
			i++
		}
	})
}

// Benchmark iteration with sync.Map
func BenchmarkSyncMapIteration(b *testing.B) {
	var m sync.Map

	for i := 0; i < 1000; i++ {
		key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i)}
		m.Store(key, &containerData{})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		m.Range(func(_, _ any) bool {
			count++
			return true
		})
	}
}

// Benchmark iteration with RWMutex and map
func BenchmarkRWMutexMapIteration(b *testing.B) {
	var mu sync.RWMutex
	m := make(map[namespacedContainerName]*containerData)

	for i := 0; i < 1000; i++ {
		key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i)}
		m[key] = &containerData{}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		count := 0
		for range m {
			count++
		}
		mu.RUnlock()
	}
}

// Benchmark mixed read/write with sync.Map
func BenchmarkSyncMapMixedReadWrite(b *testing.B) {
	var m sync.Map

	for i := 0; i < 1000; i++ {
		key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i)}
		m.Store(key, &containerData{})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%100 == 0 {
				// 1% writes
				key := namespacedContainerName{Name: fmt.Sprintf("/container/new/%d", i)}
				m.Store(key, &containerData{})
			} else {
				// 99% reads
				key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i%1000)}
				m.Load(key)
			}
			i++
		}
	})
}

// Benchmark mixed read/write with RWMutex and map
func BenchmarkRWMutexMapMixedReadWrite(b *testing.B) {
	var mu sync.RWMutex
	m := make(map[namespacedContainerName]*containerData)

	for i := 0; i < 1000; i++ {
		key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i)}
		m[key] = &containerData{}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%100 == 0 {
				// 1% writes
				key := namespacedContainerName{Name: fmt.Sprintf("/container/new/%d", i)}
				mu.Lock()
				m[key] = &containerData{}
				mu.Unlock()
			} else {
				// 99% reads
				key := namespacedContainerName{Name: fmt.Sprintf("/container/%d", i%1000)}
				mu.RLock()
				_ = m[key]
				mu.RUnlock()
			}
			i++
		}
	})
}
