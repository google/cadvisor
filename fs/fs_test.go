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

package fs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDiskStatsMap(t *testing.T) {
	diskStatsMap, err := getDiskStatsMap("test_resources/diskstats")
	if err != nil {
		t.Errorf("Error calling getDiskStatMap %s", err)
	}
	if len(diskStatsMap) != 30 {
		t.Errorf("diskStatsMap %+v not valid", diskStatsMap)
	}
	keySet := map[string]string{
		"/dev/sda":  "/dev/sda",
		"/dev/sdb":  "/dev/sdb",
		"/dev/sdc":  "/dev/sdc",
		"/dev/sdd":  "/dev/sdd",
		"/dev/sde":  "/dev/sde",
		"/dev/sdf":  "/dev/sdf",
		"/dev/sdg":  "/dev/sdg",
		"/dev/sdh":  "/dev/sdh",
		"/dev/sdb1": "/dev/sdb1",
		"/dev/sdb2": "/dev/sdb2",
		"/dev/sda1": "/dev/sda1",
		"/dev/sda2": "/dev/sda2",
		"/dev/sdc1": "/dev/sdc1",
		"/dev/sdc2": "/dev/sdc2",
		"/dev/sdc3": "/dev/sdc3",
		"/dev/sdc4": "/dev/sdc4",
		"/dev/sdd1": "/dev/sdd1",
		"/dev/sdd2": "/dev/sdd2",
		"/dev/sdd3": "/dev/sdd3",
		"/dev/sdd4": "/dev/sdd4",
		"/dev/sde1": "/dev/sde1",
		"/dev/sde2": "/dev/sde2",
		"/dev/sdf1": "/dev/sdf1",
		"/dev/sdf2": "/dev/sdf2",
		"/dev/sdg1": "/dev/sdg1",
		"/dev/sdg2": "/dev/sdg2",
		"/dev/sdh1": "/dev/sdh1",
		"/dev/sdh2": "/dev/sdh2",
		"/dev/dm-0": "/dev/dm-0",
		"/dev/dm-1": "/dev/dm-1",
	}

	for device := range diskStatsMap {
		if _, ok := keySet[device]; !ok {
			t.Errorf("Cannot find device %s", device)
		}
		delete(keySet, device)
	}
	if len(keySet) != 0 {
		t.Errorf("diskStatsMap %+v contains illegal keys %+v", diskStatsMap, keySet)
	}
}

func TestFileNotExist(t *testing.T) {
	_, err := getDiskStatsMap("/file_does_not_exist")
	if err != nil {
		t.Fatalf("getDiskStatsMap must not error for absent file: %s", err)
	}
}

func TestDirUsage(t *testing.T) {
	as := assert.New(t)
	fsInfo, err := NewFsInfo(Context{})
	as.NoError(err)
	dir, err := ioutil.TempDir(os.TempDir(), "")
	as.NoError(err)
	defer os.RemoveAll(dir)
	dataSize := 1024 * 100 //100 KB
	b := make([]byte, dataSize)
	f, err := ioutil.TempFile(dir, "")
	as.NoError(err)
	as.NoError(ioutil.WriteFile(f.Name(), b, 0700))
	fi, err := f.Stat()
	as.NoError(err)
	expectedSize := uint64(fi.Size())
	size, err := fsInfo.GetDirUsage(dir)
	as.NoError(err)
	as.True(expectedSize <= size, "expected dir size to be at-least %d; got size: %d", expectedSize, size)
}

var dmStatusTests = []struct {
	dmStatus    string
	used        uint64
	total       uint64
	errExpected bool
}{
	{`0 409534464 thin-pool 64085 3705/4161600 88106/3199488 - rw no_discard_passdown queue_if_no_space -`, 88106, 3199488, false},
	{`0 209715200 thin-pool 707 1215/524288 30282/1638400 - rw discard_passdown`, 30282, 1638400, false},
	{`Invalid status line`, 0, 0, false},
}

func TestParseDMStatus(t *testing.T) {
	for _, tt := range dmStatusTests {
		used, total, err := parseDMStatus(tt.dmStatus)
		if tt.errExpected && err != nil {
			t.Errorf("parseDMStatus(%q) expected error", tt.dmStatus)
		}
		if used != tt.used {
			t.Errorf("parseDMStatus(%q) wrong used value => %q, want %q", tt.dmStatus, used, tt.used)
		}
		if total != tt.total {
			t.Errorf("parseDMStatus(%q) wrong total value => %q, want %q", tt.dmStatus, total, tt.total)
		}
	}
}

var dmTableTests = []struct {
	dmTable     string
	major       uint
	minor       uint
	dataBlkSize uint
	errExpected bool
}{
	{`0 409534464 thin-pool 253:6 253:7 128 32768 1 skip_block_zeroing`, 253, 7, 128, false},
	{`0 409534464 thin-pool 253:6 258:9 512 32768 1 skip_block_zeroing otherstuff`, 258, 9, 512, false},
	{`Invalid status line`, 0, 0, 0, false},
}

func TestParseDMTable(t *testing.T) {
	for _, tt := range dmTableTests {
		major, minor, dataBlkSize, err := parseDMTable(tt.dmTable)
		if tt.errExpected && err != nil {
			t.Errorf("parseDMTable(%q) expected error", tt.dmTable)
		}
		if major != tt.major {
			t.Errorf("parseDMTable(%q) wrong major value => %q, want %q", tt.dmTable, major, tt.major)
		}
		if minor != tt.minor {
			t.Errorf("parseDMTable(%q) wrong minor value => %q, want %q", tt.dmTable, minor, tt.minor)
		}
		if dataBlkSize != tt.dataBlkSize {
			t.Errorf("parseDMTable(%q) wrong dataBlkSize value => %q, want %q", tt.dmTable, dataBlkSize, tt.dataBlkSize)
		}
	}
}

func TestAddLabels(t *testing.T) {
	tests := []struct {
		name                     string
		partitions               map[string]partition
		dockerImagesDevice       string
		expectedDockerDevice     string
		expectedSystemRootDevice string
	}{
		{
			name: "/ in 1 LV, images in another LV",
			partitions: map[string]partition{
				"/dev/mapper/vg_vagrant-lv_root": {
					mountpoint: "/",
					fsType:     "devicemapper",
				},
				"vg_vagrant-docker--pool": {
					mountpoint: "",
					fsType:     "devicemapper",
				},
			},
			dockerImagesDevice:       "vg_vagrant-docker--pool",
			expectedDockerDevice:     "vg_vagrant-docker--pool",
			expectedSystemRootDevice: "/dev/mapper/vg_vagrant-lv_root",
		},
		{
			name: "/ in 1 LV, images on non-devicemapper mount",
			partitions: map[string]partition{
				"/dev/mapper/vg_vagrant-lv_root": {
					mountpoint: "/",
					fsType:     "devicemapper",
				},
				"/dev/sda1": {
					mountpoint: "/var/lib/docker",
					fsType:     "ext4",
				},
			},
			dockerImagesDevice:       "",
			expectedDockerDevice:     "/dev/sda1",
			expectedSystemRootDevice: "/dev/mapper/vg_vagrant-lv_root",
		},
		{
			name: "just 1 / partition, devicemapper",
			partitions: map[string]partition{
				"/dev/mapper/vg_vagrant-lv_root": {
					mountpoint: "/",
					fsType:     "devicemapper",
				},
			},
			dockerImagesDevice:       "",
			expectedDockerDevice:     "/dev/mapper/vg_vagrant-lv_root",
			expectedSystemRootDevice: "/dev/mapper/vg_vagrant-lv_root",
		},
		{
			name: "just 1 / partition, not devicemapper",
			partitions: map[string]partition{
				"/dev/sda1": {
					mountpoint: "/",
					fsType:     "ext4",
				},
			},
			dockerImagesDevice:       "",
			expectedDockerDevice:     "/dev/sda1",
			expectedSystemRootDevice: "/dev/sda1",
		},
		{
			name: "devicemapper loopback on non-root partition",
			partitions: map[string]partition{
				"/dev/sda1": {
					mountpoint: "/",
					fsType:     "ext4",
				},
				"/dev/sdb1": {
					mountpoint: "/var/lib/docker/devicemapper",
					fsType:     "ext4",
				},
				"docker-253:0-34470016-pool": {
					// pretend that the loopback file is at /var/lib/docker/devicemapper/devicemapper/data and
					// /var/lib/docker/devicemapper is on a different partition than /
					mountpoint: "",
					fsType:     "devicemapper",
				},
			},
			// dockerImagesDevice is empty b/c this simulates a loopback setup
			dockerImagesDevice:       "",
			expectedDockerDevice:     "/dev/sdb1",
			expectedSystemRootDevice: "/dev/sda1",
		},
		{
			name: "devicemapper, not loopback, /var/lib/docker/devicemapper on non-root partition",
			partitions: map[string]partition{
				"/dev/sda1": {
					mountpoint: "/",
					fsType:     "ext4",
				},
				"/dev/sdb1": {
					mountpoint: "/var/lib/docker/devicemapper",
					fsType:     "ext4",
				},
				"vg_vagrant-docker--pool": {
					mountpoint: "",
					fsType:     "devicemapper",
				},
			},
			dockerImagesDevice:       "vg_vagrant-docker--pool",
			expectedDockerDevice:     "vg_vagrant-docker--pool",
			expectedSystemRootDevice: "/dev/sda1",
		},
		{
			name: "multiple mounts - innermost check",
			partitions: map[string]partition{
				"/dev/sda1": {
					mountpoint: "/",
					fsType:     "ext4",
				},
				"/dev/sdb1": {
					mountpoint: "/var/lib/docker",
					fsType:     "ext4",
				},
				"/dev/sdb2": {
					mountpoint: "/var/lib/docker/btrfs",
					fsType:     "btrfs",
				},
			},
			dockerImagesDevice:       "",
			expectedDockerDevice:     "/dev/sdb2",
			expectedSystemRootDevice: "/dev/sda1",
		},
	}

	for _, tt := range tests {
		fsInfo := &RealFsInfo{
			labels:     map[string]string{LabelDockerImages: tt.dockerImagesDevice},
			partitions: tt.partitions,
		}

		context := Context{DockerRoot: "/var/lib/docker"}

		fsInfo.addLabels(context)

		if e, a := tt.expectedDockerDevice, fsInfo.labels[LabelDockerImages]; e != a {
			t.Errorf("%s: docker device: expected %q, got %q", tt.name, e, a)
		}
		if e, a := tt.expectedSystemRootDevice, fsInfo.labels[LabelSystemRoot]; e != a {
			t.Errorf("%s: system root: expected %q, got %q", tt.name, e, a)
		}
	}
}

// This test ensures that NewFsInfo won't assign the devicemapper device used for Docker storage to
// LabelDockerImages if the devicemapper device is using loopback. If we did set the label for a
// loopback device, that would result in the loopback device's capacity/available/used information
// being reported back, which could greatly exceed the actual physical capacity of the actual device
// on which the loopback file resides.
func TestDockerDMDeviceReturnsErrorForLoopback(t *testing.T) {
	driverStatus := `[
        [
            "Pool Name",
            "docker-253:0-34470016-pool"
        ],
        [
            "Pool Blocksize",
            "65.54 kB"
        ],
        [
            "Base Device Size",
            "107.4 GB"
        ],
        [
            "Backing Filesystem",
            ""
        ],
        [
            "Data file",
            "/dev/loop0"
        ],
        [
            "Metadata file",
            "/dev/loop1"
        ],
        [
            "Data Space Used",
            "7.684 GB"
        ],
        [
            "Data Space Total",
            "107.4 GB"
        ],
        [
            "Data Space Available",
            "2.753 GB"
        ],
        [
            "Metadata Space Used",
            "12.21 MB"
        ],
        [
            "Metadata Space Total",
            "2.147 GB"
        ],
        [
            "Metadata Space Available",
            "2.135 GB"
        ],
        [
            "Udev Sync Supported",
            "true"
        ],
        [
            "Deferred Removal Enabled",
            "false"
        ],
        [
            "Deferred Deletion Enabled",
            "false"
        ],
        [
            "Deferred Deleted Device Count",
            "0"
        ],
        [
            "Data loop file",
            "/var/lib/docker/devicemapper/devicemapper/data"
        ],
        [
            "Metadata loop file",
            "/var/lib/docker/devicemapper/devicemapper/metadata"
        ],
        [
            "Library Version",
            "1.02.109 (2015-09-22)"
        ]
    ]`

	if _, _, _, _, err := dockerDMDevice(driverStatus); err == nil {
		t.Errorf("unexpected nil error")
	}
}
