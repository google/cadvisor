//go:build libpfm && cgo

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

package perf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	info "github.com/google/cadvisor/info/v1"
	"github.com/google/cadvisor/lib/stats"
)

type trackingCollector struct {
	updates   int
	destroyed bool
}

func (c *trackingCollector) UpdateStats(*info.ContainerStats) error {
	c.updates++
	return nil
}

func (c *trackingCollector) Destroy() {
	c.destroyed = true
}

func TestRefreshingCollectorReopensAfterCgroupRecreation(t *testing.T) {
	parent := t.TempDir()
	cgroupPath := filepath.Join(parent, "service")
	require.NoError(t, os.Mkdir(cgroupPath, 0755))

	var collectors []*trackingCollector
	factory := func(path string) (stats.Collector, cgroupIdentity, error) {
		identity, err := getCgroupIdentity(path)
		if err != nil {
			return nil, cgroupIdentity{}, err
		}
		collector := &trackingCollector{}
		collectors = append(collectors, collector)
		return collector, identity, nil
	}

	collector, err := newRefreshingCollector(cgroupPath, factory)
	require.NoError(t, err)
	require.Len(t, collectors, 1)

	require.NoError(t, collector.UpdateStats(&info.ContainerStats{}))
	assert.Equal(t, 1, collectors[0].updates)

	require.NoError(t, os.Rename(cgroupPath, filepath.Join(parent, "old-service")))
	require.NoError(t, os.Mkdir(cgroupPath, 0755))

	require.NoError(t, collector.UpdateStats(&info.ContainerStats{}))
	require.Len(t, collectors, 2)
	assert.True(t, collectors[0].destroyed)
	assert.Equal(t, 1, collectors[0].updates)
	assert.Equal(t, 1, collectors[1].updates)

	collector.Destroy()
	assert.True(t, collectors[1].destroyed)
}

func TestRefreshingCollectorRejectsNilStats(t *testing.T) {
	cgroupPath := filepath.Join(t.TempDir(), "service")
	require.NoError(t, os.Mkdir(cgroupPath, 0755))

	underlying := &trackingCollector{}
	factory := func(path string) (stats.Collector, cgroupIdentity, error) {
		identity, err := getCgroupIdentity(path)
		return underlying, identity, err
	}
	collector, err := newRefreshingCollector(cgroupPath, factory)
	require.NoError(t, err)

	assert.Error(t, collector.UpdateStats(nil))
	assert.Zero(t, underlying.updates)
}

func TestRefreshingCollectorSkipsMissingCgroup(t *testing.T) {
	cgroupPath := filepath.Join(t.TempDir(), "service")
	require.NoError(t, os.Mkdir(cgroupPath, 0755))

	underlying := &trackingCollector{}
	factory := func(path string) (stats.Collector, cgroupIdentity, error) {
		identity, err := getCgroupIdentity(path)
		return underlying, identity, err
	}
	collector, err := newRefreshingCollector(cgroupPath, factory)
	require.NoError(t, err)

	require.NoError(t, os.Remove(cgroupPath))
	containerStats := &info.ContainerStats{PerfStats: []info.PerfStat{{}}}
	err = collector.UpdateStats(containerStats)

	assert.Error(t, err)
	assert.Empty(t, containerStats.PerfStats)
	assert.Zero(t, underlying.updates)
}
