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
	"sync"
	"time"

	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/sampling"
)

type samplerFactor interface {
	String() string
	NewSampler(*StatsParameter) (sampling.Sampler, error)
}

type samplerManager struct {
	factoryMap map[string]samplerFactor
	lock       sync.RWMutex
}

func (self *samplerManager) Register(factory samplerFactor) {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.factoryMap == nil {
		self.factoryMap = make(map[string]samplerFactor, 3)
	}
	self.factoryMap[factory.String()] = factory
}

func (self *samplerManager) NewSampler(param *StatsParameter) (sampling.Sampler, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if f, ok := self.factoryMap[param.Sampler]; ok {
		return f.NewSampler(param)
	}
	return nil, fmt.Errorf("unknown sampler %v", param.Sampler)
}

var globalSamplerManager samplerManager

func NewSampler(param *StatsParameter) (sampling.Sampler, error) {
	return globalSamplerManager.NewSampler(param)
}

type reservoirSamplerFactory struct {
}

func (self *reservoirSamplerFactory) String() string {
	return "uniform"
}

func (self *reservoirSamplerFactory) NewSampler(param *StatsParameter) (sampling.Sampler, error) {
	s := sampling.NewReservoirSampler(param.NumSamples)
	if param.ResetPeriod.Seconds() > 1.0 {
		s = sampling.NewPeriodicallyResetSampler(param.ResetPeriod, s)
	}
	return s, nil
}

type esSamplerFactory struct {
	startTime time.Time
}

func (self *esSamplerFactory) String() string {
	return "weighted"
}

func (self *esSamplerFactory) NewSampler(param *StatsParameter) (sampling.Sampler, error) {
	s := sampling.NewESSampler(param.NumSamples, func(d interface{}) float64 {
		stats := d.(*info.ContainerStats)
		delta := self.startTime.Sub(stats.Timestamp)
		return delta.Seconds()
	})
	if param.ResetPeriod.Seconds() > 1.0 {
		s = sampling.NewPeriodicallyResetSampler(param.ResetPeriod, s)
	}
	return s, nil
}

type chainSamplerFactory struct {
}

func (self *chainSamplerFactory) String() string {
	return "window"
}

func (self *chainSamplerFactory) NewSampler(param *StatsParameter) (sampling.Sampler, error) {
	s := sampling.NewChainSampler(param.NumSamples, param.WindowSize)
	if param.ResetPeriod.Seconds() > 1.0 {
		s = sampling.NewPeriodicallyResetSampler(param.ResetPeriod, s)
	}
	return s, nil
}

func init() {
	globalSamplerManager.Register(&reservoirSamplerFactory{})
	globalSamplerManager.Register(&esSamplerFactory{time.Now()})
	globalSamplerManager.Register(&chainSamplerFactory{})
}
