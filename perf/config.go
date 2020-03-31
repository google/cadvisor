// Copyright 2020 Google Inc. All Rights Reserved.
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

// Configuration for perf event manager.
package perf

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"k8s.io/klog"
)

type Events struct {
	// List of perf events' names to be measured. Events in this list
	// will not be grouped. See group_fd argument in documentation
	// at man perf_event_open
	NonGrouped []string  `json:"non_grouped"`

	// List of groups of perf events' names to be measured. See group_fd
	// argument documentation at man perf_event_open.
	Grouped    [][]string  `json:"grouped"`

	// Raw allows to specify events by passing their type and configuration directly.
	// See symbolically formed events documentation at man perf-stat.
	Raw        RawEvents `json:"raw, omitempty"`
}

type RawEvents struct {
	// List of perf events to be measured. RawEvents in this list
	// will not be grouped. See group_fd argument documentation
	// at man perf_event_open.
	NonGrouped []RawEvent `json:"non_grouped"`

	// List of groups of events to be measured. See group_fd
	// argument documentation at man perf_event_open.
	Grouped [][]RawEvent `json:"grouped"`

	// Interval between each measurement.
	Interval Duration `json:"interval"`
}

type RawEvent struct {
	// Type of the event. See perf_event_attr documentation
	// at man perf_event_open.
	Type uint32 `json:"type"`

	// Symbolically formed event like:
	// pmu/config=PerfEvent.Config[0],config1=PerfEvent.Config[1],config2=PerfEvent.Config[2]
	// as described in man perf-stat.
	Config Config `json:"config"`

	// Human readable name of metric that will be created from the event.
	Name string `json:"name"`
}

type Config []uint64

func (c *Config) UnmarshalJSON(b []byte) error {
	config := []string{}
	err := json.Unmarshal(b, &config)
	if err != nil {
		klog.Errorf("Unmarshalling %s into slice of strings failed: %q", b, err)
		return fmt.Errorf("unmarshalling %s into slice of strings failed: %q", b, err)
	}
	intermediate := []uint64{}
	for _, v := range config {
		uintValue, err := strconv.ParseUint(v, 0, 64)
		if err != nil {
			klog.Errorf("Parsing %#v into uint64 failed: %q", v, err)
			return fmt.Errorf("parsing %#v into uint64 failed: %q", v, err)
		}
		intermediate = append(intermediate, uintValue)
	}
	*c = intermediate
	return nil
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var stripped string
	err := json.Unmarshal(b, &stripped)
	if err != nil {
		klog.Errorf("Unmarshalling %#v into string failed: %q", b, err)
		return fmt.Errorf("unmarshalling %#v into string failed: %q", b, err)
	}
	duration, err := time.ParseDuration(stripped)
	if err != nil {
		klog.Errorf("Parsing %q into time.Duration failed: %q", stripped, err)
		return fmt.Errorf("parsing %q into time.Duration failed: %q", stripped, err)
	}
	*d = Duration(duration)
	return nil
}
