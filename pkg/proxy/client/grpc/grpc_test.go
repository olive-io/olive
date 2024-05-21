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

package grpc_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	dsypb "github.com/olive-io/olive/apis/pb/gateway"

	"github.com/olive-io/olive/pkg/proxy/api"

	"github.com/olive-io/olive/pkg/discovery/memory"
	"github.com/olive-io/olive/pkg/proxy/client"
	"github.com/olive-io/olive/pkg/proxy/client/grpc"
	"github.com/olive-io/olive/pkg/proxy/client/selector"
)

func TestCall(t *testing.T) {
	discovery := memory.NewRegistrar()
	so, err := selector.NewSelector(selector.Discovery(discovery))
	if !assert.NoError(t, err) {
		return
	}

	cc, err := grpc.NewClient(client.Selector(so), client.Discovery(discovery))
	if !assert.NoError(t, err) {
		return
	}

	body := &dsypb.TransmitRequest{Headers: map[string]string{"a": "b"}}
	req := cc.NewRequest(api.DefaultService, "/discoverypb.Executor/Execute", body)
	rsp := &dsypb.TransmitResponse{}
	err = cc.Call(context.TODO(), req, rsp, client.WithAddress("127.0.0.1:15290"))
	if !assert.NoError(t, err) {
		return
	}

	t.Logf("%v", rsp)
}
