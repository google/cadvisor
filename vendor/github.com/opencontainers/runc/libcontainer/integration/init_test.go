package integration

import (
	"os"
	"runtime"
	"testing"

	"github.com/opencontainers/runc/libcontainer"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"

	"github.com/sirupsen/logrus"
)

// init runs the libcontainer initialization code because of the busybox style needs
// to work around the go runtime and the issues with forking
func init() {
	if len(os.Args) < 2 || os.Args[1] != "init" {
		return
	}
	runtime.GOMAXPROCS(1)
	runtime.LockOSThread()
	factory, err := libcontainer.New("")
	if err != nil {
		logrus.Fatalf("unable to initialize for container: %s", err)
	}
	if err := factory.StartInitialization(); err != nil {
		logrus.Fatal(err)
	}
}

var testRoots []string

func TestMain(m *testing.M) {
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.InfoLevel)

	// Clean up roots after running everything.
	defer func() {
		for _, root := range testRoots {
			os.RemoveAll(root)
		}
	}()

	ret := m.Run()
	os.Exit(ret)
}
