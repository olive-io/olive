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

import (
	"context"

	pb "github.com/olive-io/olive/api/discoverypb"
)

type IDiscovery interface {
	ListNode(ctx context.Context) ([]INode, error)
	Discover(ctx context.Context, options ...DiscoverOption) (INode, error)
}

type INode interface {
	Get() *pb.Node
	GetExecutor(ctx context.Context, options ...DiscoverOption) (IExecutor, error)
}

type Request struct {
	Context     context.Context
	Headers     map[string]any
	Properties  map[string]any
	DataObjects map[string]any
}

type Response struct {
	Properties  map[string]any
	DataObjects map[string]any
}

type IExecutor interface {
	Execute(ctx context.Context, req *Request, options ...ExecuteOption) (*Response, error)
}
