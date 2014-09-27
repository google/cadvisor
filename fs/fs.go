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
	"syscall"
	"unsafe"

	"github.com/docker/docker/pkg/mount"
	"github.com/golang/glog"
)

const EXT_SUPER_MAGIC = 0xEF53

type FsInfoImpl struct{}

func NewFsInfo() FsInfo {
	return &FsInfoImpl{}
}

func (*FsInfoImpl) GetFsStats(containerName string) ([]FsStat, error) {
	filesystems := make([]FsStat, 0)
	if containerName != "/" {
		return filesystems, nil
	}
	mounts, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}
	for _, mount := range mounts {
		if !strings.HasPrefix("ext", mount.FsType) || mount.Mountpoint != mount.Root {
			continue
		}
		total, free, err := getVfsStats(mount.Mountpoint)
		if err != nil {
			glog.Errorf("Statvfs failed. Error: %v", err)
		} else {
			glog.V(1).Infof("%s is an ext partition at %s. Total: %d, Free: %d", mount.Source, mount.Mountpoint, total, free)
			filesystems = append(filesystems, FsStat{mount.Source, total, free})
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
