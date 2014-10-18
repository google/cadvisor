package fs

type DeviceInfo struct {
	Device string
	Major  uint
	Minor  uint
}

type Fs struct {
	DeviceInfo
	Capacity  uint64
	Free      uint64
	DiskStats DiskStats
}

type DiskStats struct {
	ReadsMerged    float64
	WritesMerged   float64
	ReadsIssued    float64
	WritesIssued   float64
	SectorsRead    float64
	SectorsWritten float64
	AvgRequestSize float64
	AvgQueueLen    float64
	AvgWaitTime    float64
	AvgServiceTime float64
	PercentUtil    float64
}

type FsInfo interface {
	// Returns capacity and free space, in bytes, of all the ext2, ext3, ext4 filesystems on the host.
	GetGlobalFsInfo() ([]Fs, error)

	// Returns capacity and free space, in bytes, of the set of mounts passed.
	GetFsInfoForPath(mountSet map[string]struct{}) ([]Fs, error)

	// Returns number of bytes occupied by 'dir'.
	GetDirUsage(dir string) (uint64, error)

	// Returns the block device info of the filesystem on which 'dir' resides.
	GetDirFsDevice(dir string) (*DeviceInfo, error)
}
