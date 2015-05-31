// Copyright 2015 Google Inc. All Rights Reserved.
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

/*
   This client is a wrapper around Goryman,
   since Goryman doesn't support direct interaction
   with TCP connection which we need for batch-sending
   events.

   TODO: reconnect on broken pipe.
*/

package riemann

import(
	"net"
	"time"

	"github.com/bigdatadev/goryman/proto"
	riemann "github.com/bigdatadev/goryman"
)

type riemannClient struct {
	addr string
	conn *riemann.TcpTransport
}

func (c *riemannClient) Connect() error {
	conn, err := net.DialTimeout("tcp", c.addr, time.Second*5)
	if err != nil {
		return err
	}
	c.conn = riemann.NewTcpTransport(conn)
	return nil
}

func (c *riemannClient) SendMessage(msg *proto.Msg) (*proto.Msg, error) {
	return c.conn.SendRecv(msg)
}

func (c *riemannClient) Close() error {
	err := c.conn.Close()
	if err != nil {
		return err
	}
	c.conn = nil
	return nil
}
