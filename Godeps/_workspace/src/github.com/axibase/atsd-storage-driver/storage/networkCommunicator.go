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
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"

	atsdNet "github.com/axibase/atsd-api-go/net"
)

const (
	seriesCommandsChunkChannelBufferSize = 1000
	bufferSize                           = 16384
)

type counters struct {
	series, entityTag, prop, messages struct{ sent, dropped uint64 }
}

type NetworkCommunicator struct {
	seriesCommandsChunkChan chan *Chunk
	properties              chan []*atsdNet.PropertyCommand
	messageCommands         chan []*atsdNet.MessageCommand
	entityTag               chan []*atsdNet.EntityTagCommand

	protocol string
	hostport string

	counters []*counters

	goroutinesCount int

	isConnected bool

	mutex *sync.Mutex
}

func NewNetworkCommunicator(goroutineCount int, url *url.URL) (*NetworkCommunicator, error) {
	if goroutineCount <= 0 {
		return nil, errors.New(fmt.Sprintf("goroutines_count should be > 0, provided value = %v", goroutineCount))
	}
	if url.Scheme != "tcp" && url.Scheme != "udp" {
		return nil, errors.New(fmt.Sprintf("unsupported protocol: %v", url.Scheme))
	}
	if !strings.Contains(url.Host, ":") {
		if url.Scheme == "tcp" {
			url.Host += ":8081"
		} else if url.Scheme == "udp" {
			url.Host += ":8082"
		}
	}
	nc := &NetworkCommunicator{
		protocol:                url.Scheme,
		hostport:                url.Host,
		goroutinesCount:         goroutineCount,
		seriesCommandsChunkChan: make(chan *Chunk, seriesCommandsChunkChannelBufferSize),
		properties:              make(chan []*atsdNet.PropertyCommand),
		messageCommands:         make(chan []*atsdNet.MessageCommand),
		entityTag:               make(chan []*atsdNet.EntityTagCommand),
		counters:                make([]*counters, goroutineCount, goroutineCount),
		isConnected:             false,
		mutex:                   &sync.Mutex{},
	}

	for i := 0; i < goroutineCount; i++ {
		nc.counters[i] = &counters{}
	}

	for i := 0; i < goroutineCount; i++ {
		go func(threadNum int, counters *counters) {
			expBackoff := NewExpBackoff(100*time.Millisecond, 5*time.Minute)
			senderThread := senderThread{nc: nc, expBackoff: expBackoff, threadNum: threadNum, limit: bufferSize, buffer: bytes.NewBuffer(make([]byte, 0, bufferSize))}
			for {
				select {
				case entityTag := <-nc.entityTag:
					for i := range entityTag {
						senderThread.sendCommand(entityTag[i], "entity update")
						atomic.AddUint64(&counters.entityTag.sent, 1)
					}
				case properties := <-nc.properties:
					for i := range properties {
						senderThread.sendCommand(properties[i], "property")
						atomic.AddUint64(&counters.prop.sent, 1)
					}
				case messageCommands := <-nc.messageCommands:
					for i := range messageCommands {
						senderThread.sendCommand(messageCommands[i], "message")
						atomic.AddUint64(&counters.messages.sent, 1)
					}
				case seriesChunk := <-nc.seriesCommandsChunkChan:
					for el := seriesChunk.Front(); el != nil; el = seriesChunk.Front() {
						senderThread.sendCommand(el.Value.(*atsdNet.SeriesCommand), "series")
						seriesChunk.Remove(el)
						atomic.AddUint64(&counters.series.sent, 1)
					}
				}
				senderThread.flush()

			}
		}(i, nc.counters[i])
	}

	return nc, nil
}

type senderThread struct {
	nc         *NetworkCommunicator
	expBackoff *ExpBackoff
	conn       net.Conn
	threadNum  int
	buffer     *bytes.Buffer
	limit      int
}

func (self *senderThread) sendCommand(command fmt.Stringer, commandName string) {
	_, err := fmt.Fprint(self.buffer, command)
	if err != nil {
		glog.Error("Thread ", self.threadNum, " could not send", commandName, " command: ", err)
		return
	}
	if self.buffer.Len() > self.limit {
		self.flush()
	}
}

func (self *senderThread) initConnection() {
	for self.conn == nil {
		conn, err := net.DialTimeout(self.nc.protocol, self.nc.hostport, 5*time.Second)
		if err != nil {
			waitDuration := self.expBackoff.Duration()
			glog.Error("Thread ", self.threadNum, " could not init connection, waiting for ", waitDuration, " err: ", err)
			time.Sleep(waitDuration)
		} else {
			self.conn = conn
			self.expBackoff.Reset()
			self.nc.SetConnected(true)
		}
	}
}

func (self *senderThread) flush() {
	firstTime := true
	hasErrors := false
	for firstTime || hasErrors {
		firstTime = false
		if !self.nc.IsConnected() {
			self.conn = nil
		}
		if self.conn == nil {
			self.initConnection()
		}
		_, err := fmt.Fprint(self.conn, self.buffer)
		hasErrors = err != nil
		if hasErrors {
			glog.Error("Thread ", self.threadNum, " could not send buffer, size = ", self.buffer.Len(), " error: ", err)
			self.conn.Close()
			self.conn = nil
		} else {
			self.buffer.Reset()
		}
	}
	return
}

func (self *NetworkCommunicator) QueuedSendData(seriesCommandsChunk []*Chunk, entityTagCommands []*atsdNet.EntityTagCommand, properties []*atsdNet.PropertyCommand, messageCommands []*atsdNet.MessageCommand) {
	self.entityTag <- entityTagCommands

	self.properties <- properties

	self.messageCommands <- messageCommands

	for _, val := range seriesCommandsChunk {
		self.seriesCommandsChunkChan <- val
	}
}

func (self *NetworkCommunicator) PriorSendData(seriesCommands []*atsdNet.SeriesCommand, entityTagCommands []*atsdNet.EntityTagCommand, propertyCommands []*atsdNet.PropertyCommand, messageCommands []*atsdNet.MessageCommand) {
	conn, err := net.DialTimeout(self.protocol, self.hostport, 1*time.Second)
	if err != nil {
		glog.Error("Could not init connection to prior send self metrics ", err)
		self.SetConnected(false)
		return
	}
	for i := range entityTagCommands {
		_, err = fmt.Fprint(conn, entityTagCommands[i])
		if err != nil {
			glog.Error("Could not prior send entity-tag command ", err)
			self.SetConnected(false)
		}
	}
	for i := range propertyCommands {
		_, err = fmt.Fprint(conn, propertyCommands[i])
		if err != nil {
			glog.Error("Could not prior send property command ", err)
			self.SetConnected(false)
		}
	}
	for i := range seriesCommands {
		_, err = fmt.Fprint(conn, seriesCommands[i])
		if err != nil {
			glog.Error("Could not prior send series command ", err)
			self.SetConnected(false)
		}
	}
	for i := range messageCommands {
		_, err = fmt.Fprint(conn, messageCommands[i])
		if err != nil {
			glog.Error("Could not prior send message command ", err)
			self.SetConnected(false)
		}
	}
	conn.Close()
}

func (self *NetworkCommunicator) SetConnected(isConnected bool) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.isConnected = isConnected
}

func (self *NetworkCommunicator) IsConnected() bool {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.isConnected
}

func (self *NetworkCommunicator) SelfMetricValues() []*metricValue {
	metricValues := []*metricValue{}
	for i := range self.counters {
		metricValues = append(metricValues,
			&metricValue{
				name: "series-commands.sent",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].series.sent)),
			},
			&metricValue{
				name: "series-commands.dropped",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].series.dropped)),
			},
			&metricValue{
				name: "message-commands.sent",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].messages.sent)),
			},
			&metricValue{
				name: "message-commands.dropped",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].messages.dropped)),
			},
			&metricValue{
				name: "property-commands.sent",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].prop.sent)),
			},
			&metricValue{
				name: "property-commands.dropped",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].prop.dropped)),
			},
			&metricValue{
				name: "entitytag-commands.sent",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].entityTag.sent)),
			},
			&metricValue{
				name: "entitytag-commands.dropped",
				tags: map[string]string{
					"thread":    strconv.FormatInt(int64(i), 10),
					"transport": self.protocol,
				},
				value: atsdNet.Int64(atomic.LoadUint64(&self.counters[i].entityTag.dropped)),
			},
		)
	}
	return metricValues
}
