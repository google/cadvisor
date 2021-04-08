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

// Handler for Docker containers.
package docker

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/common"
	dockerutil "github.com/google/cadvisor/container/docker/utils"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/devicemapper"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/zfs"

	dockercontainer "github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"
)

const (
	// The read write layers exist here.
	aufsRWLayer     = "diff"
	overlayRWLayer  = "upper"
	overlay2RWLayer = "diff"

	// Path to the directory where docker stores log files if the json logging driver is enabled.
	pathToContainersDir = "containers"
)

type dockerContainerHandler struct {
	// machineInfoFactory provides info.MachineInfo
	machineInfoFactory info.MachineInfoFactory

	// Absolute path to the cgroup hierarchies of this container.
	// (e.g.: "cpu" -> "/sys/fs/cgroup/cpu/test")
	cgroupPaths map[string]string

	// the docker storage driver
	storageDriver    storageDriver
	fsInfo           fs.FsInfo
	rootfsStorageDir string

	// Time at which this container was created.
	creationTime time.Time

	// Metadata associated with the container.
	envs   map[string]string
	labels map[string]string

	// Image name used for this container.
	image string

	// The network mode of the container
	networkMode dockercontainer.NetworkMode

	// Filesystem handler.
	fsHandler common.FsHandler

	// The IP address of the container
	ipAddress string

	includedMetrics container.MetricSet

	// the devicemapper poolname
	poolName string

	// zfsParent is the parent for docker zfs
	zfsParent string

	// Reference to the container
	reference info.ContainerReference

	libcontainerHandler *containerlibcontainer.Handler
}

var _ container.ContainerHandler = &dockerContainerHandler{}

func getRwLayerID(containerID, storageDir string, sd storageDriver, dockerVersion []int) (string, error) {
	const (
		// Docker version >=1.10.0 have a randomized ID for the root fs of a container.
		randomizedRWLayerMinorVersion = 10
		rwLayerIDFile                 = "mount-id"
	)
	if (dockerVersion[0] <= 1) && (dockerVersion[1] < randomizedRWLayerMinorVersion) {
		return containerID, nil
	}

	bytes, err := ioutil.ReadFile(path.Join(storageDir, "image", string(sd), "layerdb", "mounts", containerID, rwLayerIDFile))
	if err != nil {
		return "", fmt.Errorf("failed to identify the read-write layer ID for container %q. - %v", containerID, err)
	}
	return string(bytes), err
}

// newDockerContainerHandler returns a new container.ContainerHandler
func newDockerContainerHandler(
	client *docker.Client,
	name string,
	machineInfoFactory info.MachineInfoFactory,
	fsInfo fs.FsInfo,
	storageDriver storageDriver,
	storageDir string,
	cgroupSubsystems *containerlibcontainer.CgroupSubsystems,
	inHostNamespace bool,
	metadataEnvs []string,
	dockerVersion []int,
	includedMetrics container.MetricSet,
	thinPoolName string,
	thinPoolWatcher *devicemapper.ThinPoolWatcher,
	zfsWatcher *zfs.ZfsWatcher,
) (container.ContainerHandler, error) {
	// Create the cgroup paths.
	cgroupPaths := common.MakeCgroupPaths(cgroupSubsystems.MountPoints, name)

	// Generate the equivalent cgroup manager for this container.
	cgroupManager, err := containerlibcontainer.NewCgroupManager(name, cgroupPaths)
	if err != nil {
		return nil, err
	}

	rootFs := "/"
	if !inHostNamespace {
		rootFs = "/rootfs"
		storageDir = path.Join(rootFs, storageDir)
	}

	id := ContainerNameToDockerId(name)

	// Add the Containers dir where the log files are stored.
	// FIXME: Give `otherStorageDir` a more descriptive name.
	otherStorageDir := path.Join(storageDir, pathToContainersDir, id)

	rwLayerID, err := getRwLayerID(id, storageDir, storageDriver, dockerVersion)
	if err != nil {
		return nil, err
	}

	// Determine the rootfs storage dir OR the pool name to determine the device.
	// For devicemapper, we only need the thin pool name, and that is passed in to this call
	var (
		rootfsStorageDir string
		zfsFilesystem    string
		zfsParent        string
	)
	switch storageDriver {
	case aufsStorageDriver:
		rootfsStorageDir = path.Join(storageDir, string(aufsStorageDriver), aufsRWLayer, rwLayerID)
	case overlayStorageDriver:
		rootfsStorageDir = path.Join(storageDir, string(storageDriver), rwLayerID, overlayRWLayer)
	case overlay2StorageDriver:
		rootfsStorageDir = path.Join(storageDir, string(storageDriver), rwLayerID, overlay2RWLayer)
	case vfsStorageDriver:
		rootfsStorageDir = path.Join(storageDir)
	case zfsStorageDriver:
		status, err := Status()
		if err != nil {
			return nil, fmt.Errorf("unable to determine docker status: %v", err)
		}
		zfsParent = status.DriverStatus[dockerutil.DriverStatusParentDataset]
		zfsFilesystem = path.Join(zfsParent, rwLayerID)
	}

	// We assume that if Inspect fails then the container is not known to docker.
	ctnr, err := client.ContainerInspect(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %q: %v", id, err)
	}

	// TODO: extract object mother method
	handler := &dockerContainerHandler{
		machineInfoFactory: machineInfoFactory,
		cgroupPaths:        cgroupPaths,
		fsInfo:             fsInfo,
		storageDriver:      storageDriver,
		poolName:           thinPoolName,
		rootfsStorageDir:   rootfsStorageDir,
		envs:               make(map[string]string),
		labels:             ctnr.Config.Labels,
		includedMetrics:    includedMetrics,
		zfsParent:          zfsParent,
	}
	// Timestamp returned by Docker is in time.RFC3339Nano format.
	handler.creationTime, err = time.Parse(time.RFC3339Nano, ctnr.Created)
	if err != nil {
		// This should not happen, report the error just in case
		return nil, fmt.Errorf("failed to parse the create timestamp %q for container %q: %v", ctnr.Created, id, err)
	}
	handler.libcontainerHandler = containerlibcontainer.NewHandler(cgroupManager, rootFs, ctnr.State.Pid, includedMetrics)

	// Add the name and bare ID as aliases of the container.
	handler.reference = info.ContainerReference{
		Id:        id,
		Name:      name,
		Aliases:   []string{strings.TrimPrefix(ctnr.Name, "/"), id},
		Namespace: DockerNamespace,
	}
	handler.image = ctnr.Config.Image
	handler.networkMode = ctnr.HostConfig.NetworkMode
	// Only adds restartcount label if it's greater than 0
	if ctnr.RestartCount > 0 {
		handler.labels["restartcount"] = strconv.Itoa(ctnr.RestartCount)
	}

	// Obtain the IP address for the container.
	// If the NetworkMode starts with 'container:' then we need to use the IP address of the container specified.
	// This happens in cases such as kubernetes where the containers doesn't have an IP address itself and we need to use the pod's address
	ipAddress := ctnr.NetworkSettings.IPAddress
	networkMode := string(ctnr.HostConfig.NetworkMode)
	if ipAddress == "" && strings.HasPrefix(networkMode, "container:") {
		containerID := strings.TrimPrefix(networkMode, "container:")
		c, err := client.ContainerInspect(context.Background(), containerID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container %q: %v", id, err)
		}
		ipAddress = c.NetworkSettings.IPAddress
	}

	handler.ipAddress = ipAddress

	if includedMetrics.Has(container.DiskUsageMetrics) {
		handler.fsHandler = &dockerFsHandler{
			fsHandler:       common.NewFsHandler(common.DefaultPeriod, rootfsStorageDir, otherStorageDir, fsInfo),
			thinPoolWatcher: thinPoolWatcher,
			zfsWatcher:      zfsWatcher,
			deviceID:        ctnr.GraphDriver.Data["DeviceId"],
			zfsFilesystem:   zfsFilesystem,
		}
	}

	// split env vars to get metadata map.
	for _, exposedEnv := range metadataEnvs {
		if exposedEnv == "" {
			// if no dockerEnvWhitelist provided, len(metadataEnvs) == 1, metadataEnvs[0] == ""
			continue
		}

		for _, envVar := range ctnr.Config.Env {
			if envVar != "" {
				splits := strings.SplitN(envVar, "=", 2)
				if len(splits) == 2 && strings.HasPrefix(splits[0], exposedEnv) {
					handler.envs[strings.ToLower(splits[0])] = splits[1]
				}
			}
		}
	}

	return handler, nil
}

// dockerFsHandler is a composite FsHandler implementation the incorporates
// the common fs handler, a devicemapper ThinPoolWatcher, and a zfsWatcher
type dockerFsHandler struct {
	fsHandler common.FsHandler

	// thinPoolWatcher is the devicemapper thin pool watcher
	thinPoolWatcher *devicemapper.ThinPoolWatcher
	// deviceID is the id of the container's fs device
	deviceID string

	// zfsWatcher is the zfs filesystem watcher
	zfsWatcher *zfs.ZfsWatcher
	// zfsFilesystem is the docker zfs filesystem
	zfsFilesystem string
}

var _ common.FsHandler = &dockerFsHandler{}

func (h *dockerFsHandler) Start() {
	h.fsHandler.Start()
}

func (h *dockerFsHandler) Stop() {
	h.fsHandler.Stop()
}

func (h *dockerFsHandler) Usage() common.FsUsage {
	usage := h.fsHandler.Usage()

	// When devicemapper is the storage driver, the base usage of the container comes from the thin pool.
	// We still need the result of the fsHandler for any extra storage associated with the container.
	// To correctly factor in the thin pool usage, we should:
	// * Usage the thin pool usage as the base usage
	// * Calculate the overall usage by adding the overall usage from the fs handler to the thin pool usage
	if h.thinPoolWatcher != nil {
		thinPoolUsage, err := h.thinPoolWatcher.GetUsage(h.deviceID)
		if err != nil {
			// TODO: ideally we should keep track of how many times we failed to get the usage for this
			// device vs how many refreshes of the cache there have been, and display an error e.g. if we've
			// had at least 1 refresh and we still can't find the device.
			klog.V(5).Infof("unable to get fs usage from thin pool for device %s: %v", h.deviceID, err)
		} else {
			usage.BaseUsageBytes = thinPoolUsage
			usage.TotalUsageBytes += thinPoolUsage
		}
	}

	if h.zfsWatcher != nil {
		zfsUsage, err := h.zfsWatcher.GetUsage(h.zfsFilesystem)
		if err != nil {
			klog.V(5).Infof("unable to get fs usage from zfs for filesystem %s: %v", h.zfsFilesystem, err)
		} else {
			usage.BaseUsageBytes = zfsUsage
			usage.TotalUsageBytes += zfsUsage
		}
	}
	return usage
}

func (h *dockerContainerHandler) Start() {
	if h.fsHandler != nil {
		h.fsHandler.Start()
	}
}

func (h *dockerContainerHandler) Cleanup() {
	if h.fsHandler != nil {
		h.fsHandler.Stop()
	}
}

func (h *dockerContainerHandler) ContainerReference() (info.ContainerReference, error) {
	return h.reference, nil
}

func (h *dockerContainerHandler) needNet() bool {
	if h.includedMetrics.Has(container.NetworkUsageMetrics) {
		return !h.networkMode.IsContainer()
	}
	return false
}

func (h *dockerContainerHandler) GetSpec() (info.ContainerSpec, error) {
	hasFilesystem := h.includedMetrics.Has(container.DiskUsageMetrics)
	spec, err := common.GetSpec(h.cgroupPaths, h.machineInfoFactory, h.needNet(), hasFilesystem)

	spec.Labels = h.labels
	spec.Envs = h.envs
	spec.Image = h.image
	spec.CreationTime = h.creationTime

	return spec, err
}

func (h *dockerContainerHandler) getFsStats(stats *info.ContainerStats) error {
	mi, err := h.machineInfoFactory.GetMachineInfo()
	if err != nil {
		return err
	}

	if h.includedMetrics.Has(container.DiskIOMetrics) {
		common.AssignDeviceNamesToDiskStats((*common.MachineInfoNamer)(mi), &stats.DiskIo)
	}

	if !h.includedMetrics.Has(container.DiskUsageMetrics) {
		return nil
	}
	var device string
	switch h.storageDriver {
	case devicemapperStorageDriver:
		// Device has to be the pool name to correlate with the device name as
		// set in the machine info filesystems.
		device = h.poolName
	case aufsStorageDriver, overlayStorageDriver, overlay2StorageDriver, vfsStorageDriver:
		deviceInfo, err := h.fsInfo.GetDirFsDevice(h.rootfsStorageDir)
		if err != nil {
			return fmt.Errorf("unable to determine device info for dir: %v: %v", h.rootfsStorageDir, err)
		}
		device = deviceInfo.Device
	case zfsStorageDriver:
		device = h.zfsParent
	default:
		return nil
	}

	var (
		limit  uint64
		fsType string
	)

	var fsInfo *info.FsInfo

	// Docker does not impose any filesystem limits for containers. So use capacity as limit.
	for _, fs := range mi.Filesystems {
		if fs.Device == device {
			limit = fs.Capacity
			fsType = fs.Type
			fsInfo = &fs
			break
		}
	}

	fsStat := info.FsStats{Device: device, Type: fsType, Limit: limit}
	usage := h.fsHandler.Usage()
	fsStat.BaseUsage = usage.BaseUsageBytes
	fsStat.Usage = usage.TotalUsageBytes
	fsStat.Inodes = usage.InodeUsage

	if fsInfo != nil {
		fileSystems, err := h.fsInfo.GetGlobalFsInfo()

		if err == nil {
			addDiskStats(fileSystems, fsInfo, &fsStat)
		} else {
			klog.Errorf("Unable to obtain diskstats for filesystem %s: %v", fsStat.Device, err)
		}
	}

	stats.Filesystem = append(stats.Filesystem, fsStat)

	return nil
}

func addDiskStats(fileSystems []fs.Fs, fsInfo *info.FsInfo, fsStats *info.FsStats) {
	if fsInfo == nil {
		return
	}

	for _, fileSys := range fileSystems {
		if fsInfo.DeviceMajor == fileSys.DiskStats.Major &&
			fsInfo.DeviceMinor == fileSys.DiskStats.Minor {
			fsStats.ReadsCompleted = fileSys.DiskStats.ReadsCompleted
			fsStats.ReadsMerged = fileSys.DiskStats.ReadsMerged
			fsStats.SectorsRead = fileSys.DiskStats.SectorsRead
			fsStats.ReadTime = fileSys.DiskStats.ReadTime
			fsStats.WritesCompleted = fileSys.DiskStats.WritesCompleted
			fsStats.WritesMerged = fileSys.DiskStats.WritesMerged
			fsStats.SectorsWritten = fileSys.DiskStats.SectorsWritten
			fsStats.WriteTime = fileSys.DiskStats.WriteTime
			fsStats.IoInProgress = fileSys.DiskStats.IoInProgress
			fsStats.IoTime = fileSys.DiskStats.IoTime
			fsStats.WeightedIoTime = fileSys.DiskStats.WeightedIoTime
			break
		}
	}
}

// TODO(vmarmol): Get from libcontainer API instead of cgroup manager when we don't have to support older Dockers.
func (h *dockerContainerHandler) GetStats() (*info.ContainerStats, error) {
	stats, err := h.libcontainerHandler.GetStats()
	if err != nil {
		return stats, err
	}
	// Clean up stats for containers that don't have their own network - this
	// includes containers running in Kubernetes pods that use the network of the
	// infrastructure container. This stops metrics being reported multiple times
	// for each container in a pod.
	if !h.needNet() {
		stats.Network = info.NetworkStats{}
	}

	// Get filesystem stats.
	err = h.getFsStats(stats)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (h *dockerContainerHandler) ListContainers(listType container.ListType) ([]info.ContainerReference, error) {
	// No-op for Docker driver.
	return []info.ContainerReference{}, nil
}

func (h *dockerContainerHandler) GetCgroupPath(resource string) (string, error) {
	path, ok := h.cgroupPaths[resource]
	if !ok {
		return "", fmt.Errorf("could not find path for resource %q for container %q", resource, h.reference.Name)
	}
	return path, nil
}

func (h *dockerContainerHandler) GetContainerLabels() map[string]string {
	return h.labels
}

func (h *dockerContainerHandler) GetContainerIPAddress() string {
	return h.ipAddress
}

func (h *dockerContainerHandler) ListProcesses(listType container.ListType) ([]int, error) {
	return h.libcontainerHandler.GetProcesses()
}

func (h *dockerContainerHandler) Exists() bool {
	return common.CgroupExists(h.cgroupPaths)
}

func (h *dockerContainerHandler) Type() container.ContainerType {
	return container.ContainerTypeDocker
}
