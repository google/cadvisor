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
	"strings"
)

type PropertyCommand struct {
	propType  string
	entity    string
	key       map[string]string
	tags      map[string]string
	timestamp *Millis
}

func NewPropertyCommand(propType, entity, tagKey, tagVal string) *PropertyCommand {
	return &PropertyCommand{propType: propType, entity: entity, key: map[string]string{}, tags: map[string]string{strings.ToLower(tagKey): tagVal}}
}
func (self *PropertyCommand) SetKey(key map[string]string) *PropertyCommand {
	self.key = key
	return self
}
func (self *PropertyCommand) SetKeyPart(name, value string) *PropertyCommand {
	self.key[name] = value
	return self
}
func (self *PropertyCommand) SetAllTags(tags map[string]string) *PropertyCommand {
	self.tags = tags
	return self
}
func (self *PropertyCommand) SetTag(name, value string) *PropertyCommand {
	self.tags[strings.ToLower(name)] = value
	return self
}
func (self *PropertyCommand) SetTimestamp(timestamp Millis) *PropertyCommand {
	self.timestamp = &timestamp
	return self
}
func (self *PropertyCommand) PropType() string {
	return self.propType
}
func (self *PropertyCommand) Entity() string {
	return self.entity
}
func (self *PropertyCommand) Key() map[string]string {
	copy := map[string]string{}
	for k, v := range self.key {
		copy[k] = v
	}
	return copy
}
func (self *PropertyCommand) Tags() map[string]string {
	copy := map[string]string{}
	for k, v := range self.tags {
		copy[k] = v
	}
	return copy
}
func (self *PropertyCommand) Timestamp() *Millis {
	return self.timestamp
}

func (self *PropertyCommand) String() string {
	str := bytes.NewBufferString("")
	fmt.Fprintf(str, "property e:%v t:%v", self.entity, self.propType)
	if self.timestamp != nil {
		fmt.Fprintf(str, " ms:%v", *self.timestamp)
	}
	for key, val := range self.key {
		fmt.Fprintf(str, " k:%v=\"%v\"", key, val)
	}
	for key, val := range self.tags {
		fmt.Fprintf(str, " v:%v=\"%v\"", key, val)
	}
	fmt.Fprint(str, "\n")
	return str.String()
}
