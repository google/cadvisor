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
	"bytes"
	"encoding/json"
	"strings"

	"github.com/axibase/atsd-api-go/net"
)

type Severity string

const (
	UNDEFINED Severity = "UNDEFINED"
	UNKNOWN   Severity = "UNKNOWN"
	NORMAL    Severity = "NORMAL"
	WARNING   Severity = "WARNING"
	MINOR     Severity = "MINOR"
	MAJOR     Severity = "MAJOR"
	CRITICAL  Severity = "CRITICAL"
	FATAL     Severity = "FATAL"
)

type Message struct {
	entity  string
	message string

	timestamp *net.Millis
	severity  *Severity
	mType     *string
	source    *string
	tags      map[string]string
}

func NewMessage(entity string) *Message {
	return &Message{
		entity: entity,
		tags:   map[string]string{},
	}
}

func (self *Message) SetEntity(entity string) *Message {
	self.entity = entity
	return self
}
func (self *Message) SetMessage(message string) *Message {
	self.message = message
	return self
}
func (self *Message) SetTimestamp(timestamp net.Millis) *Message {
	self.timestamp = &timestamp
	return self
}
func (self *Message) SetSeverity(severity Severity) *Message {
	self.severity = &severity
	return self
}
func (self *Message) SetType(mType string) *Message {
	self.mType = &mType
	return self
}
func (self *Message) SetSource(source string) *Message {
	self.source = &source
	return self
}
func (self *Message) SetTag(name, value string) *Message {
	self.tags[strings.ToLower(name)] = value
	return self
}

func (self *Message) Entity() string {
	return self.entity
}
func (self *Message) Message() string {
	return self.message
}
func (self *Message) Timestamp() *net.Millis {
	return self.timestamp
}
func (self *Message) Severity() *Severity {
	return self.severity
}
func (self *Message) Type() *string {
	return self.mType
}
func (self *Message) Source() *string {
	return self.source
}
func (self *Message) TagValue(name string) (string, bool) {
	value, ok := self.tags[strings.ToLower(name)]
	return value, ok
}

func (self *Message) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	m["entity"] = self.entity
	m["message"] = self.message
	m["tags"] = self.tags
	if self.timestamp != nil {
		m["timestamp"] = *self.Timestamp()
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
func (self *Message) UnmarshalJSON(data []byte) error {
	var jsonMap map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&jsonMap); err != nil {
		return err
	}
	if entity, ok := jsonMap["entity"]; ok {
		self.entity = entity.(string)
	}
	if message, ok := jsonMap["message"]; ok {
		self.message = message.(string)
	}
	if iTimestamp, ok := jsonMap["timestamp"]; ok {
		timestamp := iTimestamp.(json.Number)
		t, _ := timestamp.Int64()
		self.SetTimestamp(net.Millis(t))
	}
	if iSeverity, ok := jsonMap["severity"]; ok {
		severity := iSeverity.(string)
		s := Severity(severity)
		self.severity = &(s)
	}
	if iType, ok := jsonMap["type"]; ok {
		mType := iType.(string)
		self.mType = &(mType)
	}
	if iSource, ok := jsonMap["source"]; ok {
		iSource := iSource.(string)
		self.source = &(iSource)
	}
	if tags, ok := jsonMap["tags"]; ok {
		self.tags = map[string]string{}
		t := tags.(map[string]interface{})
		for key, val := range t {
			str := val.(string)
			self.tags[key] = str
		}
	}
	return nil
}

func (self *Message) String() string {
	bytes, _ := self.MarshalJSON()
	return string(bytes)
}
