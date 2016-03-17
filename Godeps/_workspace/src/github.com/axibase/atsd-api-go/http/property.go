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
	"github.com/axibase/atsd-api-go/net"
	"strings"
)

type Property struct {
	propType  string
	entity    string
	key       map[string]string
	tags      map[string]string
	timestamp *net.Millis
}

func NewProperty(propType, entity string) *Property {
	return &Property{propType: propType, entity: entity, key: map[string]string{}, tags: map[string]string{}}
}
func (self *Property) SetKeyPart(name, value string) *Property {
	self.key[name] = value
	return self
}
func (self *Property) SetKey(key map[string]string) *Property {
	self.key = key
	return self
}

func (self *Property) SetAllTags(tags map[string]string) *Property {
	self.tags = tags
	return self
}
func (self *Property) SetTag(name, value string) *Property {
	self.tags[strings.ToLower(name)] = value
	return self
}
func (self *Property) SetTimestamp(timestamp net.Millis) *Property {
	self.timestamp = &timestamp
	return self
}
func (self *Property) PropType() string {
	return self.propType
}
func (self *Property) Entity() string {
	return self.entity
}
func (self *Property) Key() map[string]string {
	copy := map[string]string{}
	for k, v := range self.key {
		copy[k] = v
	}
	return copy
}
func (self *Property) Tags() map[string]string {
	copy := map[string]string{}
	for k, v := range self.tags {
		copy[k] = v
	}
	return copy
}
func (self *Property) TagValue(name string) (string, bool) {
	val, ok := self.tags[strings.ToLower(name)]
	return val, ok
}
func (self *Property) Timestamp() *net.Millis {
	return self.timestamp
}
func (self *Property) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"key":    self.key,
		"tags":   self.tags,
		"type":   self.propType,
		"entity": self.entity,
	}
	if self.timestamp != nil {
		m["timestamp"] = *self.timestamp
	}

	return json.Marshal(m)
}
func (self *Property) String() string {
	obj, _ := self.MarshalJSON()
	return string(obj)
}
