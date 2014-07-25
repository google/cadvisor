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

package mockfs

import (
	"bytes"
	"fmt"

	"github.com/google/cadvisor/utils/fs"
	"github.com/stretchr/testify/mock"
)

type MockFile struct {
	mock.Mock
	bytes.Buffer
	checkClose bool
}

func (self *MockFile) CheckClose() *MockFile {
	self.checkClose = true
	return self
}

func (self *MockFile) Close() error {
	if self.checkClose {
		args := self.Called()
		return args.Error(0)
	}
	return nil
}

type MockFileSystem struct {
	mock.Mock
	files map[string]*MockFile
}

func (self *MockFileSystem) AddTextFile(name, content string) *MockFile {
	if self.files == nil {
		self.files = make(map[string]*MockFile, 4)
	}
	f := &MockFile{
		Buffer: *bytes.NewBufferString(content),
	}
	self.files[name] = f
	return f
}

func (self *MockFileSystem) Open(name string) (fs.File, error) {
	if f, ok := self.files[name]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("cannot open file %v", name)
}
