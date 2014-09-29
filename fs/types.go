package fs

type FsStat struct {
	Device   string `json:"device,omitempty"`
	Major    uint   `json:"major"`
	Minor    uint   `json:"minor"`
	Capacity uint64 `json:"capacity"`
	Free     uint64 `json:"free"`
}

type FsInfo interface {
	// Returns capacity and free space, in bytes, of all the ext2, ext3, ext4 filesystems on the host.
	GetFsStats() ([]FsStat, error)
}
