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
	"container/list"
	"github.com/axibase/atsd-api-go/http"
	"github.com/axibase/atsd-api-go/net"
	"sync"
	"time"
)

type metricValue struct {
	name  string
	tags  map[string]string
	value net.Number
}

type Chunk struct {
	*list.List
}

func NewChunk() *Chunk {
	return &Chunk{list.New()}
}

type IWriteCommunicator interface {
	QueuedSendData(seriesCommandsChunk []*Chunk, entityTagCommands []*net.EntityTagCommand, properties []*net.PropertyCommand, messages []*net.MessageCommand)
	PriorSendData(seriesCommands []*net.SeriesCommand, entityTagCommands []*net.EntityTagCommand, propertyCommands []*net.PropertyCommand, messageCommands []*net.MessageCommand)
	SelfMetricValues() []*metricValue
}
type Storage struct {
	selfMetricsEntity string
	metricPrefix      string

	memstore          *MemStore
	dataCompacter     *DataCompacter
	writeCommunicator IWriteCommunicator

	atsdHttpClient *http.Client

	isUpdating             bool
	updateInterval         time.Duration
	selfMetricSendInterval time.Duration
	stopUpdateTask         chan bool
	stopSelfMetricSendTask chan bool
	mutex                  sync.Mutex
}

func (self *Storage) updateTask() {
	seriesCommandsChunks := self.memstore.ReleaseSeriesCommandChunks()
	properties := self.memstore.ReleaseProperties()
	entityTagCommands := self.memstore.ReleaseEntityTagCommands()
	messageCommands := self.memstore.ReleaseMessageCommands()

	self.writeCommunicator.QueuedSendData(seriesCommandsChunks, entityTagCommands, properties, messageCommands)

}
func (self *Storage) selfMetricSendTask() {
	timestamp := net.Millis(time.Now().UnixNano() / 1e6)
	writeCommunicatorMetricValues := self.writeCommunicator.SelfMetricValues()

	seriesCommands := []*net.SeriesCommand{}
	for _, metricValue := range writeCommunicatorMetricValues {
		seriesCommand := net.NewSeriesCommand(self.selfMetricsEntity, self.metricPrefix+"."+metricValue.name, metricValue.value).
			SetTimestamp(timestamp)
		for name, val := range metricValue.tags {
			seriesCommand.SetTag(name, val)
		}
		seriesCommands = append(seriesCommands, seriesCommand)
	}

	seriesCommand := net.NewSeriesCommand(self.selfMetricsEntity, self.metricPrefix+".memstore.entities.count", net.Int64(self.memstore.EntitiesCount())).SetTimestamp(timestamp)
	seriesCommands = append(seriesCommands, seriesCommand)
	seriesCommand = net.NewSeriesCommand(self.selfMetricsEntity, self.metricPrefix+".memstore.messages.count", net.Int64(self.memstore.MessagesCount())).SetTimestamp(timestamp)
	seriesCommands = append(seriesCommands, seriesCommand)
	seriesCommand = net.NewSeriesCommand(self.selfMetricsEntity, self.metricPrefix+".memstore.properties.count", net.Int64(self.memstore.PropertiesCount())).SetTimestamp(timestamp)
	seriesCommands = append(seriesCommands, seriesCommand)
	seriesCommand = net.NewSeriesCommand(self.selfMetricsEntity, self.metricPrefix+".memstore.series-commands.count", net.Int64(self.memstore.SeriesCommandCount())).SetTimestamp(timestamp)
	seriesCommands = append(seriesCommands, seriesCommand)
	seriesCommand = net.NewSeriesCommand(self.selfMetricsEntity, self.metricPrefix+".memstore.size", net.Int64(self.memstore.Size())).SetTimestamp(timestamp)
	seriesCommands = append(seriesCommands, seriesCommand)
	self.writeCommunicator.PriorSendData(seriesCommands, nil, nil, nil)

}

func (self *Storage) SendSeriesCommands(group string, seriesCommands []*net.SeriesCommand) {
	filteredSeriesCommands := self.dataCompacter.Filter(group, seriesCommands)
	self.memstore.AppendSeriesCommands(filteredSeriesCommands)
}
func (self *Storage) SendPropertyCommands(propertyCommands []*net.PropertyCommand) {
	self.memstore.AppendPropertyCommands(propertyCommands)
}
func (self *Storage) SendEntityTagCommands(entityTagCommands []*net.EntityTagCommand) {
	self.memstore.AppendEntityTagCommands(entityTagCommands)
}
func (self *Storage) SendMessageCommands(messageCommands []*net.MessageCommand) {
	self.memstore.AppendMessageCommands(messageCommands)
}

func (self *Storage) RegisterMetric(metric *http.Metric) error {
	return self.atsdHttpClient.Metric.CreateOrReplace(metric)
}

func (self *Storage) StartPeriodicSending() {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if !self.isUpdating {
		self.stopSelfMetricSendTask = schedule(self.selfMetricSendTask, self.selfMetricSendInterval)
		self.stopUpdateTask = schedule(self.updateTask, self.updateInterval)
		self.isUpdating = true
	}
}
func (self *Storage) StopPeriodicSending() {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.isUpdating {
		self.stopSelfMetricSendTask <- true
		self.stopUpdateTask <- true
		self.isUpdating = false
	}
}
func (self *Storage) ForceSend() {
	self.updateTask()
}

func schedule(task func(), updateInterval time.Duration) chan bool {
	stop := make(chan bool)
	go func() {
		ticker := time.NewTicker(updateInterval)
		task()
		for {
			select {
			case <-ticker.C:
				task()
			case <-stop:
				ticker.Stop()
				return
			}
		}
	}()
	return stop
}
