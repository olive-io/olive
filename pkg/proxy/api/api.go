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
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	pb "github.com/olive-io/olive/api/discoverypb"
)

// Endpoint is a mapping between an RPC method and HTTP endpoint
type Endpoint struct {
	// RPC Method e.g. Greeter.Hello
	Name string `json:"name,omitempty"`
	// Endpoint Activity
	Activity pb.Activity `json:"activity,omitempty"`
	// API Handler e.g rpc, proxy
	Handler string `json:"handler,omitempty"`
	// HTTP Host e.g example.com
	Host []string `json:"host,omitempty"`
	// HTTP Methods e.g GET, POST
	Method []string `json:"method,omitempty"`
	// HTTP Path e.g /greeter. Expect POSIX regex
	Path []string `json:"path,omitempty"`
	// Body destination
	// "*" or "" - top level message value
	// "string" - inner message value
	Body string `json:"body,omitempty"`
	// Stream flag
	Stream string `json:"stream,omitempty"`
}

// Service represents an API service
type Service struct {
	// Name of service
	Name string `json:"name,omitempty"`
	// The endpoint for this service
	Endpoint *Endpoint `json:"endpoint,omitempty"`
	// Versions of this service
	Services []*pb.Service `json:"services,omitempty"`
}

// Encode encodes an endpoint to endpoint metadata
func Encode(e *Endpoint) map[string]string {
	if e == nil {
		return nil
	}

	// endpoint map
	ep := make(map[string]string)

	// set values only if they exist
	set := func(k, v string) {
		if len(v) == 0 {
			return
		}
		ep[k] = v
	}

	set("endpoint", e.Name)
	set("handler", e.Handler)
	set("method", strings.Join(e.Method, ","))
	set("path", strings.Join(e.Path, ","))
	set("host", strings.Join(e.Host, ","))
	set("stream", string(e.Stream))

	return ep
}

// Decode decodes endpoint metadata into an endpoint
func Decode(e map[string]string) *Endpoint {
	if e == nil {
		return nil
	}

	return &Endpoint{
		Name:    e["endpoint"],
		Handler: e["handler"],
		Method:  slice(e["method"]),
		Path:    slice(e["path"]),
		Host:    slice(e["host"]),
		Stream:  e["stream"],
	}
}

// Validate validates an endpoint to guarantee it won't blow up when being served
func Validate(e *Endpoint) error {
	if e == nil {
		return errors.New("endpoint is nil")
	}

	if len(e.Name) == 0 {
		return errors.New("name required")
	}

	for _, p := range e.Path {
		ps := p[0]
		pe := p[len(p)-1]

		if ps == '^' && pe == '$' {
			_, err := regexp.CompilePOSIX(p)
			if err != nil {
				return err
			}
		} else if ps == '^' && pe != '$' {
			return errors.New("invalid path")
		} else if ps != '^' && pe == '$' {
			return errors.New("invalid path")
		}
	}

	if len(e.Handler) == 0 {
		return errors.New("invalid handler")
	}

	return nil
}

func strip(s string) string {
	return strings.TrimSpace(s)
}

func slice(s string) []string {
	var sl []string

	for _, p := range strings.Split(s, ",") {
		if str := strip(p); len(str) > 0 {
			sl = append(sl, str)
		}
	}

	return sl
}
