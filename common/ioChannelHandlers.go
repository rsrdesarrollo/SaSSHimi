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
	"bufio"
	"encoding/json"
	"github.com/rsrdesarrollo/ssh-tunnel/models"
	"io"
)

func ReadInputData(inChannel chan models.DataMessage, reader io.Reader) {
	inReader := bufio.NewReader(reader)

	Logger.Debug("Reading from io.Reader to inChannel")

	for {
		var inMsg models.DataMessage
		line, err := inReader.ReadBytes('\n')
		if err != nil || len(line) == 0 {
			Logger.Error("Read ERROR: ", err)
			break
		}

		err = json.Unmarshal(line, &inMsg)
		if err != nil {
			Logger.Error("Unmarshal ERROR: ", err)
			continue
		}

		inChannel <- inMsg
	}

}

func WriteOutputData(outChannel chan models.DataMessage, writer io.Writer) {

	Logger.Debug("Writing from outChannel to io.Writer")

	for {
		outMsg := <-outChannel
		data, err := json.Marshal(outMsg)

		if err != nil {
			Logger.Error("Marshal ERROR: ", err)
		}

		data = append(data, '\n')
		writed := 0
		for writed < len(data){
			wn, err := writer.Write(data[writed:])
			writed += wn

			if err != nil {
				Logger.Error("Write ERROR: ", err)
				break
			}
		}
	}
}
func ReadFromClientToChannel(client models.Client, outChannel chan models.DataMessage) {
	for {
		data := make([]byte, 1024)
		readed, err := client.Conn.Read(data)
		if err != nil {
			client.Close()
			break
		}

		outChannel <- models.DataMessage{
			ClientId: client.Id,
			Data:     data[:readed],
		}
	}
}
