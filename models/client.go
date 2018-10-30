// Copyright © 2018 Raul Sampedro
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

package models

import (
	"github.com/rsrdesarrollo/SaSSHimi/common"
	"net"
	"sync"
)

type Client struct {
	Id           string
	Conn         net.Conn
	OutChann     chan *DataMessage
	InChann      chan *DataMessage
	readyToClose bool
	clientMutex  *sync.Mutex
}

func NewClient(id string, conn net.Conn, outChannel chan *DataMessage) *Client {
	return &Client{
		Id:           id,
		Conn:         conn,
		OutChann:     outChannel,
		readyToClose: false,
		clientMutex:  &sync.Mutex{},
	}
}

func (c *Client) Close(sendSignal bool) {
	var shallWaitToClose bool

	c.clientMutex.Lock()
	if c.readyToClose {
		shallWaitToClose = true
	} else {
		shallWaitToClose = false
		c.readyToClose = true

		common.Logger.Debug("First attempt to close", c.Id)

		if sendSignal {
			c.OutChann <- NewMessage(c.Id, []byte{})
		}
	}
	c.clientMutex.Unlock()

	if shallWaitToClose {
		common.Logger.Debug("Really closing", c.Id)
		c.Conn.Close()
	}

}

func (c *Client) ReadFromClientToChannel() {
	for {
		data := make([]byte, 1024)
		readed, err := c.Conn.Read(data)
		if err != nil {
			c.Close(true)
			break
		}

		c.OutChann <- NewMessage(c.Id, data[:readed])
	}
}
