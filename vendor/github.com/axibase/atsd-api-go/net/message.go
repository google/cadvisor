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

type Millis uint64

type MessageCommand struct {
	entity    string
	timestamp *Millis
	tags      map[string]string
	message   string
}

func NewMessageCommand(entity, message string) *MessageCommand {
	return &MessageCommand{
		entity:  entity,
		message: message,
		tags:    map[string]string{},
	}
}

func (self *MessageCommand) Entity() string {
	return self.entity
}
func (self *MessageCommand) TagValue(name string) string {
	return self.tags[strings.ToLower(name)]
}
func (self *MessageCommand) Tags() map[string]string {
	copy := map[string]string{}
	for k, v := range self.tags {
		copy[k] = v
	}
	return copy
}

func (self *MessageCommand) Message() string {
	return self.message
}
func (self *MessageCommand) Timestamp() *Millis {
	return self.timestamp
}

func (self *MessageCommand) SetEntity(entity string) *MessageCommand {
	self.entity = entity
	return self
}
func (self *MessageCommand) SetTag(name, value string) *MessageCommand {
	self.tags[strings.ToLower(name)] = value
	return self
}
func (self *MessageCommand) SetMessage(message string) *MessageCommand {
	self.message = message
	return self
}
func (self *MessageCommand) SetTimestamp(timestamp Millis) *MessageCommand {
	self.timestamp = &timestamp
	return self
}

func (self *MessageCommand) String() string {
	msg := bytes.NewBufferString("message")
	fmt.Fprintf(msg, " e:\"%v\"", escapeField(self.entity))
	fmt.Fprintf(msg, " m:\"%v\"", escapeField(self.message))
	if self.timestamp != nil {
		fmt.Fprintf(msg, " ms:%v", *self.timestamp)
	}
	for key, val := range self.tags {
		fmt.Fprintf(msg, " t:\"%v\"=\"%v\"", escapeField(key), escapeField(val))
	}
	fmt.Fprint(msg, "\n")
	return msg.String()
}
