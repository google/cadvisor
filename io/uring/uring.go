package uring

import (
	"fmt"
	"io"
	"k8s.io/klog/v2"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Params is equivalent of io_uring_params
type Params struct {
	SQEntries    uint32
	CQEntries    uint32
	Flags        ParamsFlags
	SQThreadCpu  uint32
	SQThreadIdle uint32
	Features     uint32
	WQFd         uint32
	Resv         uint32
	SQRingOffsets
	CQRingOffsets
}

type ParamsFlags uint32

const (
	SetupIopoll    ParamsFlags = 1 << 0
	SetupSqpoll    ParamsFlags = 1 << 1
	SetupSqAff     ParamsFlags = 1 << 2
	SetupCqsize    ParamsFlags = 1 << 3
	SetupClamp     ParamsFlags = 1 << 4
	SetupAttachWq  ParamsFlags = 1 << 5
	SetupRDisabled ParamsFlags = 1 << 6
	SetupSubmitAll ParamsFlags = 1 << 7
)

// SQRingOffsets is equivalent of  io_sqring_offsets.
type SQRingOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Flags       uint32
	Dropped     uint32
	Array       uint32
	Resv1       uint32
	Resv2       uint64
}

// CQRingOffsets is equivalent of io_cqring_offsets.
type CQRingOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Overflow    uint32
	CQes        uint32
	Flags       uint32
	Resv1       uint32
	Resv2       uint64
}

type Op uint32

// These are values of enum io_uring_op from Linux 6.2
const (
	OpNop Op = iota
	OpReadv
	OpWritev
	OpFsync
	OpReadFixed
	OpWriteFixed
	OpPollAdd
	OpPollRemove
	OpSyncFileRange
	OpSendmsg
	OpRecvmsg
	OpTimeout
	OpTimeoutRemove
	OpAccept
	OpAsyncCancel
	OpLinkTimeout
	OpConnect
	OpFallocate
	OpOpenat
	OpClose
	OpFilesUpdate
	OpStatx
	OpRead
	OpWrite
	OpFadvise
	OpMadvise
	OpSend
	OpRecv
	OpOpenat2
	OpEpollCtl
	OpSplice
	OpProvideBuffers
	OpRemoveBuffers
	OpTee
	OpShutdown
	OpRenameat
	OpUnlinkat
	OpMkdirat
	OpSymlinkat
	OpLinkat
	OpMsgRing
	OpFsetxattr
	OpSetxattr
	OpFgetxattr
	OpGetxattr
	OpSocket
	OpUringCmd
	OpSendZc
	OpSendmsgZc
	ngOpLast
)

type RegisterOp uint32

const (
	RegisterBuffers RegisterOp = iota
	UnregisterBuffers
	RegisterFiles
	UnregisterFiles
	RegisterEventfd
	UnregisterEventfd
	RegisterFilesUpdate
	RegisterEventfdAsync
	RegisterProbe
	RegisterPersonality
	UnregisterPersonality
	RegisterRestrictions
	RegisterEnableRings
	RegisterFiles2
	RegisterFilesUpdate2
	RegisterBuffers2
	RegisterBuffersUpdate
	RegisterIowqAff
	UnregisterIowqAff
	RegisterIowqMaxWorkers
	RegisterRingFds
	UnregisterRingFds
	RegisterPbufRing
	UnregisterPbufRing
	RegisterSyncCancel
	RegisterFileAllocRange
	RegisterLast
)

type cqe struct {
	userData uint64 /* sqe->data submission passed back */
	res      int32  /* result code for this event */
	flags    uint32
	bigCqe   uint64
}

// These constants are missing on arm64.
const (
	setup    uintptr = 425
	enter    uintptr = 426
	register uintptr = 427
)

type Ring struct {
	sq     *writer
	cq     *reader
	fd     *os.File
	params Params
}

func New(params Params, entries int) (*Ring, error) {
	r := &Ring{
		params: params,
	}

	r1, r2, errNo := unix.RawSyscall(setup, uintptr(entries), uintptr(unsafe.Pointer(&r.params)), uintptr(0))
	if errNo != 0 {
		return nil, fmt.Errorf("io_uring_setup failed: %w", errNo)
	}
	fmt.Println("inside: io_uring_setup", r1, r2, errNo)
	r.fd = os.NewFile(r1, "io_uring_fd")

	// mmap the junk
	var zero uintptr = 0
	sq, err := syscall.Mmap(0, int64(r.params.SQRingOffsets.Array), int(r.params.SQEntries*uint32(unsafe.Sizeof(zero))), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED|syscall.MAP_POPULATE)
	if err != nil {
		return nil, fmt.Errorf("submission queue mapping failed: %w", err)
	}
	r.sq = &writer{buffer: sq, capacity: len(sq)}
	cq, err := syscall.Mmap(0, int64(r.params.CQRingOffsets.CQes), int(r.params.CQEntries*uint32(unsafe.Sizeof(cqe{}))), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED|syscall.MAP_POPULATE)
	if err != nil {
		r.Destroy()
		return nil, fmt.Errorf("completion queue mapping failed: %w", err)
	}
	r.cq = &reader{buffer: cq}

	return r, nil
}

type writer struct {
	buffer   []byte
	capacity int
	lock     sync.Mutex
}

func (w *writer) Write(p []byte) (int, error) {
	var err error
	w.lock.Lock()
	defer w.lock.Unlock()
	before := len(w.buffer)
	if before+len(p) > w.capacity {
		p = p[:w.capacity-before]
		err = io.EOF
	}
	w.buffer = append(w.buffer, p...)
	after := len(w.buffer)
	return after - before, err
}

type reader struct {
	buffer []byte
	offset int
	lock   sync.Mutex
}

func (r *reader) Read(p []byte) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	pLen := len(p)
	chunk := r.buffer[r.offset:pLen]
	b := append(p, chunk...)
	r.offset = r.offset + pLen - 1
	p = b
	if pLen > len(chunk) {
		return len(chunk), io.EOF
	}
	return pLen, nil
}

// unsigned int fd, unsigned int to_submit, unsigned int min_complete, unsigned int flags, sigset_t *sig
func (r *Ring) Enter() {

}

// unsigned int fd, unsigned int opcode, void *arg, unsigned int nr_args
func (r *Ring) Register(opcode RegisterOp, args []syscall.Iovec) error {
	r1, r2, err := unix.RawSyscall6(register, r.fd.Fd(), uintptr(opcode), uintptr(unsafe.Pointer(&args)), uintptr(len(args)), 0, 0)
	fmt.Println("inside: io_uring_register", r1, r2, err)
	if err != 0 {
		return fmt.Errorf("io_uring_register failed: %w", err)
	}
	return nil
}

func (r *Ring) Destroy() {
	if r.sq != nil {
		err := syscall.Munmap(r.sq.buffer)
		if err != nil {
			klog.V(4).Infof("munmap of submission queue failed: %s", err)
		}
		r.sq = nil
	}
	if r.cq != nil {
		err := syscall.Munmap(r.cq.buffer)
		if err != nil {
			klog.V(4).Infof("munmap of completion queue failed: %s", err)
		}
		r.cq = nil
	}
}
