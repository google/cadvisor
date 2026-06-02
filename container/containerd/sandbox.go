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

// Per-container stats for sandboxed runtimes.
//
// Some container runtimes run the workload's processes inside a sandbox -- a
// user-space kernel (gVisor/runsc) or a VM (Kata, Firecracker) -- rather than
// directly on the host. For those, the per-container host cgroup that cAdvisor
// reads is empty, so libcontainer's GetStats returns zero CPU/memory/network/
// pids. The runtime, however, holds the real per-container accounting and
// exposes it out of band (the same source containerd's CRI stats path consumes
// for the kubelet /stats/summary endpoint).
//
// A sandboxStatsProvider knows how to fetch those values for one such runtime
// and overlay them onto the (zero) libcontainer base. Detection is by the
// containerd runtime name; runc and other host-cgroup runtimes have no provider
// and keep the standard libcontainer path unchanged.
package containerd

import (
	"context"
	"strings"

	"github.com/google/cadvisor/container"
	info "github.com/google/cadvisor/info/v1"
)

// sandboxStatsProvider supplies per-container stats for a sandboxed runtime
// whose processes do not live in the host cgroup.
type sandboxStatsProvider interface {
	// name identifies the provider, for logging.
	name() string

	// overlay fetches per-container stats for id and overlays them onto stats,
	// honoring includedMetrics. It should leave fields it cannot populate as
	// the libcontainer base. A returned error signals the caller to keep the
	// base stats (best-effort: a transient failure must not fail a scrape).
	overlay(ctx context.Context, id string, stats *info.ContainerStats, includedMetrics container.MetricSet) error
}

// sandboxRuntime binds a containerd runtime-name prefix to its stats provider.
type sandboxRuntime struct {
	// prefix is matched against containerd's container Runtime.Name, e.g.
	// "io.containerd.runsc" matches "io.containerd.runsc.v1".
	prefix   string
	provider sandboxStatsProvider
}

// sandboxRuntimes is the registry of runtimes whose host cgroup is empty and
// whose stats must come from the runtime instead. To support a new runtime
// (e.g. Kata or Firecracker), add an entry here with a provider -- the handler
// wiring is runtime-agnostic. A generic provider backed by containerd's task
// Metrics API (which every shim implements, returning the standard cgroup
// metrics proto) could serve any sandboxed runtime for CPU/memory/pids; runsc
// uses a dedicated provider because it also reports per-interface network
// stats, which the cgroup metrics proto does not carry.
var sandboxRuntimes = []sandboxRuntime{
	{prefix: "io.containerd.runsc", provider: runscStatsProvider{}},
}

// sandboxProviderFor returns the stats provider for a containerd runtime name,
// or nil if the runtime's processes live in the host cgroup (e.g. runc) and the
// standard libcontainer cgroup read is authoritative.
func sandboxProviderFor(runtimeName string) sandboxStatsProvider {
	for _, r := range sandboxRuntimes {
		if strings.HasPrefix(runtimeName, r.prefix) {
			return r.provider
		}
	}
	return nil
}
