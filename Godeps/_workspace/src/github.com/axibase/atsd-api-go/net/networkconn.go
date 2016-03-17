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
	"fmt"
	"net"

	"bufio"
	"errors"
	"github.com/golang/glog"
	"sync"
	"time"
)

func DialTimeout(protocol, hostport string, timeout time.Duration, bufferSize int) (*NetworkConn, error) {
	nc := &NetworkConn{protocol: protocol, hostport: hostport, timeout: timeout}
	err := nc.initConn(bufferSize)
	if err != nil {
		return nil, err
	}
	return nc, nil
}

type NetworkConn struct {
	protocol string
	hostport string
	timeout  time.Duration
	conn     net.Conn
	buffer   *bufio.Writer
	mu       sync.Mutex
}

func (self *NetworkConn) initConn(bufferSize int) error {
	var err error
	self.conn, err = net.DialTimeout(self.protocol, self.hostport, self.timeout)
	if err != nil {
		self.Close()
		glog.Error("Atsd storage network client - could not init connection: ", err)
		return err
	}
	self.buffer = bufio.NewWriterSize(self.conn, bufferSize)
	return nil
}

func (self *NetworkConn) Series(seriesCommand *SeriesCommand) error {
	err := self.writeCommand(seriesCommand)
	if err != nil {
		return err
	}
	return nil
}
func (self *NetworkConn) Message(messageCommand *MessageCommand) error {
	err := self.writeCommand(messageCommand)
	if err != nil {
		return err
	}
	return nil
}
func (self *NetworkConn) Property(propertyCommand *PropertyCommand) error {
	err := self.writeCommand(propertyCommand)
	if err != nil {
		return err
	}
	return nil
}
func (self *NetworkConn) EntityTag(entityTagCommand *EntityTagCommand) error {
	err := self.writeCommand(entityTagCommand)
	if err != nil {
		return err
	}
	return nil
}

func (self *NetworkConn) Flush() error {
	self.mu.Lock()
	defer self.mu.Unlock()
	err := self.buffer.Flush()
	if err != nil {
		glog.Error("Atsd storage network client - flush failed: ", err)
		return err
	}
	return nil
}
func (self *NetworkConn) Close() {
	self.mu.Lock()
	defer self.mu.Unlock()
	if self.conn != nil {
		self.conn.Close()
		self.conn = nil
	}
}

func (self *NetworkConn) writeCommand(w fmt.Stringer) error {
	self.mu.Lock()
	defer self.mu.Unlock()
	if self.conn == nil {
		glog.Error("Atsd storage network client - Need to open connection first")
		return errors.New("Atsd storage network client - Need to open connection first")
	}
	_, err := fmt.Fprintln(self.buffer, w)
	if err != nil {
		glog.Error("Atsd storage network client - writeCommand failed: ", err)
		return err
	}
	return nil
}
