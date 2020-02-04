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

package cli

import (
	"github.com/rsrdesarrollo/SaSSHimi/server"
	"github.com/spf13/cobra"
)


var transparentCmd = &cobra.Command{
	Use:   "transparent <tunnel_command>",
	Short: "Run local server to create tunnels executing transparent command",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		server.RunTransparent(args, bindAddress)
	},
}

func init() {
	rootCmd.AddCommand(transparentCmd)

	transparentCmd.Flags().StringVar(&bindAddress, "bind", "127.0.0.1:1080", "Set local bind address and port")
	transparentCmd.Flags().StringVarP(&idFile, "identity_file", "i", "", "Path to private key")
}
