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

package grpc

import (
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"

	"github.com/olive-io/olive/pkg/proxy/codec"
	"github.com/olive-io/olive/pkg/proxy/codec/bytes"
)

type response struct {
	conn   *grpc.ClientConn
	stream grpc.ClientStream
	codec  encoding.Codec
	gcodec codec.Codec
}

// Codec reads the response
func (r *response) Codec() codec.Reader {
	return r.gcodec
}

// Header reads the header
func (r *response) Header() map[string]string {
	md, err := r.stream.Header()
	if err != nil {
		return map[string]string{}
	}
	hdr := make(map[string]string, len(md))
	for k, v := range md {
		hdr[k] = strings.Join(v, ",")
	}
	return hdr
}

// Read the undecoded response
func (r *response) Read() ([]byte, error) {
	f := &bytes.Frame{}
	if err := r.gcodec.ReadBody(f); err != nil {
		return nil, err
	}
	return f.Data, nil
}
