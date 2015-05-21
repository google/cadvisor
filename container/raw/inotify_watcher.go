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

package raw

import (
	"sync"

	"golang.org/x/exp/inotify"
)

// Watcher for container-related inotify events in the cgroup hierarchy.
//
// Implementation is thread-safe.
type InotifyWatcher struct {
	// Underlying inotify watcher.
	watcher *inotify.Watcher

	// Containers being watched.
	containersWatched map[string]bool

	// Full cgroup paths being watched.
	cgroupsWatched map[string]bool

	// Lock for all datastructure access.
	lock sync.Mutex
}

func NewInotifyWatcher() (*InotifyWatcher, error) {
	w, err := inotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &InotifyWatcher{
		watcher:           w,
		containersWatched: make(map[string]bool),
		cgroupsWatched:    make(map[string]bool),
	}, nil
}

// Add a watch to the specified directory. Returns if the container was already being watched.
func (iw *InotifyWatcher) AddWatch(containerName, dir string) (bool, error) {
	iw.lock.Lock()
	defer iw.lock.Unlock()

	alreadyWatched := iw.containersWatched[containerName]

	// Register an inotify notification.
	if !iw.cgroupsWatched[dir] {
		err := iw.watcher.AddWatch(dir, inotify.IN_CREATE|inotify.IN_DELETE|inotify.IN_MOVE)
		if err != nil {
			return alreadyWatched, err
		}
		iw.cgroupsWatched[dir] = true
	}

	// Record our watching of the container.
	if !alreadyWatched {
		iw.containersWatched[containerName] = true
	}
	return alreadyWatched, nil
}

// Remove watch from the specified directory. Returns if the container was already being watched.
func (iw *InotifyWatcher) RemoveWatch(containerName, dir string) (bool, error) {
	iw.lock.Lock()
	defer iw.lock.Unlock()

	alreadyWatched := iw.containersWatched[containerName]

	// Remove the inotify watch if it exists.
	if iw.cgroupsWatched[dir] {
		err := iw.watcher.RemoveWatch(dir)
		if err != nil {
			return alreadyWatched, nil
		}
		delete(iw.cgroupsWatched, dir)
	}

	// Record the container as no longer being watched.
	if alreadyWatched {
		delete(iw.containersWatched, containerName)
	}

	return alreadyWatched, nil
}

// Errors are returned on this channel.
func (iw *InotifyWatcher) Error() chan error {
	return iw.watcher.Error
}

// Events are returned on this channel.
func (iw *InotifyWatcher) Event() chan *inotify.Event {
	return iw.watcher.Event
}

// Closes the inotify watcher.
func (iw *InotifyWatcher) Close() error {
	return iw.watcher.Close()
}

// Returns a list of:
// - Containers being watched.
// - Cgroup paths being watched.
func (iw *InotifyWatcher) GetWatches() ([]string, []string) {
	return mapToSlice(iw.containersWatched), mapToSlice(iw.cgroupsWatched)
}

func mapToSlice(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
