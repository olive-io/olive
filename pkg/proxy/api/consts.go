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

const (
	HeaderKeyPrefix = "ov:"
	NodeIdKey       = "ov:node_id"
	ActivityIdKey   = "ov:activity_id"
	ProtocolKey     = "ov:protocol"
	MethodKey       = "ov:method"
	SecurityKey     = "ov:security"
	ContentTypeKey  = "ov:content-type"
	HostKey         = "ov:host"
	URLKey          = "ov:url"
	HandlerKey      = "ov:handler"
	DescKey         = "ov:desc"
)

// the keys of ServiceTask Header

// the keys of ScriptTask Header

// the keys of SendTask and ReceiveTask header

// http Request Header Key
const (
	RequestActivityKey = "x-olive-activity"
)

const (
	DefaultTaskURL = "/discoverypb.Gateway/Transmit"
	DefaultService = "io.olive.gateway"
	DefaultRPC     = "Gateway"
)
