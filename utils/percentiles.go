// Copyright 2015 Google Inc. All Rights Reserved.
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
	"math"
	"sort"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
)

const milliSecondsToNanoSeconds = 1000000
const secondsToMilliSeconds = 1000

type uint64Slice []uint64

func (a uint64Slice) Len() int           { return len(a) }
func (a uint64Slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a uint64Slice) Less(i, j int) bool { return a[i] < a[j] }

// TODO(rjnagal): Move out when we update API.
type Percentiles struct {
	// Average over the collected sample.
	Mean uint64 `json:"mean"`
	// Max seen over the collected sample.
	Max uint64 `json:"max"`
	// 90th percentile over the collected sample.
	Ninety uint64 `json:"ninety"`
}

// Get 90th percentile of the provided samples. Round to integer.
func (self uint64Slice) Get90Percentile() uint64 {
	count := self.Len()
	if count == 0 {
		return 0
	}
	sort.Sort(self)
	n := float64(0.9 * (float64(count) + 1))
	idx, frac := math.Modf(n)
	index := int(idx)
	percentile := float64(self[index-1])
	if index > 1 || index < count {
		percentile += frac * float64(self[index]-self[index-1])
	}
	return uint64(percentile)
}

type Mean struct {
	// current count.
	count uint64
	// current mean.
	Mean float64
}

func (self *Mean) Add(value uint64) {
	self.count++
	if self.count == 1 {
		self.Mean = float64(value)
		return
	}
	c := float64(self.count)
	v := float64(value)
	self.Mean = (self.Mean*(c-1) + v) / c
}

// Returns cpu and memory usage percentiles.
func GetPercentiles(stats []*info.ContainerStats) (Percentiles, Percentiles) {
	lastCpu := uint64(0)
	lastTime := time.Time{}
	memorySamples := make(uint64Slice, 0, len(stats))
	cpuSamples := make(uint64Slice, 0, len(stats)-1)
	numSamples := 0
	memoryMean := Mean{count: 0, Mean: 0}
	cpuMean := Mean{count: 0, Mean: 0}
	memoryPercentiles := Percentiles{}
	cpuPercentiles := Percentiles{}
	for _, stat := range stats {
		var elapsed int64
		time := stat.Timestamp
		if !lastTime.IsZero() {
			elapsed = time.UnixNano() - lastTime.UnixNano()
			if elapsed < 10*milliSecondsToNanoSeconds {
				glog.Infof("Elapsed time too small: %d ns: time now %s last %s", elapsed, time.String(), lastTime.String())
				continue
			}
		}
		numSamples++
		cpuNs := stat.Cpu.Usage.Total
		// Ignore actual usage and only focus on working set.
		memory := stat.Memory.WorkingSet
		if memory > memoryPercentiles.Max {
			memoryPercentiles.Max = memory
		}
		glog.V(2).Infof("Read sample: cpu %d, memory %d", cpuNs, memory)
		memoryMean.Add(memory)
		memorySamples = append(memorySamples, memory)
		if lastTime.IsZero() {
			lastCpu = cpuNs
			lastTime = time
			continue
		}
		cpuRate := (cpuNs - lastCpu) * secondsToMilliSeconds / uint64(elapsed)
		if cpuRate < 0 {
			glog.Infof("cpu rate too small: %f ns", cpuRate)
			continue
		}
		glog.V(2).Infof("Adding cpu rate sample : %d", cpuRate)
		lastCpu = cpuNs
		lastTime = time
		cpuSamples = append(cpuSamples, cpuRate)
		if cpuRate > cpuPercentiles.Max {
			cpuPercentiles.Max = cpuRate
		}
		cpuMean.Add(cpuRate)
	}
	cpuPercentiles.Mean = uint64(cpuMean.Mean)
	memoryPercentiles.Mean = uint64(memoryMean.Mean)
	cpuPercentiles.Ninety = cpuSamples.Get90Percentile()
	memoryPercentiles.Ninety = memorySamples.Get90Percentile()
	return cpuPercentiles, memoryPercentiles
}
