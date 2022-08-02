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

// Handler for containerd containers.
package containerd

import (
	"context"
	"fmt"
	"testing"

	"github.com/containerd/typeurl"
	"github.com/google/cadvisor/container/containerd/containers"
	types "github.com/google/cadvisor/third_party/containerd/api/types"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	v1 "github.com/google/cadvisor/info/v1"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	testContainerSandboxID   = "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"
	testContainerSandboxName = "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"
	testContainerID          = "c6a1aa99f14d3e57417e145b897e34961145f6b6f14216a176a34bfabbf79086"
	testContainerName        = "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/c6a1aa99f14d3e57417e145b897e34961145f6b6f14216a176a34bfabbf79086"

	testLogPath = "/var/log/pods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa"

	testLogUsage    = 5000
	testBaseUsage   = 10000
	testInodesTotal = 10
	testBaseInodes  = 6
	testLogInodes   = 2
)

var (
	testContainers       map[string]*containers.Container
	testContainer        *containers.Container
	testContainerSandbox *containers.Container
	testStatus           *criapi.ContainerStatus
	testStats            *criapi.ContainerStats
)

func init() {
	typeurl.Register(&specs.Spec{}, "types.contianerd.io/opencontainers/runtime-spec", "v1", "Spec")

	testContainers = make(map[string]*containers.Container)
	testContainerSandbox = &containers.Container{
		ID: testContainerSandboxID,
		Labels: map[string]string{
			"io.cri-containerd.kind":       "sandbox",
			"io.kubernetes.container.name": "pause",
			"io.kubernetes.pod.name":       "some-pod",
			"io.kubernetes.pod.namespace":  "some-ns",
			"io.kubernetes.pod.uid":        "some-uid"},
	}
	testContainer = &containers.Container{
		ID: testContainerID,
		Labels: map[string]string{
			"io.cri-containerd.kind":       "container",
			"io.kubernetes.container.name": "some-container",
			"io.kubernetes.pod.name":       "some-pod",
			"io.kubernetes.pod.namespace":  "some-ns",
			"io.kubernetes.pod.uid":        "some-uid"},
	}
	spec := &specs.Spec{Root: &specs.Root{Path: "/test/"}, Process: &specs.Process{Env: []string{"TEST_REGION=FRA", "TEST_ZONE=A", "HELLO=WORLD"}}}
	testContainerSandbox.Spec, _ = typeurl.MarshalAny(spec)
	testContainer.Spec, _ = typeurl.MarshalAny(spec)
	testContainers[testContainerSandboxID] = testContainerSandbox
	testContainers[testContainerID] = testContainer

	testStatus = &criapi.ContainerStatus{
		Metadata: &criapi.ContainerMetadata{Attempt: 2},
		LogPath:  testLogPath,
	}

	testStats = &criapi.ContainerStats{
		WritableLayer: &criapi.FilesystemUsage{
			UsedBytes:  &criapi.UInt64Value{Value: testBaseUsage},
			InodesUsed: &criapi.UInt64Value{Value: testBaseInodes},
		},
	}
}

type mockedMachineInfo struct {
	machineInfo *v1.MachineInfo
}

func (m *mockedMachineInfo) GetMachineInfo() (*info.MachineInfo, error) {
	return m.machineInfo, nil
}

func (m *mockedMachineInfo) GetVersionInfo() (*info.VersionInfo, error) {
	return &info.VersionInfo{}, nil
}

type fsInfoMock struct {
	deviceDir  string
	deviceinfo *fs.DeviceInfo
}

func (m *fsInfoMock) GetDirFsDevice(dir string) (*fs.DeviceInfo, error) {
	if dir != m.deviceDir {
		return nil, fmt.Errorf("cannot get device for dir: %q", dir)
	}
	return m.deviceinfo, nil
}

func (m *fsInfoMock) GetGlobalFsInfo() ([]fs.Fs, error) {
	return nil, nil
}

func (m *fsInfoMock) GetFsInfoForPath(mountSet map[string]struct{}) ([]fs.Fs, error) {
	return nil, nil
}

func (m *fsInfoMock) GetDirUsage(dir string) (fs.UsageInfo, error) {
	return fs.UsageInfo{
		Bytes:  testLogUsage,
		Inodes: testLogInodes,
	}, nil
}

func (m *fsInfoMock) GetDeviceInfoByFsUUID(uuid string) (*fs.DeviceInfo, error) {
	return nil, nil
}

func (m *fsInfoMock) GetDeviceForLabel(label string) (string, error) {
	return "", nil
}

func (m *fsInfoMock) GetLabelsForDevice(device string) ([]string, error) {
	return nil, nil
}

func (m *fsInfoMock) GetMountpointForDevice(device string) (string, error) {
	return "", nil
}

func TestHandler(t *testing.T) {
	as := assert.New(t)
	type testCase struct {
		client               ContainerdClient
		name                 string
		machineInfoFactory   info.MachineInfoFactory
		fsInfo               fs.FsInfo
		cgroupSubsystems     map[string]string
		inHostNamespace      bool
		metadataEnvAllowList []string
		includedMetrics      container.MetricSet

		hasErr         bool
		errContains    string
		checkReference *info.ContainerReference
		checkEnvVars   map[string]string
	}

	for _, ts := range []testCase{
		{
			mockcontainerdClient(nil, nil, nil, nil, nil),
			testContainerSandboxName,
			nil,
			nil,
			map[string]string{},
			false,
			nil,
			nil,
			true,
			"unable to find container \"" + testContainerSandboxID + "\"",
			nil,
			nil,
		},
		{
			mockcontainerdClient(testContainers, nil, nil, nil, nil),
			testContainerSandboxName,
			&mockedMachineInfo{},
			nil,
			map[string]string{},
			false,
			nil,
			nil,
			false,
			"",
			&info.ContainerReference{
				Id:        testContainerSandboxID,
				Name:      testContainerSandboxName,
				Aliases:   []string{"k8s_POD_some-pod_some-ns_some-uid_0", testContainerSandboxID, testContainerSandboxName},
				Namespace: k8sContainerdNamespace,
			},
			map[string]string{},
		},
		{
			mockcontainerdClient(testContainers, nil, nil, nil, nil),
			testContainerSandboxName,
			&mockedMachineInfo{},
			nil,
			map[string]string{},
			false,
			[]string{"TEST"},
			nil,
			false,
			"",
			&info.ContainerReference{
				Id:        testContainerSandboxID,
				Name:      testContainerSandboxName,
				Aliases:   []string{"k8s_POD_some-pod_some-ns_some-uid_0", testContainerSandboxID, testContainerSandboxName},
				Namespace: k8sContainerdNamespace,
			},
			map[string]string{"TEST_REGION": "FRA", "TEST_ZONE": "A"},
		},
		{
			mockcontainerdClient(testContainers, testStatus, nil, nil, nil),
			testContainerName,
			&mockedMachineInfo{},
			nil,
			map[string]string{},
			false,
			nil,
			nil,
			false,
			"",
			&info.ContainerReference{
				Id:        testContainerID,
				Name:      testContainerName,
				Aliases:   []string{"k8s_some-container_some-pod_some-ns_some-uid_2", testContainerID, testContainerName},
				Namespace: k8sContainerdNamespace,
			},
			map[string]string{},
		},
	} {
		handler, err := newContainerdContainerHandler(ts.client, ts.name, ts.machineInfoFactory, ts.fsInfo, ts.cgroupSubsystems, ts.inHostNamespace, ts.metadataEnvAllowList, ts.includedMetrics)
		if ts.hasErr {
			as.NotNil(err)
			if ts.errContains != "" {
				as.Contains(err.Error(), ts.errContains)
			}
		}
		if ts.checkReference != nil {
			cr, err := handler.ContainerReference()
			as.Nil(err)
			as.Equal(*ts.checkReference, cr)
		}
		if ts.checkEnvVars != nil {
			sp, err := handler.GetSpec()
			as.Nil(err)
			as.Equal(ts.checkEnvVars, sp.Envs)
		}
	}
}

func TestHandlerDiskUssage(t *testing.T) {
	as := assert.New(t)

	metricSet := container.MetricSet{container.DiskUsageMetrics: {}}
	testMount := types.Mount{
		Type:   "overlay",
		Source: "overlay",
		Target: "",
		Options: []string{
			"index=off",
			"workdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/5001/work",
			"upperdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/5001/fs",
			"lowerdir=/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/4802/fs:/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/4801/fs",
		},
	}
	testMounts := []*types.Mount{&testMount}

	type testCase struct {
		client             ContainerdClient
		machineInfoFactory info.MachineInfoFactory
		fsInfo             fs.FsInfo
		inHostNamespace    bool

		hasErr bool
	}

	for _, ts := range []testCase{
		{
			mockcontainerdClient(testContainers, testStatus, testStats, testMounts, nil),
			&mockedMachineInfo{
				machineInfo: &v1.MachineInfo{
					Filesystems: []v1.FsInfo{
						{
							Device:   "/dev/sda",
							Capacity: 100,
							Inodes:   testInodesTotal,
							Type:     "fake type",
						},
					},
				},
			},
			&fsInfoMock{
				deviceDir:  "/rootfs/var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/5001/fs",
				deviceinfo: &fs.DeviceInfo{Device: "/dev/sda"},
			},
			false,
			false,
		},
		{
			mockcontainerdClient(testContainers, testStatus, testStats, testMounts, nil),
			&mockedMachineInfo{},
			&fsInfoMock{
				// The dummy dir will trigger an error in fsInfo.GetDirFsDevice, thus triggering error in fillDiskUsageInfo
				deviceDir:  "dummy dir",
				deviceinfo: &fs.DeviceInfo{Device: "/dev/sda"},
			},
			false,
			true, // fillDiskUsageInfo is expected to return error
		},
	} {
		handler, err := newContainerdContainerHandler(
			ts.client,
			testContainerName,
			ts.machineInfoFactory,
			ts.fsInfo,
			map[string]string{},
			ts.inHostNamespace,
			nil,
			metricSet,
		)
		// Regardless whether disk usage has error, the handler must be successfully created without error
		as.Nil(err)
		h := handler.(*containerdContainerHandler)

		if !ts.hasErr {
			mi := ts.machineInfoFactory.(*mockedMachineInfo)
			as.Equal(mi.machineInfo.Filesystems[0].Capacity, h.fsLimit)
			as.Equal(mi.machineInfo.Filesystems[0].Type, h.fsType)
			as.Equal(mi.machineInfo.Filesystems[0].Inodes, h.fsTotalInodes)

			stats := &info.ContainerStats{}

			err = h.getFsStats(stats)
			as.Nil(err)
			as.Equal(mi.machineInfo.Filesystems[0].Device, stats.Filesystem[0].Device)
		}
	}
}

func TestUsageProvider(t *testing.T) {
	as := assert.New(t)
	type testCase struct {
		client ContainerdClient
		fsInfo fs.FsInfo
	}

	for _, ts := range []testCase{
		{
			mockcontainerdClient(testContainers, testStatus, testStats, nil, nil),
			&fsInfoMock{},
		},
	} {
		up := fsUsageProvider{
			ctx:         context.Background(),
			containerID: testContainerID,
			client:      ts.client,
			fsInfo:      ts.fsInfo,
			logPath:     testLogPath,
		}
		fsUsage, err := up.Usage()
		as.Nil(err)
		as.EqualValues(testBaseUsage, fsUsage.BaseUsageBytes)
		as.EqualValues(testBaseUsage+testLogUsage, fsUsage.TotalUsageBytes)
		as.EqualValues(testBaseInodes, fsUsage.InodeUsage)
	}
}
