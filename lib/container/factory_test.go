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

package container_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/cadvisor/lib/container"
	containertest "github.com/google/cadvisor/lib/container/testing"
	"github.com/google/cadvisor/lib/fs"
	info "github.com/google/cadvisor/lib/model"
	"github.com/google/cadvisor/lib/watcher"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockContainerHandlerFactory struct {
	mock.Mock
	Name           string
	CanHandleValue bool
	CanAcceptValue bool
}

func (f *mockContainerHandlerFactory) String() string {
	return f.Name
}

func (f *mockContainerHandlerFactory) DebugInfo() map[string][]string {
	return map[string][]string{}
}

func (f *mockContainerHandlerFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	return f.CanHandleValue, f.CanAcceptValue, nil
}

func (f *mockContainerHandlerFactory) NewContainerHandler(name string, metadataEnvAllowList []string, isHostNamespace bool) (container.ContainerHandler, error) {
	args := f.Called(name)
	return args.Get(0).(container.ContainerHandler), args.Error(1)
}

const testContainerName = "/test"

var testMetadataEnvAllowList = []string{}

var mockFactory containertest.FactoryForMockContainerHandler

func TestNewContainerHandler_FirstMatches(t *testing.T) {
	container.ClearContainerHandlerFactories()

	// Register one allways yes factory.
	allwaysYes := &mockContainerHandlerFactory{
		Name:           "yes",
		CanHandleValue: true,
		CanAcceptValue: true,
	}
	container.RegisterContainerHandlerFactory(allwaysYes, []watcher.ContainerWatchSource{watcher.Raw})

	// The yes factory should be asked to create the ContainerHandler.
	mockContainer, err := mockFactory.NewContainerHandler(testContainerName, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	allwaysYes.On("NewContainerHandler", testContainerName).Return(mockContainer, nil)

	cont, _, err := container.NewContainerHandler(testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	if cont == nil {
		t.Error("Expected container to not be nil")
	}
}

func TestNewContainerHandler_SecondMatches(t *testing.T) {
	container.ClearContainerHandlerFactories()

	// Register one allways no and one always yes factory.
	allwaysNo := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	container.RegisterContainerHandlerFactory(allwaysNo, []watcher.ContainerWatchSource{watcher.Raw})
	allwaysYes := &mockContainerHandlerFactory{
		Name:           "yes",
		CanHandleValue: true,
		CanAcceptValue: true,
	}
	container.RegisterContainerHandlerFactory(allwaysYes, []watcher.ContainerWatchSource{watcher.Raw})

	// The yes factory should be asked to create the ContainerHandler.
	mockContainer, err := mockFactory.NewContainerHandler(testContainerName, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	allwaysYes.On("NewContainerHandler", testContainerName).Return(mockContainer, nil)

	cont, _, err := container.NewContainerHandler(testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error(err)
	}
	if cont == nil {
		t.Error("Expected container to not be nil")
	}
}

func TestNewContainerHandler_NoneMatch(t *testing.T) {
	container.ClearContainerHandlerFactories()

	// Register two allways no factories.
	allwaysNo1 := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	container.RegisterContainerHandlerFactory(allwaysNo1, []watcher.ContainerWatchSource{watcher.Raw})
	allwaysNo2 := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	container.RegisterContainerHandlerFactory(allwaysNo2, []watcher.ContainerWatchSource{watcher.Raw})

	_, _, err := container.NewContainerHandler(testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err == nil {
		t.Error("Expected NewContainerHandler to fail")
	}
}

func TestNewContainerHandler_Accept(t *testing.T) {
	container.ClearContainerHandlerFactories()

	// Register handler that can handle the container, but can't accept it.
	cannotHandle := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: false,
		CanAcceptValue: true,
	}
	container.RegisterContainerHandlerFactory(cannotHandle, []watcher.ContainerWatchSource{watcher.Raw})
	cannotAccept := &mockContainerHandlerFactory{
		Name:           "no",
		CanHandleValue: true,
		CanAcceptValue: false,
	}
	container.RegisterContainerHandlerFactory(cannotAccept, []watcher.ContainerWatchSource{watcher.Raw})

	_, accept, err := container.NewContainerHandler(testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	if err != nil {
		t.Error("Expected NewContainerHandler to succeed")
	}
	if accept == true {
		t.Error("Expected NewContainerHandler to ignore the container.")
	}
}

func TestRawContainerHandler_Last(t *testing.T) {
	chf1 := &mockContainerHandlerFactory{
		Name: "raw",
	}
	container.RegisterContainerHandlerFactory(chf1, []watcher.ContainerWatchSource{watcher.Raw})
	cfh2 := &mockContainerHandlerFactory{
		Name: "crio",
	}
	container.RegisterContainerHandlerFactory(cfh2, []watcher.ContainerWatchSource{watcher.Raw})

	cfh3 := &mockContainerHandlerFactory{
		Name: "containerd",
	}
	container.RegisterContainerHandlerFactory(cfh3, []watcher.ContainerWatchSource{watcher.Raw})

	list := container.GetReorderedFactoryList(watcher.Raw)

	if list[len(list)-1].String() != "raw" {
		t.Error("Expected raw container handler to be last in the list.")
	}
}

type blockingContainerHandlerFactory struct {
	mockContainerHandlerFactory
	entered chan struct{}
	release chan struct{}
}

func (f *blockingContainerHandlerFactory) CanHandleAndAccept(name string) (bool, bool, error) {
	close(f.entered)
	<-f.release
	return false, false, nil
}

func TestNewContainerHandler_DoesNotBlockRegistration(t *testing.T) {
	container.ClearContainerHandlerFactories()

	slow := &blockingContainerHandlerFactory{
		mockContainerHandlerFactory: mockContainerHandlerFactory{Name: "slow"},
		entered:                     make(chan struct{}),
		release:                     make(chan struct{}),
	}
	container.RegisterContainerHandlerFactory(slow, []watcher.ContainerWatchSource{watcher.Raw})

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = container.NewContainerHandler(testContainerName, watcher.Raw, testMetadataEnvAllowList, true)
	}()
	<-slow.entered

	// Registering a factory takes the write lock. It must not wait for the
	// slow factory's call to finish.
	registered := make(chan struct{})
	go func() {
		container.RegisterContainerHandlerFactory(&mockContainerHandlerFactory{Name: "late"}, []watcher.ContainerWatchSource{watcher.Raw})
		close(registered)
	}()
	select {
	case <-registered:
	case <-time.After(5 * time.Second):
		t.Error("RegisterContainerHandlerFactory blocked behind CanHandleAndAccept")
	}

	close(slow.release)
	<-done
}

type fakePlugin struct {
	mu      sync.Mutex
	calls   int
	errs    []error // the error for each call; nil once they run out
	watcher watcher.ContainerWatcher
	block   chan struct{}
}

func (p *fakePlugin) InitializeFSContext(context *fs.Context) error {
	return nil
}

func (p *fakePlugin) Register(factory info.MachineInfoFactory, fsInfo fs.FsInfo, includedMetrics container.MetricSet) (watcher.ContainerWatcher, error) {
	if p.block != nil {
		<-p.block
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.calls <= len(p.errs) {
		return p.watcher, p.errs[p.calls-1]
	}
	return p.watcher, nil
}

func (p *fakePlugin) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

type retryableFakePlugin struct {
	*fakePlugin
	retry bool
}

func (p *retryableFakePlugin) CanRetryRegister() bool {
	return p.retry
}

type fakeWatcher struct{}

func (fakeWatcher) Start(events chan watcher.ContainerEvent) error { return nil }
func (fakeWatcher) Stop() error                                    { return nil }

func registerTestPlugin(t *testing.T, name string, plugin container.Plugin) {
	t.Helper()
	if err := container.RegisterPlugin(name, plugin); err != nil {
		t.Fatal(err)
	}
}

func TestInitializePluginsRetriesOnlyRetryablePlugins(t *testing.T) {
	container.ResetPluginsForTest()
	t.Cleanup(container.ResetPluginsForTest)
	container.ClearContainerHandlerFactories()

	errDown := errors.New("down")
	ok := &fakePlugin{}
	once := &fakePlugin{errs: []error{errDown}}
	optOut := &retryableFakePlugin{fakePlugin: &fakePlugin{errs: []error{errDown}}, retry: false}
	retry := &retryableFakePlugin{fakePlugin: &fakePlugin{errs: []error{errDown, errDown}}, retry: true}
	registerTestPlugin(t, "ok", ok)
	registerTestPlugin(t, "once", once)
	registerTestPlugin(t, "optout", optOut)
	registerTestPlugin(t, "retry", retry)

	for i := 0; i < 3; i++ {
		container.InitializePlugins(nil, nil, nil)
	}

	assert.Equal(t, 1, ok.Calls())
	assert.Equal(t, 1, once.Calls())
	assert.Equal(t, 1, optOut.Calls())
	assert.Equal(t, 3, retry.Calls())
	assert.Equal(t, map[string]container.PluginStatus{
		"ok":     {Registered: true, Attempts: 1},
		"once":   {Attempts: 1, LastError: errDown},
		"optout": {Attempts: 1, LastError: errDown},
		"retry":  {Registered: true, Attempts: 3},
	}, container.PluginStatuses())
}

func TestInitializePluginsReturnsWatchers(t *testing.T) {
	container.ResetPluginsForTest()
	t.Cleanup(container.ResetPluginsForTest)

	// As before RetryablePlugin, a watcher returned together with an error
	// is still returned.
	registerTestPlugin(t, "failed", &fakePlugin{errs: []error{errors.New("down")}, watcher: fakeWatcher{}})
	registerTestPlugin(t, "ok", &fakePlugin{watcher: fakeWatcher{}})
	assert.Len(t, container.InitializePlugins(nil, nil, nil), 2)

	// Nothing is called again, so nothing is returned.
	assert.Empty(t, container.InitializePlugins(nil, nil, nil))
}

func TestInitializePluginsPicksUpLatePlugins(t *testing.T) {
	container.ResetPluginsForTest()
	t.Cleanup(container.ResetPluginsForTest)

	registerTestPlugin(t, "early", &fakePlugin{})
	container.InitializePlugins(nil, nil, nil)

	late := &fakePlugin{}
	registerTestPlugin(t, "late", late)
	container.InitializePlugins(nil, nil, nil)
	assert.Equal(t, 1, late.Calls())
	assert.True(t, container.PluginStatuses()["late"].Registered)
}

func TestPluginStatusesDoesNotWaitForRegister(t *testing.T) {
	container.ResetPluginsForTest()
	t.Cleanup(container.ResetPluginsForTest)

	registerTestPlugin(t, "done", &fakePlugin{})
	container.InitializePlugins(nil, nil, nil)

	slow := &fakePlugin{block: make(chan struct{})}
	registerTestPlugin(t, "slow", slow)
	initialized := make(chan struct{})
	go func() {
		container.InitializePlugins(nil, nil, nil)
		close(initialized)
	}()

	statuses := make(chan map[string]container.PluginStatus)
	go func() { statuses <- container.PluginStatuses() }()
	select {
	case s := <-statuses:
		assert.True(t, s["done"].Registered)
	case <-time.After(5 * time.Second):
		t.Error("PluginStatuses waited for a Register call")
	}
	close(slow.block)
	<-initialized
}

func TestDebugInfoListsPlugins(t *testing.T) {
	container.ResetPluginsForTest()
	t.Cleanup(container.ResetPluginsForTest)
	container.ClearContainerHandlerFactories()

	registerTestPlugin(t, "a", &fakePlugin{})
	registerTestPlugin(t, "b", &retryableFakePlugin{fakePlugin: &fakePlugin{errs: []error{errors.New("down")}}, retry: true})
	registerTestPlugin(t, "c", &fakePlugin{errs: []error{errors.New("gone")}})
	container.InitializePlugins(nil, nil, nil)

	assert.Equal(t, []string{
		"a: registered",
		"b: not registered, will retry (attempts: 1): down",
		"c: not registered (attempts: 1): gone",
	}, container.DebugInfo()["Container plugins"])
}
