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

package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/rpc/planepb"
	corev1 "github.com/olive-io/olive/api/types/core/v1"
)

type PlaneRPC interface {
	GetPlane(ctx context.Context) (*corev1.Plane, error)
	ListRunner(ctx context.Context) ([]*corev1.Runner, error)
	GetRunner(ctx context.Context, name string) (*corev1.Runner, error)
}

type planeRPC struct {
	remote   pb.PlaneRPCClient
	callOpts []grpc.CallOption
}

func NewPlaneRPC(c *Client) PlaneRPC {
	conn := c.ActiveConnection()
	api := &planeRPC{
		remote: RetryPlaneClient(conn),
	}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rpc *planeRPC) GetPlane(ctx context.Context) (*corev1.Plane, error) {
	in := &pb.GetPlaneRequest{}
	rsp, err := rpc.remote.GetPlane(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Plane, nil
}

func (rpc *planeRPC) ListRunner(ctx context.Context) ([]*corev1.Runner, error) {
	in := &pb.ListRunnerRequest{}
	rsp, err := rpc.remote.ListRunner(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Runners, nil
}

func (rpc *planeRPC) GetRunner(ctx context.Context, name string) (runner *corev1.Runner, err error) {
	//in := &pb.GetRunnerRequest{Id: id}
	//rsp, err := mc.remote.GetRunner(ctx, in, mc.callOpts...)
	//if err != nil {
	//	return nil, toErr(ctx, err)
	//}
	//return rsp.Runner, nil
	return
}
