// Copyright 2015 Google Inc. All Rights Reserved.
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

package riemann

import(
	"fmt"
	"time"
	"strconv"
	"sync"

	"github.com/google/cadvisor/storage"
	"github.com/bigdatadev/goryman/proto"

	riemann "github.com/bigdatadev/goryman"
	info    "github.com/google/cadvisor/info/v1"
)

const(
	// https://github.com/collectd/collectd/blob/master/src/write_riemann.c#L949
	ttlFactor float32 = 2.0

	// Metrics keys
	memoryUsage string = "memory:usage"
	memoryWorkingSet string = "memory:working:set"
	cpuUsageTotal string = "cpu:usage:total"
	cpuUsageUser string = "cpu:usage:user"
	cpuUsageSystem string = "cpu:usage:system"
	cpuUsagePerCpu string = "cpu:usage:cpu%d"
	networkRxBytes string = "network:rxbytes"
	networkRxErrors string = "network:rxerrors"
	networkTxErrors string = "network:txerrors"
	networkTxBytes string = "network:txbytes"
)

type riemannStorage struct {
	machineName string
	lock sync.Mutex
	lastSend time.Time
	client *riemannClient
	buffer *proto.Msg
	bufferDuration time.Duration
}

func (self *riemannStorage) readyToFlush() bool {
	return time.Since(self.lastSend) >= self.bufferDuration
}


// A Riemann metric could be int64, float32 or float64
func coerceMetricType(v interface{}) (interface{}, error) {
	var ret interface{}
	if v == nil {
		ret = 0
		return ret, nil
	}

	switch x := v.(type) {
	case uint64:
		ret = int(x)
		return ret, nil
	case int:
		if x < 0 {
			ret = 0
			return ret, fmt.Errorf("negative value: %v", x)
		}
		return int(x), nil
	case string:
		ret, err := strconv.ParseInt(x, 10, 64)
		return ret, err
	case float32, float64, int64:
		ret = x
		return ret, nil
	default:
		ret = 0
		return ret, fmt.Errorf("unknown type")
	}
}

// Calculate event's TTL
func (self *riemannStorage) calculateTtl() float32 {
	d := time.Since(self.lastSend).Seconds()
	return ttlFactor * float32(d)
}

func (self *riemannStorage) statsMapToRiemannMessage(
	values map[string]interface{},
	ref info.ContainerReference,
	timestamp int64,
) (*proto.Msg, error) {
	var containerName string
	if len(ref.Aliases) > 0 {
		containerName = ref.Aliases[0]
	} else {
		containerName = ref.Name
	}

	msg := &proto.Msg{}

	for service, metric := range values {
		metric, err := coerceMetricType(metric)
		if err != nil {
			return nil, err
		}

		event := &riemann.Event{
			Host:    self.machineName,
			Service: fmt.Sprintf("%s:%s", containerName, service),
			Metric:  metric,
			Time:    timestamp,
			Ttl:     self.calculateTtl(),
		}

		pbEvent, err := riemann.EventToProtocolBuffer(event)
		if err != nil {
			return nil, err
		}

		msg.Events = append(msg.Events, pbEvent)
	}

	return msg, nil
}

func makeStatsMap(stats *info.ContainerStats) map[string]interface{} {
	values    := make(map[string]interface{}, 0)

	values[memoryUsage] = stats.Memory.Usage
	values[memoryWorkingSet] = stats.Memory.WorkingSet

	values[cpuUsageTotal] = stats.Cpu.Usage.Total
	values[cpuUsageUser] = stats.Cpu.Usage.User
	values[cpuUsageSystem] = stats.Cpu.Usage.System

	for i, usage := range stats.Cpu.Usage.PerCpu {
		k := fmt.Sprintf(cpuUsagePerCpu, i)
		values[k] = usage
	}

	values[networkRxBytes] = stats.Network.RxBytes
	values[networkRxErrors] = stats.Network.RxErrors
	values[networkTxBytes] = stats.Network.TxBytes
	values[networkTxErrors] = stats.Network.TxErrors

	return values
}


func (self *riemannStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	var messageToFlush *proto.Msg

	timestamp    := stats.Timestamp.UnixNano()/1E3
	statsMap     := makeStatsMap(stats)
	message, err := self.statsMapToRiemannMessage(statsMap, ref, timestamp)

	self.lock.Lock()

	self.buffer.Events = append(self.buffer.Events, message.Events...)

	if self.readyToFlush() {
		messageToFlush = self.buffer
		self.buffer = newEmptyBuffer()
		self.lastSend = time.Now()
	}

	self.lock.Unlock()

	if messageToFlush != nil {
		_, err = self.client.SendMessage(messageToFlush)

		if err != nil {
			return fmt.Errorf("Failed to send data to Riemann: %s", err)
		}
	}

	return nil
}

// Same as redis: we need only push.
func (self *riemannStorage) RecentStats(containerName string, numStats int) ([]*info.ContainerStats, error) {
	return nil, nil
}

func (self *riemannStorage) Close() error {
	err := self.client.Close()
	return err
}

func newEmptyBuffer() *proto.Msg {
	return &proto.Msg{
		Events: make([]*proto.Event, 0),
	}
}

// Create a new Riemann storage driver.
// machineName: A unique identifier to identify the host that current cAdvisor
// instance is running on.
// riemannAddr: Riemann network address.
func New(
	machineName,
	riemannAddr string,
	bufferDuration time.Duration,
) (storage.StorageDriver, error) {
	client := &riemannClient{
		addr: riemannAddr,
	}

	err := client.Connect()
	if err != nil {
		return nil, err
	}

	ret := &riemannStorage{
		client: client,
		machineName: machineName,
		bufferDuration: bufferDuration,
		buffer: newEmptyBuffer(),
		lastSend: time.Now(),
	}

	return ret, nil
}
