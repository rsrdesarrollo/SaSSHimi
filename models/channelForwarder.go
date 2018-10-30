package models

import (
	"bufio"
	"encoding/json"
	"github.com/rsrdesarrollo/SaSSHimi/common"
	"io"
	"sync"
)

type ChannelForwarder struct {
	InChannel   chan *DataMessage
	OutChannel  chan *DataMessage
	Reader      io.Reader
	Writer      io.Writer
	ChannelOpen bool

	Clients     map[string]*Client
	ClientsLock *sync.Mutex
}

func (c *ChannelForwarder) ReadInputData() {
	inReader := bufio.NewReader(c.Reader)

	common.Logger.Debug("Reading from io.Reader to InChannel")

	for {
		var inMsg DataMessage
		line, err := inReader.ReadBytes('\n')
		if err != nil || len(line) == 0 {
			common.Logger.Error("Read ERROR: ", err)
			break
		}

		err = json.Unmarshal(line, &inMsg)
		if err != nil {
			common.Logger.Error("Unmarshal ERROR: ", err)
			continue
		}

		c.InChannel <- &inMsg
	}

	c.Close()
}

func (c *ChannelForwarder) WriteOutputData() {

	common.Logger.Debug("Writing from OutChannel to io.Writer")

	for {
		outMsg := <-c.OutChannel
		data, err := json.Marshal(*outMsg)

		if err != nil {
			common.Logger.Error("Marshal ERROR: ", err)
		}

		data = append(data, '\n')
		writed := 0
		for writed < len(data) {
			wn, err := c.Writer.Write(data[writed:])
			writed += wn

			if err != nil {
				common.Logger.Error("Write ERROR: ", err)
				break
			}
		}
	}

	c.Close()
}

func (c *ChannelForwarder) Close() {
	c.ChannelOpen = false
}
