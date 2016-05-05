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

// Package container defines types for sub-container events and also
// defines an interface for container operation handlers.
package rkt

import (
	"path/filepath"
	"time"

	"github.com/google/cadvisor/container/rkt"
	"github.com/google/cadvisor/manager/watcher"

	"github.com/golang/glog"
	"golang.org/x/net/context"

	rktapi "github.com/coreos/rkt/api/v1alpha"
)

type rktContainerWatcher struct {
	// Signal for watcher thread to stop.
	stopWatcher chan error
}

func NewRawContainerWatcher() (watcher.ContainerWatcher, error) {
	watcher := &rktContainerWatcher{
		stopWatcher: make(chan error),
	}

	return watcher, nil
}

func (self *rktContainerWatcher) Start(events chan watcher.ContainerEvent) error {
	go self.detectRktContainers(events)

	return nil
}

func (self *rktContainerWatcher) Stop() error {
	// Rendezvous with the watcher thread.
	self.stopWatcher <- nil
	return <-self.stopWatcher
}

func (self *rktContainerWatcher) detectRktContainers(events chan watcher.ContainerEvent) {
	glog.Infof("starting detectRktContainers thread")
	ticker := time.Tick(30 * time.Second)
	curpods := make(map[string]*rktapi.Pod)

	for {
		select {
		case _ = <-ticker:
			glog.Infof("rkt ticker")
			pods, err := ListRunningPods()
			if err != nil {
				continue
			}

			newpods := make(map[string]*rktapi.Pod)
			for _, pod := range pods {
				newpods[pod.Id] = pod
				//if pods become mutable, have to handle this better
				if _, ok := curpods[pod.Id]; !ok {
					//should create all cgroups not including system.slice
					//i.e. /system.slice/rkt-test.service/ and /system.slice/rkt-test.service/system.slice/pause.service
					for _, cgroup := range PodToCgroup(pod) {
						glog.Infof("cgroup to upate = %v", cgroup)
						self.sendUpdateEvent(cgroup, events)
					}
				}
			}

			for id, pod := range curpods {
				if _, ok := newpods[id]; !ok {
					for _, cgroup := range PodToCgroup(pod) {
						glog.Infof("cgroup to delete = %v", cgroup)
						self.sendDestroyEvent(cgroup, events)
					}
				}
			}

			curpods = newpods

		case <-self.stopWatcher:
			self.stopWatcher <- nil
			glog.Infof("Exiting rktContainer Thread")
			return
		}
	}
}

func (self *rktContainerWatcher) sendUpdateEvent(cgroup string, events chan watcher.ContainerEvent) {
	events <- watcher.ContainerEvent{
		EventType:   watcher.ContainerOverride,
		Name:        cgroup,
		WatchSource: watcher.Rkt,
	}
}

func (self *rktContainerWatcher) sendDestroyEvent(cgroup string, events chan watcher.ContainerEvent) {
	events <- watcher.ContainerEvent{
		EventType:   watcher.ContainerDelete,
		Name:        cgroup,
		WatchSource: watcher.Rkt,
	}
}

func ListRunningPods() ([]*rktapi.Pod, error) {
	client, err := rkt.Client()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListPods(context.Background(), &rktapi.ListPodsRequest{
		// Specify the request: Fetch and print only running pods and their details.
		Detail: true,
		Filters: []*rktapi.PodFilter{
			{
				States: []rktapi.PodState{rktapi.PodState_POD_STATE_RUNNING},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return resp.Pods, nil
}

func PodToCgroup(pod *rktapi.Pod) []string {
	cgroups := make([]string, 0, 1+len(pod.Apps))

	baseCgroup := pod.Cgroup
	cgroups = append(cgroups, baseCgroup)

	for _, app := range pod.Apps {
		cgroups = append(cgroups, filepath.Join(baseCgroup, "system.slice", app.Name+".service"))
	}

	return cgroups
}
