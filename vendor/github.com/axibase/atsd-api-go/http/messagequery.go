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
)

type MessagesQuery struct {
	entity        string
	startDateTime *time.Time
	endDateTime   *time.Time
	limit         *uint64
	severity      *Severity
	mType         *string
	source        *string
	tags          map[string][]string
}

func NewMessagesQuery(entity string) *MessagesQuery {
	return &MessagesQuery{
		entity: entity,
		tags:   map[string][]string{},
	}
}

func (self *MessagesQuery) SetEntity(entity string) *MessagesQuery {
	self.entity = entity
	return self
}
func (self *MessagesQuery) SetStartDateTime(startDateTime time.Time) *MessagesQuery {
	self.startDateTime = &startDateTime
	return self
}
func (self *MessagesQuery) SetEndDateTime(endDateTime time.Time) *MessagesQuery {
	self.endDateTime = &endDateTime
	return self
}
func (self *MessagesQuery) SetLimit(limit uint64) *MessagesQuery {
	self.limit = &limit
	return self
}
func (self *MessagesQuery) SetSeverity(severity Severity) *MessagesQuery {
	self.severity = &severity
	return self
}
func (self *MessagesQuery) SetType(mType string) *MessagesQuery {
	self.mType = &mType
	return self
}
func (self *MessagesQuery) SetSource(source string) *MessagesQuery {
	self.source = &source
	return self
}
func (self *MessagesQuery) SetTag(name string, values []string) *MessagesQuery {
	copyValues := make([]string, len(values), len(values))
	copy(copyValues, values)
	self.tags[strings.ToLower(name)] = copyValues
	return self
}

func (self *MessagesQuery) Entity() string {
	return self.entity
}
func (self *MessagesQuery) StartDateTime() *time.Time {
	return self.startDateTime
}
func (self *MessagesQuery) EndDateTime() *time.Time {
	return self.endDateTime
}
func (self *MessagesQuery) Limit() *uint64 {
	return self.limit
}
func (self *MessagesQuery) Severity() *Severity {
	return self.severity
}
func (self *MessagesQuery) Type() *string {
	return self.mType
}
func (self *MessagesQuery) Source() *string {
	return self.source
}
func (self *MessagesQuery) TagValue(name string) ([]string, bool) {
	value, ok := self.tags[strings.ToLower(name)]
	return value, ok
}

func (self *MessagesQuery) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	m["entity"] = self.entity
	m["tags"] = self.tags
	if self.startDateTime != nil {
		m["startTime"] = self.startDateTime.UnixNano() / 1e6
	}
	if self.endDateTime != nil {
		m["endTime"] = self.endDateTime.UnixNano() / 1e6
	}
	if self.limit != nil {
		m["limit"] = *self.limit
	}
	if self.severity != nil {
		m["severity"] = *self.severity
	}
	if self.mType != nil {
		m["type"] = *self.mType
	}
	if self.source != nil {
		m["source"] = *self.source
	}
	return json.Marshal(m)
}
func (self *MessagesQuery) String() string {
	obj, _ := self.MarshalJSON()
	return string(obj)
}
