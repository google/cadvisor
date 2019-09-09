// +build linux

package libcontainer

import (
	"reflect"
	"testing"
)

var states = map[containerState]Status{
	&createdState{}:          Created,
	&runningState{}:          Running,
	&restoredState{}:         Running,
	&pausedState{}:           Paused,
	&stoppedState{}:          Stopped,
	&loadedState{s: Running}: Running,
}

func TestStateStatus(t *testing.T) {
	for s, status := range states {
		if s.status() != status {
			t.Fatalf("state returned %s but expected %s", s.status(), status)
		}
	}
}

func isStateTransitionError(err error) bool {
	_, ok := err.(*stateTransitionError)
	return ok
}

func testTransitions(t *testing.T, initialState containerState, valid []containerState) {
	validMap := map[reflect.Type]interface{}{}
	for _, validState := range valid {
		validMap[reflect.TypeOf(validState)] = nil
		t.Run(validState.status().String(), func(t *testing.T) {
			if err := initialState.transition(validState); err != nil {
				t.Fatal(err)
			}
		})
	}
	for state := range states {
		if _, ok := validMap[reflect.TypeOf(state)]; ok {
			continue
		}
		t.Run(state.status().String(), func(t *testing.T) {
			err := initialState.transition(state)
			if err == nil {
				t.Fatal("transition should fail")
			}
			if !isStateTransitionError(err) {
				t.Fatal("expected stateTransitionError")
			}
		})
	}
}

func TestStoppedStateTransition(t *testing.T) {
	testTransitions(
		t,
		&stoppedState{c: &linuxContainer{}},
		[]containerState{
			&stoppedState{},
			&runningState{},
			&restoredState{},
		},
	)
}

func TestPausedStateTransition(t *testing.T) {
	testTransitions(
		t,
		&pausedState{c: &linuxContainer{}},
		[]containerState{
			&pausedState{},
			&runningState{},
			&stoppedState{},
		},
	)
}

func TestRestoredStateTransition(t *testing.T) {
	testTransitions(
		t,
		&restoredState{c: &linuxContainer{}},
		[]containerState{
			&stoppedState{},
			&runningState{},
		},
	)
}

func TestRunningStateTransition(t *testing.T) {
	testTransitions(
		t,
		&runningState{c: &linuxContainer{}},
		[]containerState{
			&stoppedState{},
			&pausedState{},
			&runningState{},
		},
	)
}

func TestCreatedStateTransition(t *testing.T) {
	testTransitions(
		t,
		&createdState{c: &linuxContainer{}},
		[]containerState{
			&stoppedState{},
			&pausedState{},
			&runningState{},
			&createdState{},
		},
	)
}
