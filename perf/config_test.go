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
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfigParsing(t *testing.T) {
	file, err := os.Open("testing/perf.json")
	assert.Nil(t, err)
	defer file.Close()

	events, err := parseConfig(file)

	assert.Nil(t, err)
	assert.Len(t, events.Events, 2)
	assert.Len(t, events.Events[0], 1)
	assert.Equal(t, Event("instructions"), events.Events[0][0])
	assert.Len(t, events.Events[1], 1)
	assert.Equal(t, Event("instructions_retired"), events.Events[1][0])

	assert.Len(t, events.CustomEvents, 1)
	assert.Equal(t, Config{5439680}, events.CustomEvents[0].Config)
	assert.Equal(t, uint32(4), events.CustomEvents[0].Type)
	assert.Equal(t, Event("instructions_retired"), events.CustomEvents[0].Name)
}
