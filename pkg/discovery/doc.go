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

package discovery

import "github.com/cockroachdb/errors"

const (
	HeaderKeyPrefix = "ov:"
	NodeIdKey       = "ov:node_id"
	ActivityIdKey   = "ov:activity_id"
	ProtocolKey     = "ov:protocol"
)

// the keys of ServiceTask Header
const (
	ServiceMethodKey      = "ov:method"
	ServiceContentTypeKey = "ov:content-type"
	ServiceURLKey         = "ov:url"
	ServiceProtocolKey    = "ov:protocol"
)

var (
	ErrNotFoundNode = errors.New("node not found")
	ErrNoNode       = errors.New("no node")
	ErrNoExecutor   = errors.New("no executor")
)
