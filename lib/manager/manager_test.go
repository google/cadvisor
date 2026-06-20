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

package manager

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/cadvisor/lib/cache/memory"
	"github.com/google/cadvisor/lib/container"
	containertest "github.com/google/cadvisor/lib/container/testing"
	info "github.com/google/cadvisor/lib/model"
	itest "github.com/google/cadvisor/lib/model/test"
	"github.com/google/cadvisor/lib/stats"
	"github.com/google/cadvisor/lib/utils/sysfs/fakesysfs"

	"github.com/stretchr/testify/assert"

	"k8s.io/utils/clock"
	clocktesting "k8s.io/utils/clock/testing"
)

// mockHandler is a minimal container.ContainerHandler for exercising the
// manager's query/seam surface without the full (pruned) mock packages.
type mockHandler struct {
	ref  info.ContainerReference
	spec info.ContainerSpec
}

func (h *mockHandler) ContainerReference() (info.ContainerReference, error) { return h.ref, nil }
func (h *mockHandler) GetSpec() (info.ContainerSpec, error)                 { return h.spec, nil }
func (h *mockHandler) GetStats() (*info.ContainerStats, error) {
	return &info.ContainerStats{Timestamp: time.Now()}, nil
}
func (h *mockHandler) ListContainers(container.ListType) ([]info.ContainerReference, error) {
	return nil, nil
}
func (h *mockHandler) ListProcesses(container.ListType) ([]int, error) { return nil, nil }
func (h *mockHandler) GetCgroupPath(string) (string, error)            { return "/", nil }
func (h *mockHandler) GetContainerLabels() map[string]string           { return nil }
func (h *mockHandler) GetContainerIPAddress() string                   { return "" }
func (h *mockHandler) GetExitCode() (int, error)                       { return 0, nil }
func (h *mockHandler) Exists() bool                                    { return true }
func (h *mockHandler) Cleanup()                                        {}
func (h *mockHandler) Start()                                          {}
func (h *mockHandler) Type() container.ContainerType                   { return container.ContainerTypeRaw }

func newTestManager() *manager {
	return &manager{memoryCache: memory.New(time.Minute, nil)}
}

// addContainer registers a tracked container backed by a mock handler and seeds
// one stats sample so the query methods return data.
func (m *manager) addContainer(t *testing.T, name, namespace string) *containerData {
	t.Helper()
	h := &mockHandler{
		ref:  info.ContainerReference{Name: name, Namespace: namespace},
		spec: info.ContainerSpec{HasCpu: true},
	}
	cont, err := newContainerData(name, m.memoryCache, h, time.Minute, false, clock.RealClock{})
	if err != nil {
		t.Fatalf("newContainerData(%q): %v", name, err)
	}
	if err := m.memoryCache.AddStats(&info.ContainerInfo{ContainerReference: h.ref}, &info.ContainerStats{Timestamp: time.Now()}); err != nil {
		t.Fatalf("AddStats(%q): %v", name, err)
	}
	m.containers.Store(namespacedContainerName{Namespace: namespace, Name: name}, cont)
	return cont
}

func TestExists(t *testing.T) {
	m := newTestManager()
	m.addContainer(t, "/x", "")
	if !m.Exists("/x") {
		t.Errorf("Exists(/x) = false, want true")
	}
	if m.Exists("/y") {
		t.Errorf("Exists(/y) = true, want false")
	}
}

func TestGetContainerInfo(t *testing.T) {
	m := newTestManager()
	m.addContainer(t, "/x", "")
	cinfo, err := m.GetContainerInfo("/x", &info.ContainerInfoRequest{NumStats: 10})
	if err != nil {
		t.Fatalf("GetContainerInfo: %v", err)
	}
	if cinfo.Name != "/x" {
		t.Errorf("Name = %q, want /x", cinfo.Name)
	}
	if !cinfo.Spec.HasCpu {
		t.Errorf("Spec.HasCpu = false, want true")
	}
	if len(cinfo.Stats) == 0 {
		t.Errorf("Stats empty, want >=1")
	}
	if _, err := m.GetContainerInfo("/missing", &info.ContainerInfoRequest{}); err == nil {
		t.Errorf("GetContainerInfo(/missing) err = nil, want unknown-container error")
	}
}

func TestAllDockerContainers(t *testing.T) {
	m := newTestManager()
	m.addContainer(t, "/docker/abc", DockerNamespace)
	m.addContainer(t, "/x", "") // non-docker, must be excluded
	docker, err := m.AllDockerContainers(&info.ContainerInfoRequest{NumStats: 1})
	if err != nil {
		t.Fatalf("AllDockerContainers: %v", err)
	}
	if _, ok := docker["/docker/abc"]; !ok {
		t.Errorf("docker container /docker/abc missing from %v", keys(docker))
	}
	if _, ok := docker["/x"]; ok {
		t.Errorf("non-docker /x leaked into AllDockerContainers")
	}
}

// TestGetDerivedStatsNotEnabled exercises the summary seam default: with no
// SummaryReaderFactory wired (the kubelet case) each container reports "not
// enabled" rather than panicking.
func TestGetDerivedStatsNotEnabled(t *testing.T) {
	m := newTestManager()
	m.addContainer(t, "/x", "")
	_, err := m.GetDerivedStats("/x", info.RequestOptions{IdType: info.TypeName, Count: 1})
	if err == nil {
		t.Errorf("GetDerivedStats err = nil, want 'not enabled' (no summary reader wired)")
	}
}

// TestGetProcessListSeam exercises the ps seam: nil provider yields an empty
// list; a wired provider is invoked with the container's identity.
func TestGetProcessListSeam(t *testing.T) {
	m := newTestManager()
	m.addContainer(t, "/x", "")

	ProcessListProvider = nil
	got, err := m.GetProcessList("/x", info.RequestOptions{IdType: info.TypeName, Count: 1})
	if err != nil || len(got) != 0 {
		t.Errorf("GetProcessList(nil provider) = (%v, %v), want ([], nil)", got, err)
	}

	var gotName string
	ProcessListProvider = func(name string, isRoot bool, _ string, _ bool) ([]info.ProcessInfo, error) {
		gotName = name
		return []info.ProcessInfo{{Pid: 42}}, nil
	}
	defer func() { ProcessListProvider = nil }()
	got, err = m.GetProcessList("/x", info.RequestOptions{IdType: info.TypeName, Count: 1})
	if err != nil {
		t.Fatalf("GetProcessList(provider): %v", err)
	}
	if len(got) != 1 || got[0].Pid != 42 {
		t.Errorf("GetProcessList = %v, want one proc pid=42", got)
	}
	if gotName != "/x" {
		t.Errorf("provider got container %q, want /x", gotName)
	}
}

func keys(m map[string]info.ContainerInfo) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TODO(vmarmol): Refactor these tests.

func createManagerAndAddContainers(
	memoryCache *memory.InMemoryCache,
	sysfs *fakesysfs.FakeSysFs,
	containers []string,
	f func(*containertest.MockContainerHandler),
	t *testing.T,
) *manager {
	container.ClearContainerHandlerFactories()
	mif := &manager{
		quitChannels: make([]chan error, 0, 2),
		memoryCache:  memoryCache,
	}
	for _, name := range containers {
		mockHandler := containertest.NewMockContainerHandler(name)
		spec := itest.GenerateRandomContainerSpec(4)
		mockHandler.On("GetSpec").Return(
			spec,
			nil,
		).Once()
		cont, err := newContainerData(name, memoryCache, mockHandler, 60*time.Second, true, clocktesting.NewFakeClock(time.Now()))
		if err != nil {
			t.Fatal(err)
		}
		mif.containers.Store(namespacedContainerName{
			Name: name,
		}, cont)
		// Add Docker containers under their namespace.
		if strings.HasPrefix(name, "/docker") {
			mif.containers.Store(namespacedContainerName{
				Namespace: DockerNamespace,
				Name:      strings.TrimPrefix(name, "/docker/"),
			}, cont)
		}
		f(mockHandler)
	}
	return mif
}

func createManagerAndAddSubContainers(
	memoryCache *memory.InMemoryCache,
	sysfs *fakesysfs.FakeSysFs,
	containers []string,
	f func(*containertest.MockContainerHandler),
	t *testing.T,
) *manager {
	container.ClearContainerHandlerFactories()
	mif := &manager{
		quitChannels: make([]chan error, 0, 2),
		memoryCache:  memoryCache,
	}

	subcontainers1 := []info.ContainerReference{
		{Name: "/kubepods/besteffort"},
		{Name: "/kubepods/burstable"},
	}
	subcontainers2 := []info.ContainerReference(nil)
	subcontainers3 := []info.ContainerReference{
		{Name: "/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12cd"},
		{Name: "/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12ce"},
	}
	subcontainers4 := []info.ContainerReference{
		{Name: "/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12cd/22f44d2a517778590e2d8bcafafe501f79e8a509e5b6de70b7700c4d37722bce"},
		{Name: "/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12cd/ae9465f98d275998e148b6fc12f5f92e5d4a64fca0d255f6dc3a13cc6f93a10f"},
	}

	subcontainers5 := []info.ContainerReference(nil)
	subcontainers6 := []info.ContainerReference(nil)

	subcontainerList := [][]info.ContainerReference{subcontainers1, subcontainers2, subcontainers3, subcontainers4, subcontainers5, subcontainers6}

	for idx, name := range containers {
		mockHandler := containertest.NewMockContainerHandler(name)
		spec := itest.GenerateRandomContainerSpec(4)
		mockHandler.On("GetSpec").Return(
			spec,
			nil,
		).Once()
		mockHandler.On("ListContainers", container.ListSelf).Return(
			subcontainerList[idx],
			nil,
		)
		cont, err := newContainerData(name, memoryCache, mockHandler, 60*time.Second, true, clocktesting.NewFakeClock(time.Now()))
		if err != nil {
			t.Fatal(err)
		}
		mif.containers.Store(namespacedContainerName{
			Name: name,
		}, cont)
		// Add Docker containers under their namespace.
		if strings.HasPrefix(name, "/docker") {
			mif.containers.Store(namespacedContainerName{
				Namespace: DockerNamespace,
				Name:      strings.TrimPrefix(name, "/docker/"),
			}, cont)
		}
		f(mockHandler)
	}
	return mif
}

// Expect a manager with the specified containers and query. Returns the manager, map of ContainerInfo objects,
// and map of MockContainerHandler objects.}
func expectManagerWithContainers(containers []string, query *info.ContainerInfoRequest, t *testing.T) (*manager, map[string]*info.ContainerInfo, map[string]*containertest.MockContainerHandler) {
	infosMap := make(map[string]*info.ContainerInfo, len(containers))
	handlerMap := make(map[string]*containertest.MockContainerHandler, len(containers))

	for _, container := range containers {
		infosMap[container] = itest.GenerateRandomContainerInfo(container, 4, query, 1*time.Second)
	}

	memoryCache := memory.New(time.Duration(query.NumStats)*time.Second, nil)
	sysfs := &fakesysfs.FakeSysFs{}
	m := createManagerAndAddContainers(
		memoryCache,
		sysfs,
		containers,
		func(h *containertest.MockContainerHandler) {
			cinfo := infosMap[h.Name]
			ref, err := h.ContainerReference()
			if err != nil {
				t.Error(err)
			}

			cInfo := info.ContainerInfo{
				ContainerReference: ref,
			}
			for _, stat := range cinfo.Stats {
				err = memoryCache.AddStats(&cInfo, stat)
				if err != nil {
					t.Error(err)
				}
			}
			spec := cinfo.Spec

			h.On("ListContainers", container.ListSelf).Return(
				[]info.ContainerReference(nil),
				nil,
			)
			h.On("GetSpec").Return(
				spec,
				nil,
			).Once()
			handlerMap[h.Name] = h
		},
		t,
	)

	return m, infosMap, handlerMap
}

// Expect a manager with the specified containers and query. Returns the manager, map of ContainerInfo objects,
// and map of MockContainerHandler objects.}
func expectManagerWithSubContainers(containers []string, query *info.ContainerInfoRequest, t *testing.T) (*manager, map[string]*info.ContainerInfo, map[string]*containertest.MockContainerHandler) {
	infosMap := make(map[string]*info.ContainerInfo, len(containers))
	handlerMap := make(map[string]*containertest.MockContainerHandler, len(containers))

	for _, container := range containers {
		infosMap[container] = itest.GenerateRandomContainerInfo(container, 4, query, 1*time.Second)
	}

	memoryCache := memory.New(time.Duration(query.NumStats)*time.Second, nil)
	sysfs := &fakesysfs.FakeSysFs{}
	m := createManagerAndAddSubContainers(
		memoryCache,
		sysfs,
		containers,
		func(h *containertest.MockContainerHandler) {
			cinfo := infosMap[h.Name]
			ref, err := h.ContainerReference()
			if err != nil {
				t.Error(err)
			}

			cInfo := info.ContainerInfo{
				ContainerReference: ref,
			}
			for _, stat := range cinfo.Stats {
				err = memoryCache.AddStats(&cInfo, stat)
				if err != nil {
					t.Error(err)
				}
			}
			spec := cinfo.Spec
			h.On("GetSpec").Return(
				spec,
				nil,
			).Once()
			handlerMap[h.Name] = h
		},
		t,
	)

	return m, infosMap, handlerMap
}

// Expect a manager with the specified containers and query. Returns the manager, map of ContainerInfo objects,
// and map of MockContainerHandler objects.}
func expectManagerWithContainersV2(containers []string, query *info.ContainerInfoRequest, t *testing.T) (*manager, map[string]*info.ContainerInfo, map[string]*containertest.MockContainerHandler) {
	infosMap := make(map[string]*info.ContainerInfo, len(containers))
	handlerMap := make(map[string]*containertest.MockContainerHandler, len(containers))

	for _, container := range containers {
		infosMap[container] = itest.GenerateRandomContainerInfo(container, 4, query, 1*time.Second)
	}

	memoryCache := memory.New(time.Duration(query.NumStats)*time.Second, nil)
	sysfs := &fakesysfs.FakeSysFs{}
	m := createManagerAndAddContainers(
		memoryCache,
		sysfs,
		containers,
		func(h *containertest.MockContainerHandler) {
			cinfo := infosMap[h.Name]
			ref, err := h.ContainerReference()
			if err != nil {
				t.Error(err)
			}

			cInfo := info.ContainerInfo{
				ContainerReference: ref,
			}

			for _, stat := range cinfo.Stats {
				err = memoryCache.AddStats(&cInfo, stat)
				if err != nil {
					t.Error(err)
				}
			}
			spec := cinfo.Spec

			h.On("GetSpec").Return(
				spec,
				nil,
			).Once()
			handlerMap[h.Name] = h
		},
		t,
	)

	return m, infosMap, handlerMap
}

func TestGetContainerInfoV2(t *testing.T) {
	containers := []string{
		"/",
		"/c1",
		"/c2",
	}

	options := info.RequestOptions{
		IdType:    info.TypeName,
		Count:     1,
		Recursive: true,
	}
	query := &info.ContainerInfoRequest{
		NumStats: 2,
	}

	m, _, handlerMap := expectManagerWithContainersV2(containers, query, t)

	infos, err := m.GetContainerInfoV2("/", options)
	if err != nil {
		t.Fatalf("GetContainerInfoV2 failed: %v", err)
	}

	for container, handler := range handlerMap {
		handler.AssertExpectations(t)
		cInfo, ok := infos[container]
		assert.True(t, ok, "Missing info for container %q", container)
		assert.NotEqual(t, info.ContainerSpec{}, cInfo.Spec, "Empty spec for container %q", container)
		assert.NotEmpty(t, cInfo.Stats, "Missing stats for container %q", container)
	}
}

func TestGetContainerInfoV2Failure(t *testing.T) {
	successful := "/"
	statless := "/c1"
	failing := "/c2"
	containers := []string{
		successful, statless, failing,
	}

	options := info.RequestOptions{
		IdType:    info.TypeName,
		Count:     1,
		Recursive: true,
	}
	query := &info.ContainerInfoRequest{
		NumStats: 2,
	}

	m, _, handlerMap := expectManagerWithContainers(containers, query, t)

	// Remove /c1 stats
	err := m.memoryCache.RemoveContainer(statless)
	if err != nil {
		t.Fatalf("RemoveContainer failed: %v", err)
	}

	// Make GetSpec fail on /c2
	mockErr := fmt.Errorf("intentional GetSpec failure")
	_, err = handlerMap[failing].GetSpec()
	assert.NoError(t, err) // Use up default GetSpec call, and replace below
	handlerMap[failing].On("GetSpec").Return(info.ContainerSpec{}, mockErr)
	handlerMap[failing].On("Exists").Return(true)
	// Force GetSpec by resetting infoLastUpdatedTime to zero.
	if cont, ok := m.containers.Load(namespacedContainerName{Name: failing}); ok {
		cont.infoLastUpdatedTime.Store(0)
	}

	infos, err := m.GetContainerInfoV2("/", options)
	if err == nil {
		t.Error("Expected error calling GetContainerInfoV2")
	}

	// Successful containers still successful.
	cInfo, ok := infos[successful]
	assert.True(t, ok, "Missing info for container %q", successful)
	assert.NotEqual(t, info.ContainerSpec{}, cInfo.Spec, "Empty spec for container %q", successful)
	assert.NotEmpty(t, cInfo.Stats, "Missing stats for container %q", successful)

	// "/c1" present with spec.
	cInfo, ok = infos[statless]
	assert.True(t, ok, "Missing info for container %q", statless)
	assert.NotEqual(t, info.ContainerSpec{}, cInfo.Spec, "Empty spec for container %q", statless)
	assert.Empty(t, cInfo.Stats, "Missing stats for container %q", successful)

	// "/c2" should be present but empty.
	cInfo, ok = infos[failing]
	assert.True(t, ok, "Missing info for failed container")
	assert.Equal(t, info.ContainerInfo{}, cInfo, "Empty spec for failed container")
	assert.Empty(t, cInfo.Stats, "Missing stats for failed container")
}

func TestSubcontainersInfo(t *testing.T) {
	containers := []string{
		"/c1",
		"/c2",
	}

	query := &info.ContainerInfoRequest{
		NumStats: 64,
	}

	m, _, _ := expectManagerWithContainers(containers, query, t)

	result, err := m.SubcontainersInfo("/", query)
	if err != nil {
		t.Fatalf("expected to succeed: %s", err)
	}
	if len(result) != len(containers) {
		t.Errorf("expected to received containers: %v, but received: %v", containers, result)
	}
	for _, res := range result {
		found := false
		for _, name := range containers {
			if res.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected container %q in result, expected one of %v", res.Name, containers)
		}
	}
}

func TestSubcontainersInfoError(t *testing.T) {
	containers := []string{
		"/kubepods",
		"/kubepods/besteffort",
		"/kubepods/burstable",
		"/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12cd",
		"/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12cd/22f44d2a517778590e2d8bcafafe501f79e8a509e5b6de70b7700c4d37722bce",
		"/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12cd/ae9465f98d275998e148b6fc12f5f92e5d4a64fca0d255f6dc3a13cc6f93a10f",
	}
	query := &info.ContainerInfoRequest{
		NumStats: 1,
	}

	m, _, _ := expectManagerWithSubContainers(containers, query, t)
	result, err := m.SubcontainersInfo("/kubepods", query)
	if err != nil {
		t.Fatalf("expected to succeed: %s", err)
	}

	if len(result) != len(containers) {
		t.Errorf("expected to received containers: %v, but received: %v", containers, result)
	}

	totalBurstable := 0
	burstableCount := 0
	totalBesteffort := 0
	besteffortCount := 0

	for _, res := range result {
		found := false
		if res.Name == "/kubepods/burstable" {
			totalBurstable = len(res.Subcontainers)
		} else if res.Name == "/kubepods/besteffort" {
			totalBesteffort = len(res.Subcontainers)
		} else if strings.HasPrefix(res.Name, "/kubepods/burstable") && len(res.Name) == len("/kubepods/burstable/pod01042b28-179d-446a-954a-7266557e12cd") {
			burstableCount++
		} else if strings.HasPrefix(res.Name, "/kubepods/besteffort") && len(res.Name) == len("/kubepods/besteffort/pod01042b28-179d-446a-954a-7266557e12cd") {
			besteffortCount++
		}
		for _, name := range containers {
			if res.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected container %q in result, expected one of %v", res.Name, containers)
		}
	}

	assert.NotEqual(t, totalBurstable, burstableCount)
	assert.Equal(t, totalBesteffort, besteffortCount)
}

func TestDockerContainersInfo(t *testing.T) {
	containers := []string{
		"/docker/c1a",
		"/docker/c2a",
	}

	query := &info.ContainerInfoRequest{
		NumStats: 2,
	}

	m, _, _ := expectManagerWithContainers(containers, query, t)

	result, err := m.DockerContainer("c1a", query)
	if err != nil {
		t.Fatalf("expected to succeed: %s", err)
	}
	if result.Name != containers[0] {
		t.Errorf("Unexpected container %q in result. Expected container %q", result.Name, containers[0])
	}

	result, err = m.DockerContainer("c2", query)
	if err != nil {
		t.Fatalf("expected to succeed: %s", err)
	}
	if result.Name != containers[1] {
		t.Errorf("Unexpected container %q in result. Expected container %q", result.Name, containers[1])
	}

	result, err = m.DockerContainer("c", query)
	expectedError := "unable to find container. Container \"c\" is not unique"
	if err == nil {
		t.Errorf("expected error %q but received %q", expectedError, err)
	}
}

func TestDestroyContainerWithExitCode(t *testing.T) {
	mockHandler := containertest.NewMockContainerHandler("/test")
	mockHandler.On("GetExitCode").Return(42, nil)

	memoryCache := memory.New(60*time.Second, nil)
	m := &manager{
		quitChannels: make([]chan error, 0, 2),
		memoryCache:  memoryCache,
	}

	cont := &containerData{
		handler:          mockHandler,
		memoryCache:      memoryCache,
		perfCollector:    &stats.NoopCollector{},
		resctrlCollector: &stats.NoopCollector{},
		info: containerInfo{
			ContainerReference: info.ContainerReference{
				Name: "/test",
			},
		},
		stop: make(chan struct{}),
	}

	m.containers.Store(namespacedContainerName{Name: "/test"}, cont)

	mockEventHandler := &mockEventHandler{
		events: make([]*info.Event, 0),
	}
	m.eventSink = mockEventHandler

	err := m.destroyContainer("/test")
	if err != nil {
		t.Logf("destroyContainer error: %v", err)
	}
	assert.Nil(t, err)

	assert.Len(t, mockEventHandler.events, 1)
	event := mockEventHandler.events[0]
	assert.Equal(t, info.EventContainerDeletion, event.EventType)
	assert.NotNil(t, event.EventData.ContainerDeletion)
	assert.Equal(t, 42, event.EventData.ContainerDeletion.ExitCode)

	mockHandler.AssertExpectations(t)
}

func TestDestroyContainerExitCodeUnavailable(t *testing.T) {
	mockHandler := containertest.NewMockContainerHandler("/test")
	mockHandler.On("GetExitCode").Return(-1, fmt.Errorf("container not found"))

	memoryCache := memory.New(60*time.Second, nil)
	m := &manager{
		quitChannels: make([]chan error, 0, 2),
		memoryCache:  memoryCache,
	}

	cont := &containerData{
		handler:          mockHandler,
		memoryCache:      memoryCache,
		perfCollector:    &stats.NoopCollector{},
		resctrlCollector: &stats.NoopCollector{},
		info: containerInfo{
			ContainerReference: info.ContainerReference{
				Name: "/test",
			},
		},
		stop: make(chan struct{}),
	}

	m.containers.Store(namespacedContainerName{Name: "/test"}, cont)

	mockEventHandler := &mockEventHandler{
		events: make([]*info.Event, 0),
	}
	m.eventSink = mockEventHandler

	err := m.destroyContainer("/test")
	if err != nil {
		t.Logf("destroyContainer error: %v", err)
	}
	assert.Nil(t, err)

	assert.Len(t, mockEventHandler.events, 1)
	event := mockEventHandler.events[0]
	assert.Equal(t, info.EventContainerDeletion, event.EventType)
	assert.NotNil(t, event.EventData.ContainerDeletion)
	assert.Equal(t, -1, event.EventData.ContainerDeletion.ExitCode)

	mockHandler.AssertExpectations(t)
}

type mockEventHandler struct {
	events []*info.Event
}

func (m *mockEventHandler) AddEvent(e *info.Event) error {
	m.events = append(m.events, e)
	return nil
}

// threadSafeEventHandler is a thread-safe version of mockEventHandler for concurrent tests.
type threadSafeEventHandler struct {
	mu     sync.Mutex
	events []*info.Event
}

func (m *threadSafeEventHandler) AddEvent(e *info.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	return nil
}

// TestContainerDataStopConcurrent verifies that concurrent calls to Stop()
// do not cause a panic. This is a regression test for a race condition where
// multiple goroutines could call Stop() on the same containerData, causing a
// "close of closed channel" panic.
func TestContainerDataStopConcurrent(t *testing.T) {
	memoryCache := memory.New(60*time.Second, nil)

	// Create a minimal containerData with the fields needed for Stop()
	cd := &containerData{
		info: containerInfo{
			ContainerReference: info.ContainerReference{
				Name: "/test-concurrent",
			},
		},
		memoryCache:      memoryCache,
		stop:             make(chan struct{}),
		perfCollector:    &stats.NoopCollector{},
		resctrlCollector: &stats.NoopCollector{},
	}

	// Launch multiple goroutines that all try to call Stop() simultaneously
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Use a channel to synchronize goroutines to start at the same time
	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			<-start // Wait for signal to start
			// This should not panic even if called multiple times concurrently
			_ = cd.Stop()
		}()
	}

	// Signal all goroutines to start simultaneously
	close(start)

	// Wait for all goroutines to complete - if there's a panic, the test will fail
	wg.Wait()
}

// TestDestroyContainerConcurrent verifies that concurrent calls to destroyContainer
// for the same container do not cause a panic.
func TestDestroyContainerConcurrent(t *testing.T) {
	memoryCache := memory.New(60*time.Second, nil)
	m := &manager{
		quitChannels: make([]chan error, 0, 2),
		memoryCache:  memoryCache,
	}

	mockEventHandler := &threadSafeEventHandler{}
	mockHandler := containertest.NewMockContainerHandler("/test-concurrent")
	mockHandler.On("Start").Return(nil)
	mockHandler.On("Cleanup").Return()
	mockHandler.On("Stop").Return()
	// GetExitCode may be called multiple times due to concurrent access
	mockHandler.On("GetExitCode").Return(-1, nil).Maybe()

	// Create the container
	cd := &containerData{
		handler: mockHandler,
		info: containerInfo{
			ContainerReference: info.ContainerReference{
				Name: "/test-concurrent",
			},
		},
		memoryCache:      memoryCache,
		stop:             make(chan struct{}),
		perfCollector:    &stats.NoopCollector{},
		resctrlCollector: &stats.NoopCollector{},
	}

	// Add to manager's container map
	m.containers.Store(namespacedContainerName{Name: "/test-concurrent"}, cd)

	// Register event handler
	m.eventSink = mockEventHandler

	// Launch multiple goroutines that all try to destroy the same container
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	start := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			_ = m.destroyContainer("/test-concurrent")
		}()
	}

	close(start)
	wg.Wait()

	// With sync.Once protecting only the channel close, multiple goroutines
	// can still process the container and add events. The key assertion is
	// that no panic occurred (test would fail otherwise).
	// At least one event should be recorded.
	assert.GreaterOrEqual(t, len(mockEventHandler.events), 1, "at least one destruction event should be recorded")
}
