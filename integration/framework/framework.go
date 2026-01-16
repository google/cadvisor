// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package framework

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/klog/v2"

	"github.com/google/cadvisor/client"
	v2 "github.com/google/cadvisor/client/v2"
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

	// Returns the CRI-O actions for the test framework.
	Crio() CrioActions

	// Returns the containerd actions for the test framework.
	Containerd() ContainerdActions

	// Returns the shell actions for the test framework.
	Shell() ShellActions

	// Returns the cAdvisor actions for the test framework.
	Cadvisor() CadvisorActions
}

// Instantiates a Framework. Cleanup *must* be called. Class is thread-compatible.
// All framework actions report fatal errors on the t specified at creation time.
//
// Typical use:
//
//	func TestFoo(t *testing.T) {
//		fm := framework.New(t)
//		defer fm.Cleanup()
//	     ... actual test ...
//	}
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
	fm.crioActions = crioActions{
		fm:        fm,
		podSeqNum: 0,
	}
	fm.containerdActions = containerdActions{
		fm:          fm,
		seqNum:      0,
		namespace:   "k8s.io",
		socket:      getContainerdSocket(),
		snapshotter: "native",
	}

	return fm
}

// getContainerdSocket returns the containerd socket path from CONTAINERD_SOCK env var.
func getContainerdSocket() string {
	if sock := os.Getenv("CONTAINERD_SOCK"); sock != "" {
		return sock
	}
	return "/run/containerd/containerd.sock"
}

const (
	Aufs         string = "aufs"
	Overlay      string = "overlay"
	Overlay2     string = "overlay2"
	DeviceMapper string = "devicemapper"
	Unknown      string = ""
)

type DockerActions interface {
	// Run the no-op pause Docker container and return its ID.
	RunPause() string

	// Run the specified command in a Docker busybox container and return its ID.
	RunBusybox(cmd ...string) string

	// Runs a Docker container in the background. Uses the specified DockerRunArgs and command.
	// Returns the ID of the new container.
	//
	// e.g.:
	// Run(DockerRunArgs{Image: "busybox"}, "ping", "www.google.com")
	//   -> docker run busybox ping www.google.com
	Run(args DockerRunArgs, cmd ...string) string
	RunStress(args DockerRunArgs, cmd ...string) string

	Version() []string
	StorageDriver() string
}

// CrioActions provides methods for managing CRI-O containers in tests.
// CRI-O containers run inside pod sandboxes, so each container requires
// a pod to be created first.
type CrioActions interface {
	// Run the no-op pause CRI-O container and return its ID.
	RunPause() string

	// Run the specified command in a CRI-O busybox container and return its ID.
	RunBusybox(cmd ...string) string

	// Runs a CRI-O container in the background. Uses the specified CrioRunArgs and command.
	// Returns the ID of the new container.
	Run(args CrioRunArgs, cmd ...string) string
}

// CrioRunArgs contains arguments for running a CRI-O container.
type CrioRunArgs struct {
	// Image to use.
	Image string

	// Container name (optional, auto-generated if empty).
	Name string
}

// ContainerdActions provides methods for managing containerd containers in tests.
// Containerd containers are created directly using the ctr CLI tool.
type ContainerdActions interface {
	// Run the no-op pause containerd container and return its ID.
	RunPause() string

	// Run the specified command in a containerd busybox container and return its ID.
	RunBusybox(cmd ...string) string

	// Runs a containerd container in the background. Uses the specified ContainerdRunArgs and command.
	// Returns the ID of the new container.
	Run(args ContainerdRunArgs, cmd ...string) string
}

// ContainerdRunArgs contains arguments for running a containerd container.
type ContainerdRunArgs struct {
	// Image to use.
	Image string

	// Container name (optional, auto-generated if empty).
	Name string

	// Labels to add to the container.
	Labels map[string]string
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

	shellActions      shellActions
	dockerActions     dockerActions
	crioActions       crioActions
	containerdActions containerdActions

	// Cleanup functions to call on Cleanup()
	cleanups []func()
}

type shellActions struct {
	fm *realFramework
}

type dockerActions struct {
	fm *realFramework
}

type crioActions struct {
	fm        *realFramework
	podSeqNum int // For generating unique pod names
}

type containerdActions struct {
	fm          *realFramework
	seqNum      int    // For generating unique container names
	namespace   string // containerd namespace (default: k8s.io)
	socket      string // containerd socket path
	snapshotter string // containerd snapshotter (default: native)
}

type HostnameInfo struct {
	Host string
	Port int
}

// Returns: http://<host>:<port>/
func (h HostnameInfo) FullHostname() string {
	return fmt.Sprintf("http://%s:%d/", h.Host, h.Port)
}

func (f *realFramework) T() *testing.T {
	return f.t
}

func (f *realFramework) Hostname() HostnameInfo {
	return f.hostname
}

func (f *realFramework) Shell() ShellActions {
	return f.shellActions
}

func (f *realFramework) Docker() DockerActions {
	return f.dockerActions
}

func (f *realFramework) Crio() CrioActions {
	return &f.crioActions
}

func (f *realFramework) Containerd() ContainerdActions {
	return &f.containerdActions
}

func (f *realFramework) Cadvisor() CadvisorActions {
	return f
}

// Call all cleanup functions.
func (f *realFramework) Cleanup() {
	for _, cleanupFunc := range f.cleanups {
		cleanupFunc()
	}
}

// Gets a client to the cAdvisor being tested.
func (f *realFramework) Client() *client.Client {
	if f.cadvisorClient == nil {
		cadvisorClient, err := client.NewClient(f.Hostname().FullHostname())
		if err != nil {
			f.t.Fatalf("Failed to instantiate the cAdvisor client: %v", err)
		}
		f.cadvisorClient = cadvisorClient
	}
	return f.cadvisorClient
}

// Gets a v2 client to the cAdvisor being tested.
func (f *realFramework) ClientV2() *v2.Client {
	if f.cadvisorClientV2 == nil {
		cadvisorClientV2, err := v2.NewClient(f.Hostname().FullHostname())
		if err != nil {
			f.t.Fatalf("Failed to instantiate the cAdvisor client: %v", err)
		}
		f.cadvisorClientV2 = cadvisorClientV2
	}
	return f.cadvisorClientV2
}

func (a dockerActions) RunPause() string {
	return a.Run(DockerRunArgs{
		Image: "registry.k8s.io/pause",
	})
}

// Run the specified command in a Docker busybox container.
func (a dockerActions) RunBusybox(cmd ...string) string {
	return a.Run(DockerRunArgs{
		Image: "registry.k8s.io/busybox:1.27",
	}, cmd...)
}

type DockerRunArgs struct {
	// Image to use.
	Image string

	// Arguments to the Docker CLI.
	Args []string

	InnerArgs []string
}

// TODO(vmarmol): Use the Docker remote API.
// TODO(vmarmol): Refactor a set of "RunCommand" actions.
// Runs a Docker container in the background. Uses the specified DockerRunArgs and command.
//
// e.g.:
// RunDockerContainer(DockerRunArgs{Image: "busybox"}, "ping", "www.google.com")
//
//	-> docker run busybox ping www.google.com
func (a dockerActions) Run(args DockerRunArgs, cmd ...string) string {
	dockerCommand := append(append([]string{"docker", "run", "-d"}, args.Args...), args.Image)
	dockerCommand = append(dockerCommand, cmd...)
	output, _ := a.fm.Shell().Run("sudo", dockerCommand...)

	// The last line is the container ID.
	elements := strings.Fields(output)
	containerID := elements[len(elements)-1]

	a.fm.cleanups = append(a.fm.cleanups, func() {
		a.fm.Shell().Run("sudo", "docker", "rm", "-f", containerID)
	})
	return containerID
}
func (a dockerActions) Version() []string {
	dockerCommand := []string{"docker", "version", "-f", "'{{.Server.Version}}'"}
	output, _ := a.fm.Shell().Run("sudo", dockerCommand...)
	output = strings.TrimSpace(output)
	ret := strings.Split(output, ".")
	if len(ret) != 3 {
		a.fm.T().Fatalf("invalid version %v", output)
	}
	return ret
}

func (a dockerActions) StorageDriver() string {
	dockerCommand := []string{"docker", "info"}
	output, _ := a.fm.Shell().Run("sudo", dockerCommand...)
	if len(output) < 1 {
		a.fm.T().Fatalf("failed to find docker storage driver - %v", output)
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Storage Driver: ") {
			idx := strings.LastIndex(line, ": ") + 2
			driver := line[idx:]
			switch driver {
			case Aufs, Overlay, Overlay2, DeviceMapper:
				return driver
			default:
				return Unknown
			}
		}
	}
	a.fm.T().Fatalf("failed to find docker storage driver from info - %v", output)
	return Unknown
}

func (a dockerActions) RunStress(args DockerRunArgs, cmd ...string) string {
	dockerCommand := append(append(append(append([]string{"docker", "run", "-m=4M", "-d", "-t", "-i"}, args.Args...), args.Image), args.InnerArgs...), cmd...)

	output, _ := a.fm.Shell().RunStress("sudo", dockerCommand...)

	// The last line is the container ID.
	if len(output) < 1 {
		a.fm.T().Fatalf("need 1 arguments in output %v to get the name but have %v", output, len(output))
	}
	elements := strings.Fields(output)
	containerID := elements[len(elements)-1]

	a.fm.cleanups = append(a.fm.cleanups, func() {
		a.fm.Shell().Run("sudo", "docker", "rm", "-f", containerID)
	})
	return containerID
}

func (a shellActions) wrapSSH(command string, args ...string) *exec.Cmd {
	cmd := []string{a.fm.Hostname().Host, "--", "sh", "-c", "\"", command}
	cmd = append(cmd, args...)
	cmd = append(cmd, "\"")
	if *sshOptions != "" {
		cmd = append(strings.Split(*sshOptions, " "), cmd...)
	}
	return exec.Command("ssh", cmd...)
}

func (a shellActions) Run(command string, args ...string) (string, string) {
	var cmd *exec.Cmd
	if a.fm.Hostname().Host == "localhost" {
		// Just run locally.
		cmd = exec.Command(command, args...)
	} else {
		// We must SSH to the remote machine and run the command.
		cmd = a.wrapSSH(command, args...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	klog.Infof("About to run - %v", cmd.Args)
	err := cmd.Run()
	if err != nil {
		a.fm.T().Fatalf("Failed to run %q %v in %q with error: %q. Stdout: %q, Stderr: %s", command, args, a.fm.Hostname().Host, err, stdout.String(), stderr.String())
		return "", ""
	}
	return stdout.String(), stderr.String()
}

func (a shellActions) RunStress(command string, args ...string) (string, string) {
	var cmd *exec.Cmd
	if a.fm.Hostname().Host == "localhost" {
		// Just run locally.
		cmd = exec.Command(command, args...)
	} else {
		// We must SSH to the remote machine and run the command.
		cmd = a.wrapSSH(command, args...)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		a.fm.T().Logf("Ran %q %v in %q and received error: %q. Stdout: %q, Stderr: %s", command, args, a.fm.Hostname().Host, err, stdout.String(), stderr.String())
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

// CRI-O pod sandbox configuration for crictl
type crioPodConfig struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		UID       string `json:"uid"`
	} `json:"metadata"`
	Linux struct{} `json:"linux"`
}

// CRI-O container configuration for crictl
type crioContainerConfig struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Image struct {
		Image string `json:"image"`
	} `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	Linux   struct{} `json:"linux"`
}

func (a *crioActions) RunPause() string {
	return a.Run(CrioRunArgs{
		Image: "registry.k8s.io/pause:3.9",
	})
}

func (a *crioActions) RunBusybox(cmd ...string) string {
	return a.Run(CrioRunArgs{
		Image: "registry.k8s.io/busybox:1.27",
	}, cmd...)
}

func (a *crioActions) Run(args CrioRunArgs, cmd ...string) string {
	// Generate unique names for pod and container
	a.podSeqNum++
	podName := fmt.Sprintf("test-pod-%d-%d", os.Getpid(), a.podSeqNum)
	containerName := args.Name
	if containerName == "" {
		containerName = fmt.Sprintf("test-container-%d-%d", os.Getpid(), a.podSeqNum)
	}

	// Create temporary directory for config files
	tmpDir, err := os.MkdirTemp("", "crio-test-")
	if err != nil {
		a.fm.T().Fatalf("Failed to create temp directory: %v", err)
	}

	// Create pod config JSON
	podConfig := crioPodConfig{}
	podConfig.Metadata.Name = podName
	podConfig.Metadata.Namespace = "default"
	podConfig.Metadata.UID = fmt.Sprintf("uid-%d-%d", os.Getpid(), a.podSeqNum)

	podConfigPath := filepath.Join(tmpDir, "pod-config.json")
	podConfigData, err := json.Marshal(podConfig)
	if err != nil {
		a.fm.T().Fatalf("Failed to marshal pod config: %v", err)
	}
	if err := os.WriteFile(podConfigPath, podConfigData, 0644); err != nil {
		a.fm.T().Fatalf("Failed to write pod config: %v", err)
	}

	// Create container config JSON
	containerConfig := crioContainerConfig{}
	containerConfig.Metadata.Name = containerName
	containerConfig.Image.Image = args.Image
	if len(cmd) > 0 {
		containerConfig.Command = cmd[:1]
		if len(cmd) > 1 {
			containerConfig.Args = cmd[1:]
		}
	}

	containerConfigPath := filepath.Join(tmpDir, "container-config.json")
	containerConfigData, err := json.Marshal(containerConfig)
	if err != nil {
		a.fm.T().Fatalf("Failed to marshal container config: %v", err)
	}
	if err := os.WriteFile(containerConfigPath, containerConfigData, 0644); err != nil {
		a.fm.T().Fatalf("Failed to write container config: %v", err)
	}

	// Pull the image first
	klog.Infof("Pulling image %s", args.Image)
	a.fm.Shell().Run("sudo", "crictl", "pull", args.Image)

	// Create pod sandbox
	klog.Infof("Creating pod sandbox %s", podName)
	podOutput, _ := a.fm.Shell().Run("sudo", "crictl", "runp", podConfigPath)
	podID := strings.TrimSpace(podOutput)
	if podID == "" {
		a.fm.T().Fatalf("Failed to create pod sandbox, got empty pod ID")
	}
	klog.Infof("Created pod sandbox with ID: %s", podID)

	// Create container
	klog.Infof("Creating container %s in pod %s", containerName, podID)
	containerOutput, _ := a.fm.Shell().Run("sudo", "crictl", "create", podID, containerConfigPath, podConfigPath)
	containerID := strings.TrimSpace(containerOutput)
	if containerID == "" {
		a.fm.T().Fatalf("Failed to create container, got empty container ID")
	}
	klog.Infof("Created container with ID: %s", containerID)

	// Start container
	klog.Infof("Starting container %s", containerID)
	a.fm.Shell().Run("sudo", "crictl", "start", containerID)

	// Register cleanup function (in reverse order: container first, then pod)
	a.fm.cleanups = append(a.fm.cleanups, func() {
		klog.Infof("Cleaning up container %s and pod %s", containerID, podID)
		// Stop and remove container
		a.fm.Shell().Run("sudo", "crictl", "stop", containerID)
		a.fm.Shell().Run("sudo", "crictl", "rm", containerID)
		// Stop and remove pod
		a.fm.Shell().Run("sudo", "crictl", "stopp", podID)
		a.fm.Shell().Run("sudo", "crictl", "rmp", podID)
		// Clean up temp directory
		os.RemoveAll(tmpDir)
	})

	return containerID
}

// Containerd actions implementation

func (a *containerdActions) RunPause() string {
	return a.Run(ContainerdRunArgs{
		Image: "registry.k8s.io/pause:3.9",
	})
}

func (a *containerdActions) RunBusybox(cmd ...string) string {
	return a.Run(ContainerdRunArgs{
		Image: "registry.k8s.io/busybox:1.27",
	}, cmd...)
}

// Run creates and starts a containerd container using the ctr CLI.
// It uses the configured namespace (default "moby" for Docker-in-Docker environments).
func (a *containerdActions) Run(args ContainerdRunArgs, cmd ...string) string {
	a.seqNum++
	containerName := args.Name
	if containerName == "" {
		// Generate a unique 64-char hex container ID
		// cAdvisor's containerd handler expects container IDs to match this format
		// Use timestamp in nanoseconds to ensure uniqueness across test runs
		containerName = fmt.Sprintf("%016x%016x%016x%016x", os.Getpid(), a.seqNum, time.Now().UnixNano(), time.Now().UnixNano()%1000000)
	}

	// Build the ctr command
	// ctr -a <socket> -n <namespace> run -d <image> <container-id> [cmd...]
	ctrArgs := []string{
		"ctr",
		"--address", a.socket,
		"--namespace", a.namespace,
	}

	// Pull the image first
	klog.Infof("Pulling containerd image %s", args.Image)
	pullArgs := append(ctrArgs, "image", "pull", args.Image)
	a.fm.Shell().Run("sudo", pullArgs...)

	// Build the run command
	// Use the configured snapshotter (from CONTAINERD_SNAPSHOTTER env var, default overlayfs)
	runArgs := append(ctrArgs, "run", "-d", "--snapshotter", a.snapshotter)

	// Add labels if specified
	for key, value := range args.Labels {
		runArgs = append(runArgs, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add the image and container name
	runArgs = append(runArgs, args.Image, containerName)

	// Add the command if specified
	if len(cmd) > 0 {
		runArgs = append(runArgs, cmd...)
	}

	klog.Infof("Creating containerd container %s", containerName)
	a.fm.Shell().Run("sudo", runArgs...)

	// ctr run returns the container ID (which is the same as the name we provided)
	containerID := containerName

	klog.Infof("Created containerd container with ID: %s", containerID)

	// Register cleanup function
	// Use RunStress for cleanup commands to avoid test failures when containers have already exited
	a.fm.cleanups = append(a.fm.cleanups, func() {
		klog.Infof("Cleaning up containerd container %s", containerID)
		// Kill the task with SIGKILL to ensure it stops immediately
		// Use RunStress so we don't fail if the task has already exited
		killArgs := append([]string{"ctr", "--address", a.socket, "--namespace", a.namespace},
			"task", "kill", "--signal", "SIGKILL", containerID)
		a.fm.Shell().RunStress("sudo", killArgs...)
		// Wait a moment for the task to stop
		time.Sleep(500 * time.Millisecond)
		// Delete the task (with force flag)
		deleteTaskArgs := append([]string{"ctr", "--address", a.socket, "--namespace", a.namespace},
			"task", "delete", "-f", containerID)
		a.fm.Shell().RunStress("sudo", deleteTaskArgs...)
		// Delete the container
		deleteArgs := append([]string{"ctr", "--address", a.socket, "--namespace", a.namespace},
			"container", "delete", containerID)
		a.fm.Shell().RunStress("sudo", deleteArgs...)
	})

	return containerID
}
