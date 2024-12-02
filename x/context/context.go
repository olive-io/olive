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

package context

import (
	"context"
	"net/http"
	"net/textproto"
	"strings"

	cxmd "github.com/olive-io/olive/x/context/metadata"
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
