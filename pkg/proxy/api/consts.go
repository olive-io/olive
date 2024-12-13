/*
Copyright 2023 The olive Authors

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

package api

import (
	"strings"
)

const (
	HeaderKeyPrefix = "ov:"
	ActivityKey     = "ov:activity"
	TaskTypeKey     = "ov:task-type"
	IdKey           = "ov:id"
	EndpointKey     = "ov:endpoint"
	ProtocolKey     = "ov:protocol"
	MethodKey       = "ov:method"
	SecurityKey     = "ov:security"
	ContentTypeKey  = "ov:content-type"
	HostKey         = "ov:host"
	URLKey          = "ov:url"
	HandlerKey      = "ov:handler"
	DescKey         = "ov:desc"
)

const (
	RPCHandler  = "rpc"
	HTTPHandler = "http"
)

// the keys of ServiceTask Header

// the keys of ScriptTask Header

// the keys of SendTask and ReceiveTask header

const (
	// RequestPrefix http Request Header Key
	RequestPrefix = "X-Olive-"
	// HttpNativePrefix http native header key
	HttpNativePrefix = "X-Native-"
)

const (
	DefaultTaskURL = "/gatewaypb.Gateway/Transmit"
	DefaultService = "io.olive.gateway"
)

func OliveHttpKey(key string) string {
	s := strings.TrimPrefix(key, HeaderKeyPrefix)
	s = strings.ToUpper(string(s[0])) + s[1:]
	return RequestPrefix + s
}

func HttpOliveKey(key string) string {
	s := strings.TrimPrefix(key, RequestPrefix)
	s = strings.ToLower(string(s[0])) + s[1:]
	return HeaderKeyPrefix + s
}
