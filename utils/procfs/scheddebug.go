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

package procfs

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/google/cadvisor/utils/fs"
)

type SchedulerLoadReader interface {
	// Load() returns the load of each core of given container. If there's
	// no information of given container, it should return nil, nil. If the
	// returned load is not nil, the number of elements in the returned
	// slice must be the same as the number of cores on the machine.  Each
	// element in the returned slices represents number of tasks/threads in
	// the container and waiting for the CPU, i.e. number of runnable
	// tasks.
	Load(container string) ([]int, error)

	AllContainers() ([]string, error)
}

func NewSchedulerLoadReader() (SchedulerLoadReader, error) {
	schedDebug, err := fs.Open("/proc/sched_debug")
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(schedDebug)
	stateMachine := newSchedDebugReader()
	for scanner.Scan() {
		line := scanner.Text()
		err = stateMachine.ProcessLine(line)
		if err != nil {
			return nil, err
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return stateMachine.Load()
}

type schedDebugReaderStateMachine struct {
	currentState schedDebugReaderState
	context      *schedDebugContext
}

func newSchedDebugReader() *schedDebugReaderStateMachine {
	return &schedDebugReaderStateMachine{
		currentState: &schedDebugReaderStateReadingVersion{},
		context: &schedDebugContext{
			loadMap: make(map[string][]int, 8),
		},
	}
}

func (self *schedDebugReaderStateMachine) ProcessLine(line string) error {
	nextState, err := self.currentState.Transit(self.context, line)
	if err != nil {
		return err
	}
	self.currentState = nextState
	return nil
}

func (self *schedDebugReaderStateMachine) Load() (SchedulerLoadReader, error) {
	return self.context.loadMap, nil
}

type schedDebugContext struct {
	loadMap  simpleSchedulerLoadReader
	numCores int
}

// Key: container name
// Value: number of runnable processes in each core
type simpleSchedulerLoadReader map[string][]int

func (self simpleSchedulerLoadReader) Load(container string) ([]int, error) {
	if load, ok := self[container]; ok {
		return load, nil
	}
	return nil, nil
}

func (self simpleSchedulerLoadReader) AllContainers() ([]string, error) {
	ret := make([]string, 0, len(self))
	for c := range self {
		ret = append(ret, c)
	}
	return ret, nil
}

// This interface is used to represent a state when reading sched_debug. The
// reader of sched_debug is a state machine and will transit to another state
// (and change its behavior accordingly) when a certain line is read.
// The state machine is implemented using the State Pattern.
type schedDebugReaderState interface {
	// Transit() consumes a line from sched_debug and returns a new state
	// for the next line. It may change the context accordingly.
	Transit(context *schedDebugContext, line string) (schedDebugReaderState, error)
}

// State: Reading version
// In this state, the state machine is waiting for the version of the Sched Debug message.
// If the version is supported, it will transit to WaitingHeader state.
// Otherwise, it will report error
type schedDebugReaderStateReadingVersion struct {
}

func (self *schedDebugReaderStateReadingVersion) Transit(context *schedDebugContext, line string) (schedDebugReaderState, error) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "Sched Debug Version") {
		if !strings.HasPrefix(line, "Sched Debug Version: v0.11") {
			return nil, fmt.Errorf("unsupported sched_debug version: %v", line)
		}
		return &schedDebugReaderStateWaitingHeader{}, nil
	}
	return self, nil
}

// State: WaitingHeader
// In this state the state machine is waiting for the header of the stats table.
// It will transit to ReadingTask state once it received a header.
type schedDebugReaderStateWaitingHeader struct {
}

func (self *schedDebugReaderStateWaitingHeader) isSeparator(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return false
	}
	for _, ch := range line {
		if ch != '-' {
			return false
		}
	}
	return true
}

func (self *schedDebugReaderStateWaitingHeader) Transit(context *schedDebugContext, line string) (schedDebugReaderState, error) {
	if self.isSeparator(line) {
		// A new table of runnable tasks
		context.numCores++
		loadMap := make(map[string][]int, len(context.loadMap))
		for container, loads := range context.loadMap {
			for len(loads) < context.numCores {
				loads = append(loads, 0)
			}
			loadMap[container] = loads
		}
		context.loadMap = loadMap
		ret := &schedDebugReaderStateReadingTasks{}
		return ret, nil
	}
	return self, nil
}

// State: ReadingTasks
// In this state, the state machine expects that each line contains a stats info of a thread.
// It will transit to WaitingHeader state once it reads an empty line.
type schedDebugReaderStateReadingTasks struct {
}

func (self *schedDebugReaderStateReadingTasks) Transit(context *schedDebugContext, line string) (schedDebugReaderState, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		// End of a table. Waiting for another table.
		ret := &schedDebugReaderStateWaitingHeader{}
		return ret, nil
	}
	fields := strings.Fields(line)

	// Example of one row:
	// rcuos/2    10 153908711.511953   9697302   120 153908711.511953    462044.084299 4409304806.819551 0 /

	container := fields[len(fields)-1]
	var loads []int
	ok := false
	if loads, ok = context.loadMap[container]; !ok {
		loads = make([]int, context.numCores)
	}
	for len(loads) < context.numCores {
		loads = append(loads, 0)
	}
	loads[len(loads)-1]++
	context.loadMap[container] = loads
	return self, nil
}
