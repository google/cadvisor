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

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	assert := assert.New(t)

	//Create an nginx collector using the config file 'sample_config.json'
	collector, err := NewCollector("nginx", "config/sample_config.json")
	assert.NoError(err)
	assert.Equal(collector.name, "nginx")
	assert.Equal(collector.configFile.Endpoint, "host:port/nginx_status")
	assert.Equal(collector.configFile.MetricsConfig[0].Name, "activeConnections")
}
