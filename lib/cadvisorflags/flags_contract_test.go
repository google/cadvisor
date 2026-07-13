// Copyright 2024 Google Inc. All Rights Reserved.
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

package cadvisorflags_test

import (
	"flag"
	"testing"

	"github.com/google/cadvisor/lib/cadvisorflags"

	// Mirror exactly the cAdvisor packages the kubelet blank-imports in
	// cmd/kubelet/app/options/globalflags_linux.go so their init() functions
	// register the global flags this contract pins.
	_ "github.com/google/cadvisor/lib/container/common"
	_ "github.com/google/cadvisor/lib/container/containerd"
	_ "github.com/google/cadvisor/lib/container/raw"
	_ "github.com/google/cadvisor/lib/machine"
	_ "github.com/google/cadvisor/lib/manager"
	_ "github.com/google/cadvisor/lib/storage"
)

// TestPinnedFlagsResolve guarantees every flag name exported by this package
// still resolves to a flag registered on the global flag set. If this fails,
// the kubelet (which looks these names up by string) would panic at startup —
// so the failure belongs here, in this repo, not in the consumer.
func TestPinnedFlagsResolve(t *testing.T) {
	names := append(cadvisorflags.Kept(), cadvisorflags.Deprecated()...)
	for _, name := range names {
		if flag.CommandLine.Lookup(name) == nil {
			t.Errorf("flag %q is pinned by the kubelet but not registered on the global flag set", name)
		}
	}
}
