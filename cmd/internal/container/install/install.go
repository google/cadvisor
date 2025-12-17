// Copyright 2019 Google Inc. All Rights Reserved.
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

// The install package registers all included container providers when imported
package install

import (
	// Register all included container providers.
	_ "github.com/google/cadvisor/container/containerd/install"
	_ "github.com/google/cadvisor/container/crio/install"
	_ "github.com/google/cadvisor/container/docker/install"
	_ "github.com/google/cadvisor/container/podman/install"
	_ "github.com/google/cadvisor/container/systemd/install"

	// Register all filesystem plugins.
	_ "github.com/google/cadvisor/fs/btrfs/install"
	_ "github.com/google/cadvisor/fs/devicemapper/install"
	_ "github.com/google/cadvisor/fs/nfs/install"
	_ "github.com/google/cadvisor/fs/overlay/install"
	_ "github.com/google/cadvisor/fs/tmpfs/install"
	_ "github.com/google/cadvisor/fs/vfs/install"
	_ "github.com/google/cadvisor/fs/zfs/install"
)
