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

type FsInfoImpl struct{}

func NewFsInfo() FsInfo {
	return &FsInfoImpl{}
}

func (*FsInfoImpl) GetFsStats() ([]FsStat, error) {
	filesystems := make([]FsStat, 0)
	mounts, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}
	processedPartitions := make(map[string]bool, 0)
	for _, mount := range mounts {
		if !strings.HasPrefix(mount.Fstype, "ext") {
			continue
		}
		// Avoid bind mounts.
		if _, ok := processedPartitions[mount.Source]; ok {
			continue
		}
		total, free, err := getVfsStats(mount.Mountpoint)
		if err != nil {
			glog.Errorf("Statvfs failed. Error: %v", err)
		} else {
			glog.V(1).Infof("%s is an %s partition at %s. Total: %d, Free: %d", mount.Source, mount.Fstype, mount.Mountpoint, total, free)
			fsStat := FsStat{
				Device:   mount.Source,
				Major:    uint(mount.Major),
				Minor:    uint(mount.Minor),
				Capacity: total,
				Free:     free,
			}
			filesystems = append(filesystems, fsStat)
			processedPartitions[mount.Source] = true
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
