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

package container

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/cadvisor/info"
	"github.com/stretchr/testify/mock"
)

type mockContainerHandlerFactory struct {
	mock.Mock
	Name string
}

func (self *mockContainerHandlerFactory) String() string {
	return self.Name
}

func (self *mockContainerHandlerFactory) NewContainerHandler(name string) (ContainerHandler, error) {
	args := self.Called(name)
	return args.Get(0).(ContainerHandler), args.Error(1)
}

func testExpectedFactory(root *factoryTreeNode, path, expectedFactory string, t *testing.T) {
	elems := dropEmptyString(strings.Split(path, "/")...)
	factory := root.find(elems...)
	if factory.String() != expectedFactory {
		t.Errorf("factory %v should be used to create container %v. but %v is selected",
			expectedFactory,
			path,
			factory)
	}
}

func testAddFactory(root *factoryTreeNode, path string) *factoryTreeNode {
	elems := dropEmptyString(strings.Split(path, "/")...)
	if root == nil {
		root = &factoryTreeNode{
			defaultFactory: nil,
		}
	}
	f := &mockContainerHandlerFactory{
		Name: path,
	}
	root.add(f, elems...)
	return root
}

func TestFactoryTree(t *testing.T) {
	root := testAddFactory(nil, "/")
	root = testAddFactory(root, "/docker")
	root = testAddFactory(root, "/user")
	root = testAddFactory(root, "/user/special/containers")

	testExpectedFactory(root, "/docker/container", "/docker", t)
	testExpectedFactory(root, "/docker", "/docker", t)
	testExpectedFactory(root, "/", "/", t)
	testExpectedFactory(root, "/user/deep/level/container", "/user", t)
	testExpectedFactory(root, "/user/special/containers", "/user/special/containers", t)
	testExpectedFactory(root, "/user/special/containers/container", "/user/special/containers", t)
	testExpectedFactory(root, "/other", "/", t)
}

type mockContainerHandlerDecorator struct {
	name string
}

func (self *mockContainerHandlerDecorator) Decorate(container ContainerHandler) (ContainerHandler, error) {
	return &containerNameSuffixAdder{
		suffix:  self.name,
		handler: container,
	}, nil
}

type containerNameSuffixAdder struct {
	suffix  string
	handler ContainerHandler
	mockContainer
}

func (self *containerNameSuffixAdder) GetStats() (*info.ContainerStats, error) {
	return nil, fmt.Errorf("GetStats() should never be called")
}

func (self *containerNameSuffixAdder) StatsSummary() (*info.ContainerStatsPercentiles, error) {
	return nil, fmt.Errorf("StatsSummary() should never be called")
}

func (self *containerNameSuffixAdder) ContainerReference() (info.ContainerReference, error) {
	if self.handler == nil {
		return info.ContainerReference{
			Name: self.suffix,
		}, nil
	}
	ref, err := self.handler.ContainerReference()
	if err != nil {
		return ref, err
	}
	return info.ContainerReference{
		Name: fmt.Sprintf("%v.%v", ref.Name, self.suffix),
	}, nil
}

type containerNameSuffixAdderFactory struct {
	name string
}

func (self *containerNameSuffixAdderFactory) String() string {
	return self.name
}

func (self *containerNameSuffixAdderFactory) NewContainerHandler(name string) (ContainerHandler, error) {
	return &containerNameSuffixAdder{
		suffix: self.name,
	}, nil
}

func TestContainerHandlerDecorator(t *testing.T) {
	factoryName := "factory"
	prefixes := []string{
		"a",
		"b",
		"c",
	}
	factory := &containerNameSuffixAdderFactory{
		name: factoryName,
	}
	RegisterContainerHandlerFactory("/", factory)
	decorators := make([]ContainerHandlerDecorator, 0, len(prefixes))
	for _, suffix := range prefixes {
		dec := &mockContainerHandlerDecorator{
			name: suffix,
		}
		decorators = append(decorators, dec)
	}
	RegisterContainerHandlerDecorators(decorators...)
	N := 10
	var wg sync.WaitGroup

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			container, err := NewContainerHandler("/somecontainer")
			containerName := strings.Join(prefixes, ".")
			containerName = fmt.Sprintf("%v.%v", factoryName, containerName)
			if err != nil {
				t.Error(err)
			}
			ref, err := container.ContainerReference()
			if err != nil {
				t.Error(err)
			}
			if ref.Name != containerName {
				t.Errorf("Wrong container name. should be %v. got %v", containerName, ref.Name)
			}
		}()
	}
	wg.Wait()
}
