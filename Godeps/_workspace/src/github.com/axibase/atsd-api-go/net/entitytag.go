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

type EntityTagCommand struct {
	entity string
	tags   map[string]string
}

func NewEntityTagCommand(entity, tagName, tagValue string) *EntityTagCommand {
	return &EntityTagCommand{entity: entity, tags: map[string]string{tagName: tagValue}}
}

func (self *EntityTagCommand) Entity() string {
	return self.entity
}
func (self *EntityTagCommand) Tags() map[string]string {
	copy := map[string]string{}
	for k, v := range self.tags {
		copy[k] = v
	}
	return copy
}

func (self *EntityTagCommand) SetTag(name, value string) *EntityTagCommand {
	self.tags[strings.ToLower(name)] = value
	return self
}

func (self *EntityTagCommand) String() string {
	result := bytes.NewBufferString("entity-tag")
	fmt.Fprintf(result, " e:%v", self.entity)
	for name, value := range self.tags {
		fmt.Fprintf(result, " t:%v=\"%v\"", name, value)
	}
	fmt.Fprint(result, "\n")
	return result.String()
}
