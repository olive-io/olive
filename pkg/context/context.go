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

package context

import (
	"context"
	"net/http"
	"net/textproto"
	"strings"

	cxmd "github.com/olive-io/olive/pkg/context/metadata"
)

func FromRequest(r *http.Request) context.Context {
	md, ok := cxmd.FromContext(r.Context())
	if !ok {
		md = make(cxmd.Metadata)
	}
	for key, values := range r.Header {
		switch key {
		case "Connection", "Content-Length":
			continue
		}
		md.Set(textproto.CanonicalMIMEHeaderKey(key), strings.Join(values, ","))
	}
	if v, ok := md.Get("X-Forwarded-For"); ok {
		md["X-Forwarded-For"] = v + ", " + r.RemoteAddr
	} else {
		md["X-Forwarded-For"] = r.RemoteAddr
	}
	if _, ok = md.Get("Host"); !ok {
		md.Set("Host", r.Host)
	}
	md.Set("Method", r.Method)
	if _, ok = md.Get("X-Olive-Path"); !ok {
		if r.URL != nil {
			md.Set("X-Olive-Path", r.URL.Path)
		}
	}
	return cxmd.NewContext(r.Context(), md)
}
