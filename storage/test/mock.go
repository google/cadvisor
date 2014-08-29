// Copyright 2014 Google Inc. All Rights Reserved.
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

package test

import (
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/storage"
	"github.com/stretchr/testify/mock"
)

type MockStorageDriver struct {
	mock.Mock
	MockCloseMethod bool
}

func (self *MockStorageDriver) AddStats(pairs ...storage.ContainerRefStatsPair) error {
	args := self.Called(pairs)
	return args.Error(0)
}

func (self *MockStorageDriver) RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error) {
	args := self.Called(containerName, numStats)
	return args.Get(0).([]*info.ContainerStats), args.Error(1)
}

func (self *MockStorageDriver) Percentiles(
	containerName string,
	cpuUsagePercentiles []int,
	memUsagePercentiles []int,
) (*info.ContainerStatsPercentiles, error) {
	args := self.Called(containerName, cpuUsagePercentiles, memUsagePercentiles)
	return args.Get(0).(*info.ContainerStatsPercentiles), args.Error(1)
}

func (self *MockStorageDriver) Samples(containerName string, numSamples int) ([]*info.ContainerStatsSample, error) {
	args := self.Called(containerName, numSamples)
	return args.Get(0).([]*info.ContainerStatsSample), args.Error(1)
}

func (self *MockStorageDriver) Close() error {
	if self.MockCloseMethod {
		args := self.Called()
		return args.Error(0)
	}
	return nil
}
