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
)

func TestStringToUint64Unmarshaling(t *testing.T) {
	configContents, err := ioutil.ReadFile("testing/perf.json")
	assert.Nil(t, err)

	events := &Events{}
	err = json.Unmarshal(configContents, events)

	assert.Nil(t, err)
	assert.Equal(t, events.Raw.NonGrouped[0].Config, Config{5439680})
	assert.Equal(t, events.Raw.NonGrouped[0].Type, uint32(4))
	assert.Equal(t, events.Raw.NonGrouped[0].Name, "instructions_retired")
}
