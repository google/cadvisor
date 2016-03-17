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
	"time"
)

type Entity struct {
	name string

	enabled *bool

	tags map[string]string

	lastInsertTime *time.Time
}

func NewEntity(name string) *Entity {
	return &Entity{name: name, tags: make(map[string]string)}
}

func (self *Entity) Name() string {
	return self.name
}
func (self *Entity) Enabled() *bool {
	return self.enabled
}
func (self *Entity) Tags() map[string]string {
	copy := map[string]string{}
	for k, v := range self.tags {
		copy[k] = v
	}
	return copy
}

func (self *Entity) LastIsertTime() *time.Time {
	return self.lastInsertTime
}

func (self *Entity) SetEnabled(isEnabled bool) *Entity {
	self.enabled = &isEnabled
	return self
}
func (self *Entity) SetTag(key, val string) *Entity {
	self.tags[strings.ToLower(key)] = val
	return self
}

func (self *Entity) UnmarshalJSON(data []byte) error {
	var jsonMap map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&jsonMap); err != nil {
		return err
	}
	self.name, _ = jsonMap["name"].(string)

	enabled, _ := jsonMap["enabled"].(bool)
	self.enabled = &enabled
	if _, ok := jsonMap["lastInsertTime"]; ok {
		lastInsertTimeString, _ := jsonMap["lastInsertTime"].(json.Number)
		lastInsertTimeInt, _ := lastInsertTimeString.Int64()
		lastInsertTime := time.Unix(0, lastInsertTimeInt*1e6)
		self.lastInsertTime = &lastInsertTime
	}
	m, _ := jsonMap["tags"].(map[string]interface{})
	if self.tags == nil {
		self.tags = map[string]string{}
	}
	for key, val := range m {
		self.tags[key], _ = val.(string)
	}
	return nil
}
