// +build linux

package libcontainer

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/docker/libcontainer/cgroups"
	"github.com/docker/libcontainer/configs"
	"github.com/docker/libcontainer/label"
)

const defaultMountFlags = syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

// setupRootfs sets up the devices, mount points, and filesystems for use inside a
// new mount namespace.
func setupRootfs(config *configs.Config, console *linuxConsole) (err error) {
	if err := prepareRoot(config); err != nil {
		return newSystemError(err)
	}
	for _, m := range config.Mounts {
		for _, precmd := range m.PremountCmds {
			if err := mountCmd(precmd); err != nil {
				return newSystemError(err)
			}
		}
		if err := mountToRootfs(m, config.Rootfs, config.MountLabel); err != nil {
			return newSystemError(err)
		}

		for _, postcmd := range m.PostmountCmds {
			if err := mountCmd(postcmd); err != nil {
				return newSystemError(err)
			}
		}
	}
	if err := createDevices(config); err != nil {
		return newSystemError(err)
	}
	if err := setupPtmx(config, console); err != nil {
		return newSystemError(err)
	}
	// stdin, stdout and stderr could be pointing to /dev/null from parent namespace.
	// re-open them inside this namespace.
	if err := reOpenDevNull(config.Rootfs); err != nil {
		return newSystemError(err)
	}
	if err := setupDevSymlinks(config.Rootfs); err != nil {
		return newSystemError(err)
	}
	if err := syscall.Chdir(config.Rootfs); err != nil {
		return newSystemError(err)
	}
	if config.NoPivotRoot {
		err = msMoveRoot(config.Rootfs)
	} else {
		err = pivotRoot(config.Rootfs, config.PivotDir)
	}
	if err != nil {
		return newSystemError(err)
	}
	if config.Readonlyfs {
		if err := setReadonly(); err != nil {
			return newSystemError(err)
		}
	}
	syscall.Umask(0022)
	return nil
}

func mountCmd(cmd configs.Command) error {

	command := exec.Command(cmd.Path, cmd.Args[:]...)
	command.Env = cmd.Env
	command.Dir = cmd.Dir
	if out, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%#v failed: %s: %v", cmd, string(out), err)
	}

	return nil
}

func mountToRootfs(m *configs.Mount, rootfs, mountLabel string) error {
	var (
		dest = m.Destination
		data = label.FormatMountLabel(m.Data, mountLabel)
	)
	if !strings.HasPrefix(dest, rootfs) {
		dest = filepath.Join(rootfs, dest)
	}

	switch m.Device {
	case "proc", "sysfs":
		if err := os.MkdirAll(dest, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		return syscall.Mount(m.Source, dest, m.Device, uintptr(m.Flags), "")
	case "mqueue":
		if err := os.MkdirAll(dest, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		if err := syscall.Mount(m.Source, dest, m.Device, uintptr(m.Flags), ""); err != nil {
			return err
		}
		return label.SetFileLabel(dest, mountLabel)
	case "tmpfs":
		stat, err := os.Stat(dest)
		if err != nil {
			if err := os.MkdirAll(dest, 0755); err != nil && !os.IsExist(err) {
				return err
			}
		}
		if err := syscall.Mount(m.Source, dest, m.Device, uintptr(m.Flags), data); err != nil {
			return err
		}
		if stat != nil {
			if err = os.Chmod(dest, stat.Mode()); err != nil {
				return err
			}
		}
		return nil
	case "devpts":
		if err := os.MkdirAll(dest, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		return syscall.Mount(m.Source, dest, m.Device, uintptr(m.Flags), data)
	case "bind":
		stat, err := os.Stat(m.Source)
		if err != nil {
			// error out if the source of a bind mount does not exist as we will be
			// unable to bind anything to it.
			return err
		}
		if err := createIfNotExists(dest, stat.IsDir()); err != nil {
			return err
		}
		if err := syscall.Mount(m.Source, dest, m.Device, uintptr(m.Flags), data); err != nil {
			return err
		}
		if m.Flags&syscall.MS_RDONLY != 0 {
			if err := syscall.Mount(m.Source, dest, m.Device, uintptr(m.Flags|syscall.MS_REMOUNT), ""); err != nil {
				return err
			}
		}
		if m.Relabel != "" {
			if err := label.Relabel(m.Source, mountLabel, m.Relabel); err != nil {
				return err
			}
		}
		if m.Flags&syscall.MS_PRIVATE != 0 {
			if err := syscall.Mount("", dest, "none", uintptr(syscall.MS_PRIVATE), ""); err != nil {
				return err
			}
		}
	case "cgroup":
		mounts, err := cgroups.GetCgroupMounts()
		if err != nil {
			return err
		}
		var binds []*configs.Mount
		for _, mm := range mounts {
			dir, err := mm.GetThisCgroupDir()
			if err != nil {
				return err
			}
			binds = append(binds, &configs.Mount{
				Device:      "bind",
				Source:      filepath.Join(mm.Mountpoint, dir),
				Destination: filepath.Join(m.Destination, strings.Join(mm.Subsystems, ",")),
				Flags:       syscall.MS_BIND | syscall.MS_REC | syscall.MS_RDONLY,
			})
		}
		tmpfs := &configs.Mount{
			Device:      "tmpfs",
			Destination: m.Destination,
			Flags:       syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV,
		}
		if err := mountToRootfs(tmpfs, rootfs, mountLabel); err != nil {
			return err
		}
		for _, b := range binds {
			if err := mountToRootfs(b, rootfs, mountLabel); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown mount device %q to %q", m.Device, m.Destination)
	}
	return nil
}

func setupDevSymlinks(rootfs string) error {
	var links = [][2]string{
		{"/proc/self/fd", "/dev/fd"},
		{"/proc/self/fd/0", "/dev/stdin"},
		{"/proc/self/fd/1", "/dev/stdout"},
		{"/proc/self/fd/2", "/dev/stderr"},
	}
	// kcore support can be toggled with CONFIG_PROC_KCORE; only create a symlink
	// in /dev if it exists in /proc.
	if _, err := os.Stat("/proc/kcore"); err == nil {
		links = append(links, [2]string{"/proc/kcore", "/dev/kcore"})
	}
	for _, link := range links {
		var (
			src = link[0]
			dst = filepath.Join(rootfs, link[1])
		)
		if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
			return fmt.Errorf("symlink %s %s %s", src, dst, err)
		}
	}
	return nil
}

// If stdin, stdout or stderr are pointing to '/dev/null' in the global mount namespace,
// this method will make them point to '/dev/null' in this namespace.
func reOpenDevNull(rootfs string) error {
	var stat, devNullStat syscall.Stat_t
	file, err := os.Open(filepath.Join(rootfs, "/dev/null"))
	if err != nil {
		return fmt.Errorf("Failed to open /dev/null - %s", err)
	}
	defer file.Close()
	if err := syscall.Fstat(int(file.Fd()), &devNullStat); err != nil {
		return err
	}
	for fd := 0; fd < 3; fd++ {
		if err := syscall.Fstat(fd, &stat); err != nil {
			return err
		}
		if stat.Rdev == devNullStat.Rdev {
			// Close and re-open the fd.
			if err := syscall.Dup2(int(file.Fd()), fd); err != nil {
				return err
			}
		}
	}
	return nil
}

// Create the device nodes in the container.
func createDevices(config *configs.Config) error {
	oldMask := syscall.Umask(0000)
	for _, node := range config.Devices {
		// containers running in a user namespace are not allowed to mknod
		// devices so we can just bind mount it from the host.
		if err := createDeviceNode(config.Rootfs, node, config.Namespaces.Contains(configs.NEWUSER)); err != nil {
			syscall.Umask(oldMask)
			return err
		}
	}
	syscall.Umask(oldMask)
	return nil
}

// Creates the device node in the rootfs of the container.
func createDeviceNode(rootfs string, node *configs.Device, bind bool) error {
	dest := filepath.Join(rootfs, node.Path)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	if bind {
		f, err := os.Create(dest)
		if err != nil && !os.IsExist(err) {
			return err
		}
		if f != nil {
			f.Close()
		}
		return syscall.Mount(node.Path, dest, "bind", syscall.MS_BIND, "")
	}
	if err := mknodDevice(dest, node); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func mknodDevice(dest string, node *configs.Device) error {
	fileMode := node.FileMode
	switch node.Type {
	case 'c':
		fileMode |= syscall.S_IFCHR
	case 'b':
		fileMode |= syscall.S_IFBLK
	default:
		return fmt.Errorf("%c is not a valid device type for device %s", node.Type, node.Path)
	}
	if err := syscall.Mknod(dest, uint32(fileMode), node.Mkdev()); err != nil {
		return err
	}
	return syscall.Chown(dest, int(node.Uid), int(node.Gid))
}

func prepareRoot(config *configs.Config) error {
	flag := syscall.MS_SLAVE | syscall.MS_REC
	if config.Privatefs {
		flag = syscall.MS_PRIVATE | syscall.MS_REC
	}
	if err := syscall.Mount("", "/", "", uintptr(flag), ""); err != nil {
		return err
	}
	return syscall.Mount(config.Rootfs, config.Rootfs, "bind", syscall.MS_BIND|syscall.MS_REC, "")
}

func setReadonly() error {
	return syscall.Mount("/", "/", "bind", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_REC, "")
}

func setupPtmx(config *configs.Config, console *linuxConsole) error {
	ptmx := filepath.Join(config.Rootfs, "dev/ptmx")
	if err := os.Remove(ptmx); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink("pts/ptmx", ptmx); err != nil {
		return fmt.Errorf("symlink dev ptmx %s", err)
	}
	if console != nil {
		return console.mount(config.Rootfs, config.MountLabel, 0, 0)
	}
	return nil
}

func pivotRoot(rootfs, pivotBaseDir string) error {
	if pivotBaseDir == "" {
		pivotBaseDir = "/"
	}
	tmpDir := filepath.Join(rootfs, pivotBaseDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("can't create tmp dir %s, error %v", tmpDir, err)
	}
	pivotDir, err := ioutil.TempDir(tmpDir, ".pivot_root")
	if err != nil {
		return fmt.Errorf("can't create pivot_root dir %s, error %v", pivotDir, err)
	}
	if err := syscall.PivotRoot(rootfs, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %s", err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %s", err)
	}
	// path to pivot dir now changed, update
	pivotDir = filepath.Join(pivotBaseDir, filepath.Base(pivotDir))
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %s", err)
	}
	return os.Remove(pivotDir)
}

func msMoveRoot(rootfs string) error {
	if err := syscall.Mount(rootfs, "/", "", syscall.MS_MOVE, ""); err != nil {
		return err
	}
	if err := syscall.Chroot("."); err != nil {
		return err
	}
	return syscall.Chdir("/")
}

// createIfNotExists creates a file or a directory only if it does not already exist.
func createIfNotExists(path string, isDir bool) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if isDir {
				return os.MkdirAll(path, 0755)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE, 0755)
			if err != nil {
				return err
			}
			f.Close()
		}
	}
	return nil
}

// remountReadonly will bind over the top of an existing path and ensure that it is read-only.
func remountReadonly(path string) error {
	for i := 0; i < 5; i++ {
		if err := syscall.Mount("", path, "", syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil && !os.IsNotExist(err) {
			switch err {
			case syscall.EINVAL:
				// Probably not a mountpoint, use bind-mount
				if err := syscall.Mount(path, path, "", syscall.MS_BIND, ""); err != nil {
					return err
				}
				return syscall.Mount(path, path, "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_REC|defaultMountFlags, "")
			case syscall.EBUSY:
				time.Sleep(100 * time.Millisecond)
				continue
			default:
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("unable to mount %s as readonly max retries reached", path)
}

// maskFile bind mounts /dev/null over the top of the specified path inside a container
// to avoid security issues from processes reading information from non-namespace aware mounts ( proc/kcore ).
func maskFile(path string) error {
	if err := syscall.Mount("/dev/null", path, "", syscall.MS_BIND, ""); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// writeSystemProperty writes the value to a path under /proc/sys as determined from the key.
// For e.g. net.ipv4.ip_forward translated to /proc/sys/net/ipv4/ip_forward.
func writeSystemProperty(key, value string) error {
	keyPath := strings.Replace(key, ".", "/", -1)
	return ioutil.WriteFile(path.Join("/proc/sys", keyPath), []byte(value), 0644)
}
