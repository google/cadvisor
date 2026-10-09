// Copyright 2026 Google Inc. All Rights Reserved.
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

package influxdb

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the token-authenticated InfluxDB 2.x / InfluxDB 3 Core write
// path. Unlike influxdb_test.go (build tag influxdb_test), these tests do not
// require a running InfluxDB instance and run as part of the default
// `go test ./...` invocation.

func tokenModeTestStats() (*info.ContainerInfo, *info.ContainerStats) {
	cInfo := &info.ContainerInfo{
		ContainerReference: info.ContainerReference{
			Name:    "testContainername",
			Aliases: []string{"testContainerAlias"},
		},
	}
	stats := &info.ContainerStats{
		Timestamp: time.Unix(1700000000, 0).UTC(),
		Cpu: &info.CpuStats{
			Usage: info.CpuUsage{Total: 100, User: 40, System: 60},
		},
		Memory: &info.MemoryStats{Usage: 2048, WorkingSet: 1024},
		Network: &info.NetworkStats{
			InterfaceStats: info.InterfaceStats{
				RxBytes:  11,
				RxErrors: 1,
				TxBytes:  22,
				TxErrors: 2,
			},
		},
	}
	return cInfo, stats
}

func TestNewStorageLegacyModeWithoutAuthToken(t *testing.T) {
	storage, err := newStorage("machineA",
		"cadvisor_table",
		"cadvisor_test",
		"cadvisor_test_rp",
		"root",
		"root",
		"",
		"",
		"",
		"localhost:8086",
		false,
		time.Minute)
	require.NoError(t, err)
	defer storage.Close()

	assert.NotNil(t, storage.client)
	assert.Nil(t, storage.v2Client)
	assert.Nil(t, storage.writeAPI)
}

func TestNewStorageTokenMode(t *testing.T) {
	storage, err := newStorage("machineA",
		"cadvisor_table",
		"cadvisor_test",
		"cadvisor_test_rp",
		"root",
		"root",
		"my-token",
		"my-org",
		"",
		"localhost:8086",
		true,
		time.Minute)
	require.NoError(t, err)
	defer storage.Close()

	assert.Nil(t, storage.client)
	assert.NotNil(t, storage.v2Client)
	assert.NotNil(t, storage.writeAPI)
	// The bucket falls back to the database name when unset.
	assert.Equal(t, "cadvisor_test", storage.bucket)
}

func TestNewStorageTokenModeExplicitBucket(t *testing.T) {
	storage, err := newStorage("machineA",
		"cadvisor_table",
		"cadvisor_test",
		"cadvisor_test_rp",
		"root",
		"root",
		"my-token",
		"my-org",
		"my-bucket",
		"localhost:8086",
		false,
		time.Minute)
	require.NoError(t, err)
	defer storage.Close()

	assert.Equal(t, "my-bucket", storage.bucket)
}

func TestAddStatsWithAuthTokenWritesToV2API(t *testing.T) {
	var gotPath, gotAuthHeader, gotOrg, gotBucket, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthHeader = r.Header.Get("Authorization")
		gotOrg = r.URL.Query().Get("org")
		gotBucket = r.URL.Query().Get("bucket")
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	storage, err := newStorage("machineA",
		"cadvisor_table",
		"cadvisor_db",
		"",
		"ignored-user",
		"ignored-password",
		"my-token",
		"my-org",
		"",
		strings.TrimPrefix(server.URL, "http://"),
		false,
		0)
	require.NoError(t, err)
	defer storage.Close()

	cInfo, stats := tokenModeTestStats()
	require.NoError(t, storage.AddStats(cInfo, stats))

	assert.Equal(t, "/api/v2/write", gotPath)
	assert.Equal(t, "Token my-token", gotAuthHeader)
	assert.Equal(t, "my-org", gotOrg)
	// The bucket falls back to the -storage_driver_db value.
	assert.Equal(t, "cadvisor_db", gotBucket)

	// The batch is written as line protocol with the cAdvisor tags attached.
	assert.Contains(t, gotBody, "cpu_usage_total")
	assert.Contains(t, gotBody, "memory_usage")
	assert.Contains(t, gotBody, "rx_bytes")
	assert.Contains(t, gotBody, "machine=machineA")
	assert.Contains(t, gotBody, "container_name=testContainerAlias")
}

func TestAddStatsWithAuthTokenReturnsWriteErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"code":"unauthorized","message":"Unauthorized"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	storage, err := newStorage("machineA",
		"cadvisor_table",
		"cadvisor_db",
		"",
		"ignored-user",
		"ignored-password",
		"invalid-token",
		"my-org",
		"",
		strings.TrimPrefix(server.URL, "http://"),
		false,
		0)
	require.NoError(t, err)
	defer storage.Close()

	cInfo, stats := tokenModeTestStats()
	err = storage.AddStats(cInfo, stats)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write stats to influxDb")
}
