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
	"github.com/rsrdesarrollo/SaSSHimi/utils"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"net"
	"os"
	user2 "os/user"
	"strings"
	"sync"
	"syscall"
	"time"
)

type tunnel struct {
	common.ChannelForwarder
	sshClient  *ssh.Client
	sshSession *ssh.Session
	viper      *viper.Viper
}

func newTunnel(viper *viper.Viper) *tunnel {
	return &tunnel{
		ChannelForwarder: common.ChannelForwarder{
			OutChannel: make(chan *common.DataMessage, 10),
			InChannel:  make(chan *common.DataMessage, 10),

			ChannelOpen: true,
			ClientsLock: &sync.Mutex{},
			Clients:     make(map[string]*common.Client),

			NotifyCousure: make(chan struct{}),
		},
		viper: viper,
	}
}

func (t *tunnel) getRemoteHost() string {
	remoteHost := t.viper.GetString("RemoteHost")
	if !strings.Contains(remoteHost, ":") {
		remoteHost = remoteHost + ":22"
	}

	utils.Logger.Debug("SSH Remote Host:", remoteHost)
	return remoteHost
}

func (t *tunnel) getUsername() string {
	user := t.viper.GetString("User")
	if user == "" {
		user, _ := user2.Current()
		return user.Name
	}
	utils.Logger.Debug("SSH User:", user)
	return user
}

func (t *tunnel) getPassword() string {
	password := t.viper.GetString("Password")
	if password == "" {
		fmt.Printf("%s@%s's password: ", t.getUsername(), t.getRemoteHost())
		bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println("")
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
		utils.Logger.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		utils.Logger.Fatalf("unable to parse private key: %v", err)
	}

	return signer
}

func (t *tunnel) uploadForwarder() error {
	session, err := t.sshClient.NewSession()
	defer session.Close()
	if err != nil {
		return errors.New("Failed to create session: " + err.Error())
	}

	selfFilePath, _ := os.Executable()
	selfFile, err := os.Open(selfFilePath)
	session.Stdin = selfFile

	if err != nil {
		return errors.New("Failed to open current binary " + err.Error())
	}

	err = session.Run("cat > ./.daemon && chmod +x ./.daemon")

	return err
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

	err = t.uploadForwarder()
	if err != nil {
		return errors.New("Failed to upload forwarder " + err.Error())
	}

	t.sshSession, err = t.sshClient.NewSession()
	defer t.sshSession.Close()

	if err != nil {
		return errors.New("Failed to create session: " + err.Error())
	}

	t.Writer, err = t.sshSession.StdinPipe()
	if err != nil {
		return errors.New("Failed to pipe STDIN on session: " + err.Error())
	}

	t.Reader, err = t.sshSession.StdoutPipe()
	if err != nil {
		return errors.New("Failed to pipe STDOUT on session: " + err.Error())
	}

	t.sshSession.Stderr = os.Stderr

	go t.ReadInputData()
	go t.WriteOutputData()

	utils.Logger.Notice("SSH Tunnel Open")

	var runCommand = "./.daemon agent %s"
	var commandOps = ""

	if verboseLevel != 0 {
		commandOps = "-" + strings.Repeat("v", verboseLevel)
	}

	t.sshSession.Run(fmt.Sprintf(runCommand, commandOps))

	t.ChannelOpen = false
	t.NotifyCousure <- struct{}{}

	return errors.New("Remote process is dead")
}

func (t *tunnel) handleClients() {
	for t.ChannelOpen {
		msg := <-t.InChannel

		if msg.KeepAlive {
			continue
		}

		t.ClientsLock.Lock()

		client, prs := t.Clients[msg.ClientId]

		if prs == false {
			utils.Logger.Warning("Received data from closed client", msg.ClientId)
		} else {
			if msg.DeadClient {
				// ACK for client termination
				client.NotifyEOF(false)
				client.Terminate()
				delete(t.Clients, msg.ClientId)
			} else if msg.CloseClient {
				client.Close()
				delete(t.Clients, msg.ClientId)
			} else if !client.IsDead() {
				err := client.Write(msg.Data)

				if err != nil {
					client.Terminate()
					client.NotifyEOF(true)

					utils.Logger.Errorf("Error Writing: %s\n", err.Error())
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

	utils.Logger.Notice("Proxy bind at", bindAddress)

	tunnel := newTunnel(viper)

	termios, _ := unix.IoctlGetTermios(int(syscall.Stdin), unix.TCGETS)
	onExit := func() {
		unix.IoctlSetTermios(int(syscall.Stdin), unix.TCGETS, termios)
		tunnel.Terminate()

		utils.Logger.Notice("Waiting to remote process to clean up...")
		select {
		case <-tunnel.NotifyCousure:
		case <-time.After(5 * time.Second):
			tunnel.sshSession.Signal(ssh.SIGTERM)
			utils.Logger.Warning("Remote close timeout. Sending TERM signal.")
		}

		select {
		case <-tunnel.NotifyCousure:
		case <-time.After(5 * time.Second):
			utils.Logger.Error("Remote process don't respond. Force close channel.")
			utils.Logger.Error("IMPORTANT: This might leave files in remote host.")
			tunnel.sshSession.Close()
		}

		tunnel.sshClient.Close()
		ln.Close()
	}

	utils.ExitCallback(onExit)

	go func() {
		err = tunnel.openTunnel(verboseLevel)

		if err != nil {
			utils.Logger.Fatal("Failed to open tunnel ", err.Error())
		}
	}()

	go tunnel.handleClients()
	go tunnel.KeepAlive()

	for tunnel.ChannelOpen {
		conn, err := ln.Accept()
		if err != nil {
			utils.Logger.Fatalf("Error in conncetion accept: %s", err.Error())
			continue
		}

		utils.Logger.Debug("New connection from ", conn.RemoteAddr().String())

		client := common.NewClient(
			conn.RemoteAddr().String(),
			conn,
			tunnel.OutChannel,
		)

		tunnel.Clients[client.Id] = client
		go client.ReadFromClientToChannel()
	}
}
