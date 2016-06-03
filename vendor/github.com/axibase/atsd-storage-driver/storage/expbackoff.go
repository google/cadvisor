/*
* Copyright 2015 Axibase Corporation or its affiliates. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License").
* You may not use this file except in compliance with the License.
* A copy of the License is located at
*
* https://www.axibase.com/atsd/axibase-apache-2.0.pdf
*
* or in the "license" file accompanying this file. This file is distributed
* on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
* express or implied. See the License for the specific language governing
* permissions and limitations under the License.
 */

package storage

import (
	"math"
	"math/rand"
	"time"
)

const MaxDuration time.Duration = 1<<63 - 1
const maxPowerBeforeOverflow = 62

type ExpBackoff struct {
	counter  int
	limit    time.Duration
	timespan time.Duration
	randGen  *rand.Rand
}

func NewExpBackoff(timespan, limit time.Duration) *ExpBackoff {
	src := rand.NewSource(time.Now().UTC().UnixNano())
	randGen := rand.New(src)
	return &ExpBackoff{counter: 1, limit: limit, timespan: timespan, randGen: randGen}
}
func (self *ExpBackoff) Duration() time.Duration {
	var maxRand int64 = math.MaxInt64
	if self.counter <= maxPowerBeforeOverflow {
		maxRand = int64(math.Pow(2, float64(self.counter)))
		self.counter++
	}
	randNumber := self.randGen.Int63n(maxRand)
	duration := self.limit
	if time.Duration(randNumber) <= self.limit/self.timespan {
		duration = time.Duration(randNumber) * self.timespan
	}
	return duration
}
func (self *ExpBackoff) Reset() {
	self.counter = 1
}
