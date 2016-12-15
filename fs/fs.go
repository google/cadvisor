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

// +build linux

// Provides Filesystem Stats
package fs

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/mount"
	"github.com/golang/glog"
	"github.com/google/cadvisor/devicemapper"
	dockerutil "github.com/google/cadvisor/utils/docker"
	zfs "github.com/mistifyio/go-zfs"
)

const (
	LabelSystemRoot   = "root"
	LabelDockerImages = "docker-images"
	LabelRktImages    = "rkt-images"
)

// The maximum number of `du` and `find` tasks that can be running at once.
const maxConcurrentOps = 20

// A pool for restricting the number of consecutive `du` and `find` tasks running.
var pool = make(chan struct{}, maxConcurrentOps)

func init() {
	for i := 0; i < maxConcurrentOps; i++ {
		releaseToken()
	}
}

func claimToken() {
	<-pool
}

func releaseToken() {
	pool <- struct{}{}
}

type partition struct {
	mountpoint string
	major      uint
	minor      uint
	fsType     string
	blockSize  uint
}

type RealFsInfo struct {
	// Map from block device path to partition information.
	partitions map[string]partition
	// Map from label to block device path.
	// Labels are intent-specific tags that are auto-detected.
	labels map[string]string
	// devicemapper client
	dmsetup devicemapper.DmsetupClient
}

type Context struct {
	// docker root directory.
	Docker  DockerContext
	RktPath string
}

type DockerContext struct {
	Root         string
	Driver       string
	DriverStatus map[string]string
}

func NewFsInfo(context Context) (FsInfo, error) {
	mounts, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}

	// Avoid devicemapper container mounts - these are tracked by the ThinPoolWatcher
	excluded := []string{fmt.Sprintf("%s/devicemapper/mnt", context.Docker.Root)}
	fsInfo := &RealFsInfo{
		partitions: processMounts(mounts, excluded),
		labels:     make(map[string]string, 0),
		dmsetup:    devicemapper.NewDmsetupClient(),
	}

	fsInfo.addRktImagesLabel(context, mounts)
	// need to call this before the log line below printing out the partitions, as this function may
	// add a "partition" for devicemapper to fsInfo.partitions
	fsInfo.addDockerImagesLabel(context, mounts)

	glog.Infof("Filesystem partitions: %+v", fsInfo.partitions)
	fsInfo.addSystemRootLabel(mounts)
	return fsInfo, nil
}

func processMounts(mounts []*mount.Info, excludedMountpointPrefixes []string) map[string]partition {
	partitions := make(map[string]partition, 0)

	supportedFsType := map[string]bool{
		// all ext systems are checked through prefix.
		"btrfs": true,
		"xfs":   true,
		"zfs":   true,
	}

	for _, mount := range mounts {
		if !strings.HasPrefix(mount.Fstype, "ext") && !supportedFsType[mount.Fstype] {
			continue
		}
		// Avoid bind mounts.
		if _, ok := partitions[mount.Source]; ok {
			continue
		}

		hasPrefix := false
		for _, prefix := range excludedMountpointPrefixes {
			if strings.HasPrefix(mount.Mountpoint, prefix) {
				hasPrefix = true
				break
			}
		}
		if hasPrefix {
			continue
		}

		partitions[mount.Source] = partition{
			fsType:     mount.Fstype,
			mountpoint: mount.Mountpoint,
			major:      uint(mount.Major),
			minor:      uint(mount.Minor),
		}
	}

	return partitions
}

// getDockerDeviceMapperInfo returns information about the devicemapper device and "partition" if
// docker is using devicemapper for its storage driver. If a loopback device is being used, don't
// return any information or error, as we want to report based on the actual partition where the
// loopback file resides, inside of the loopback file itself.
func (self *RealFsInfo) getDockerDeviceMapperInfo(context DockerContext) (string, *partition, error) {
	if context.Driver != DeviceMapper.String() {
		return "", nil, nil
	}

	dataLoopFile := context.DriverStatus[dockerutil.DriverStatusDataLoopFile]
	if len(dataLoopFile) > 0 {
		return "", nil, nil
	}

	dev, major, minor, blockSize, err := dockerDMDevice(context.DriverStatus, self.dmsetup)
	if err != nil {
		return "", nil, err
	}

	return dev, &partition{
		fsType:    DeviceMapper.String(),
		major:     major,
		minor:     minor,
		blockSize: blockSize,
	}, nil
}

// addSystemRootLabel attempts to determine which device contains the mount for /.
func (self *RealFsInfo) addSystemRootLabel(mounts []*mount.Info) {
	for _, m := range mounts {
		if m.Mountpoint == "/" {
			self.partitions[m.Source] = partition{
				fsType:     m.Fstype,
				mountpoint: m.Mountpoint,
				major:      uint(m.Major),
				minor:      uint(m.Minor),
			}
			self.labels[LabelSystemRoot] = m.Source
			return
		}
	}
}

// addDockerImagesLabel attempts to determine which device contains the mount for docker images.
func (self *RealFsInfo) addDockerImagesLabel(context Context, mounts []*mount.Info) {
	dockerDev, dockerPartition, err := self.getDockerDeviceMapperInfo(context.Docker)
	if err != nil {
		glog.Warningf("Could not get Docker devicemapper device: %v", err)
	}
	if len(dockerDev) > 0 && dockerPartition != nil {
		self.partitions[dockerDev] = *dockerPartition
		self.labels[LabelDockerImages] = dockerDev
	} else {
		self.updateContainerImagesPath(LabelDockerImages, mounts, getDockerImagePaths(context))
	}
}

func (self *RealFsInfo) addRktImagesLabel(context Context, mounts []*mount.Info) {
	if context.RktPath != "" {
		rktPath := context.RktPath
		rktImagesPaths := map[string]struct{}{
			"/": {},
		}
		for rktPath != "/" && rktPath != "." {
			rktImagesPaths[rktPath] = struct{}{}
			rktPath = filepath.Dir(rktPath)
		}
		self.updateContainerImagesPath(LabelRktImages, mounts, rktImagesPaths)
	}
}

// Generate a list of possible mount points for docker image management from the docker root directory.
// Right now, we look for each type of supported graph driver directories, but we can do better by parsing
// some of the context from `docker info`.
func getDockerImagePaths(context Context) map[string]struct{} {
	dockerImagePaths := map[string]struct{}{
		"/": {},
	}

	// TODO(rjnagal): Detect docker root and graphdriver directories from docker info.
	dockerRoot := context.Docker.Root
	for _, dir := range []string{"devicemapper", "btrfs", "aufs", "overlay", "zfs"} {
		dockerImagePaths[path.Join(dockerRoot, dir)] = struct{}{}
	}
	for dockerRoot != "/" && dockerRoot != "." {
		dockerImagePaths[dockerRoot] = struct{}{}
		dockerRoot = filepath.Dir(dockerRoot)
	}
	return dockerImagePaths
}

// This method compares the mountpoints with possible container image mount points. If a match is found,
// the label is added to the partition.
func (self *RealFsInfo) updateContainerImagesPath(label string, mounts []*mount.Info, containerImagePaths map[string]struct{}) {
	var useMount *mount.Info
	for _, m := range mounts {
		if _, ok := containerImagePaths[m.Mountpoint]; ok {
			if useMount == nil || (len(useMount.Mountpoint) < len(m.Mountpoint)) {
				useMount = m
			}
		}
	}
	if useMount != nil {
		self.partitions[useMount.Source] = partition{
			fsType:     useMount.Fstype,
			mountpoint: useMount.Mountpoint,
			major:      uint(useMount.Major),
			minor:      uint(useMount.Minor),
		}
		self.labels[label] = useMount.Source
	}
}

func (self *RealFsInfo) GetDeviceForLabel(label string) (string, error) {
	dev, ok := self.labels[label]
	if !ok {
		return "", fmt.Errorf("non-existent label %q", label)
	}
	return dev, nil
}

func (self *RealFsInfo) GetLabelsForDevice(device string) ([]string, error) {
	labels := []string{}
	for label, dev := range self.labels {
		if dev == device {
			labels = append(labels, label)
		}
	}
	return labels, nil
}

func (self *RealFsInfo) GetMountpointForDevice(dev string) (string, error) {
	p, ok := self.partitions[dev]
	if !ok {
		return "", fmt.Errorf("no partition info for device %q", dev)
	}
	return p.mountpoint, nil
}

func (self *RealFsInfo) GetFsInfoForPath(mountSet map[string]struct{}) ([]Fs, error) {
	filesystems := make([]Fs, 0)
	deviceSet := make(map[string]struct{})
	diskStatsMap, err := getDiskStatsMap("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	for device, partition := range self.partitions {
		_, hasMount := mountSet[partition.mountpoint]
		_, hasDevice := deviceSet[device]
		if mountSet == nil || (hasMount && !hasDevice) {
			var (
				err error
				fs  Fs
			)
			switch partition.fsType {
			case DeviceMapper.String():
				fs.Capacity, fs.Free, fs.Available, err = getDMStats(device, partition.blockSize)
				glog.V(5).Infof("got devicemapper fs capacity stats: capacity: %v free: %v available: %v:", fs.Capacity, fs.Free, fs.Available)
				fs.Type = DeviceMapper
			case ZFS.String():
				fs.Capacity, fs.Free, fs.Available, err = getZfstats(device)
				fs.Type = ZFS
			default:
				var inodes, inodesFree uint64
				fs.Capacity, fs.Free, fs.Available, inodes, inodesFree, err = getVfsStats(partition.mountpoint)
				fs.Inodes = &inodes
				fs.InodesFree = &inodesFree
				fs.Type = VFS
			}
			if err != nil {
				glog.Errorf("Stat fs failed. Error: %v", err)
			} else {
				deviceSet[device] = struct{}{}
				fs.DeviceInfo = DeviceInfo{
					Device: device,
					Major:  uint(partition.major),
					Minor:  uint(partition.minor),
				}
				fs.DiskStats = diskStatsMap[device]
				filesystems = append(filesystems, fs)
			}
		}
	}
	return filesystems, nil
}

var partitionRegex = regexp.MustCompile(`^(?:(?:s|xv)d[a-z]+\d*|dm-\d+)$`)

func getDiskStatsMap(diskStatsFile string) (map[string]DiskStats, error) {
	diskStatsMap := make(map[string]DiskStats)
	file, err := os.Open(diskStatsFile)
	if err != nil {
		if os.IsNotExist(err) {
			glog.Infof("not collecting filesystem statistics because file %q was not available", diskStatsFile)
			return diskStatsMap, nil
		}
		return nil, err
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if !partitionRegex.MatchString(words[2]) {
			continue
		}
		// 8      50 sdd2 40 0 280 223 7 0 22 108 0 330 330
		deviceName := path.Join("/dev", words[2])
		wordLength := len(words)
		offset := 3
		var stats = make([]uint64, wordLength-offset)
		if len(stats) < 11 {
			return nil, fmt.Errorf("could not parse all 11 columns of /proc/diskstats")
		}
		var error error
		for i := offset; i < wordLength; i++ {
			stats[i-offset], error = strconv.ParseUint(words[i], 10, 64)
			if error != nil {
				return nil, error
			}
		}
		diskStats := DiskStats{
			ReadsCompleted:  stats[0],
			ReadsMerged:     stats[1],
			SectorsRead:     stats[2],
			ReadTime:        stats[3],
			WritesCompleted: stats[4],
			WritesMerged:    stats[5],
			SectorsWritten:  stats[6],
			WriteTime:       stats[7],
			IoInProgress:    stats[8],
			IoTime:          stats[9],
			WeightedIoTime:  stats[10],
		}
		diskStatsMap[deviceName] = diskStats
	}
	return diskStatsMap, nil
}

func (self *RealFsInfo) GetGlobalFsInfo() ([]Fs, error) {
	return self.GetFsInfoForPath(nil)
}

func major(devNumber uint64) uint {
	return uint((devNumber >> 8) & 0xfff)
}

func minor(devNumber uint64) uint {
	return uint((devNumber & 0xff) | ((devNumber >> 12) & 0xfff00))
}

func (self *RealFsInfo) GetDirFsDevice(dir string) (*DeviceInfo, error) {
	buf := new(syscall.Stat_t)
	err := syscall.Stat(dir, buf)
	if err != nil {
		return nil, fmt.Errorf("stat failed on %s with error: %s", dir, err)
	}
	major := major(buf.Dev)
	minor := minor(buf.Dev)
	for device, partition := range self.partitions {
		if partition.major == major && partition.minor == minor {
			return &DeviceInfo{device, major, minor}, nil
		}
	}
	return nil, fmt.Errorf("could not find device with major: %d, minor: %d in cached partitions map", major, minor)
}

func (self *RealFsInfo) GetDirDiskUsage(dir string, timeout time.Duration) (uint64, error) {
	if dir == "" {
		return 0, fmt.Errorf("invalid directory")
	}
	claimToken()
	defer releaseToken()
	cmd := exec.Command("nice", "-n", "19", "du", "-s", dir)
	stdoutp, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to setup stdout for cmd %v - %v", cmd.Args, err)
	}
	stderrp, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to setup stderr for cmd %v - %v", cmd.Args, err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to exec du - %v", err)
	}
	timer := time.AfterFunc(timeout, func() {
		glog.Infof("killing cmd %v due to timeout(%s)", cmd.Args, timeout.String())
		cmd.Process.Kill()
	})
	stdoutb, souterr := ioutil.ReadAll(stdoutp)
	if souterr != nil {
		glog.Errorf("failed to read from stdout for cmd %v - %v", cmd.Args, souterr)
	}
	stderrb, _ := ioutil.ReadAll(stderrp)
	err = cmd.Wait()
	timer.Stop()
	if err != nil {
		return 0, fmt.Errorf("du command failed on %s with output stdout: %s, stderr: %s - %v", dir, string(stdoutb), string(stderrb), err)
	}
	stdout := string(stdoutb)
	usageInKb, err := strconv.ParseUint(strings.Fields(stdout)[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse 'du' output %s - %s", stdout, err)
	}
	return usageInKb * 1024, nil
}

func (self *RealFsInfo) GetDirInodeUsage(dir string, timeout time.Duration) (uint64, error) {
	if dir == "" {
		return 0, fmt.Errorf("invalid directory")
	}
	var stdout, stdwcerr, stdfinderr bytes.Buffer
	var err error
	claimToken()
	defer releaseToken()
	findCmd := exec.Command("find", dir, "-xdev", "-printf", ".")
	wcCmd := exec.Command("wc", "-c")
	if wcCmd.Stdin, err = findCmd.StdoutPipe(); err != nil {
		return 0, fmt.Errorf("failed to setup stdout for cmd %v - %v", findCmd.Args, err)
	}
	wcCmd.Stdout, wcCmd.Stderr, findCmd.Stderr = &stdout, &stdwcerr, &stdfinderr
	if err = findCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to exec cmd %v - %v; stderr: %v", findCmd.Args, err, stdfinderr.String())
	}

	if err = wcCmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to exec cmd %v - %v; stderr %v", wcCmd.Args, err, stdwcerr.String())
	}
	timer := time.AfterFunc(timeout, func() {
		glog.Infof("killing cmd %v, and cmd %v due to timeout(%s)", findCmd.Args, wcCmd.Args, timeout.String())
		wcCmd.Process.Kill()
		findCmd.Process.Kill()
	})
	err = findCmd.Wait()
	if err != nil {
		return 0, fmt.Errorf("cmd %v failed. stderr: %s; err: %v", findCmd.Args, stdfinderr.String(), err)
	}
	err = wcCmd.Wait()
	if err != nil {
		return 0, fmt.Errorf("cmd %v failed. stderr: %s; err: %v", wcCmd.Args, stdwcerr.String(), err)
	}
	timer.Stop()
	inodeUsage, err := strconv.ParseUint(strings.TrimSpace(stdout.String()), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse cmds: %v, %v output %s - %s", findCmd.Args, wcCmd.Args, stdout.String(), err)
	}
	return inodeUsage, nil
}

func getVfsStats(path string) (total uint64, free uint64, avail uint64, inodes uint64, inodesFree uint64, err error) {
	var s syscall.Statfs_t
	if err = syscall.Statfs(path, &s); err != nil {
		return 0, 0, 0, 0, 0, err
	}
	total = uint64(s.Frsize) * s.Blocks
	free = uint64(s.Frsize) * s.Bfree
	avail = uint64(s.Frsize) * s.Bavail
	inodes = uint64(s.Files)
	inodesFree = uint64(s.Ffree)
	return total, free, avail, inodes, inodesFree, nil
}

// Devicemapper thin provisioning is detailed at
// https://www.kernel.org/doc/Documentation/device-mapper/thin-provisioning.txt
func dockerDMDevice(driverStatus map[string]string, dmsetup devicemapper.DmsetupClient) (string, uint, uint, uint, error) {
	poolName, ok := driverStatus[dockerutil.DriverStatusPoolName]
	if !ok || len(poolName) == 0 {
		return "", 0, 0, 0, fmt.Errorf("Could not get dm pool name")
	}

	out, err := dmsetup.Table(poolName)
	if err != nil {
		return "", 0, 0, 0, err
	}

	major, minor, dataBlkSize, err := parseDMTable(string(out))
	if err != nil {
		return "", 0, 0, 0, err
	}

	return poolName, major, minor, dataBlkSize, nil
}

// parseDMTable parses a single line of `dmsetup table` output and returns the
// major device, minor device, block size, and an error.
func parseDMTable(dmTable string) (uint, uint, uint, error) {
	dmTable = strings.Replace(dmTable, ":", " ", -1)
	dmFields := strings.Fields(dmTable)

	if len(dmFields) < 8 {
		return 0, 0, 0, fmt.Errorf("Invalid dmsetup status output: %s", dmTable)
	}

	major, err := strconv.ParseUint(dmFields[5], 10, 32)
	if err != nil {
		return 0, 0, 0, err
	}
	minor, err := strconv.ParseUint(dmFields[6], 10, 32)
	if err != nil {
		return 0, 0, 0, err
	}
	dataBlkSize, err := strconv.ParseUint(dmFields[7], 10, 32)
	if err != nil {
		return 0, 0, 0, err
	}

	return uint(major), uint(minor), uint(dataBlkSize), nil
}

func getDMStats(poolName string, dataBlkSize uint) (uint64, uint64, uint64, error) {
	out, err := exec.Command("dmsetup", "status", poolName).Output()
	if err != nil {
		return 0, 0, 0, err
	}

	used, total, err := parseDMStatus(string(out))
	if err != nil {
		return 0, 0, 0, err
	}

	used *= 512 * uint64(dataBlkSize)
	total *= 512 * uint64(dataBlkSize)
	free := total - used

	return total, free, free, nil
}

func parseDMStatus(dmStatus string) (uint64, uint64, error) {
	dmStatus = strings.Replace(dmStatus, "/", " ", -1)
	dmFields := strings.Fields(dmStatus)

	if len(dmFields) < 8 {
		return 0, 0, fmt.Errorf("Invalid dmsetup status output: %s", dmStatus)
	}

	used, err := strconv.ParseUint(dmFields[6], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	total, err := strconv.ParseUint(dmFields[7], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return used, total, nil
}

// getZfstats returns ZFS mount stats using zfsutils
func getZfstats(poolName string) (uint64, uint64, uint64, error) {
	dataset, err := zfs.GetDataset(poolName)
	if err != nil {
		return 0, 0, 0, err
	}

	total := dataset.Used + dataset.Avail + dataset.Usedbydataset

	return total, dataset.Avail, dataset.Avail, nil
}
