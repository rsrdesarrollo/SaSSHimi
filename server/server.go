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
	"fmt"
	"github.com/rsrdesarrollo/SaSSHimi/common"
	"github.com/rsrdesarrollo/SaSSHimi/models"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"net"
	"os"
	user2 "os/user"
	"strings"
	"sync"
	"syscall"
)

type tunnel struct {
	isOpen      bool
	sshClient   *ssh.Client
	clients     map[string]models.Client
	clientsLock *sync.Mutex
	inChan      chan models.DataMessage
	outChan     chan models.DataMessage
	viper       *viper.Viper
}

func newTunnel(viper *viper.Viper) *tunnel {
	return &tunnel{
		isOpen:      true,
		clients:     make(map[string]models.Client),
		clientsLock: &sync.Mutex{},
		inChan:      make(chan models.DataMessage, 10),
		outChan:     make(chan models.DataMessage, 10),
		viper:       viper,
	}
}

func (t *tunnel) getRemoteHost() string {
	remoteHost := t.viper.GetString("RemoteHost")
	if !strings.Contains(remoteHost, ":") {
		remoteHost = remoteHost + ":22"
	}
	return remoteHost
}

func (t *tunnel) getUsername() string {
	user := t.viper.GetString("User")
	if user == "" {
		user, _ := user2.Current()
		return user.Name
	}
	return user
}

func (t *tunnel) getPassword() string {
	password := t.viper.GetString("Password")
	if password == "" {
		fmt.Printf("%s@%s's password: ", t.getUsername(), t.getRemoteHost())
		bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
		password = string(bytePassword)
	}
	return password
}

func (t *tunnel) getPublicKey() ssh.Signer {
	pkFilePath := t.viper.GetString("PrivateKey")

	if pkFilePath == "" {
		return nil
	}

	key, err := ioutil.ReadFile(pkFilePath)
	if err != nil {
		common.Logger.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		common.Logger.Fatalf("unable to parse private key: %v", err)
	}

	return signer
}

func (t *tunnel) openTunnel(verboseLevel int) error {
	var err error

	var authMethods = []ssh.AuthMethod{}

	pkSigner := t.getPublicKey()
	if pkSigner != nil {
		authMethods = append(authMethods, ssh.PublicKeys(pkSigner))
	}
	authMethods = append(authMethods, ssh.Password(t.getPassword()))

	config := &ssh.ClientConfig{
		User:            t.getUsername(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            authMethods,
	}

	t.sshClient, err = ssh.Dial("tcp", t.getRemoteHost(), config)

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

	if verboseLevel == 0 {
		session.Run("./daemon agent")
	} else {
		verbose := strings.Repeat("v", verboseLevel)
		session.Run("./daemon -" + verbose + " agent")
	}

	t.isOpen = false
	return errors.New("Remote process is dead")
}

func (t *tunnel) handleClients() {
	for t.isOpen {
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

func Run(viper *viper.Viper, bindAddress string, verboseLevel int) {

	ln, err := net.Listen("tcp", bindAddress)

	if err != nil {
		panic("Failed to bind local port " + err.Error())
	}

	tunnel := newTunnel(viper)
	go func() {
		err = tunnel.openTunnel(verboseLevel)

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

	for tunnel.isOpen {
		conn, err := ln.Accept()
		if err != nil {
			common.Logger.Fatalf("Error in conncetion accept: %s", err.Error())
		}

		common.Logger.Debug("New connection from ", conn.RemoteAddr().String())

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
