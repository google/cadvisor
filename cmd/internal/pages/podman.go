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

package pages

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	dockerutil "github.com/google/cadvisor/container/docker/utils"
	"github.com/google/cadvisor/container/podman"
	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/manager"

	"k8s.io/klog/v2"
)

const PodmanPage = "/podman/"

func servePodmanPage(m manager.Manager, w http.ResponseWriter, u *url.URL) {
	start := time.Now()

	containerName := u.Path[len(PodmanPage)-1:]
	rootDir := getRootDir(containerName)

	var data *pageData

	if containerName == "/" {
		// Scenario for all containers.
		status, err := podman.Status()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get podman info: %v", err), http.StatusInternalServerError)
			return
		}
		images, err := podman.Images()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get podman images: %v", err), http.StatusInternalServerError)
			return
		}

		reqParams := info.ContainerInfoRequest{
			NumStats: 0,
		}
		conts, err := m.AllPodmanContainers(&reqParams)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get container %q with error: %v", containerName, err), http.StatusNotFound)
			return
		}
		subcontainers := make([]link, 0, len(conts))
		for _, cont := range conts {
			subcontainers = append(subcontainers, link{
				Text: getContainerDisplayName(cont.ContainerReference),
				Link: path.Join(rootDir, PodmanPage, dockerutil.ContainerNameToId(cont.ContainerReference.Name)),
			})
		}

		podmanStatus, driverStatus := toStatusKV(status)

		podmanContainerText := "Podman Containers"
		data = &pageData{
			DisplayName: podmanContainerText,
			ParentContainers: []link{
				{
					Text: podmanContainerText,
					Link: path.Join(rootDir, PodmanPage),
				}},
			Subcontainers:      subcontainers,
			Root:               rootDir,
			DockerStatus:       podmanStatus,
			DockerDriverStatus: driverStatus,
			DockerImages:       images,
		}
	} else {
		// Scenario for specific container.
		machineInfo, err := m.GetMachineInfo()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get machine info: %v", err), http.StatusInternalServerError)
			return
		}

		reqParams := info.ContainerInfoRequest{
			NumStats: 60,
		}
		cont, err := m.PodmanContainer(containerName[1:], &reqParams)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get container %v with error: %v", containerName, err), http.StatusNotFound)
			return
		}
		displayName := getContainerDisplayName(cont.ContainerReference)

		var parentContainers []link
		parentContainers = append(parentContainers, link{
			Text: "Podman Containers",
			Link: path.Join(rootDir, PodmanPage),
		})
		parentContainers = append(parentContainers, link{
			Text: displayName,
			Link: path.Join(rootDir, PodmanPage, dockerutil.ContainerNameToId(cont.Name)),
		})

		data = &pageData{
			DisplayName:            displayName,
			ContainerName:          escapeContainerName(cont.Name),
			ParentContainers:       parentContainers,
			Spec:                   cont.Spec,
			Stats:                  cont.Stats,
			MachineInfo:            machineInfo,
			ResourcesAvailable:     cont.Spec.HasCpu || cont.Spec.HasMemory || cont.Spec.HasNetwork,
			CpuAvailable:           cont.Spec.HasCpu,
			MemoryAvailable:        cont.Spec.HasMemory,
			NetworkAvailable:       cont.Spec.HasNetwork,
			FsAvailable:            cont.Spec.HasFilesystem,
			CustomMetricsAvailable: cont.Spec.HasCustomMetrics,
			Root:                   rootDir,
		}
	}

	err := pageTemplate.Execute(w, data)
	if err != nil {
		klog.Errorf("Failed to apply template: %s", err)
	}

	klog.V(5).Infof("Request took %s", time.Since(start))
}
