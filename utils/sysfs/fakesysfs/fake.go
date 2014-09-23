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

package fakesysfs

import (
	"os"
	"time"
)

// If we extend sysfs to support more interfaces, it might be worth making this a mock instead of a fake.
type FileInfo struct {
}

func (self *FileInfo) Name() string {
	return "sda"
}

func (self *FileInfo) Size() int64 {
	return 1234567
}

func (self *FileInfo) Mode() os.FileMode {
	return 0
}

func (self *FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (self *FileInfo) IsDir() bool {
	return true
}

func (self *FileInfo) Sys() interface{} {
	return nil
}

type FakeSysFs struct {
	info FileInfo
}

func (self *FakeSysFs) GetBlockDevices() ([]os.FileInfo, error) {
	return []os.FileInfo{&self.info}, nil
}

func (self *FakeSysFs) GetBlockDeviceSize(name string) (string, error) {
	return "1234567", nil
}

func (self *FakeSysFs) GetBlockDeviceNumbers(name string) (string, error) {
	return "8:0\n", nil
}
