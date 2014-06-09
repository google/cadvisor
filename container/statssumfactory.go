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

import "fmt"

type statsSummaryFactory struct {
	factory ContainerHandlerFactory
}

func (self *statsSummaryFactory) String() string {
	return fmt.Sprintf("%v/stats", self.factory)
}

var globalStatsParameter StatsParameter

func (self *statsSummaryFactory) NewContainerHandler(name string) (ContainerHandler, error) {
	h, err := self.factory.NewContainerHandler(name)
	if err != nil {
		return nil, err
	}
	return AddStatsSummary(h, &globalStatsParameter)
}

// This is a decorator for container factory. If the container handler created
// by a container factory does not implement stats summary method, then the factory
// could be decorated with this structure.
func AddStatsSummaryToFactory(factory ContainerHandlerFactory) ContainerHandlerFactory {
	return &statsSummaryFactory{
		factory: factory,
	}
}

func SetStatsParameter(param *StatsParameter) {
	globalStatsParameter = *param
}
