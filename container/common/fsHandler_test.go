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

// Handler for Docker containers.
package common

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/cadvisor/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsHandlerEmptyPath(t *testing.T) {
	d, err := ioutil.TempDir("", "")
	require.Nil(t, err)
	defer os.RemoveAll(d)
	f, err := ioutil.TempFile(d, "")
	require.Nil(t, err)

	_, err = f.Write([]byte("test message"))
	require.Nil(t, err)

	fh := &realFsHandler{
		lastUpdate:     time.Time{},
		usageBytes:     0,
		baseUsageBytes: 0,
		period:         time.Second,
		minPeriod:      time.Second,
		rootfs:         "",
		extraDir:       d,
		fsInfo:         &fs.RealFsInfo{},
		stopChan:       make(chan struct{}, 1),
	}
	fh.update()
	a, b := fh.Usage()
	assert.True(t, a == 0, "expected %d to be equal to 0", a)
	assert.True(t, b > 0, "expected %d to be greater than 0", b)

	fh = &realFsHandler{
		lastUpdate:     time.Time{},
		usageBytes:     0,
		baseUsageBytes: 0,
		period:         time.Second,
		minPeriod:      time.Second,
		extraDir:       "",
		rootfs:         d,
		fsInfo:         &fs.RealFsInfo{},
		stopChan:       make(chan struct{}, 1),
	}

	fh.update()
	a, b = fh.Usage()
	assert.True(t, a > 0, "expected %d to be greater than 0", a)
	assert.Equal(t, b, a, "expected %d to be equal to %d", b, a)
}
