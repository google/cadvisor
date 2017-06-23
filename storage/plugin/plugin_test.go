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

package plugin

import (
	"flag"
	"testing"
	"time"

	"github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage/plugin/client"
	"github.com/google/cadvisor/storage/plugin/server"
	"github.com/stretchr/testify/assert"
)

func TestClientServer(t *testing.T) {
	statsChan := make(chan v1.ContainerInfo, 2)
	driver := &testDriver{statsChan}

	const socket = "cadvisor-test.sock"
	flag.Set("storage_driver_socket", socket)

	s, err := server.Start(socket, driver)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, s.Close())
	}()

	client, err := client.NewClient()
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, client.Close())
	}()

	ref := v1.ContainerReference{
		Id:      "12345",
		Name:    "test",
		Aliases: []string{"test-alias"},
	}
	stats := v1.ContainerStats{
		Timestamp: time.Now(),
		Cpu: v1.CpuStats{
			LoadAverage: 12,
		},
	}

	// Send the stats.
	assert.NoError(t, client.AddStats(ref, &stats))

	const timeout = 30 * time.Second
	select {
	case v := <-statsChan:
		assert.Equal(t, ref, v.ContainerReference, "Received ContainerReference")
		assert.Len(t, v.Stats, 1)
		assert.Equal(t, stats, *v.Stats[0], "Received ContainerStats")
	case <-time.After(timeout):
		assert.Fail(t, "Timed out after %s", timeout.String())
	}
}

type testDriver struct {
	statsChan chan v1.ContainerInfo
}

func (d *testDriver) AddStats(ref v1.ContainerReference, stats *v1.ContainerStats) error {
	d.statsChan <- v1.ContainerInfo{
		ContainerReference: ref,
		Stats:              []*v1.ContainerStats{stats},
	}
	return nil
}

func (d *testDriver) Close() error {
	return nil
}
