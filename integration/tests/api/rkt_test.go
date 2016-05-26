// Copyright 2016 Google Inc. All Rights Reserved.
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

package api

import (
	//        "fmt"
	//        "os"
	//        "strconv"
	"testing"
	//        "time"

	"github.com/google/cadvisor/container/rkt"
	info "github.com/google/cadvisor/info/v1"
	//        "github.com/google/cadvisor/info/v2"
	"github.com/google/cadvisor/integration/framework"

	//        "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A Rkt container by id
func TestRktContainerById(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerId := fm.Rkt().RunPause()

	// Wait for the container to show up.
	waitForContainer(rkt.RktNamespace, containerId, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().NamespacedContainer(rkt.RktNamespace, containerId, request)
	require.NoError(t, err)

	sanityCheck(containerId, containerInfo, t)
}
