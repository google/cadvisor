package nsenter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/vishvananda/netlink/nl"

	"golang.org/x/sys/unix"
)

type pid struct {
	Pid int `json:"Pid"`
}

type logentry struct {
	Msg   string `json:"msg"`
	Level string `json:"level"`
}

func TestNsenterValidPaths(t *testing.T) {
	args := []string{"nsenter-exec"}
	parent, child, err := newPipe()
	if err != nil {
		t.Fatalf("failed to create pipe %v", err)
	}

	namespaces := []string{
		// join pid ns of the current process
		fmt.Sprintf("pid:/proc/%d/ns/pid", os.Getpid()),
	}
	cmd := &exec.Cmd{
		Path:       os.Args[0],
		Args:       args,
		ExtraFiles: []*os.File{child},
		Env:        []string{"_LIBCONTAINER_INITPIPE=3"},
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("nsenter failed to start %v", err)
	}
	// write cloneFlags
	r := nl.NewNetlinkRequest(int(libcontainer.InitMsg), 0)
	r.AddData(&libcontainer.Int32msg{
		Type:  libcontainer.CloneFlagsAttr,
		Value: uint32(unix.CLONE_NEWNET),
	})
	r.AddData(&libcontainer.Bytemsg{
		Type:  libcontainer.NsPathsAttr,
		Value: []byte(strings.Join(namespaces, ",")),
	})
	if _, err := io.Copy(parent, bytes.NewReader(r.Serialize())); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(parent)
	var pid *pid

	if err := cmd.Wait(); err != nil {
		t.Fatalf("nsenter exits with a non-zero exit status")
	}
	if err := decoder.Decode(&pid); err != nil {
		dir, _ := ioutil.ReadDir(fmt.Sprintf("/proc/%d/ns", os.Getpid()))
		for _, d := range dir {
			t.Log(d.Name())
		}
		t.Fatalf("%v", err)
	}

	p, err := os.FindProcess(pid.Pid)
	if err != nil {
		t.Fatalf("%v", err)
	}
	p.Wait()
}

func TestNsenterInvalidPaths(t *testing.T) {
	args := []string{"nsenter-exec"}
	parent, child, err := newPipe()
	if err != nil {
		t.Fatalf("failed to create pipe %v", err)
	}

	namespaces := []string{
		// join pid ns of the current process
		fmt.Sprintf("pid:/proc/%d/ns/pid", -1),
	}
	cmd := &exec.Cmd{
		Path:       os.Args[0],
		Args:       args,
		ExtraFiles: []*os.File{child},
		Env:        []string{"_LIBCONTAINER_INITPIPE=3"},
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// write cloneFlags
	r := nl.NewNetlinkRequest(int(libcontainer.InitMsg), 0)
	r.AddData(&libcontainer.Int32msg{
		Type:  libcontainer.CloneFlagsAttr,
		Value: uint32(unix.CLONE_NEWNET),
	})
	r.AddData(&libcontainer.Bytemsg{
		Type:  libcontainer.NsPathsAttr,
		Value: []byte(strings.Join(namespaces, ",")),
	})
	if _, err := io.Copy(parent, bytes.NewReader(r.Serialize())); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatalf("nsenter exits with a zero exit status")
	}
}

func TestNsenterIncorrectPathType(t *testing.T) {
	args := []string{"nsenter-exec"}
	parent, child, err := newPipe()
	if err != nil {
		t.Fatalf("failed to create pipe %v", err)
	}

	namespaces := []string{
		// join pid ns of the current process
		fmt.Sprintf("net:/proc/%d/ns/pid", os.Getpid()),
	}
	cmd := &exec.Cmd{
		Path:       os.Args[0],
		Args:       args,
		ExtraFiles: []*os.File{child},
		Env:        []string{"_LIBCONTAINER_INITPIPE=3"},
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// write cloneFlags
	r := nl.NewNetlinkRequest(int(libcontainer.InitMsg), 0)
	r.AddData(&libcontainer.Int32msg{
		Type:  libcontainer.CloneFlagsAttr,
		Value: uint32(unix.CLONE_NEWNET),
	})
	r.AddData(&libcontainer.Bytemsg{
		Type:  libcontainer.NsPathsAttr,
		Value: []byte(strings.Join(namespaces, ",")),
	})
	if _, err := io.Copy(parent, bytes.NewReader(r.Serialize())); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatalf("nsenter exits with a zero exit status")
	}
}

func TestNsenterChildLogging(t *testing.T) {
	args := []string{"nsenter-exec"}
	parent, child, err := newPipe()
	if err != nil {
		t.Fatalf("failed to create exec pipe %v", err)
	}
	logread, logwrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create log pipe %v", err)
	}
	defer logread.Close()
	defer logwrite.Close()

	namespaces := []string{
		// join pid ns of the current process
		fmt.Sprintf("pid:/proc/%d/ns/pid", os.Getpid()),
	}
	cmd := &exec.Cmd{
		Path:       os.Args[0],
		Args:       args,
		ExtraFiles: []*os.File{child, logwrite},
		Env:        []string{"_LIBCONTAINER_INITPIPE=3", "_LIBCONTAINER_LOGPIPE=4"},
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("nsenter failed to start %v", err)
	}
	// write cloneFlags
	r := nl.NewNetlinkRequest(int(libcontainer.InitMsg), 0)
	r.AddData(&libcontainer.Int32msg{
		Type:  libcontainer.CloneFlagsAttr,
		Value: uint32(unix.CLONE_NEWNET),
	})
	r.AddData(&libcontainer.Bytemsg{
		Type:  libcontainer.NsPathsAttr,
		Value: []byte(strings.Join(namespaces, ",")),
	})
	if _, err := io.Copy(parent, bytes.NewReader(r.Serialize())); err != nil {
		t.Fatal(err)
	}

	logsDecoder := json.NewDecoder(logread)
	var logentry *logentry

	err = logsDecoder.Decode(&logentry)
	if err != nil {
		t.Fatalf("child log: %v", err)
	}
	if logentry.Level == "" || logentry.Msg == "" {
		t.Fatalf("child log: empty log fileds: level=\"%s\" msg=\"%s\"", logentry.Level, logentry.Msg)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("nsenter exits with a non-zero exit status")
	}
}

func init() {
	if strings.HasPrefix(os.Args[0], "nsenter-") {
		os.Exit(0)
	}
	return
}

func newPipe() (parent *os.File, child *os.File, err error) {
	fds, err := unix.Socketpair(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), "parent"), os.NewFile(uintptr(fds[0]), "child"), nil
}
