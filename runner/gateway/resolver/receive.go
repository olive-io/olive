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
	dsypb "github.com/olive-io/olive/api/discoverypb"
)

// the Resolver implementation for ReceiveTask
type receiveResolver struct{}

func (r *receiveResolver) Activity() dsypb.Activity {
	return dsypb.Activity_ReceiveTask
}

func (r *receiveResolver) Resolve(req *http.Request) (*Endpoint, error) {
	// /foo.Bar/Service
	if req.URL.Path == "/" {
		return nil, errors.New("unknown name")
	}
	// [foo.Bar, Service]
	parts := strings.Split(req.URL.Path[1:], "/")
	// [foo, Bar]
	name := strings.Split(parts[0], ".")
	// foo
	endpoint := &Endpoint{
		Name:   strings.Join(name[:len(name)-1], "."),
		Host:   req.Host,
		Method: req.Method,
		Path:   req.URL.Path,
	}

	return endpoint, nil
}
