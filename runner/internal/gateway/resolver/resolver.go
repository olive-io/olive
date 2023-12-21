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

package resolver

import (
	"net/http"

	"github.com/cockroachdb/errors"
	pb "github.com/olive-io/olive/api/discoverypb"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrInvalidPath = errors.New("invalid path")
)

// Resolver resolves requests to endpoints
type Resolver interface {
	Activity() pb.Activity
	Resolve(r *http.Request) (*Endpoint, error)
}

// Endpoint is the endpoint for a http request
type Endpoint struct {
	// e.g greeter
	Name string
	// HTTP Host e.g example.com
	Host string
	// HTTP Methods e.g GET, POST
	Method string
	// HTTP Path e.g /greeter.
	Path string
}

type Factory struct {
	resolver map[pb.Activity]Resolver
}

// NewFactory returns a factory about Resolver
func NewFactory() *Factory {
	resolvers := map[pb.Activity]Resolver{}
	resolvers[pb.Activity_ServiceTask] = &serviceResolver{}
	resolvers[pb.Activity_ScriptTask] = &scriptResolver{}
	resolvers[pb.Activity_UserTask] = &userResolver{}
	resolvers[pb.Activity_SendTask] = &sendResolver{}
	resolvers[pb.Activity_ReceiveTask] = &receiveResolver{}
	resolvers[pb.Activity_CallActivity] = &callActivityResolver{}

	return &Factory{
		resolver: resolvers,
	}
}

func (f *Factory) GetResolver(act pb.Activity) (Resolver, bool) {
	r, ok := f.resolver[act]
	return r, ok
}
