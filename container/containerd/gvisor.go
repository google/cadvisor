// Copyright 2026 Google Inc. All Rights Reserved.
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

//go:build linux

// gVisor (runsc) stats overlay for the containerd handler.
//
// A gVisor container's processes run inside the sandbox (a single host
// process), so the per-container host cgroup that cAdvisor reads is empty and
// libcontainer reports zero CPU/memory/network/pids. gVisor's sentry holds the
// real per-container accounting and exposes it via `runsc events --stats`,
// which is the same source containerd's CRI stats path consumes for the
// kubelet `/stats/summary` endpoint.
//
// When the containerd handler detects a runsc-managed container, it overlays
// the values from `runsc events --stats` on top of the (zero) libcontainer
// stats so that /metrics/cadvisor reports real numbers, matching runc.
//
// See gVisor google/gvisor#13067 (problem) and #13070 (host-side compat dirs
// that make these containers discoverable by cAdvisor in the first place).
package containerd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
)

var (
	argRunscBinary = flag.String("runsc", "runsc", "path to the runsc binary, used to collect per-container stats for gVisor containers")
	argRunscRoot   = flag.String("runsc_root", "/run/containerd/runsc/k8s.io", "runsc root directory (containerd's runsc state dir for the configured namespace), used to collect per-container stats for gVisor containers")
)

// gvisorRuntimePrefix matches the containerd runtime names used by the runsc
// shim, e.g. "io.containerd.runsc.v1".
const gvisorRuntimePrefix = "io.containerd.runsc"

// gvisorStatsTimeout bounds a single `runsc events --stats` invocation.
const gvisorStatsTimeout = 5 * time.Second

// isGVisorRuntime reports whether the containerd runtime name belongs to the
// runsc (gVisor) shim.
func isGVisorRuntime(runtimeName string) bool {
	return strings.HasPrefix(runtimeName, gvisorRuntimePrefix)
}

// gvisorEvent mirrors the JSON emitted by `runsc events --stats <id>` (gVisor's
// runsc/boot.Event). Only the fields consumed here are declared.
type gvisorEvent struct {
	Data gvisorEventData `json:"data"`
}

type gvisorEventData struct {
	CPU               gvisorCPU                `json:"cpu"`
	Memory            gvisorMemory             `json:"memory"`
	Pids              gvisorPids               `json:"pids"`
	NetworkInterfaces []gvisorNetworkInterface `json:"network_interfaces"`
}

type gvisorCPU struct {
	Usage gvisorCPUUsage `json:"usage"`
}

type gvisorCPUUsage struct {
	// Nanoseconds.
	Kernel uint64 `json:"kernel"`
	User   uint64 `json:"user"`
	Total  uint64 `json:"total"`
}

type gvisorMemory struct {
	Cache uint64            `json:"cache"`
	Usage gvisorMemoryEntry `json:"usage"`
}

type gvisorMemoryEntry struct {
	Usage uint64 `json:"usage"`
	Max   uint64 `json:"max"`
}

type gvisorPids struct {
	Current uint64 `json:"current"`
}

// gvisorNetworkInterface mirrors runsc/boot.NetworkInterface, which has no JSON
// tags -- so the wire keys are the exported Go field names.
type gvisorNetworkInterface struct {
	Name      string
	RxBytes   uint64
	RxPackets uint64
	RxErrors  uint64
	RxDropped uint64
	TxBytes   uint64
	TxPackets uint64
	TxErrors  uint64
	TxDropped uint64
}

// gvisorContainerStats runs `runsc --root <root> events --stats <id>` once and
// parses its JSON output.
func gvisorContainerStats(ctx context.Context, id string) (*gvisorEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, gvisorStatsTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, *argRunscBinary, "--root", *argRunscRoot, "events", "--stats", id)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running %q events --stats %s (root %q): %w", *argRunscBinary, id, *argRunscRoot, err)
	}
	var ev gvisorEvent
	if err := json.Unmarshal(out, &ev); err != nil {
		return nil, fmt.Errorf("parsing runsc events output for %s: %w", id, err)
	}
	return &ev, nil
}

// applyGVisorStats overwrites the cgroup-derived (zero) values in stats with
// the per-container values gVisor reports, honoring includedMetrics. Only the
// metrics gVisor reliably provides are overlaid; the rest are left as the
// libcontainer base so consumers that read them degrade gracefully.
func applyGVisorStats(stats *info.ContainerStats, ev *gvisorEvent, includedMetrics container.MetricSet) {
	d := ev.Data

	if includedMetrics.Has(container.CpuUsageMetrics) {
		stats.Cpu.Usage.Total = d.CPU.Usage.Total
		stats.Cpu.Usage.User = d.CPU.Usage.User
		stats.Cpu.Usage.System = d.CPU.Usage.Kernel
	}

	if includedMetrics.Has(container.MemoryUsageMetrics) {
		stats.Memory.Usage = d.Memory.Usage.Usage
		stats.Memory.MaxUsage = d.Memory.Usage.Max
		stats.Memory.Cache = d.Memory.Cache
		// gVisor does not break out inactive_file, so approximate the
		// working set as usage minus page cache (clamped at >= 0).
		if d.Memory.Usage.Usage > d.Memory.Cache {
			stats.Memory.WorkingSet = d.Memory.Usage.Usage - d.Memory.Cache
		} else {
			stats.Memory.WorkingSet = d.Memory.Usage.Usage
		}
	}

	if includedMetrics.Has(container.ProcessMetrics) {
		stats.Processes.ProcessCount = d.Pids.Current
	}

	if includedMetrics.Has(container.NetworkUsageMetrics) && len(d.NetworkInterfaces) > 0 {
		ifaces := make([]info.InterfaceStats, 0, len(d.NetworkInterfaces))
		for _, ni := range d.NetworkInterfaces {
			ifaces = append(ifaces, info.InterfaceStats{
				Name:      ni.Name,
				RxBytes:   ni.RxBytes,
				RxPackets: ni.RxPackets,
				RxErrors:  ni.RxErrors,
				RxDropped: ni.RxDropped,
				TxBytes:   ni.TxBytes,
				TxPackets: ni.TxPackets,
				TxErrors:  ni.TxErrors,
				TxDropped: ni.TxDropped,
			})
		}
		stats.Network.Interfaces = ifaces
		// cAdvisor's inline InterfaceStats reports the primary interface;
		// prefer the first non-loopback one.
		stats.Network.InterfaceStats = ifaces[0]
		for _, is := range ifaces {
			if is.Name != "lo" {
				stats.Network.InterfaceStats = is
				break
			}
		}
	}
}

// overlayGVisorStats fetches gVisor's per-container stats and overlays them on
// the libcontainer base. It is best-effort: on any error the base (zeroed)
// stats are left untouched so a transient runsc failure never fails a scrape.
func (h *containerdContainerHandler) overlayGVisorStats(stats *info.ContainerStats) {
	ev, err := gvisorContainerStats(context.Background(), h.reference.Id)
	if err != nil {
		klog.V(4).Infof("containerd: could not get gVisor stats for %q, leaving cgroup-derived values: %v", h.reference.Name, err)
		return
	}
	applyGVisorStats(stats, ev, h.includedMetrics)
}
