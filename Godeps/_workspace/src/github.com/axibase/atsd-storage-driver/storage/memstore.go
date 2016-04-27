/*
* Copyright 2015 Axibase Corporation or its affiliates. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License").
* You may not use this file except in compliance with the License.
* A copy of the License is located at
*
* https://www.axibase.com/atsd/axibase-apache-2.0.pdf
*
* or in the "license" file accompanying this file. This file is distributed
* on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
* express or implied. See the License for the specific language governing
* permissions and limitations under the License.
 */

package storage

import (
	"fmt"
	"sort"
	"sync"

	"github.com/axibase/atsd-api-go/net"
)

const (
	minMemoryLimit = uint(10000)
)

type MemStore struct {
	seriesCommandMap *map[string]*Chunk

	properties []*net.PropertyCommand

	messages []*net.MessageCommand

	entityTagCommands []*net.EntityTagCommand

	sync.Mutex

	Limit uint
}

func NewMemStore(limit uint) (*MemStore, error) {
	if limit < minMemoryLimit {
		return nil, fmt.Errorf("Memstore limit should be >= 10000. Current limit = %v", limit)
	}
	ms := &MemStore{
		seriesCommandMap: &map[string]*Chunk{},
		Limit:            limit,
	}
	return ms, nil
}
func (self *MemStore) AppendSeriesCommands(commands []*net.SeriesCommand) {
	self.Lock()
	defer self.Unlock()
	if uint(self.unsafeSize()) < self.Limit {
		for i := 0; i < len(commands); i++ {
			key := self.getKey(commands[i])
			if _, ok := (*self.seriesCommandMap)[key]; !ok {
				(*self.seriesCommandMap)[key] = NewChunk()
			}
			(*self.seriesCommandMap)[key].PushBack(commands[i])
		}
	}
}
func (self *MemStore) AppendPropertyCommands(propertyCommands []*net.PropertyCommand) {
	self.Lock()
	defer self.Unlock()
	if self.unsafeSize() < self.Limit {
		self.properties = append(self.properties, propertyCommands...)
	}
}
func (self *MemStore) AppendEntityTagCommands(entityUpdateCommands []*net.EntityTagCommand) {
	self.Lock()
	defer self.Unlock()
	if self.unsafeSize() < self.Limit {
		self.entityTagCommands = append(self.entityTagCommands, entityUpdateCommands...)
	}
}
func (self *MemStore) AppendMessageCommands(messageCommands []*net.MessageCommand) {
	self.Lock()
	defer self.Unlock()
	if self.unsafeSize() < self.Limit {
		self.messages = append(self.messages, messageCommands...)
	}

}

func (self *MemStore) ReleaseSeriesCommandChunks() []*Chunk {
	self.Lock()
	defer self.Unlock()
	smap := self.seriesCommandMap
	self.seriesCommandMap = &map[string]*Chunk{}
	seriesCommandsChunks := []*Chunk{}
	for _, val := range *smap {
		seriesCommandsChunks = append(seriesCommandsChunks, val)
	}
	return seriesCommandsChunks
}
func (self *MemStore) ReleaseProperties() []*net.PropertyCommand {
	self.Lock()
	defer self.Unlock()
	properties := self.properties
	self.properties = nil
	return properties
}
func (self *MemStore) ReleaseEntityTagCommands() []*net.EntityTagCommand {
	self.Lock()
	defer self.Unlock()
	entityTagCommands := self.entityTagCommands
	self.entityTagCommands = nil
	return entityTagCommands
}
func (self *MemStore) SeriesCommandCount() uint {
	self.Lock()
	defer self.Unlock()
	return self.unsafeSeriesCommandCount()
}
func (self *MemStore) unsafeSeriesCommandCount() uint {
	commandCount := uint(0)

	for _, val := range *(self.seriesCommandMap) {
		commandCount += uint(val.Len())
	}
	return commandCount
}
func (self *MemStore) PropertiesCount() uint {
	self.Lock()
	defer self.Unlock()

	return self.unsafePropertiesCount()
}
func (self *MemStore) unsafePropertiesCount() uint {
	return uint(len(self.properties))
}

func (self *MemStore) MessagesCount() uint {
	self.Lock()
	defer self.Unlock()

	return self.unsafeMessagesCount()
}
func (self *MemStore) unsafeMessagesCount() uint {
	return uint(len(self.messages))
}

func (self *MemStore) EntitiesCount() uint {
	self.Lock()
	defer self.Unlock()

	return self.unsafeEntitiesCount()
}

func (self *MemStore) unsafeEntitiesCount() uint {
	return uint(len(self.entityTagCommands))
}

func (self *MemStore) Size() uint {
	return self.EntitiesCount() + self.PropertiesCount() + self.SeriesCommandCount() + self.MessagesCount()
}
func (self *MemStore) unsafeSize() uint {
	return self.unsafeEntitiesCount() + self.unsafePropertiesCount() + self.unsafeSeriesCommandCount() + self.unsafeMessagesCount()
}

func (self *MemStore) getKey(sc *net.SeriesCommand) string {
	key := sc.Entity()
	metrics := []string{}
	for metricName := range sc.Metrics() {
		metrics = append(metrics, metricName)
	}
	sort.Strings(metrics)
	for i := range metrics {
		key += metrics[i]
	}

	tags := []string{}
	for tagName, tagValue := range sc.Tags() {
		tags = append(tags, tagName+"="+tagValue)
	}
	sort.Strings(tags)
	for i := range tags {
		key += tags[i]
	}

	return key
}

func (self *MemStore) ReleaseMessageCommands() []*net.MessageCommand {
	self.Lock()
	defer self.Unlock()
	messages := self.messages
	self.messages = nil
	return messages
}
