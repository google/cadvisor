// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fake

import (
	"testing"
)

type DmsetupCommand struct {
	Name   string
	Result string
	Err    error
}

// NewFakeDmsetupClient returns a new fake DmsetupClient.
func NewFakeDmsetupClient(t *testing.T, commands ...DmsetupCommand) *DmsetupClient {
	if len(commands) == 0 {
		commands = make([]DmsetupCommand, 0)
	}
	return &DmsetupClient{t: t, commands: commands}
}

// DmsetupClient is a thread-unsafe fake implementation of the
// DmsetupClient interface
type DmsetupClient struct {
	t        *testing.T
	commands []DmsetupCommand
}

func (c *DmsetupClient) Table(deviceName string) ([]byte, error) {
	return c.dmsetup("table")
}

func (c *DmsetupClient) Message(deviceName string, sector int, message string) ([]byte, error) {
	return c.dmsetup("message")
}

func (c *DmsetupClient) Status(deviceName string) ([]byte, error) {
	return c.dmsetup("status")
}

func (c *DmsetupClient) AddCommand(name string, result string, err error) {
	c.commands = append(c.commands, DmsetupCommand{name, result, err})
}

func (c *DmsetupClient) dmsetup(inputCommand string) ([]byte, error) {
	var nextCommand DmsetupCommand
	nextCommand, c.commands = c.commands[0], c.commands[1:]
	if nextCommand.Name != inputCommand {
		c.t.Fatalf("unexpected dmsetup command; expected: %q, got %q", nextCommand.Name, inputCommand)
		// should be unreachable in a test context.
	}

	return []byte(nextCommand.Result), nextCommand.Err
}
