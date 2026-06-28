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

package http

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/manager"
)

// mockManager implements a minimal manager.Manager for testing.
type mockManager struct{}

func (m *mockManager) Start() error { return nil }
func (m *mockManager) Stop() error  { return nil }

// TestAPIRequiresAuthWhenConfigured verifies that /api/ routes require authentication
// when an auth file is configured.
func TestAPIRequiresAuthWhenConfigured(t *testing.T) {
	// Create a temporary htpasswd file with test credentials (user:pass = test:test)
	// Format: username:hashed_password (using htpasswd -c)
	// "test" hashed with default crypt: $apr1$6RXm1v7P$G3XC9xGP1MaQrV/Y.OVIB1 (example hash)
	tmpFile, err := ioutil.TempFile("", "htpasswd")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write a simple htpasswd entry (user:password with plaintext for testing)
	// Actually, go-http-auth reads htpasswd format. We'll use a basic entry.
	// This is user "testuser" with password "testpass" (apache format: user:$apr1$...$...)
	// For simplicity, we create a file that the library can parse.
	content := "testuser:$apr1$6RXm1v7P$G3XC9xGP1MaQrV/Y.OVIB1\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create a test mux
	mux := http.NewServeMux()
	mockMgr := &mockManager{}

	// Register handlers with auth file
	err = RegisterHandlers(mux, mockMgr, tmpFile.Name(), "TestRealm", "", "", "")
	if err != nil {
		t.Fatalf("Failed to register handlers: %v", err)
	}

	// Create a test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// Test 1: Request to /api/ without credentials should return 401
	resp, err := http.Get(server.URL + "/api/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /api/ without auth: expected %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	// Test 2: Health check should NOT require auth
	resp, err = http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("GET /healthz should not require auth, got %d", resp.StatusCode)
	}
}

// TestPrometheusRequiresAuthWhenConfigured verifies that Prometheus endpoint
// requires authentication when an auth file is configured.
func TestPrometheusRequiresAuthWhenConfigured(t *testing.T) {
	// Create a temporary htpasswd file
	tmpFile, err := ioutil.TempFile("", "htpasswd")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := "testuser:$apr1$6RXm1v7P$G3XC9xGP1MaQrV/Y.OVIB1\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Create a test mux
	mux := http.NewServeMux()
	mockMgr := &mockManager{}

	prometheusEndpoint := "/metrics"
	labelFunc := func(containerName string) map[string]string { return map[string]string{} }
	includedMetrics := container.MetricSet{}

	// Register Prometheus handler with auth
	RegisterPrometheusHandler(mux, mockMgr, prometheusEndpoint, labelFunc, includedMetrics,
		tmpFile.Name(), "TestRealm", "", "")

	// Create a test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// Request to /metrics without credentials should return 401
	resp, err := http.Get(server.URL + prometheusEndpoint)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET %s without auth: expected %d, got %d", prometheusEndpoint, http.StatusUnauthorized, resp.StatusCode)
	}
}

// TestHandlersWithoutAuth verifies that API and Prometheus handlers work without auth.
func TestHandlersWithoutAuth(t *testing.T) {
	// Create a test mux without auth
	mux := http.NewServeMux()
	mockMgr := &mockManager{}

	// Register handlers WITHOUT auth file
	err := RegisterHandlers(mux, mockMgr, "", "", "", "", "")
	if err != nil {
		t.Fatalf("Failed to register handlers: %v", err)
	}

	prometheusEndpoint := "/metrics"
	labelFunc := func(containerName string) map[string]string { return map[string]string{} }
	includedMetrics := container.MetricSet{}

	// Register Prometheus handler WITHOUT auth
	RegisterPrometheusHandler(mux, mockMgr, prometheusEndpoint, labelFunc, includedMetrics, "", "", "", "")

	// Create a test server
	server := httptest.NewServer(mux)
	defer server.Close()

	// Request to /api/ without auth configured should NOT return 401
	resp, err := http.Get(server.URL + "/api/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("GET /api/ without auth configured should not return 401, got %d", resp.StatusCode)
	}

	// Request to /metrics without auth configured should NOT return 401
	resp, err = http.Get(server.URL + prometheusEndpoint)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("GET %s without auth configured should not return 401, got %d", prometheusEndpoint, resp.StatusCode)
	}
}
