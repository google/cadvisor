package framework

import (
	"bytes"
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/client"
	"github.com/google/cadvisor/client/v2"
)

var host = flag.String("host", "localhost", "Address of the host being tested")
var port = flag.Int("port", 8080, "Port of the application on the host being tested")
var sshOptions = flag.String("ssh-options", "", "Command line options for ssh")

// Integration test framework.
type Framework interface {
	// Clean the framework state.
	Cleanup()

	// The testing.T used by the framework and the current test.
	T() *testing.T

	// Returns the hostname being tested.
	Hostname() HostnameInfo

	// Returns the Docker actions for the test framework.
	Docker() DockerActions

	// Returns the shell actions for the test framework.
	Shell() ShellActions

	// Returns the Rkt actions for the test framework.
	Rkt() RktActions

	// Returns the cAdvisor actions for the test framework.
	Cadvisor() CadvisorActions
}

// Instantiates a Framework. Cleanup *must* be called. Class is thread-compatible.
// All framework actions report fatal errors on the t specified at creation time.
//
// Typical use:
//
// func TestFoo(t *testing.T) {
// 	fm := framework.New(t)
// 	defer fm.Cleanup()
//      ... actual test ...
// }
func New(t *testing.T) Framework {
	// All integration tests are large.
	if testing.Short() {
		t.Skip("Skipping framework test in short mode")
	}

	// Try to see if non-localhost hosts are GCE instances.
	fm := &realFramework{
		hostname: HostnameInfo{
			Host: *host,
			Port: *port,
		},
		t:        t,
		cleanups: make([]func(), 0),
	}
	fm.shellActions = shellActions{
		fm: fm,
	}
	fm.dockerActions = dockerActions{
		fm: fm,
	}
	fm.rktActions = rktActions{
		fm: fm,
	}

	return fm
}

const (
	Aufs         string = "aufs"
	Overlay      string = "overlay"
	DeviceMapper string = "devicemapper"
	Unknown      string = ""
)

type RunArgs struct {
	// Image to use.
	Image string

	// Arguments to the CLI.
	Args []string

	InnerArgs []string
}

type ShellActions interface {
	// Runs a specified command and arguments. Returns the stdout and stderr.
	Run(cmd string, args ...string) (string, string)
	RunStress(cmd string, args ...string) (string, string)
}

type CadvisorActions interface {
	// Returns a cAdvisor client to the machine being tested.
	Client() *client.Client
	ClientV2() *v2.Client
}

type realFramework struct {
	hostname         HostnameInfo
	t                *testing.T
	cadvisorClient   *client.Client
	cadvisorClientV2 *v2.Client

	shellActions  shellActions
	dockerActions dockerActions
	rktActions    rktActions

	// Cleanup functions to call on Cleanup()
	cleanups []func()
}

type shellActions struct {
	fm *realFramework
}

type HostnameInfo struct {
	Host string
	Port int
}

// Returns: http://<host>:<port>/
func (self HostnameInfo) FullHostname() string {
	return fmt.Sprintf("http://%s:%d/", self.Host, self.Port)
}

func (self *realFramework) T() *testing.T {
	return self.t
}

func (self *realFramework) Hostname() HostnameInfo {
	return self.hostname
}

func (self *realFramework) Shell() ShellActions {
	return self.shellActions
}

func (self *realFramework) Docker() DockerActions {
	return self.dockerActions
}

func (self *realFramework) Rkt() RktActions {
	return self.rktActions
}

func (self *realFramework) Cadvisor() CadvisorActions {
	return self
}

// Call all cleanup functions.
func (self *realFramework) Cleanup() {
	for _, cleanupFunc := range self.cleanups {
		cleanupFunc()
	}
}

// Gets a client to the cAdvisor being tested.
func (self *realFramework) Client() *client.Client {
	if self.cadvisorClient == nil {
		cadvisorClient, err := client.NewClient(self.Hostname().FullHostname())
		if err != nil {
			self.t.Fatalf("Failed to instantiate the cAdvisor client: %v", err)
		}
		self.cadvisorClient = cadvisorClient
	}
	return self.cadvisorClient
}

// Gets a v2 client to the cAdvisor being tested.
func (self *realFramework) ClientV2() *v2.Client {
	if self.cadvisorClientV2 == nil {
		cadvisorClientV2, err := v2.NewClient(self.Hostname().FullHostname())
		if err != nil {
			self.t.Fatalf("Failed to instantiate the cAdvisor client: %v", err)
		}
		self.cadvisorClientV2 = cadvisorClientV2
	}
	return self.cadvisorClientV2
}

func (self shellActions) wrapSsh(command string, args ...string) *exec.Cmd {
	cmd := []string{self.fm.Hostname().Host, "--", "sh", "-c", "\"", command}
	cmd = append(cmd, args...)
	cmd = append(cmd, "\"")
	if *sshOptions != "" {
		cmd = append(strings.Split(*sshOptions, " "), cmd...)
	}
	return exec.Command("ssh", cmd...)
}

func (self shellActions) Run(command string, args ...string) (string, string) {
	var cmd *exec.Cmd
	if self.fm.Hostname().Host == "localhost" {
		// Just run locally.
		cmd = exec.Command(command, args...)
	} else {
		// We must SSH to the remote machine and run the command.
		cmd = self.wrapSsh(command, args...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	glog.Infof("About to run - %v", cmd.Args)
	err := cmd.Run()
	if err != nil {
		self.fm.T().Fatalf("Failed to run %q %v in %q with error: %q. Stdout: %q, Stderr: %s", command, args, self.fm.Hostname().Host, err, stdout.String(), stderr.String())
		return "", ""
	}
	return stdout.String(), stderr.String()
}

func (self shellActions) RunStress(command string, args ...string) (string, string) {
	var cmd *exec.Cmd
	if self.fm.Hostname().Host == "localhost" {
		// Just run locally.
		cmd = exec.Command(command, args...)
	} else {
		// We must SSH to the remote machine and run the command.
		cmd = self.wrapSsh(command, args...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		self.fm.T().Logf("Ran %q %v in %q and received error: %q. Stdout: %q, Stderr: %s", command, args, self.fm.Hostname().Host, err, stdout.String(), stderr.String())
		return stdout.String(), stderr.String()
	}
	return stdout.String(), stderr.String()
}

// Runs retryFunc until no error is returned. After dur time the last error is returned.
// Note that the function does not timeout the execution of retryFunc when the limit is reached.
func RetryForDuration(retryFunc func() error, dur time.Duration) error {
	waitUntil := time.Now().Add(dur)
	var err error
	for time.Now().Before(waitUntil) {
		err = retryFunc()
		if err == nil {
			return nil
		}
	}
	return err
}
