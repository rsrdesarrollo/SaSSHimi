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

package utils

import (
	"github.com/op/go-logging"
	"os"
)

var Logger = logging.MustGetLogger("SaSSHimi")

func init() {
	var format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{program:10s} - %{shortfunc:-20s} ▶ %{level:-8s} %{id:03x}%{color:reset} %{message}`,
	)

	logfile, _ := os.Open("/tmp/" + os.Args[0] + ".log")

	stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
	fileBackend := logging.NewLogBackend(logfile, "", 0)

	stderrBackendFormater := logging.NewBackendFormatter(stderrBackend, format)

	stderrBackendLeveled := logging.AddModuleLevel(stderrBackendFormater)

	logging.SetBackend(stderrBackendLeveled, fileBackend)

}
