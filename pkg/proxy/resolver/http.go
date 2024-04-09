/*
   Copyright 2024 The olive Authors

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
