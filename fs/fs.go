// +build linux
//
// Provides Filesystem Stats
package fs

/*
 extern int getBytesFree(const char *path, unsigned long long *bytes);
 extern int getBytesTotal(const char *path, unsigned long long *bytes);
*/
import "C"

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/docker/docker/pkg/mount"
	"github.com/golang/glog"
)

type partition struct {
	mountpoint string
	major      uint
	minor      uint
}

type RealFsInfo struct {
	partitions map[string]partition
}

func NewFsInfo() (FsInfo, error) {
	mounts, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}
	partitions := make(map[string]partition, 0)
	for _, mount := range mounts {
		if !strings.HasPrefix(mount.Fstype, "ext") {
			continue
		}
		// Avoid bind mounts.
		if _, ok := partitions[mount.Source]; ok {
			continue
		}
		partitions[mount.Source] = partition{mount.Mountpoint, uint(mount.Major), uint(mount.Minor)}
	}
	return &RealFsInfo{partitions}, nil
}

func (self *RealFsInfo) GetFsInfoForPath(mountSet map[string]struct{}) ([]Fs, error) {
	filesystems := make([]Fs, 0)
	deviceSet := make(map[string]struct{})

	for device, partition := range self.partitions {
		_, hasMount := mountSet[partition.mountpoint]
		_, hasDevice := deviceSet[device]
		if mountSet == nil || hasMount && !hasDevice {
			total, free, err := getVfsStats(partition.mountpoint)
			if err != nil {
				glog.Errorf("Statvfs failed. Error: %v", err)
			} else {
				deviceSet[device] = struct{}{}
				deviceInfo := DeviceInfo{
					Device: device,
					Major:  uint(partition.major),
					Minor:  uint(partition.minor),
				}
				var diskStats DiskStats
				out, err := exec.Command("iostat", "-d", "-x", device).CombinedOutput()
				if err != nil {
					glog.Errorf("iostat failed on device %s. Error: %v", device, err)
				} else {
					diskStats, err = getDiskStats(string(out))
					if err != nil {
						glog.Errorf("Error parsing iostat output %s", out)
					}
				}

				fs := Fs{deviceInfo, total, free, diskStats}
				filesystems = append(filesystems, fs)
			}
		}
	}
	return filesystems, nil
}

func getDiskStats(ioStatOutput string) (DiskStats, error) {
	partitionRegex, _ := regexp.Compile("^sd[a-z]\\d\\s[\\d+\\.\\s+]+")
	var diskStats DiskStats
	for _, line := range strings.Split(ioStatOutput, "\n") {
		if !partitionRegex.MatchString(line) {
			continue
		}
		words := strings.Fields(line)

		// 8      50 sdd2 40 0 280 223 7 0 22 108 0 330 330
		offset := 1
		wordLength := len(words)
		var stats = make([]float64, wordLength-offset)
		var error error
		for i := offset; i < wordLength; i++ {
			stats[i-offset], error = strconv.ParseFloat(words[i], 64)
			if error != nil {
				return diskStats, error
			}
		}
		diskStats = DiskStats{
			ReadsMerged:    stats[0],
			WritesMerged:   stats[1],
			ReadsIssued:    stats[2],
			WritesIssued:   stats[3],
			SectorsRead:    stats[4],
			SectorsWritten: stats[5],
			AvgRequestSize: stats[6],
			AvgQueueLen:    stats[7],
			AvgWaitTime:    stats[8],
			AvgServiceTime: stats[9],
			PercentUtil:    stats[10],
		}
	}
	return diskStats, nil
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
	var buf syscall.Stat_t
	err := syscall.Stat(dir, &buf)
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

func (self *RealFsInfo) GetDirUsage(dir string) (uint64, error) {
	out, err := exec.Command("du", "-s", dir).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("du command failed on %s with output %s - %s", dir, out, err)
	}
	usageInKb, err := strconv.ParseUint(strings.Fields(string(out))[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse 'du' output %s - %s", out, err)
	}
	return usageInKb * 1024, nil
}

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
