// Copyright 2017 Google Inc. All Rights Reserved.
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

package crio

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/google/cadvisor/container"
	"github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/manager/watcher"

	"k8s.io/klog"
)

// The namespace under which crio aliases are unique.
const CrioNamespace = "crio"

// Regexp that identifies CRI-O cgroups
var crioCgroupRegexp = regexp.MustCompile(`([a-z0-9]{64})`)

type storageDriver string

const (
	// TODO add full set of supported drivers in future..
	overlayStorageDriver  storageDriver = "overlay"
	overlay2StorageDriver storageDriver = "overlay2"
)

type crioFactory struct {
	machineInfoFactory info.MachineInfoFactory

	storageDriver storageDriver
	storageDir    string

	// Information about the mounted cgroup subsystems.
	cgroupSubsystems libcontainer.CgroupSubsystems

	// Information about mounted filesystems.
	fsInfo fs.FsInfo

	includedMetrics container.MetricSet

	client crioClient
}

func (self *crioFactory) String() string {
	return CrioNamespace
}

func (self *crioFactory) NewContainerHandler(name string, inHostNamespace bool) (handler container.ContainerHandler, err error) {
	client, err := Client()
	if err != nil {
		return
	}
	// TODO are there any env vars we need to white list, if so, do it here...
	metadataEnvs := []string{}
	handler, err = newCrioContainerHandler(
		client,
		name,
		self.machineInfoFactory,
		self.fsInfo,
		self.storageDriver,
		self.storageDir,
		&self.cgroupSubsystems,
		inHostNamespace,
		metadataEnvs,
		self.includedMetrics,
	)
	return
}

// Returns the CRIO ID from the full container name.
func ContainerNameToCrioId(name string) string {
	id := path.Base(name)

	if matches := crioCgroupRegexp.FindStringSubmatch(id); matches != nil {
		return matches[1]
	}

	return id
}

// isContainerName returns true if the cgroup with associated name
// corresponds to a crio container.
func isContainerName(name string) bool {
	// always ignore .mount cgroup even if associated with crio and delegate to systemd
	if strings.HasSuffix(name, ".mount") {
		return false
	}
	return crioCgroupRegexp.MatchString(path.Base(name))
}

// crio handles all containers under /crio
func (self *crioFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	if strings.HasPrefix(path.Base(name), "crio-conmon") {
		// TODO(runcom): should we include crio-conmon cgroups?
		return false, false, nil
	}
	if !strings.HasPrefix(path.Base(name), CrioNamespace) {
		return false, false, nil
	}
	// if the container is not associated with CRI-O, we can't handle it or accept it.
	if !isContainerName(name) {
		return false, false, nil
	}
	return true, true, nil
}

func (self *crioFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

var (
	// TODO(runcom): handle versioning in CRI-O
	version_regexp_string    = `(\d+)\.(\d+)\.(\d+)`
	version_re               = regexp.MustCompile(version_regexp_string)
	apiversion_regexp_string = `(\d+)\.(\d+)`
	apiversion_re            = regexp.MustCompile(apiversion_regexp_string)
)

// Register root container before running this function!
func Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) error {
	client, err := Client()
	if err != nil {
		return err
	}

	info, err := client.Info()
	if err != nil {
		return err
	}

	// TODO determine crio version so we can work differently w/ future versions if needed

	cgroupSubsystems, err := libcontainer.GetCgroupSubsystems()
	if err != nil {
		return fmt.Errorf("failed to get cgroup subsystems: %v", err)
	}

	klog.V(1).Infof("Registering CRI-O factory")
	f := &crioFactory{
		client:             client,
		cgroupSubsystems:   cgroupSubsystems,
		fsInfo:             fsInfo,
		machineInfoFactory: factory,
		storageDriver:      storageDriver(info.StorageDriver),
		storageDir:         info.StorageRoot,
		includedMetrics:    includedMetrics,
	}

	container.RegisterContainerHandlerFactory(f, []watcher.ContainerWatchSource{watcher.Raw})
	return nil
}
