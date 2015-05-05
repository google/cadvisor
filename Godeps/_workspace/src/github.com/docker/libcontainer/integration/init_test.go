package integration

import (
	"os"
	"runtime"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups/systemd"
	_ "github.com/docker/libcontainer/nsenter"
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
		log.Fatalf("unable to initialize for container: %s", err)
	}
	if err := factory.StartInitialization(); err != nil {
		log.Fatal(err)
	}
}

var (
	factory libcontainer.Factory
	systemdFactory libcontainer.Factory
)

func TestMain(m *testing.M) {
	var (
		err error
		ret int = 0
	)

	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)

	factory, err = libcontainer.New(".", libcontainer.Cgroupfs)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	if systemd.UseSystemd() {
		systemdFactory, err = libcontainer.New(".", libcontainer.SystemdCgroups)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	ret = m.Run()
	os.Exit(ret)
}
