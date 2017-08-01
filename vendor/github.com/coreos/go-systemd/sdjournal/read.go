// Copyright 2015 RedHat, Inc.
// Copyright 2015 CoreOS, Inc.
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

package sdjournal

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"
)

var (
	ErrExpired = errors.New("Timeout expired")
)

// JournalReaderConfig represents options to drive the behavior of a JournalReader.
type JournalReaderConfig struct {
	// The Since, NumFromTail and Cursor options are mutually exclusive and
	// determine where the reading begins within the journal. The order in which
	// options are written is exactly the order of precedence.
	Since       time.Duration // start relative to a Duration from now
	NumFromTail uint64        // start relative to the tail
	Cursor      string        // start relative to the cursor

	// Show only journal entries whose fields match the supplied values. If
	// the array is empty, entries will not be filtered.
	Matches []Match

	// If not empty, the journal instance will point to a journal residing
	// in this directory. The supplied path may be relative or absolute.
	Path string
}

// JournalReader is an io.ReadCloser which provides a simple interface for iterating through the
// systemd journal.
type JournalReader struct {
	journal *Journal
}

// NewJournalReader creates a new JournalReader with configuration options that are similar to the
// systemd journalctl tool's iteration and filtering features.
func NewJournalReader(config JournalReaderConfig) (*JournalReader, error) {
	r := &JournalReader{}

	// Open the journal
	var err error
	if config.Path != "" {
		r.journal, err = NewJournalFromDir(config.Path)
	} else {
		r.journal, err = NewJournal()
	}
	if err != nil {
		return nil, err
	}

	// Add any supplied matches
	for _, m := range config.Matches {
		r.journal.AddMatch(m.String())
	}

	// Set the start position based on options
	if config.Since != 0 {
		// Start based on a relative time
		start := time.Now().Add(config.Since)
		if err := r.journal.SeekRealtimeUsec(uint64(start.UnixNano() / 1000)); err != nil {
			return nil, err
		}
	} else if config.NumFromTail != 0 {
		// Start based on a number of lines before the tail
		if err := r.journal.SeekTail(); err != nil {
			return nil, err
		}

		// Move the read pointer into position near the tail. Go one further than
		// the option so that the initial cursor advancement positions us at the
		// correct starting point.
		if _, err := r.journal.PreviousSkip(config.NumFromTail + 1); err != nil {
			return nil, err
		}
	} else if config.Cursor != "" {
		// Start based on a custom cursor
		if err := r.journal.SeekCursor(config.Cursor); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *JournalReader) Read(b []byte) (int, error) {
	var err error
	var c int

	// Advance the journal cursor
	c, err = r.journal.Next()

	// An unexpected error
	if err != nil {
		return 0, err
	}

	// EOF detection
	if c == 0 {
		return 0, io.EOF
	}

	// Build a message
	var msg string
	msg, err = r.buildMessage()

	if err != nil {
		return 0, err
	}

	// Copy and return the message
	copy(b, []byte(msg))

	return len(msg), nil
}

// Close closes the JournalReader's handle to the journal.
func (r *JournalReader) Close() error {
	return r.journal.Close()
}

// Rewind attempts to rewind the JournalReader to the first entry.
func (r *JournalReader) Rewind() error {
	return r.journal.SeekHead()
}

// Follow synchronously follows the JournalReader, writing each new journal entry to writer. The
// follow will continue until a single time.Time is received on the until channel.
func (r *JournalReader) Follow(until <-chan time.Time, writer io.Writer) (err error) {

	// Process journal entries and events. Entries are flushed until the tail or
	// timeout is reached, and then we wait for new events or the timeout.
	var msg = make([]byte, 64*1<<(10))
process:
	for {
		c, err := r.Read(msg)
		if err != nil && err != io.EOF {
			break process
		}

		select {
		case <-until:
			return ErrExpired
		default:
			if c > 0 {
				writer.Write(msg[:c])
				continue process
			}
		}

		// We're at the tail, so wait for new events or time out.
		// Holds journal events to process. Tightly bounded for now unless there's a
		// reason to unblock the journal watch routine more quickly.
		events := make(chan int, 1)
		pollDone := make(chan bool, 1)
		go func() {
			for {
				select {
				case <-pollDone:
					return
				default:
					events <- r.journal.Wait(time.Duration(1) * time.Second)
				}
			}
		}()

		select {
		case <-until:
			pollDone <- true
			return ErrExpired
		case e := <-events:
			pollDone <- true
			switch e {
			case SD_JOURNAL_NOP, SD_JOURNAL_APPEND, SD_JOURNAL_INVALIDATE:
				// TODO: need to account for any of these?
			default:
				log.Printf("Received unknown event: %d\n", e)
			}
			continue process
		}
	}

	return
}

// buildMessage returns a string representing the current journal entry in a simple format which
// includes the entry timestamp and MESSAGE field.
func (r *JournalReader) buildMessage() (string, error) {
	var msg string
	var usec uint64
	var err error

	if msg, err = r.journal.GetData("MESSAGE"); err != nil {
		return "", err
	}

	if usec, err = r.journal.GetRealtimeUsec(); err != nil {
		return "", err
	}

	timestamp := time.Unix(0, int64(usec)*int64(time.Microsecond))

	return fmt.Sprintf("%s %s\n", timestamp, msg), nil
}
