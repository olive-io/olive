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

package server

import (
	"context"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

func (e *Executor) Ping(ctx context.Context, req *dsypb.PingRequest) (*dsypb.PingResponse, error) {
	resp := &dsypb.PingResponse{}

	return resp, nil
}

func (e *Executor) Execute(ctx context.Context, req *dsypb.ExecuteRequest) (*dsypb.ExecuteResponse, error) {
	resp := &dsypb.ExecuteResponse{}
	resp.Response = &dsypb.Response{
		Properties: map[string]*dsypb.Box{
			"a": dsypb.BoxFromT("a"),
		},
		DataObjects: nil,
	}
	return resp, nil
}
