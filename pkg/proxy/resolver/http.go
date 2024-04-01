// Copyright 2024 The olive Authors
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

	"github.com/olive-io/olive/pkg/proxy/api"
)

// httpResolver the Resolver implementation for IResolver
type httpResolver struct{}

func NewHTTPResolver() IResolver {
	return new(httpResolver)
}

func (r *httpResolver) Resolve(req *http.Request) (*Endpoint, error) {
	// /foo.Bar/Service
	path := req.URL.Path

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
		Handler: api.HTTPHandler,
	}

	return endpoint, nil
}
