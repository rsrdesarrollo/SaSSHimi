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
	models.ChannelForwarder
	sshClient *ssh.Client
	viper     *viper.Viper
}

func newTunnel(viper *viper.Viper) *tunnel {
	return &tunnel{
		ChannelForwarder: models.ChannelForwarder{
			OutChannel: make(chan *models.DataMessage, 10),
			InChannel:  make(chan *models.DataMessage, 10),

			ChannelOpen: true,
			ClientsLock: &sync.Mutex{},
			Clients:     make(map[string]*models.Client),
		},
		viper: viper,
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

	t.Writer, err = session.StdinPipe()
	if err != nil {
		return errors.New("Failed to pipe STDIN on session: " + err.Error())
	}

	t.Reader, err = session.StdoutPipe()
	if err != nil {
		return errors.New("Failed to pipe STDOUT on session: " + err.Error())
	}

	session.Stderr = os.Stderr

	go t.ReadInputData()
	go t.WriteOutputData()

	common.Logger.Info("SSH Tunnel Open :)")

	if verboseLevel == 0 {
		session.Run("./daemon agent")
	} else {
		verbose := strings.Repeat("v", verboseLevel)
		session.Run("./daemon -" + verbose + " agent")
	}

	t.ChannelOpen = false
	return errors.New("Remote process is dead")
}

func (t *tunnel) handleClients() {
	for t.ChannelOpen {
		msg := <-t.InChannel

		t.ClientsLock.Lock()

		client, prs := t.Clients[msg.ClientId]

		if prs == false {
			common.Logger.Warning("Received data from closed client", msg.ClientId)
			// Send an empty message to close remote connection
			t.OutChannel <- models.NewMessage(client.Id, []byte{})

		} else if len(msg.Data) == 0 {
			client.Close(false)
			delete(t.Clients, msg.ClientId)
		} else {
			var writed = 0
			for writed < len(msg.Data) {
				wn, err := client.Conn.Write(msg.Data[writed:])
				writed += wn

				if err != nil {
					client.Close(true)
					delete(t.Clients, msg.ClientId)

					common.Logger.Errorf("Error Writing: %s\n", err.Error())
					break
				}
			}

		}

		t.ClientsLock.Unlock()
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

	//onExit := func() {
	//	tunnel.sshClient.Close()
	//	ln.Close()
	//}

	//common.ExitCallback(onExit)

	go tunnel.handleClients()

	for tunnel.ChannelOpen {
		conn, err := ln.Accept()
		if err != nil {
			common.Logger.Fatalf("Error in conncetion accept: %s", err.Error())
		}

		common.Logger.Debug("New connection from ", conn.RemoteAddr().String())

		client := models.NewClient(
			conn.RemoteAddr().String(),
			conn,
			tunnel.OutChannel,
		)

		tunnel.ClientsLock.Lock()
		tunnel.Clients[client.Id] = client
		tunnel.ClientsLock.Unlock()
		go client.ReadFromClientToChannel()
	}
}
