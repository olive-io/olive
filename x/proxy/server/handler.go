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

package server

import (
	"google.golang.org/grpc"
)

type HandlerOption func(*HandlerOptions)

type HandlerOptions struct {
	Internal    bool
	Metadata    map[string]map[string]string
	ServiceDesc *grpc.ServiceDesc
}

// EndpointMetadata is a Handler option that allows metadata to be added to
// individual endpoints.
func EndpointMetadata(name string, md map[string]string) HandlerOption {
	return func(opts *HandlerOptions) {
		if opts.Metadata == nil {
			opts.Metadata = map[string]map[string]string{}
		}
		opts.Metadata[name] = md
	}
}

func WithServerDesc(desc *grpc.ServiceDesc) HandlerOption {
	return func(opts *HandlerOptions) {
		opts.ServiceDesc = desc
	}
}
