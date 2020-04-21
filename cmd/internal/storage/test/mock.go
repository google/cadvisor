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
	info "github.com/google/cadvisor/info/v1"

	"github.com/stretchr/testify/mock"
)

type MockStorageDriver struct {
	mock.Mock
	MockCloseMethod bool
}

func (d *MockStorageDriver) AddStats(cInfo *info.ContainerInfo, stats *info.ContainerStats) error {
	args := d.Called(cInfo.ContainerReference, stats)
	return args.Error(0)
}

func (d *MockStorageDriver) Close() error {
	if d.MockCloseMethod {
		args := d.Called()
		return args.Error(0)
	}
	return nil
}
