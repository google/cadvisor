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

// Page for /containers/
package pages

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
)

const ContainersPage = "/containers/"

// from http://golang.org/doc/effective_go.html#constants
type ByteSize float64

const (
	_ = iota
	// KB - kilobyte
	KB ByteSize = 1 << (10 * iota)
	// MB - megabyte
	MB
	// GB - gigabyte
	GB
	// TB - terabyte
	TB
	// PB - petabyte
	PB
	// EB - exabyte
	EB
	// ZB - zettabyte
	ZB
	// YB - yottabyte
	YB
)

func (b ByteSize) Size() string {
	for _, i := range [...]ByteSize{YB, ZB, EB, PB, TB, GB, MB, KB} {
		if b >= i {
			return fmt.Sprintf("%.2f", b/i)
		}
	}
	return fmt.Sprintf("%.2f", b)
}

func (b ByteSize) Unit() string {
	switch {
	case b >= YB:
		return "YB"
	case b >= ZB:
		return "ZB"
	case b >= EB:
		return "EB"
	case b >= PB:
		return "PB"
	case b >= TB:
		return "TB"
	case b >= GB:
		return "GB"
	case b >= MB:
		return "MB"
	case b >= KB:
		return "KB"
	}
	return "B"
}

var funcMap = template.FuncMap{
	"containerLink":         containerLink,
	"printMask":             printMask,
	"printCores":            printCores,
	"printShares":           printShares,
	"printSize":             printSize,
	"printUnit":             printUnit,
	"getMemoryUsage":        getMemoryUsage,
	"getMemoryUsagePercent": getMemoryUsagePercent,
	"getHotMemoryPercent":   getHotMemoryPercent,
	"getColdMemoryPercent":  getColdMemoryPercent,
	"getFsStats":            getFsStats,
	"getFsUsagePercent":     getFsUsagePercent,
}

// TODO(vmarmol): Consider housekeeping Spec too so we can show changes through time. We probably don't need it ever second though.

var pageTemplate *template.Template

type pageData struct {
	ContainerName      string
	ParentContainers   []info.ContainerReference
	Subcontainers      []info.ContainerReference
	Spec               info.ContainerSpec
	Stats              []*info.ContainerStats
	MachineInfo        *info.MachineInfo
	ResourcesAvailable bool
	CpuAvailable       bool
	MemoryAvailable    bool
	NetworkAvailable   bool
	FsAvailable        bool
}

func init() {
	pageTemplate = template.New("containersTemplate").Funcs(funcMap)
	_, err := pageTemplate.Parse(containersHtmlTemplate)
	if err != nil {
		glog.Fatalf("Failed to parse template: %s", err)
	}
}

// TODO(vmarmol): Escape this correctly.
func containerLink(container info.ContainerReference, basenameOnly bool, cssClasses string) interface{} {
	var displayName string
	containerName := container.Name
	if len(container.Aliases) > 0 {
		displayName = container.Aliases[0]
	} else if basenameOnly {
		displayName = path.Base(string(container.Name))
	} else {
		displayName = string(container.Name)
	}
	if container.Name == "root" {
		containerName = "/"
	} else if strings.Contains(container.Name, " ") {
		// If it has a space, it is an a.k.a, so keep the base-name
		containerName = container.Name[:strings.Index(container.Name, " ")]
	}
	return template.HTML(fmt.Sprintf("<a class=\"%s\" href=\"%s%s\">%s</a>", cssClasses, ContainersPage[:len(ContainersPage)-1], containerName, displayName))
}

func printMask(mask string, numCores int) interface{} {
	masks := make([]string, numCores)
	activeCores := getActiveCores(mask)
	for i := 0; i < numCores; i++ {
		coreClass := "inactive-cpu"
		if activeCores[i] {
			coreClass = "active-cpu"
		}
		masks[i] = fmt.Sprintf("<span class=\"%s\">%d</span>", coreClass, i)
	}
	return template.HTML(strings.Join(masks, "&nbsp;"))
}

func getActiveCores(mask string) map[int]bool {
	activeCores := make(map[int]bool)
	for _, corebits := range strings.Split(mask, ",") {
		cores := strings.Split(corebits, "-")
		if len(cores) == 1 {
			index, err := strconv.Atoi(cores[0])
			if err != nil {
				// Ignore malformed strings.
				continue
			}
			activeCores[index] = true
		} else if len(cores) == 2 {
			start, err := strconv.Atoi(cores[0])
			if err != nil {
				continue
			}
			end, err := strconv.Atoi(cores[1])
			if err != nil {
				continue
			}
			for i := start; i <= end; i++ {
				activeCores[i] = true
			}
		}
	}
	return activeCores
}

func printCores(millicores *uint64) string {
	cores := float64(*millicores) / 1000
	return strconv.FormatFloat(cores, 'f', 3, 64)
}

func printShares(shares *uint64) string {
	return fmt.Sprintf("%d", *shares)
}

func toMegabytes(bytes uint64) float64 {
	return float64(bytes) / (1 << 20)
}

func printSize(bytes uint64) string {
	if bytes >= math.MaxInt64 {
		return "unlimited"
	}
	return ByteSize(bytes).Size()
}

func printUnit(bytes uint64) string {
	if bytes >= math.MaxInt64 {
		return ""
	}
	return ByteSize(bytes).Unit()
}

func toMemoryPercent(usage uint64, spec *info.ContainerSpec, machine *info.MachineInfo) int {
	// Saturate limit to the machine size.
	limit := uint64(spec.Memory.Limit)
	if limit > uint64(machine.MemoryCapacity) {
		limit = uint64(machine.MemoryCapacity)
	}

	return int((usage * 100) / limit)
}

func getMemoryUsage(stats []*info.ContainerStats) string {
	if len(stats) == 0 {
		return "0.0"
	}
	return strconv.FormatFloat(toMegabytes((stats[len(stats)-1].Memory.Usage)), 'f', 2, 64)
}

func getMemoryUsagePercent(spec *info.ContainerSpec, stats []*info.ContainerStats, machine *info.MachineInfo) int {
	if len(stats) == 0 {
		return 0
	}
	return toMemoryPercent((stats[len(stats)-1].Memory.Usage), spec, machine)
}

func getHotMemoryPercent(spec *info.ContainerSpec, stats []*info.ContainerStats, machine *info.MachineInfo) int {
	if len(stats) == 0 {
		return 0
	}
	return toMemoryPercent((stats[len(stats)-1].Memory.WorkingSet), spec, machine)
}

func getColdMemoryPercent(spec *info.ContainerSpec, stats []*info.ContainerStats, machine *info.MachineInfo) int {
	if len(stats) == 0 {
		return 0
	}
	latestStats := stats[len(stats)-1].Memory
	return toMemoryPercent((latestStats.Usage)-(latestStats.WorkingSet), spec, machine)
}

func getFsStats(stats []*info.ContainerStats) []info.FsStats {
	if len(stats) == 0 {
		return []info.FsStats{}
	}
	return stats[len(stats)-1].Filesystem
}

func getFsUsagePercent(limit, used uint64) uint64 {
	return uint64((float64(used) / float64(limit)) * 100)
}

func ServerContainersPage(m manager.Manager, w http.ResponseWriter, u *url.URL) error {
	start := time.Now()

	// The container name is the path after the handler
	containerName := u.Path[len(ContainersPage)-1:]

	// Get the container.
	reqParams := info.ContainerInfoRequest{
		NumStats: 60,
	}
	cont, err := m.GetContainerInfo(containerName, &reqParams)
	if err != nil {
		return fmt.Errorf("Failed to get container \"%s\" with error: %s", containerName, err)
	}

	// Get the MachineInfo
	machineInfo, err := m.GetMachineInfo()
	if err != nil {
		return err
	}

	// Make a list of the parent containers and their links
	var parentContainers []info.ContainerReference
	parentContainers = append(parentContainers, info.ContainerReference{Name: "root"})
	parentName := ""
	for _, part := range strings.Split(string(cont.Name), "/") {
		if part == "" {
			continue
		}
		parentName += "/" + part
		parentContainers = append(parentContainers, info.ContainerReference{Name: parentName})
	}

	// Pick the shortest name of the container as the display name.
	displayName := cont.Name
	for _, alias := range cont.Aliases {
		if len(displayName) >= len(alias) {
			displayName = alias
		}
	}

	// Replace the last part of the parent containers with the displayName.
	if displayName != cont.Name {
		parentContainers[len(parentContainers)-1] = info.ContainerReference{
			Name: fmt.Sprintf("%s (%s)", displayName, path.Base(cont.Name)),
		}
	}

	data := &pageData{
		ContainerName: displayName,
		// TODO(vmarmol): Only use strings for this.
		ParentContainers:   parentContainers,
		Subcontainers:      cont.Subcontainers,
		Spec:               cont.Spec,
		Stats:              cont.Stats,
		MachineInfo:        machineInfo,
		ResourcesAvailable: cont.Spec.HasCpu || cont.Spec.HasMemory || cont.Spec.HasNetwork,
		CpuAvailable:       cont.Spec.HasCpu,
		MemoryAvailable:    cont.Spec.HasMemory,
		NetworkAvailable:   cont.Spec.HasNetwork,
		FsAvailable:        cont.Spec.HasFilesystem,
	}
	err = pageTemplate.Execute(w, data)
	if err != nil {
		glog.Errorf("Failed to apply template: %s", err)
	}

	glog.V(1).Infof("Request took %s", time.Since(start))
	return nil
}
