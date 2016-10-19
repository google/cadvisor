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

// Handler for VMDK Docker volume driver.
package drivers

import (
	info "github.com/google/cadvisor/info/v1"
	docker "github.com/docker/engine-api/client"
	"golang.org/x/net/context"
)

const io_stats="iostats"

type VmdkVolumeStats struct {
}

func NewVmdkDriver() *VmdkVolumeStats {
	return &VmdkVolumeStats{}
}

// Stats returned for a VMDK volume.
func (stats *VmdkVolumeStats) GetStats(client *docker.Client, name string) (info.VolumeIoStats, error) {
	volume, err := client.VolumeInspect(context.Background(), name)

	if err == nil && volume.Status[io_stats] != nil {
                io := info.VolumeIoStats{}
		for k, v := range volume.Status[io_stats].(map[string]interface{}) {
			io.IoStats[k] = v
		}
		io.Name = name

		return io, nil
	} else {
		return info.VolumeIoStats{}, err
	}
}
