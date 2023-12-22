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
	"errors"
	"net/http"
	"strings"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

// the Resolver implementation for ServiceTask
type serviceResolver struct{}

func (r *serviceResolver) Activity() dsypb.Activity {
	return dsypb.Activity_ServiceTask
}

func (r *serviceResolver) Resolve(req *http.Request) (*Endpoint, error) {
	// /foo.Bar/Service
	if req.URL.Path == "/" {
		return nil, errors.New("unknown name")
	}
	// [foo.Bar, Service]
	parts := strings.Split(req.URL.Path[1:], "/")
	// [foo, Bar]
	names := strings.Split(parts[0], ".")
	var ename string
	if len(names) > 3 {
		ename = strings.Join(names[:len(names)-2], ".")
	} else if len(names) > 2 {
		ename = strings.Join(names[:len(names)-1], ".")
	} else {
		ename = strings.Join(names, ".")
	}

	// foo
	endpoint := &Endpoint{
		Name:   ename,
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}

	return endpoint, nil
}
