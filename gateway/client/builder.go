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

package client

import (
	dsypb "github.com/olive-io/olive/api/pb/discovery"
	"github.com/olive-io/olive/pkg/proxy/api"
)

type CBuilder struct {
	cs *dsypb.Consumer
}

func (c *Client) ConsumerBuilder() *CBuilder {
	builder := &CBuilder{
		cs: &dsypb.Consumer{
			Activity: dsypb.ActivityType_ServiceTask,
			Action:   api.HTTPHandler,
			Metadata: map[string]string{
				api.MethodKey: "POST",
			},
		},
	}
	return builder
}

func (b *CBuilder) Handle(method string, url string) *CBuilder {
	b.cs.Id = url
	b.cs.Metadata[api.MethodKey] = method
	return b
}

func (b *CBuilder) Metadata(key, value string) *CBuilder {
	b.cs.Metadata[key] = value
	return b
}

func (b *CBuilder) In(in any) *CBuilder {
	b.cs.Request = dsypb.BoxFromAny(in)
	return b
}

func (b *CBuilder) Out(out any) *CBuilder {
	b.cs.Response = dsypb.BoxFromAny(out)
	return b
}

func (b *CBuilder) Build() *dsypb.Consumer {
	return b.cs
}
