// +build linux

package libcontainer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/opencontainers/runc/libcontainer/apparmor"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/keys"
	"github.com/opencontainers/runc/libcontainer/label"
	"github.com/opencontainers/runc/libcontainer/seccomp"
	"github.com/opencontainers/runc/libcontainer/system"
)

type linuxStandardInit struct {
	pipe       io.ReadWriteCloser
	parentPid  int
	stateDirFD int
	config     *initConfig
}

func (l *linuxStandardInit) getSessionRingParams() (string, uint32, uint32) {
	var newperms uint32

	if l.config.Config.Namespaces.Contains(configs.NEWUSER) {
		// with user ns we need 'other' search permissions
		newperms = 0x8
	} else {
		// without user ns we need 'UID' search permissions
		newperms = 0x80000
	}

	// create a unique per session container name that we can
	// join in setns; however, other containers can also join it
	return fmt.Sprintf("_ses.%s", l.config.ContainerId), 0xffffffff, newperms
}

// PR_SET_NO_NEW_PRIVS isn't exposed in Golang so we define it ourselves copying the value
// the kernel
const PR_SET_NO_NEW_PRIVS = 0x26

func (l *linuxStandardInit) Init() error {
	if !l.config.Config.NoNewKeyring {
		ringname, keepperms, newperms := l.getSessionRingParams()

		// do not inherit the parent's session keyring
		sessKeyId, err := keyctl.JoinSessionKeyring(ringname)
		if err != nil {
			return err
		}
		// make session keyring searcheable
		if err := keyctl.ModKeyringPerm(sessKeyId, keepperms, newperms); err != nil {
			return err
		}
	}

	var console *linuxConsole
	if l.config.Console != "" {
		console = newConsoleFromPath(l.config.Console)
		if err := console.dupStdio(); err != nil {
			return err
		}
	}
	if console != nil {
		if err := system.Setctty(); err != nil {
			return err
		}
	}
	if err := setupNetwork(l.config); err != nil {
		return err
	}
	if err := setupRoute(l.config.Config); err != nil {
		return err
	}

	label.Init()
	// InitializeMountNamespace() can be executed only for a new mount namespace
	if l.config.Config.Namespaces.Contains(configs.NEWNS) {
		if err := setupRootfs(l.config.Config, console, l.pipe); err != nil {
			return err
		}
	}
	if hostname := l.config.Config.Hostname; hostname != "" {
		if err := syscall.Sethostname([]byte(hostname)); err != nil {
			return err
		}
	}
	if err := apparmor.ApplyProfile(l.config.AppArmorProfile); err != nil {
		return err
	}
	if err := label.SetProcessLabel(l.config.ProcessLabel); err != nil {
		return err
	}

	for key, value := range l.config.Config.Sysctl {
		if err := writeSystemProperty(key, value); err != nil {
			return err
		}
	}
	for _, path := range l.config.Config.ReadonlyPaths {
		if err := remountReadonly(path); err != nil {
			return err
		}
	}
	for _, path := range l.config.Config.MaskPaths {
		if err := maskFile(path); err != nil {
			return err
		}
	}
	pdeath, err := system.GetParentDeathSignal()
	if err != nil {
		return err
	}
	if l.config.NoNewPrivileges {
		if err := system.Prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
			return err
		}
	}
	// Tell our parent that we're ready to Execv. This must be done before the
	// Seccomp rules have been applied, because we need to be able to read and
	// write to a socket.
	if err := syncParentReady(l.pipe); err != nil {
		return err
	}
	// Without NoNewPrivileges seccomp is a privileged operation, so we need to
	// do this before dropping capabilities; otherwise do it as late as possible
	// just before execve so as few syscalls take place after it as possible.
	if l.config.Config.Seccomp != nil && !l.config.NoNewPrivileges {
		if err := seccomp.InitSeccomp(l.config.Config.Seccomp); err != nil {
			return err
		}
	}
	if err := finalizeNamespace(l.config); err != nil {
		return err
	}
	// finalizeNamespace can change user/group which clears the parent death
	// signal, so we restore it here.
	if err := pdeath.Restore(); err != nil {
		return err
	}
	// compare the parent from the inital start of the init process and make sure that it did not change.
	// if the parent changes that means it died and we were reparened to something else so we should
	// just kill ourself and not cause problems for someone else.
	if syscall.Getppid() != l.parentPid {
		return syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
	}
	// check for the arg before waiting to make sure it exists and it is returned
	// as a create time error.
	name, err := exec.LookPath(l.config.Args[0])
	if err != nil {
		return err
	}
	// close the pipe to signal that we have completed our init.
	l.pipe.Close()
	// wait for the fifo to be opened on the other side before
	// exec'ing the users process.
	fd, err := syscall.Openat(l.stateDirFD, execFifoFilename, os.O_WRONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return newSystemErrorWithCause(err, "openat exec fifo")
	}
	if _, err := syscall.Write(fd, []byte("0")); err != nil {
		return newSystemErrorWithCause(err, "write 0 exec fifo")
	}
	if l.config.Config.Seccomp != nil && l.config.NoNewPrivileges {
		if err := seccomp.InitSeccomp(l.config.Config.Seccomp); err != nil {
			return newSystemErrorWithCause(err, "init seccomp")
		}
	}
	if err := syscall.Exec(name, l.config.Args[0:], os.Environ()); err != nil {
		return newSystemErrorWithCause(err, "exec user process")
	}
	return nil
}
