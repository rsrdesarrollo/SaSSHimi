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
	"github.com/elazarl/goproxy"
	"github.com/rsrdesarrollo/SaSSHimi/common"
	"github.com/rsrdesarrollo/SaSSHimi/utils"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type agent struct {
	common.ChannelForwarder
	sockFilePath string
	sockFamily   string
}

func newAgent() agent {
	return agent{
		ChannelForwarder: common.ChannelForwarder{
			OutChannel:  make(chan *common.DataMessage, 10),
			InChannel:   make(chan *common.DataMessage, 10),
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ChannelOpen: false,
			Clients:     make(map[string]*common.Client),
			ClientsLock: &sync.Mutex{},
		},
		sockFamily:   "unix",
		sockFilePath: "./daemon_" + utils.RandStringRunes(10),
	}
}

func (a *agent) runProxyServer(done chan struct{}, useHttpProxy bool) {
	ln, err := net.Listen(a.sockFamily, a.sockFilePath)

	if err != nil {
		utils.Logger.Fatal("Failed to bind local socket " + err.Error())
	}

	if useHttpProxy {
		proxy := goproxy.NewProxyHttpServer()

		done <- struct{}{}

		http.Serve(ln, proxy)
	} else {
		conf := &socks5.Config{
			Logger: log.New(os.Stderr, "", log.LstdFlags),
		}

		server, err := socks5.New(conf)

		if err != nil {
			utils.Logger.Error("ERROR Creating socks socksServer: " + err.Error())
		}

		done <- struct{}{}
		err = server.Serve(ln)

		if err != nil {
			utils.Logger.Error("ERROR Running socks socksServer: " + err.Error())
		}
	}
}

func (a *agent) handleInOutData() {
	for a.ChannelOpen {
		msg := <-a.InChannel

		a.ClientsLock.Lock()
		client, prs := a.Clients[msg.ClientId]

		if prs == false {
			conn, err := net.Dial(a.sockFamily, a.sockFilePath)

			if err != nil {
				utils.Logger.Error("Connection dial error: ", err)
			} else {
				client = common.NewClient(
					msg.ClientId,
					conn,
					a.OutChannel,
				)

				utils.Logger.Debug("New connection to socks proxy from", conn.LocalAddr().String(), "for client", client.Id)
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
			utils.Logger.Debug("Closing client sock connection for ", client.Id)
			client.Close()

			a.ClientsLock.Lock()
			delete(a.Clients, msg.ClientId)
			a.ClientsLock.Unlock()
		} else {
			err := client.Write(msg.Data)

			if err != nil {
				utils.Logger.Error("Error writing to client connection: ", err.Error())
				client.SetReadyToClose(true)
				client.Close()
				client.NotifyEOF()

				a.ClientsLock.Lock()
				delete(a.Clients, msg.ClientId)
				a.ClientsLock.Unlock()
			}
		}

	}
}

func Run(useHttpProxy bool) {

	agent := newAgent()

	onExit := func() {
		utils.Logger.Warning("Agent is closing")
		selfFilePath, _ := os.Executable()
		os.Remove(agent.sockFilePath)
		os.Remove(selfFilePath)
	}

	defer onExit()
	utils.ExitCallback(onExit)

	listeningSignal := make(chan struct{})
	go agent.runProxyServer(listeningSignal, useHttpProxy)
	<-listeningSignal

	agent.ChannelOpen = true

	go agent.ReadInputData()
	go agent.WriteOutputData()

	go agent.handleInOutData()

	for agent.ChannelOpen {
		time.Sleep(1 * time.Second)
	}
}
