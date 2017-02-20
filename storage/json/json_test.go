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

package json

import (
	"net"
	"reflect"
	"testing"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/storage"
	"github.com/google/cadvisor/storage/test"
)

func newJsonTestStorage(connection net.Conn) *jsonStorage {
	machineName := "machineA"
	description := "test_description"
	newJsonStorage := &jsonStorage{
		machineName: machineName,
		description: description,
		connection:  connection,
	}
	return newJsonStorage
}

type jsonTestStorage struct {
	base storage.StorageDriver
}

func (self *jsonTestStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	return self.base.AddStats(ref, stats)
}

func (self *jsonTestStorage) StatsEq(a, b *info.ContainerStats) bool {
	if !test.TimeEq(a.Timestamp, b.Timestamp, 10*time.Millisecond) {
		return false
	}
	if !reflect.DeepEqual(a.Cpu, b.Cpu) {
		return false
	}
	if !reflect.DeepEqual(a.Memory, b.Memory) {
		return false
	}
	if !reflect.DeepEqual(a.Network, b.Network) {
		return false
	}
	if !reflect.DeepEqual(a.Filesystem, b.Filesystem) {
		return false
	}
	return true
}

func (self *jsonTestStorage) Close() error {
	return self.base.Close()
}

func runStorageTest(f func(test.TestStorageDriver, *testing.T), t *testing.T) {
	// Generate a fake server/client
	server, client := net.Pipe()

	driver := newJsonTestStorage(client)

	defer driver.Close()
	defer server.Close()
	testDriver := &jsonTestStorage{}
	testDriver.base = driver

	// Generate another container's data on same machine.
	test.StorageDriverFillRandomStatsFunc("containerOnSameMachine", 100, testDriver, t)

	// Generate a second fake server/client
	serverTwo, clientTwo := net.Pipe()

	// Generate another container's data on another machine.
	driverForAnotherMachine := newJsonTestStorage(clientTwo)

	defer driverForAnotherMachine.Close()
	defer serverTwo.Close()
	testDriverOtherMachine := &jsonTestStorage{}
	testDriverOtherMachine.base = driverForAnotherMachine

	test.StorageDriverFillRandomStatsFunc("containerOnAnotherMachine", 100, testDriverOtherMachine, t)

	f(testDriver, t)
}
