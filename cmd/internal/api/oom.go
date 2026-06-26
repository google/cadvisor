// Copyright 2026 Google Inc. All Rights Reserved.
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

package api

import (
	"sync"

	"github.com/google/cadvisor/events"
	"github.com/google/cadvisor/lib/manager"
	info "github.com/google/cadvisor/lib/model"
	"github.com/google/cadvisor/utils/oomparser"

	"k8s.io/klog/v2"
)

// oomCounts holds the cumulative per-container OOM-kill count. The OOM watcher
// writes it and the prometheus collector reads it through oomInfoProvider to
// emit container_oom_events_total. The lean library manager no longer watches
// for OOMs, so the binary owns this (one manager per process, so a
// package-level value is enough).
var oomCounts = struct {
	sync.Mutex
	byContainer map[string]uint64
}{byContainer: map[string]uint64{}}

// startOOMWatcher reads OOM kills from the kernel log via oomparser and (a)
// emits OOM and OOM-kill events to the event manager and (b) maintains the
// per-container OOM counter. It is best-effort: if the kernel log cannot be
// read, OOM events and the metric are disabled with a warning.
func startOOMWatcher(em events.EventManager) {
	oomLog, err := oomparser.New()
	if err != nil {
		klog.Warningf("Could not configure a source for OOM detection, disabling OOM events: %v", err)
		return
	}
	outStream := make(chan *oomparser.OomInstance, 10)
	go oomLog.StreamOoms(outStream)

	go func() {
		for oomInstance := range outStream {
			_ = em.AddEvent(&info.Event{
				ContainerName: oomInstance.ContainerName,
				Timestamp:     oomInstance.TimeOfDeath,
				EventType:     info.EventOom,
			})
			_ = em.AddEvent(&info.Event{
				ContainerName: oomInstance.VictimContainerName,
				Timestamp:     oomInstance.TimeOfDeath,
				EventType:     info.EventOomKill,
				EventData: info.EventData{
					OomKill: &info.OomKillEventData{
						Pid:         oomInstance.Pid,
						ProcessName: oomInstance.ProcessName,
						Constraint:  oomInstance.Constraint,
					},
				},
			})

			oomCounts.Lock()
			oomCounts.byContainer[oomInstance.ContainerName]++
			oomCounts.Unlock()
		}
	}()
}

// WrapManagerForOOM wraps a manager so the prometheus collector reports the
// per-container OOM-kill count maintained by the OOM watcher. The lean library
// manager always reports zero for OOMEvents, so without this wrapper
// container_oom_events_total would be stuck at zero.
func WrapManagerForOOM(m manager.Manager) manager.Manager {
	return &oomInfoProvider{Manager: m}
}

type oomInfoProvider struct {
	manager.Manager
}

func (p *oomInfoProvider) GetRequestedContainersInfo(containerName string, options info.RequestOptions) (map[string]*info.ContainerInfo, error) {
	infos, err := p.Manager.GetRequestedContainersInfo(containerName, options)
	oomCounts.Lock()
	defer oomCounts.Unlock()
	for _, ci := range infos {
		count := oomCounts.byContainer[ci.Name]
		for _, stats := range ci.Stats {
			stats.OOMEvents = count
		}
	}
	return infos, err
}
