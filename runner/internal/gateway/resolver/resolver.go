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
	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/execute"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrInvalidPath = errors.New("invalid path")
)

// IResolver resolves requests to endpoints
type IResolver interface {
	Activity() dsypb.Activity
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

// resolver the Resolver implementation for IResolver
type resolver struct{}

func NewResolver() IResolver {
	return new(resolver)
}

func (r *resolver) Activity() dsypb.Activity {
	return dsypb.Activity_ServiceTask
}

func (r *resolver) Resolve(req *http.Request) (*Endpoint, error) {
	// foo
	endpoint := &Endpoint{
		Name:   execute.DefaultExecuteName,
		Host:   req.Host,
		Method: req.Method,
		Path:   execute.DefaultTaskURL,
	}

	return endpoint, nil
}
