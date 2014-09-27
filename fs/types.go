package fs

type FsStat struct {
	Name     string `json:"name"`
	Capacity uint64 `json:"capacity"`
	Free     uint64 `json:"free"`
}

type FsInfo interface {
	// Returns capacity and free space, in bytes, of all the ext2, ext3, ext4 filesystems used by container 'containerName'.
	GetFsStats(containerName string) ([]FsStat, error)
}
