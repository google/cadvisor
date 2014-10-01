package fs

import "github.com/google/cadvisor/info"

type FsInfo interface {
	// Returns capacity and free space, in bytes, of all the ext2, ext3, ext4 filesystems on the host.
	GetFsStats() ([]info.FsStats, error)
}
