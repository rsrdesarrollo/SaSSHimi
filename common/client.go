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

package common

import (
	"github.com/rsrdesarrollo/SaSSHimi/utils"
	"net"
	"sync"
)

type Client struct {
	Id           string
	conn         net.Conn
	outChann     chan *DataMessage
	inChann      chan *DataMessage
	readyToClose bool
	clientMutex  *sync.Mutex
}

func (c *Client) ReadyToClose() bool {
	return c.readyToClose
}

func (c *Client) SetReadyToClose(readyToClose bool) {
	c.readyToClose = readyToClose
}

func NewClient(id string, conn net.Conn, outChannel chan *DataMessage) *Client {
	return &Client{
		Id:           id,
		conn:         conn,
		outChann:     outChannel,
		readyToClose: false,
		clientMutex:  &sync.Mutex{},
	}
}

func (c *Client) Close() {
	var mustBeClosed bool

	c.clientMutex.Lock()
	if c.ReadyToClose() {
		mustBeClosed = true
	} else {
		mustBeClosed = false
		c.readyToClose = true

		utils.Logger.Debug("First attempt to close", c.Id)
	}
	c.clientMutex.Unlock()

	if mustBeClosed {
		utils.Logger.Debug("Really closing", c.Id)
		c.conn.Close()
	}

}

func (c *Client) Write(data []byte) error {
	var writed = 0
	for writed < len(data) {
		wn, err := c.conn.Write(data)
		writed += wn

		if writed < len(data) {
			utils.Logger.Debugf("******* Need second write of %d bytes on client %s", len(data)-writed, c.Id)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) NotifyEOF() {
	c.outChann <- NewMessage(c.Id, []byte{})
}

func (c *Client) ReadFromClientToChannel() {
	for {
		data := make([]byte, 1024)
		readed, err := c.conn.Read(data)
		if err != nil {
			c.Close()
			c.NotifyEOF()
			break
		}

		c.outChann <- NewMessage(c.Id, data[:readed])
	}
}
