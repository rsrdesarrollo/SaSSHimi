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
	"time"
)

type agent struct {
	channelOpen  bool
	sockFilePath string
	inChannel    chan models.DataMessage
	outChannel   chan models.DataMessage
}

func newAgent() agent {
	return agent{
		channelOpen:  false,
		sockFilePath: "./daemon_" + common.RandStringRunes(10),
		inChannel:    make(chan models.DataMessage, 10),
		outChannel:   make(chan models.DataMessage, 10),
	}
}

func (a *agent) close() {
	a.channelOpen = false
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

	ln, err := net.Listen("unix", a.sockFilePath)

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
	clientsMap := make(map[string]models.Client)

	go func() {
		common.ReadInputData(a.inChannel, os.Stdin)
		a.close()
	}()
	go func() {
		common.WriteOutputData(a.outChannel, os.Stdout)
		a.close()
	}()

	for a.channelOpen {
		msg := <-a.inChannel
		client, prs := clientsMap[msg.ClientId]

		if prs == false {
			//conn, err := net.Dial("unix", a.sockFilePath) "tcp", "127.0.1.1:8888"
			conn, err := net.Dial("unix", a.sockFilePath)

			if err != nil {
				common.Logger.Error("Connection dial error: ", err)
			}

			client = models.Client{
				Id:       msg.ClientId,
				Conn:     conn,
				OutChann: a.outChannel,
			}

			clientsMap[msg.ClientId] = client

			go common.ReadFromClientToChannel(client, a.outChannel)
		}

		if len(msg.Data) == 0 {
			client.Conn.Close()
			delete(clientsMap, msg.ClientId)
		} else {
			var writed = 0
			for writed < len(msg.Data) {
				wn, err := client.Conn.Write(msg.Data[writed:])
				writed += wn

				if err != nil {
					client.Close()
					delete(clientsMap, msg.ClientId)
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

	agent.channelOpen = true

	go agent.handleInOutData()

	for agent.channelOpen {
		time.Sleep(1 * time.Second)
	}
}
