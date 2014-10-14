package fs

type DeviceInfo struct {
	Device string
	Major  uint
	Minor  uint
}

type Fs struct {
	DeviceInfo
	Capacity uint64
	Free     uint64
}

type FsInfo interface {
	// Returns capacity and free space, in bytes, of all the ext2, ext3, ext4 filesystems on the host.
	GetGlobalFsInfo() ([]Fs, error)

	// Returns number of bytes occupied by 'dir'.
	GetDirUsage(dir string) (uint64, error)

	// Returns the block device info of the filesystem on which 'dir' resides.
	GetDirFsDevice(dir string) (*DeviceInfo, error)
}
