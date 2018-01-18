package mountcap

import (
	"github.com/golang/glog"
	syscall "golang.org/x/sys/unix"
	"os"
	"unsafe"
)

func PollMount(changed chan bool, quit chan error) {
	f, _ := os.Open("/proc/self/mountinfo")
	defer f.Close()
	fdOri := f.Fd()
	var fd *int32 = (*int32)(unsafe.Pointer(&fdOri))
	pollFds := make([]syscall.PollFd, 1)
	pollFds[0] = syscall.PollFd{
		Fd:      *fd,
		Events:  syscall.POLLERR | syscall.POLLPRI,
		Revents: 0,
	}
	for {
		//set a minute so that this loop will check if it should exit
		ret, _ := syscall.Poll(pollFds, 60000)
		if ret >= 0 {
			if (pollFds[0].Revents & syscall.POLLERR) == 8 {
				changed <- true
			}
		}
		pollFds[0].Revents = 0

		select {
		case <-quit:
			glog.Infoln("recive quit msg.")
			return
		default:
			//do nothing and go next poll.
		}
	}
}

func PollMountEver() bool {
	f, _ := os.Open("/proc/self/mountinfo")
	defer f.Close()
	fdOri := f.Fd()
	var fd *int32 = (*int32)(unsafe.Pointer(&fdOri))
	pollFds := make([]syscall.PollFd, 1)
	pollFds[0] = syscall.PollFd{
		Fd:      *fd,
		Events:  syscall.POLLERR | syscall.POLLPRI,
		Revents: 0,
	}
	for {
		//set a minute so that this loop will check if it should exit
		ret, _ := syscall.Poll(pollFds, 60000)
		if ret >= 0 {
			if (pollFds[0].Revents & syscall.POLLERR) == 8 {
				return true
			}
		}
		pollFds[0].Revents = 0
	}
}