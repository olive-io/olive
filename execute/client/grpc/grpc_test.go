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

package grpc_test

import (
	"context"
	"testing"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/execute"
	"github.com/olive-io/olive/execute/client"
	"github.com/olive-io/olive/execute/client/grpc"
	"github.com/olive-io/olive/execute/client/selector"
	"github.com/olive-io/olive/pkg/discovery/memory"
	"github.com/stretchr/testify/assert"
)

func TestCall(t *testing.T) {
	discovery := memory.NewRegistry()
	so, err := selector.NewSelector(selector.Discovery(discovery))
	if !assert.NoError(t, err) {
		return
	}

	cc, err := grpc.NewClient(client.Selector(so), client.Discovery(discovery))
	if !assert.NoError(t, err) {
		return
	}

	body := &dsypb.ExecuteRequest{Headers: map[string]string{"a": "b"}}
	req := cc.NewRequest(execute.DefaultExecuteName, "/discoverypb.Executor/Execute", body)
	rsp := &dsypb.ExecuteResponse{}
	err = cc.Call(context.TODO(), req, rsp, client.WithAddress("127.0.0.1:15290"))
	if !assert.NoError(t, err) {
		return
	}

	t.Logf("%v", rsp)
}
