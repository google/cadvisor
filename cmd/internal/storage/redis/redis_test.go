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

package redis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
)

func TestAddStatsAuthenticatesWhenPasswordConfigured(t *testing.T) {
	commands, address, stop := startFakeRedis(t)
	defer stop()

	driver, err := newStorage(
		"test-machine",
		"cadvisor",
		address,
		"redisPswrd",
		0,
	)
	if err != nil {
		t.Fatalf("newStorage() error = %v", err)
	}
	defer driver.Close()

	storage := driver.(*redisStorage)
	storage.readyToFlush = func() bool { return true }

	if err := storage.AddStats(
		&info.ContainerInfo{
			ContainerReference: info.ContainerReference{
				Name: "/docker/test-container",
			},
		},
		&info.ContainerStats{Timestamp: time.Unix(1, 0)},
	); err != nil {
		t.Fatalf("AddStats() error = %v", err)
	}

	assertCommand(t, <-commands, "AUTH", "redisPswrd")
	lpush := <-commands
	if len(lpush) != 3 {
		t.Fatalf("LPUSH command length = %d, want 3: %#v", len(lpush), lpush)
	}
	assertCommand(t, lpush[:2], "LPUSH", "cadvisor")

	var detail detailSpec
	if err := json.Unmarshal([]byte(lpush[2]), &detail); err != nil {
		t.Fatalf("LPUSH payload is not detailSpec JSON: %v", err)
	}
	if detail.MachineName != "test-machine" {
		t.Fatalf("MachineName = %q, want test-machine", detail.MachineName)
	}
	if detail.ContainerName != "/docker/test-container" {
		t.Fatalf(
			"ContainerName = %q, want /docker/test-container",
			detail.ContainerName,
		)
	}
}

func TestAddStatsDoesNotAuthenticateWithoutPassword(t *testing.T) {
	commands, address, stop := startFakeRedis(t)
	defer stop()

	driver, err := newStorage("test-machine", "cadvisor", address, "", 0)
	if err != nil {
		t.Fatalf("newStorage() error = %v", err)
	}
	defer driver.Close()

	storage := driver.(*redisStorage)
	storage.readyToFlush = func() bool { return true }

	if err := storage.AddStats(
		&info.ContainerInfo{
			ContainerReference: info.ContainerReference{
				Name: "/docker/test-container",
			},
		},
		&info.ContainerStats{Timestamp: time.Unix(1, 0)},
	); err != nil {
		t.Fatalf("AddStats() error = %v", err)
	}

	lpush := <-commands
	assertCommand(t, lpush[:2], "LPUSH", "cadvisor")
}

func startFakeRedis(t *testing.T) (<-chan []string, string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	commands := make(chan []string, 4)
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			command, err := readRESPCommand(reader)
			if err != nil {
				return
			}
			commands <- command
			switch strings.ToUpper(command[0]) {
			case "AUTH":
				fmt.Fprint(conn, "+OK\r\n")
			case "LPUSH":
				fmt.Fprint(conn, ":1\r\n")
			default:
				fmt.Fprintf(conn, "-ERR unsupported command %s\r\n", command[0])
			}
		}
	}()

	stop := func() {
		listener.Close()
		<-done
	}
	return commands, listener.Addr().String(), stop
}

func readRESPCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected array, got %q", line)
	}
	count, err := strconv.Atoi(strings.TrimPrefix(line, "*"))
	if err != nil {
		return nil, err
	}

	command := make([]string, 0, count)
	for i := 0; i < count; i++ {
		lengthLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		lengthLine = strings.TrimSuffix(lengthLine, "\r\n")
		if !strings.HasPrefix(lengthLine, "$") {
			return nil, fmt.Errorf("expected bulk string, got %q", lengthLine)
		}
		length, err := strconv.Atoi(strings.TrimPrefix(lengthLine, "$"))
		if err != nil {
			return nil, err
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return nil, err
		}
		command = append(command, string(buf[:length]))
	}
	return command, nil
}

func assertCommand(t *testing.T, command []string, want ...string) {
	t.Helper()
	if len(command) != len(want) {
		t.Fatalf("command length = %d, want %d: %#v", len(command), len(want), command)
	}
	for i := range want {
		if command[i] != want[i] {
			t.Fatalf("command[%d] = %q, want %q: %#v", i, command[i], want[i], command)
		}
	}
}
