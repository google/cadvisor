// Copyright 2016 Google Inc. All Rights Reserved.
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

// Package tail implements "tail -F" functionality following rotated logs
package tail

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"golang.org/x/exp/inotify"
)

const (
	readerStateOpening = 1 << iota
	readerStateOpened
	readerStateError
)

type Tail struct {
	reader      *bufio.Reader
	readerState int
	readerLock  sync.Mutex
	filename    string
	file        *os.File
	stop        chan bool
	watcher     *inotify.Watcher
}

const retryOpenInterval = time.Second
const maxOpenAttempts = 3

// NewTail starts opens the given file and watches it for deletion/rotation
func NewTail(filename string) (*Tail, error) {
	t := &Tail{
		filename: filename,
	}
	var err error
	t.stop = make(chan bool)
	t.watcher, err = inotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("inotify init failed on %s: %v", t.filename, err)
	}
	t.readerState = readerStateOpening
	go t.watchLoop()
	return t, nil
}

// Read implements the io.Reader interface for Tail
func (t *Tail) Read(p []byte) (int, error) {
	t.readerLock.Lock()
	defer t.readerLock.Unlock()
	if t.reader == nil {
		if t.readerState == readerStateOpening {
			return 0, nil
		}
		return 0, fmt.Errorf("can't open log file %s", t.filename)
	}
	return t.reader.Read(p)
}

var _ io.Reader = &Tail{}

// Close stops watching and closes the file
func (t *Tail) Close() {
	close(t.stop)
}

func isEvent(event *inotify.Event, flag uint32) bool {
	return event.Mask&flag == flag
}

func (t *Tail) fileChanged() error {
	for {
		select {
		case event := <-t.watcher.Event:
			// We don't get IN_DELETE because we are holding the file open
			if isEvent(event, inotify.IN_ATTRIB) || isEvent(event, inotify.IN_MOVE_SELF) {
				return nil
			}
		case <-t.stop:
			return fmt.Errorf("watch was cancelled")
		}
	}
}

func (t *Tail) attemptOpen() (err error) {
	t.readerLock.Lock()
	defer t.readerLock.Unlock()
	for attempt := 1; attempt <= maxOpenAttempts; attempt++ {
		glog.V(4).Infof("Opening %s (attempt %d of %d)", t.filename, attempt, maxOpenAttempts)
		t.file, err = os.Open(t.filename)
		if err == nil {
			// TODO: not interested in old events?
			//t.file.Seek(0, os.SEEK_END)
			t.reader = bufio.NewReader(t.file)
			t.readerState = readerStateOpened
			return nil
		}
		select {
		case <-time.After(retryOpenInterval):
		case <-t.stop:
			t.readerState = readerStateError
			return fmt.Errorf("watch was cancelled")
		}
	}
	t.readerState = readerStateError
	return err
}

func (t *Tail) watchLoop() {
	for {
		err := t.watchFile()
		if err != nil {
			glog.Errorf("Tail failed on %s: %v", t.filename, err)
			break
		}
	}
}

func (t *Tail) watchFile() error {
	err := t.attemptOpen()
	if err != nil {
		return err
	}
	defer t.file.Close()
	err = t.watcher.Watch(t.filename)
	if err != nil {
		return err
	}
	defer t.watcher.RemoveWatch(t.filename)
	err = t.fileChanged()
	if err != nil {
		return err
	}
	glog.V(4).Infof("Log file %s moved/deleted", t.filename)
	t.readerLock.Lock()
	defer t.readerLock.Unlock()
	t.readerState = readerStateOpening
	return nil
}
