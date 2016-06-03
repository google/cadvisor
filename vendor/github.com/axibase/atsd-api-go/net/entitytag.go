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

const entityTagType = "$entity_tags"

type EntityTagCommand struct {
	property *PropertyCommand
}

func NewEntityTagCommand(entity, tagName, tagValue string) *EntityTagCommand {
	return &EntityTagCommand{property: NewPropertyCommand(entityTagType, entity, tagName, tagValue)}
}

func (self *EntityTagCommand) Entity() string {
	return self.property.Entity()
}
func (self *EntityTagCommand) Tags() map[string]string {
	return self.property.tags
}

func (self *EntityTagCommand) SetTag(name, value string) *EntityTagCommand {
	self.property.SetTag(name, value)
	return self
}

func (self *EntityTagCommand) String() string {
	return self.property.String()
}
