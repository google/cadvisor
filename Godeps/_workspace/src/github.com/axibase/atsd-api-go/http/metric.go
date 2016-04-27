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

package http

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/axibase/atsd-api-go/net"
)

type Days uint
type DataType int

const (
	SHORT DataType = iota
	INTEGER
	LONG
	FLOAT
	DOUBLE
)

func (self DataType) String() string {
	switch self {
	case SHORT:
		return "SHORT"
	case INTEGER:
		return "INTEGER"
	case LONG:
		return "LONG"
	case FLOAT:
		return "FLOAT"
	case DOUBLE:
		return "DOUBLE"
	default:
		return "FLOAT"
	}
}

type TimePrecision int

const (
	SECONDS TimePrecision = iota
	MILLISECONDS
)

func (self TimePrecision) String() string {
	switch self {
	case SECONDS:
		return "SECONDS"
	case MILLISECONDS:
		return "MILLISECONDS"
	default:
		return "MILLISECONDS"
	}
}

type InvalidAction int

const (
	NONE InvalidAction = iota
	DISCARD
	TRANSFORM
	RAISE_ERROR
)

func (self InvalidAction) String() string {
	switch self {
	case NONE:
		return "NONE"
	case DISCARD:
		return "DISCARD"
	case TRANSFORM:
		return "TRANSFORM"
	case RAISE_ERROR:
		return "RAISE_ERROR"
	default:
		return "NONE"
	}
}

type Metric struct {
	name              string            //Metric name (unique)
	label             *string           //Metric label
	enabled           bool              //Enabled status. Incoming data is discarded for disabled metrics
	dataType          DataType          //short, integer, float, long, double
	timePrecision     TimePrecision     //seconds, milliseconds
	persistent        bool              //Persistence status. Non-persistent metrics are not stored in the database and are only used in rule engine.
	counter           bool              //Metrics with continuously incrementing value should be defined as counters
	filter            *string           //If filter is specified, metric puts that do not match the filter are discarded
	minValue          *net.Number       //Minimum value. If value is less than Minimum value, Invalid Action is triggered
	maxValue          *net.Number       //Maximum value. If value is greater than Maximum value, Invalid Action is triggered
	invalidAction     InvalidAction     //None - retain value as is; Discard - don’t process the incoming put, discard it; Transform - set value to min_value or max_value; Raise_Error - log error in ATSD log
	description       *string           //Metric description
	retentionInterval Days              //Number of days to retain values for this metric in the database
	lastInsertTime    *time.Time        //Last time value was received by ATSD for this metric. Time specified in epoch milliseconds.
	tags              map[string]string //as requested by tags parameter
}

func NewMetric(name string) *Metric {
	return &Metric{
		name:              name,
		enabled:           true,
		dataType:          FLOAT,
		counter:           false,
		persistent:        true,
		tags:              map[string]string{},
		timePrecision:     MILLISECONDS,
		retentionInterval: 0,
		invalidAction:     NONE,
	}
}

func (self *Metric) SetName(name string) *Metric {
	self.name = name
	return self
}
func (self *Metric) SetEnabled(isEnabled bool) *Metric {
	self.enabled = isEnabled
	return self
}
func (self *Metric) SetDataType(dataType DataType) *Metric {
	self.dataType = dataType
	return self
}
func (self *Metric) SetCounter(isCounter bool) *Metric {
	self.counter = isCounter
	return self
}
func (self *Metric) SetPersistent(isPersistent bool) *Metric {
	self.persistent = isPersistent
	return self
}
func (self *Metric) SetTag(name, value string) *Metric {
	self.tags[strings.ToLower(name)] = value
	return self
}
func (self *Metric) SetTimePrecision(timePrecision TimePrecision) *Metric {
	self.timePrecision = timePrecision
	return self
}
func (self *Metric) SetRetentionInterval(retentionInterval Days) *Metric {
	self.retentionInterval = retentionInterval
	return self
}
func (self *Metric) SetInvalidAction(invalidAction InvalidAction) *Metric {
	self.invalidAction = invalidAction
	return self
}
func (self *Metric) SetLabel(label string) *Metric {
	self.label = &label
	return self
}
func (self *Metric) SetFilter(filter string) *Metric {
	self.filter = &filter
	return self
}
func (self *Metric) SetMinValue(minValue net.Number) *Metric {
	self.minValue = &minValue
	return self
}
func (self *Metric) SetMaxValue(maxValue net.Number) *Metric {
	self.maxValue = &maxValue
	return self
}
func (self *Metric) SetDescription(description string) *Metric {
	self.description = &description
	return self
}

func (self *Metric) Name() string {
	return self.name
}
func (self *Metric) Enabled() bool {
	return self.enabled
}
func (self *Metric) DataType() DataType {
	return self.dataType
}
func (self *Metric) Counter() bool {
	return self.counter
}
func (self *Metric) Persistent() bool {
	return self.persistent
}
func (self *Metric) TagValue(name string) (string, bool) {
	val, ok := self.tags[strings.ToLower(name)]
	return val, ok
}
func (self *Metric) TimePrecision() TimePrecision {
	return self.timePrecision
}
func (self *Metric) RetentionInterval() Days {
	return self.retentionInterval
}
func (self *Metric) InvalidAction() InvalidAction {
	return self.invalidAction
}
func (self *Metric) Label() *string {
	return self.label
}
func (self *Metric) Filter() *string {
	return self.filter
}
func (self *Metric) MinValue() *net.Number {
	return self.minValue
}
func (self *Metric) MaxValue() *net.Number {
	return self.maxValue
}
func (self *Metric) Description() *string {
	return self.description
}
func (self *Metric) GetLastInsertTime() *time.Time {
	return self.lastInsertTime
}
func (self *Metric) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	m["name"] = self.name
	m["enabled"] = self.enabled
	m["counter"] = self.counter
	m["persistent"] = self.persistent
	m["tags"] = self.tags
	m["retentionInterval"] = self.retentionInterval

	m["dataType"] = self.dataType.String()
	m["timePrecision"] = self.timePrecision.String()
	m["invalidAction"] = self.invalidAction.String()

	if self.label != nil {
		m["label"] = *self.label
	}
	if self.filter != nil {
		m["filter"] = *self.filter
	}
	if self.minValue != nil {
		m["minValue"] = *self.minValue
	}
	if self.maxValue != nil {
		m["maxValue"] = *self.maxValue
	}
	if self.description != nil {
		m["description"] = *self.description
	}
	if self.lastInsertTime != nil {
		m["lastInsertTime"] = self.lastInsertTime.UnixNano() / 1e6
	}
	return json.Marshal(m)
}
