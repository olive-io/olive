/*
Copyright 2024 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package config

import (
	restclient "k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type RestConfig struct {
	restclient.Config
}

type CmdAPIConfig struct {
	clientcmdapi.Config

	Endpoints string `json:"endpoints"`
	Timeout   int64  `json:"timeout"`
}

func FromCmdAPIConfig(config *clientcmdapi.Config) *CmdAPIConfig {
	return &CmdAPIConfig{Config: *config}
}

func NewCmdAPIConfig() *CmdAPIConfig {
	return &CmdAPIConfig{Config: *clientcmdapi.NewConfig()}
}
