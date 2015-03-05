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

// +build cgo,linux

package fs

/*
 extern int getBytesFree(const char *path, unsigned long long *bytes);
 extern int getBytesTotal(const char *path, unsigned long long *bytes);
*/
import "C"

import (
	"syscall"
	"unsafe"
)

func getVfsStats(path string) (total uint64, free uint64, err error) {
	_p0, err := syscall.BytePtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	res, err := C.getBytesFree((*C.char)(unsafe.Pointer(_p0)), (*_Ctype_ulonglong)(unsafe.Pointer(&free)))
	if res != 0 {
		return 0, 0, err
	}
	res, err = C.getBytesTotal((*C.char)(unsafe.Pointer(_p0)), (*_Ctype_ulonglong)(unsafe.Pointer(&total)))
	if res != 0 {
		return 0, 0, err
	}
	return total, free, nil
}
