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
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestStringToUint64Unmarshaling(t *testing.T) {
	configContents, err := ioutil.ReadFile("testing/perf.json")
	assert.Nil(t, err)

	events := &RawEvents{}
	err = json.Unmarshal(configContents, events)

	assert.Nil(t, err)
	assert.Equal(t, Duration(5*time.Second), events.Interval)
	assert.Equal(t, events.NonGrouped[0].Config, Config{192})
	assert.Equal(t, events.NonGrouped[0].Type, uint32(4))
	assert.Equal(t, events.NonGrouped[0].Name, "instructions_retired")
	assert.Equal(t, events.Grouped[0][0].Config, Config{3076})
	assert.Equal(t, events.Grouped[0][0].Type, uint32(13))
	assert.Equal(t, events.Grouped[0][0].Name, "UNC_M_CAS_COUNT_WRITE")
	assert.Equal(t, events.Grouped[0][1].Config, Config{49924})
	assert.Equal(t, events.Grouped[0][1].Type, uint32(13))
	assert.Equal(t, events.Grouped[0][1].Name, "UNC_M_CAS_COUNT_READ")
}
