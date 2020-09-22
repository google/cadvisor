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

package perf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigParsing(t *testing.T) {
	file, err := os.Open("testing/perf.json")
	assert.Nil(t, err)
	defer file.Close()

	events, err := parseConfig(file)

	assert.Nil(t, err)
	assert.Len(t, events.Core.Events, 2)
	assert.Len(t, events.Core.Events[0].events, 2)
	assert.Equal(t, true, events.Core.Events[0].array)
	assert.Equal(t, Event("instructions"), events.Core.Events[0].events[0])
	assert.Equal(t, Event("instructions_retired"), events.Core.Events[0].events[1])
	assert.Len(t, events.Core.Events[1].events, 1)
	assert.Equal(t, false, events.Core.Events[1].array)
	assert.Equal(t, Event("cycles"), events.Core.Events[1].events[0])

	assert.Len(t, events.Uncore.Events, 3)
	assert.Equal(t, Event("cas_count_write"), events.Uncore.Events[0].events[0])
	assert.Equal(t, Event("uncore_imc_0/UNC_M_CAS_COUNT:RD"), events.Uncore.Events[1].events[0])
	assert.Equal(t, Event("uncore_ubox/UNC_U_EVENT_MSG"), events.Uncore.Events[2].events[0])

	assert.Len(t, events.Uncore.CustomEvents, 1)
	assert.Equal(t, Config{0x5300}, events.Uncore.CustomEvents[0].Config)
	assert.Equal(t, uint32(0x12), events.Uncore.CustomEvents[0].Type)
	assert.Equal(t, Event("cas_count_write"), events.Uncore.CustomEvents[0].Name)

}
