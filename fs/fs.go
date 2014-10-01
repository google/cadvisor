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
	"strings"
	"syscall"
	"unsafe"

	"github.com/docker/docker/pkg/mount"
	"github.com/golang/glog"
)

type partition struct {
	mountpoint string
	major      uint32
	minor      uint32
}

type FsInfoImpl struct {
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
		partitions[mount.Source] = partition{mount.Mountpoint, uint32(mount.Major), uint32(mount.Minor)}
	}
	return &FsInfoImpl{partitions}, nil
}

func (self *FsInfoImpl) GetFsStats() ([]FsStats, error) {
	filesystems := make([]FsStats, 0)
	for device, partition := range self.partitions {
		total, free, err := getVfsStats(partition.mountpoint)
		if err != nil {
			glog.Errorf("Statvfs failed. Error: %v", err)
		} else {
			fsStat := FsStats{
				Device:   device,
				Major:    uint(partition.major),
				Minor:    uint(partition.minor),
				Capacity: total,
				Free:     free,
			}
			filesystems = append(filesystems, fsStat)
		}
	}
	return filesystems, nil
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
