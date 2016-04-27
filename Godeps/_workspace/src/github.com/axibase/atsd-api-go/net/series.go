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

package net

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type Number interface {
	String() string
	Float64() float64
	Int64() int64
}

type Float64 float64

func (self Float64) String() string {
	return strconv.FormatFloat(float64(self), 'f', -1, 64)
}
func (self Float64) Float64() float64 {
	return float64(self)
}
func (self Float64) Int64() int64 {
	return int64(self)
}

type Float32 float32

func (self Float32) String() string {
	return strconv.FormatFloat(float64(self), 'f', -1, 32)
}
func (self Float32) Float64() float64 {
	return float64(self)
}
func (self Float32) Int64() int64 {
	return int64(self)
}

type Int64 int64

func (self Int64) String() string {
	return strconv.FormatInt(int64(self), 10)
}
func (self Int64) Int64() int64 {
	return int64(self)
}
func (self Int64) Float64() float64 {
	return float64(self)
}

type Int32 int32

func (self Int32) String() string {
	return strconv.FormatInt(int64(self), 10)
}
func (self Int32) Int64() int64 {
	return int64(self)
}
func (self Int32) Float64() float64 {
	return float64(self)
}

type Int16 int16

func (self Int16) String() string {
	return strconv.FormatInt(int64(self), 10)
}
func (self Int16) Int64() int64 {
	return int64(self)
}
func (self Int16) Float64() float64 {
	return float64(self)
}

type Uint64 uint64

func (self Uint64) String() string {
	return strconv.FormatUint(uint64(self), 10)
}
func (self Uint64) Int64() int64 {
	return int64(self)
}
func (self Uint64) Float64() float64 {
	return float64(self)
}

type Uint32 uint32

func (self Uint32) String() string {
	return strconv.FormatUint(uint64(self), 10)
}
func (self Uint32) Int64() int64 {
	return int64(self)
}
func (self Uint32) Float64() float64 {
	return float64(self)
}

type Uint16 uint16

func (self Uint16) String() string {
	return strconv.FormatUint(uint64(self), 10)
}
func (self Uint16) Int64() int64 {
	return int64(self)
}
func (self Uint16) Float64() float64 {
	return float64(self)
}

type SeriesCommand struct {
	timestamp    *Millis
	entity       string //todo: verify entity name
	metricValues map[string]Number
	tags         map[string]string
}

func NewSeriesCommand(entity, metricName string, metricValue Number) *SeriesCommand {
	return &SeriesCommand{entity: entity, metricValues: map[string]Number{strings.ToLower(metricName): metricValue}, tags: map[string]string{}}
}
func (self *SeriesCommand) Metrics() map[string]Number {
	copy := map[string]Number{}
	for k, v := range self.metricValues {
		copy[k] = v
	}
	return copy
}
func (self *SeriesCommand) Entity() string {
	return self.entity
}
func (self *SeriesCommand) Timestamp() *Millis {
	return self.timestamp
}
func (self *SeriesCommand) Tags() map[string]string {
	copy := map[string]string{}
	for k, v := range self.tags {
		copy[k] = v
	}
	return copy
}
func (self *SeriesCommand) SetTimestamp(timestamp Millis) *SeriesCommand {
	self.timestamp = &timestamp
	return self
}
func (self *SeriesCommand) SetMetricValue(metric string, value Number) *SeriesCommand {
	self.metricValues[strings.ToLower(metric)] = value
	return self
}
func (self *SeriesCommand) SetTag(tag, value string) *SeriesCommand {
	self.tags[strings.ToLower(tag)] = value
	return self
}

func (self *SeriesCommand) String() string {

	msg := bytes.NewBufferString("")
	fmt.Fprintf(msg, "series e:%v", self.entity)
	if self.timestamp != nil {
		fmt.Fprintf(msg, " ms:%v", *self.timestamp)
	}
	for key, val := range self.tags {
		fmt.Fprintf(msg, " t:%v=\"%v\"", key, val)
	}
	for key, val := range self.metricValues {
		fmt.Fprintf(msg, " m:%v=%v", key, val)
	}
	fmt.Fprint(msg, "\n")
	return msg.String()
}
