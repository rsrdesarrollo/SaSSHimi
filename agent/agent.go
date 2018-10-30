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

package agent

import (
	"github.com/armon/go-socks5"
	"github.com/rsrdesarrollo/SaSSHimi/common"
	"github.com/rsrdesarrollo/SaSSHimi/models"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type agent struct {
	models.ChannelForwarder
	sockFilePath string
}

func newAgent() agent {
	return agent{
		ChannelForwarder: models.ChannelForwarder{
			OutChannel:  make(chan *models.DataMessage, 10),
			InChannel:   make(chan *models.DataMessage, 10),
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ChannelOpen: false,
			Clients:     make(map[string]*models.Client),
			ClientsLock: &sync.Mutex{},
		},
		sockFilePath: "127.0.1.1:8888", //"./daemon_" + common.RandStringRunes(10),
	}
}

func (a *agent) startSocksServer(done chan struct{}) {
	conf := &socks5.Config{
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	server, err := socks5.New(conf)

	if err != nil {
		common.Logger.Fatal("ERROR Creating socks socksServer: " + err.Error())
	}

	common.Logger.Info("Handling sock connection")

	ln, err := net.Listen("tcp", a.sockFilePath)

	if err != nil {
		common.Logger.Fatal("Failed to bind local socket " + err.Error())
	}

	common.Logger.Infof("Socks porxy listening on unix://%s", a.sockFilePath)
	done <- struct{}{}
	for {
		conn, err := ln.Accept()
		if err != nil {
			common.Logger.Fatal("Error accepting socks connection: " + err.Error())
		}

		go server.ServeConn(conn)
	}
}

func (a *agent) handleInOutData() {
	go a.ReadInputData()
	go a.WriteOutputData()

	for a.ChannelOpen {
		msg := <-a.InChannel

		a.ClientsLock.Lock()
		client, prs := a.Clients[msg.ClientId]

		if prs == false {
			conn, err := net.Dial("tcp", a.sockFilePath)

			if err != nil {
				common.Logger.Error("Connection dial error: ", err)
			} else {
				client = models.NewClient(
					msg.ClientId,
					conn,
					a.OutChannel,
				)

				common.Logger.Debug("New connection to socks proxy from", conn.LocalAddr().String(), "for client", client.Id)
				a.Clients[msg.ClientId] = client
				prs = true

				go client.ReadFromClientToChannel()
			}
		}
		a.ClientsLock.Unlock()

		if prs == false {
			continue
		}

		if len(msg.Data) == 0 {
			common.Logger.Debug("Closing client sock connection for ", client.Id)
			client.Close(false)

			a.ClientsLock.Lock()
			delete(a.Clients, msg.ClientId)
			a.ClientsLock.Unlock()
		} else {
			var writed = 0
			for writed < len(msg.Data) {
				wn, err := client.Conn.Write(msg.Data[writed:])
				writed += wn

				if writed < len(msg.Data) {
					common.Logger.Debugf("Need second write of %d bytes", len(msg.Data)-writed)
				}

				if err != nil {
					common.Logger.Error("Error writing to client connection: ", err.Error())
					client.Close(true)

					a.ClientsLock.Lock()
					delete(a.Clients, msg.ClientId)
					a.ClientsLock.Unlock()
					break
				}
			}

		}
	}

}

func Run() {

	agent := newAgent()

	onExit := func() {
		selfFilePath, _ := os.Executable()
		os.Remove(agent.sockFilePath)
		os.Remove(selfFilePath)
	}

	defer onExit()
	common.ExitCallback(onExit)

	done := make(chan struct{})
	go agent.startSocksServer(done)
	<-done

	agent.ChannelOpen = true

	go agent.handleInOutData()

	for agent.ChannelOpen {
		time.Sleep(1 * time.Second)
	}
}
