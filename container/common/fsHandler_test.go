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

package common

import (
	"fmt"
	"github.com/google/cadvisor/fs"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testFsInfo struct {
	allowDirUsage bool
	t             *testing.T
}

func (self *testFsInfo) RefreshCache() {}

var (
	sda1 = fs.Fs{
		DeviceInfo: fs.DeviceInfo{
			Device: "/dev/sda1",
		},
		Type:      "ext4",
		Capacity:  5000,
		Available: 2000,
	}
	sda2 = fs.Fs{
		DeviceInfo: fs.DeviceInfo{
			Device: "/dev/sda2",
		},
		Type:      "ext4",
		Capacity:  1000,
		Available: 500,
	}
	sdb1 = fs.Fs{
		DeviceInfo: fs.DeviceInfo{
			Device: "/dev/sdb1",
		},
		Type:      "xfs",
		Capacity:  3000,
		Available: 500,
	}
	sdc1 = fs.Fs{
		DeviceInfo: fs.DeviceInfo{
			Device: "/dev/sdc1",
		},
		Type:      "zfs",
		Capacity:  4000,
		Available: 4000,
	}
)

func (self *testFsInfo) GetGlobalFsInfo(_ bool) ([]fs.Fs, error) {
	return []fs.Fs{sda1, sda2, sdb1, sdc1}, nil
}

func (self *testFsInfo) GetFsInfoForMounts(mountSet map[string]struct{}, _ bool) ([]fs.Fs, error) {
	return nil, fmt.Errorf("Not implemented: GetFsInfoForMounts(%+v)", mountSet)
}

func (self *testFsInfo) GetFsInfoForDevices(deviceSet map[string]struct{}, _ bool) ([]fs.Fs, error) {
	fsMap := map[string]fs.Fs{
		"/dev/sda1": sda1,
		"/dev/sda2": sda2,
		"/dev/sdb1": sdb1,
		// /dev/sdc1 isn't there because we should never be looking at it
	}
	fsOut := make([]fs.Fs, 0)

	for device := range deviceSet {
		fs, ok := fsMap[device]
		if !ok {
			return nil, fmt.Errorf("GetFsInfoForDevices was called with unexpected device: %q", device)
		}
		fsOut = append(fsOut, fs)
	}

	return fsOut, nil
}

func (self *testFsInfo) GetDirUsage(dir string, timeout time.Duration) (uint64, error) {
	if !self.allowDirUsage {
		return uint64(0), fmt.Errorf("Dir usage is disabled for this test!")
	}
	if dir == "/var/lib/docker/aufs/diff/aa" {
		return uint64(100), nil
	}
	if dir == "/var/lib/docker/containers/aa" {
		return uint64(50), nil
	}
	if dir == "/some/mount" {
		return uint64(2000), nil
	}
	if dir == "/other/mount" {
		return uint64(200), nil
	}
	// /sdcmount isn't there because we should never be looking at it
	return uint64(0), fmt.Errorf("Not implemented: GetDirUsage(%s, ...)", dir)
}

func (self *testFsInfo) GetDirFsDevice(dir string) (*fs.DeviceInfo, error) {
	if dir == "/var/lib/docker/aufs/diff/aa" {
		return &fs.DeviceInfo{Device: "/dev/sda1"}, nil
	}
	if dir == "/var/lib/docker/containers/aa" {
		return &fs.DeviceInfo{Device: "/dev/sda2"}, nil
	}
	if dir == "/some/mount" {
		return &fs.DeviceInfo{Device: "/dev/sdb1"}, nil
	}
	if dir == "/other/mount" {
		return &fs.DeviceInfo{Device: "/dev/sdb1"}, nil
	}
	if dir == "/sdcmount" {
		return &fs.DeviceInfo{Device: "/dev/sdc1"}, nil
	}
	return nil, fmt.Errorf("Not implemented: GetDirFsDevice(%s)", dir)
}

func (self *testFsInfo) GetDeviceForLabel(label string) (string, error) {
	return "", fmt.Errorf("Not implemented: GetDeviceForLabel(%s)", label)
}

func (self *testFsInfo) GetLabelsForDevice(device string) ([]string, error) {
	return nil, fmt.Errorf("Not implemented: GetLabelsForDevice(%s)", device)
}

func (self *testFsInfo) GetMountpointForDevice(device string) (string, error) {
	return "", fmt.Errorf("Not implemented: GetMountpointForDevice(%s)", device)
}

var testBaseDirs = []string{
	"/var/lib/docker/aufs/diff/aa",
	"/some/mount",
	"/other/mount",
}

var testExtraDirs = []string{
	"/var/lib/docker/containers/aa",
}

func TestCollectionWithDu(t *testing.T) {
	as := assert.New(t)

	(*skipDuFlag) = false
	hdlr := NewFsHandler(time.Second, testBaseDirs, testExtraDirs, &testFsInfo{
		allowDirUsage: true,
		t:             t,
	})

	// Usage before trackUsage should return an error
	usage, err := hdlr.Usage()
	as.Error(err)
	as.Equal(0, len(usage))

	err = hdlr.update()
	as.NoError(err)

	usage, err = hdlr.Usage()
	as.NoError(err)

	foundSda1 := false
	foundSda2 := false
	foundSdb1 := false

	for _, stat := range usage {
		if stat.Device == "/dev/sda1" {
			// Only the aufs layer
			as.Equal("ext4", stat.Type)
			as.Equal(uint64(5000), stat.Limit)
			as.Equal(uint64(100), stat.BaseUsage)
			as.Equal(uint64(100), stat.Usage)
			foundSda1 = true
			continue
		}

		if stat.Device == "/dev/sda2" {
			// Only logs
			as.Equal("ext4", stat.Type)
			as.Equal(uint64(1000), stat.Limit)
			as.Equal(uint64(0), stat.BaseUsage)
			as.Equal(uint64(50), stat.Usage)
			foundSda2 = true
			continue
		}

		if stat.Device == "/dev/sdb1" {
			// Two mounts
			as.Equal("xfs", stat.Type)
			as.Equal(uint64(3000), stat.Limit)
			as.Equal(uint64(2200), stat.BaseUsage)
			as.Equal(uint64(2200), stat.Usage)
			foundSdb1 = true
			continue
		}

		t.Errorf("Unexpected device in results: %q", stat.Device)
	}

	as.True(foundSda1)
	as.True(foundSda2)
	as.True(foundSdb1)
}

func TestCollectionWithDf(t *testing.T) {
	as := assert.New(t)

	(*skipDuFlag) = true
	hdlr := NewFsHandler(time.Second, testBaseDirs, testExtraDirs, &testFsInfo{
		allowDirUsage: false,
		t:             t,
	})

	err := hdlr.update()
	as.NoError(err)

	usage, err := hdlr.Usage()
	as.NoError(err)

	foundSda1 := false
	foundSda2 := false
	foundSdb1 := false

	for _, stat := range usage {
		if stat.Device == "/dev/sda1" {
			// Only the aufs layer
			as.Equal(uint64(5000), stat.Limit)
			as.Equal(uint64(3000), stat.Usage)
			foundSda1 = true
			continue
		}

		if stat.Device == "/dev/sda2" {
			// Only logs
			as.Equal(uint64(1000), stat.Limit)
			as.Equal(uint64(500), stat.Usage)
			foundSda2 = true
			continue
		}

		if stat.Device == "/dev/sdb1" {
			// Two mounts
			as.Equal(uint64(3000), stat.Limit)
			as.Equal(uint64(2500), stat.Usage)
			foundSdb1 = true
			continue
		}

		t.Errorf("Unexpected device in results: %q", stat.Device)
	}

	as.True(foundSda1)
	as.True(foundSda2)
	as.True(foundSdb1)
}

func TestCollectionWithDfAndIgnoredDevices(t *testing.T) {
	as := assert.New(t)

	(*skipDuFlag) = true
	skipDevicesFlag.Set("/dev/sda\\d")
	hdlr := NewFsHandler(time.Second, testBaseDirs, testExtraDirs, &testFsInfo{
		allowDirUsage: false,
		t:             t,
	})

	err := hdlr.update()
	as.NoError(err)

	usage, err := hdlr.Usage()
	as.NoError(err)

	foundSdb1 := false

	for _, stat := range usage {
		if stat.Device == "/dev/sda1" {
			t.Errorf("Metrics for /dev/sda1 reported despite being ignored!")
			continue
		}

		if stat.Device == "/dev/sda2" {
			t.Errorf("Metrics for /dev/sda2 reported despite being ignored!")
			continue
		}

		if stat.Device == "/dev/sdb1" {
			foundSdb1 = true
			continue
		}

		t.Errorf("Unexpected device in results: %q", stat.Device)
	}

	as.True(foundSdb1)
}

func TestCollectionWithDuAndIgnoredDevices(t *testing.T) {
	as := assert.New(t)

	(*skipDuFlag) = false
	skipDevicesFlag.Set("/dev/sdc1")
	hdlr := NewFsHandler(time.Second, []string{"/sdcmount"}, []string{}, &testFsInfo{
		allowDirUsage: true,
		t:             t,
	})

	err := hdlr.update()
	as.NoError(err)

	usage, err := hdlr.Usage()
	as.NoError(err)

	for _, stat := range usage {
		if stat.Device == "/dev/sdc1" {
			t.Errorf("Metrics for /dev/sdc1 reported despite being ignored!")
			continue
		}
	}
}
