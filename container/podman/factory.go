// Copyright 2021 Google Inc. All Rights Reserved.
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

package podman

import (
	"flag"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/watcher"

	docker "github.com/docker/docker/client"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"
)

var ArgPodmanEndpoint = flag.String("podman", "unix:///run/podman/podman.sock", "podman endpoint")
var ArgPodmanTLS = flag.Bool("podman-tls", false, "use TLS to connect to podman")
var ArgPodmanCert = flag.String("podman-tls-cert", "cert.pem", "path to client certificate")
var ArgPodmanKey = flag.String("podman-tls-key", "key.pem", "path to private key")
var ArgPodmanCA = flag.String("podman-tls-ca", "ca.pem", "path to trusted CA")

// The namespace under which podman aliases are unique.
const PodmanNamespace = "podman"

// The retry times for getting podman root dir
const rootDirRetries = 5

// The retry period for getting podman root dir, Millisecond
const rootDirRetryPeriod time.Duration = 1000 * time.Millisecond

// Regexp that identifies podman cgroups
// --cgroup-parent have another prefix than 'libpod'
var podmanCgroupRegexp = regexp.MustCompile(`([a-z0-9]{64})`)

var podmanEnvWhitelist = flag.String("podman_env_metadata_whitelist", "", "a comma-separated list of environment variable keys matched with specified prefix that needs to be collected for podman containers")

var (
	// Basepath to all container specific information that libcontainer stores.
	podmanRootDir string

	podmanRootDirOnce sync.Once
)

func RootDir() string {
	podmanRootDirOnce.Do(func() {
		for i := 0; i < rootDirRetries; i++ {
			status, err := Status()
			if err == nil && status.RootDir != "" {
				podmanRootDir = status.RootDir
				break
			} else {
				time.Sleep(rootDirRetryPeriod)
			}
		}
	})
	return podmanRootDir
}

type podmanFactory struct {
	machineInfoFactory info.MachineInfoFactory

	client *docker.Client

	// Information about the mounted cgroup subsystems.
	cgroupSubsystems libcontainer.CgroupSubsystems

	// Information about mounted filesystems.
	fsInfo fs.FsInfo

	podmanVersion []int

	podmanAPIVersion []int

	includedMetrics container.MetricSet
}

func (f *podmanFactory) String() string {
	return PodmanNamespace
}

func (f *podmanFactory) NewContainerHandler(name string, inHostNamespace bool) (handler container.ContainerHandler, err error) {
	client, err := Client()
	if err != nil {
		return
	}

	metadataEnvs := strings.Split(*podmanEnvWhitelist, ",")

	handler, err = newPodmanContainerHandler(
		client,
		name,
		f.machineInfoFactory,
		f.fsInfo,
		&f.cgroupSubsystems,
		inHostNamespace,
		metadataEnvs,
		f.podmanVersion,
		f.includedMetrics,
	)
	return
}

// Returns the Podman ID from the full container name.
func CgroupNameToPodmanId(name string) string {
	id := path.Base(name)

	if matches := podmanCgroupRegexp.FindStringSubmatch(id); matches != nil {
		return matches[1]
	}

	return id
}

// isContainerName returns true if the cgroup with associated name
// could be a podman container.
// the actual decision is made by running a ContainerInspect API call
func isContainerName(name string) bool {
	// always ignore .mount cgroup even if associated with podman and delegate to systemd
	if strings.HasSuffix(name, ".mount") {
		return false
	}
	return podmanCgroupRegexp.MatchString(path.Base(name))
}

// Podman handles all containers prefixed with libpod-
func (f *podmanFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	// if the container is not associated with podman, we can't handle it or accept it.
	if !isContainerName(name) {
		return false, false, nil
	}

	// Check if the container is known to podman and it is active.
	id := CgroupNameToPodmanId(name)

	// We assume that if Inspect fails then the container is not known to podman.
	ctnr, err := f.client.ContainerInspect(context.Background(), id)
	if err != nil || !ctnr.State.Running {
		return false, true, fmt.Errorf("error inspecting container: %v", err)
	}

	return true, true, nil
}

func (f *podmanFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

var (
	versionRegexpString    = `(\d+)\.(\d+)\.(\d+)`
	versionRe              = regexp.MustCompile(versionRegexpString)
	apiVersionRegexpString = `(\d+)\.(\d+)`
	apiVersionRe           = regexp.MustCompile(apiVersionRegexpString)
)

// Register root container before running this function!
func Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) error {
	client, err := Client()
	if err != nil {
		return fmt.Errorf("unable to communicate with podman: %v", err)
	}

	podmanInfo, err := ValidateInfo()
	if err != nil {
		return fmt.Errorf("failed to validate Podman info: %v", err)
	}

	// Version already validated above, assume no error here.
	podmanVersion, _ := parseVersion(podmanInfo.ServerVersion, versionRe, 3)

	podmanAPIVersion, _ := APIVersion()

	cgroupSubsystems, err := libcontainer.GetCgroupSubsystems(includedMetrics)
	if err != nil {
		return fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}

	klog.V(1).Infof("Registering Podman factory")
	f := &podmanFactory{
		cgroupSubsystems:   cgroupSubsystems,
		client:             client,
		podmanVersion:      podmanVersion,
		podmanAPIVersion:   podmanAPIVersion,
		fsInfo:             fsInfo,
		machineInfoFactory: factory,
		includedMetrics:    includedMetrics,
	}

	container.RegisterContainerHandlerFactory(f, []watcher.ContainerWatchSource{watcher.Raw})
	return nil
}
