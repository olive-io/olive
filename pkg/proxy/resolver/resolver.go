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
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/olive-io/olive/pkg/proxy/api"
)

var (
	ErrUnknown     = errors.New("unknown name from URL")
	ErrNotFound    = errors.New("not found")
	ErrInvalidPath = errors.New("invalid path")
)

type HandlerType string

const (
	HTTP HandlerType = "http"
	RPC  HandlerType = "rpc"
)

// IResolver resolves requests to endpoints
type IResolver interface {
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
	// the Handler of router
	Handler HandlerType
}

// resolver the Resolver implementation for IResolver
type resolver struct{}

func NewResolver() IResolver {
	return new(resolver)
}

func (r *resolver) Resolve(req *http.Request) (*Endpoint, error) {
	// /foo.Bar/Service
	path := req.URL.Path
	switch path {
	case "/":
		return nil, ErrUnknown
	case api.DefaultTaskURL:
		return &Endpoint{
			Name:    api.DefaultService,
			Host:    req.Host,
			Method:  req.Method,
			Path:    path,
			Handler: HTTP,
		}, nil
	}

	// [foo.Bar, Service]
	parts := strings.Split(path[1:], "/")
	// [foo, Bar]
	names := strings.Split(parts[0], ".")
	// foo
	name := strings.Join(names[:len(names)-1], ".")
	endpoint := &Endpoint{
		Name:    name,
		Host:    req.Host,
		Method:  req.Method,
		Path:    path,
		Handler: RPC,
	}

	return endpoint, nil
}
