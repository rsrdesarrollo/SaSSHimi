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

package server

import (
	"errors"
	"github.com/rsrdesarrollo/SaSSHimi/common"
	"github.com/rsrdesarrollo/SaSSHimi/models"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"sync"
)

type tunnel struct {
	isOpen      bool
	sshClient   *ssh.Client
	clients     map[string]models.Client
	clientsLock *sync.Mutex
	inChan      chan models.DataMessage
	outChan     chan models.DataMessage
}

func newTunnel() *tunnel {
	return &tunnel{
		isOpen:      false,
		clients:     make(map[string]models.Client),
		clientsLock: &sync.Mutex{},
		inChan:      make(chan models.DataMessage, 10),
		outChan:     make(chan models.DataMessage, 10),
	}
}

func (t *tunnel) openTunnel() error {
	var err error

	config := &ssh.ClientConfig{
		User:            viper.GetString("User"),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(viper.GetString("Password")),
		},
	}

	t.sshClient, err = ssh.Dial("tcp", viper.GetString("RemoteHost"), config)

	if err != nil {
		return errors.New("Dial error: " + err.Error())
	}

	defer t.sshClient.Close()

	session, err := t.sshClient.NewSession()
	if err != nil {
		return errors.New("Failed to create session: " + err.Error())
	}

	selfFilePath, _ := os.Executable()
	selfFile, err := os.Open(selfFilePath)
	session.Stdin = selfFile

	if err != nil {
		return errors.New("Failed to open current binary " + err.Error())
	}

	err = session.Run("cat > ./daemon")

	session, err = t.sshClient.NewSession()
	if err != nil {
		return errors.New("Failed to create session: " + err.Error())
	}

	err = session.Run("chmod +x ./daemon")
	session.Close()

	if err != nil {
		return errors.New("Failed to make daemon executable " + err.Error())
	}

	session, err = t.sshClient.NewSession()
	defer session.Close()

	if err != nil {
		return errors.New("Failed to create session: " + err.Error())
	}

	remoteStdIn, err := session.StdinPipe()
	if err != nil {
		return errors.New("Failed to pipe STDIN on session: " + err.Error())
	}

	remoteStdOut, err := session.StdoutPipe()
	if err != nil {
		return errors.New("Failed to pipe STDOUT on session: " + err.Error())
	}

	session.Stderr = os.Stderr

	go common.ReadInputData(t.inChan, remoteStdOut)
	go common.WriteOutputData(t.outChan, remoteStdIn)

	session.Run("./daemon agent")

	t.isOpen = false
	return errors.New("Remote process is dead")
}

func (t *tunnel) handleClients() {
	for {
		msg := <-t.inChan

		t.clientsLock.Lock()

		client, prs := t.clients[msg.ClientId]

		if prs == false {
			common.Logger.Warningf("Received data from closed client %s", client.Id)
			// Send an empty message to close remote connection
			t.outChan <- models.DataMessage{
				ClientId: client.Id,
				Data:     []byte{},
			}

		} else if len(msg.Data) == 0 {
			client.Conn.Close()
			delete(t.clients, msg.ClientId)
		} else {
			var writed = 0
			for writed < len(msg.Data) {
				wn, err := client.Conn.Write(msg.Data[writed:])
				writed += wn

				if err != nil {
					client.Close()
					delete(t.clients, msg.ClientId)

					common.Logger.Errorf("Error Writing: %s\n", err.Error())
					break
				}
			}

		}

		t.clientsLock.Unlock()
	}
}

func Run() {

	ln, err := net.Listen("tcp", "127.0.0.1:8080")

	if err != nil {
		panic("Failed to bind local port " + err.Error())
	}

	tunnel := newTunnel()
	go func() {
		err = tunnel.openTunnel()

		if err != nil {
			common.Logger.Fatal("Failed to open tunnel ", err.Error())
		}
	}()

	onExit := func() {
		tunnel.sshClient.Close()
		ln.Close()
	}

	common.ExitCallback(onExit)
	defer onExit()

	go tunnel.handleClients()

	for {
		conn, err := ln.Accept()
		if err != nil {
			common.Logger.Fatalf("Error in conncetion accept: %s", err.Error())
		}

		common.Logger.Info("New connection from ", conn.RemoteAddr().String())

		client := models.Client{
			Id:       conn.RemoteAddr().String(),
			Conn:     conn,
			OutChann: tunnel.outChan,
		}

		tunnel.clientsLock.Lock()
		tunnel.clients[client.Id] = client
		tunnel.clientsLock.Unlock()
		go common.ReadFromClientToChannel(client, tunnel.outChan)
	}
}
