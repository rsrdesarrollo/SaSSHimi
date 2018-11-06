package common

import (
	"encoding/gob"
	"github.com/rsrdesarrollo/SaSSHimi/utils"
	"io"
	"sync"
	"time"
)

type ChannelForwarder struct {
	InChannel   chan *DataMessage
	OutChannel  chan *DataMessage
	Reader      io.Reader
	Writer      io.Writer
	ChannelOpen bool

	NotifyCousure chan struct{}

	Clients     map[string]*Client
	ClientsLock *sync.Mutex
}

func (c *ChannelForwarder) ReadInputData() {
	decoder := gob.NewDecoder(c.Reader)

	utils.Logger.Debug("Reading from io.Reader to InChannel")

	for c.ChannelOpen {
		var inMsg DataMessage
		err := decoder.Decode(&inMsg)
		if err != nil {
			utils.Logger.Error("Read ERROR: ", err)
			break
		}
		c.InChannel <- &inMsg
	}

	c.Close()
}

func (c *ChannelForwarder) WriteOutputData() {
	encoder := gob.NewEncoder(c.Writer)

	utils.Logger.Debug("Writing from OutChannel to io.Writer")

	for c.ChannelOpen {
		outMsg := <-c.OutChannel
		err := encoder.Encode(outMsg)

		if err != nil {
			utils.Logger.Error("Write ERROR: ", err)
			break
		}
	}

	c.Close()
}

func (c *ChannelForwarder) Close() {
	c.ChannelOpen = false
}

func (c *ChannelForwarder) Terminate() {
	msg := NewMessage("", nil)
	msg.CloseChannel = true

	c.OutChannel <- msg
}

func (c *ChannelForwarder) KeepAlive(){
	for c.ChannelOpen {
		c.sendKeepAlive()
		time.Sleep(30 * time.Second)
	}
}

func (c *ChannelForwarder) sendKeepAlive() {
	msg := NewMessage("", nil)
	msg.KeepAlive = true

	c.OutChannel <- msg
}
