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
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/x/proxy/server"
)

// Endpoint is a mapping between an RPC method and HTTP endpoint
type Endpoint struct {
	// RPC Method e.g. Greeter.Hello
	Name string `json:"name,omitempty"`
	// API Handler e.g rpc, proxy
	Handler string `json:"handler,omitempty"`
	// HTTP Host e.g example.com
	Host []string `json:"host,omitempty"`
	// HTTP Methods e.g GET, POST
	Method []string `json:"method,omitempty"`
	// HTTP Path e.g /greeter. Expect POSIX regex
	Path []string `json:"path,omitempty"`
	// Security HTTP authorization, e.g. Basic, Bearer
	Security string `json:"security,omitempty"`

	Request  *dsypb.Box `json:"request,omitempty"`
	Response *dsypb.Box `json:"response,omitempty"`

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
	Services []*dsypb.Service `json:"services,omitempty"`
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

	set(EndpointKey, e.Name)
	set(HandlerKey, e.Handler)
	set(MethodKey, strings.Join(e.Method, ","))
	set(URLKey, strings.Join(e.Path, ","))
	set(SecurityKey, e.Security)
	set(HostKey, strings.Join(e.Host, ","))
	set("stream", string(e.Stream))

	return ep
}

// Decode decodes endpoint metadata into an endpoint
func Decode(e map[string]string) *Endpoint {
	if e == nil {
		return nil
	}

	return &Endpoint{
		Name:     e[EndpointKey],
		Handler:  e[HandlerKey],
		Method:   slice(e[MethodKey]),
		Path:     slice(e[URLKey]),
		Security: e[SecurityKey],
		Host:     slice(e[HostKey]),
		Stream:   e["stream"],
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

// WithEndpoint returns a server.HandlerOption with endpoint metadata set
//
// Usage:
//
//	proto.RegisterHandler(service.Server(), new(Handler), api.WithEndpoint(
//		&api.Endpoint{
//			Name: "Greeter.Hello",
//			Path: []string{"/greeter"},
//		},
//	))
func WithEndpoint(e *Endpoint) server.HandlerOption {
	return server.EndpointMetadata(e.Name, Encode(e))
}
