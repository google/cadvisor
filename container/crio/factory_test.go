// Copyright 2017 Google Inc. All Rights Reserved.
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

package crio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type canHandleAndAccept struct {
	canHandle bool
	canAccept bool
}

func TestCanHandleAndAccept(t *testing.T) {
	as := assert.New(t)
	f := &crioFactory{
		client:             nil,
		cgroupSubsystems:   nil,
		fsInfo:             nil,
		machineInfoFactory: nil,
		storageDriver:      "",
		storageDir:         "",
		includedMetrics:    nil,
	}
	for k, v := range map[string]canHandleAndAccept{
		"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f":           {true, false},
		"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.scope":     {true, true},
		"/system.slice/system-systemd\\\\x2dcoredump.slice":                                                                                 {true, false},
		"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f.mount":     {false, false},
		"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-conmon-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f":    {false, false},
		"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/no-crio-conmon-81e5c2990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75d5f": {false, false},
		"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/crio-990803c383229c9680ce964738d5e566d97f5bd436ac34808d2ec75":                    {false, false},
	} {
		b1, b2, err := f.CanHandleAndAccept(k)
		as.Nil(err)
		as.Equal(b1, v.canHandle)
		as.Equal(b2, v.canAccept)
	}
}
