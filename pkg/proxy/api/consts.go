// Copyright 2023 The olive Authors
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

package api

import (
	"strings"
)

const (
	HeaderKeyPrefix = "ov:"
	ActivityKey     = "ov:activity"
	EndpointKey     = "ov:endpoint"
	ProtocolKey     = "ov:protocol"
	MethodKey       = "ov:method"
	SecurityKey     = "ov:security"
	ContentTypeKey  = "ov:content_type"
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
