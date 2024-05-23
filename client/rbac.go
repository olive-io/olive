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
	"context"

	"google.golang.org/grpc"

	authv1 "github.com/olive-io/olive/apis/pb/auth"
	pb "github.com/olive-io/olive/apis/pb/olive"
)

type (
	ListPolicyResponse        pb.ListPolicyResponse
	AddPolicyResponse         pb.AddPolicyResponse
	RemovePolicyResponse      pb.RemovePolicyResponse
	AddGroupPolicyResponse    pb.AddGroupPolicyResponse
	RemoveGroupPolicyResponse pb.RemoveGroupPolicyResponse
	AdmitResponse             pb.AdmitResponse
)

type RbacRPC interface {
	ListPolicy(ctx context.Context, ptype authv1.PType, sub string) (*ListPolicyResponse, error)
	AddPolicy(ctx context.Context, policy *authv1.Policy) (*AddPolicyResponse, error)
	RemovePolicy(ctx context.Context, policy *authv1.Policy) (*RemovePolicyResponse, error)
	AddGroupPolicy(ctx context.Context, policy *authv1.Policy) (*AddGroupPolicyResponse, error)
	RemoveGroupPolicy(ctx context.Context, policy *authv1.Policy) (*RemoveGroupPolicyResponse, error)
	Admit(ctx context.Context, policy *authv1.Policy) (*AdmitResponse, error)
}

type rbacRPC struct {
	client   *Client
	callOpts []grpc.CallOption
}

func NewRbacRPC(c *Client) RbacRPC {
	api := &rbacRPC{client: c}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rc *rbacRPC) ListPolicy(ctx context.Context, ptype authv1.PType, sub string) (*ListPolicyResponse, error) {
	conn := rc.client.conn
	in := pb.ListPolicyRequest{
		Ptyle: ptype,
		Sub:   sub,
	}
	rsp, err := rc.remoteClient(conn).ListPolicy(ctx, &in, rc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*ListPolicyResponse)(rsp), nil
}

func (rc *rbacRPC) AddPolicy(ctx context.Context, policy *authv1.Policy) (*AddPolicyResponse, error) {
	conn := rc.client.conn
	leaderEndpoints, err := rc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = rc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}
	in := pb.AddPolicyRequest{
		Policy: policy,
	}
	rsp, err := rc.remoteClient(conn).AddPolicy(ctx, &in, rc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*AddPolicyResponse)(rsp), nil
}

func (rc *rbacRPC) RemovePolicy(ctx context.Context, policy *authv1.Policy) (*RemovePolicyResponse, error) {
	conn := rc.client.conn
	leaderEndpoints, err := rc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = rc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.RemovePolicyRequest{
		Policy: policy,
	}
	resp, err := rc.remoteClient(conn).RemovePolicy(ctx, in, rc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*RemovePolicyResponse)(resp), nil
}

func (rc *rbacRPC) AddGroupPolicy(ctx context.Context, policy *authv1.Policy) (*AddGroupPolicyResponse, error) {
	conn := rc.client.conn
	leaderEndpoints, err := rc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = rc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := pb.AddGroupPolicyRequest{
		Policy: policy,
	}
	rsp, err := rc.remoteClient(conn).AddGroupPolicy(ctx, &in, rc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*AddGroupPolicyResponse)(rsp), nil
}

func (rc *rbacRPC) RemoveGroupPolicy(ctx context.Context, policy *authv1.Policy) (*RemoveGroupPolicyResponse, error) {
	conn := rc.client.conn
	leaderEndpoints, err := rc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = rc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.RemoveGroupPolicyRequest{Policy: policy}
	resp, err := rc.remoteClient(conn).RemoveGroupPolicy(ctx, in, rc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*RemoveGroupPolicyResponse)(resp), nil
}

func (rc *rbacRPC) Admit(ctx context.Context, policy *authv1.Policy) (*AdmitResponse, error) {
	conn := rc.client.conn
	in := &pb.AdmitRequest{
		Policy: policy,
	}
	rsp, err := rc.remoteClient(conn).Admit(ctx, in, rc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*AdmitResponse)(rsp), nil
}

func (rc *rbacRPC) remoteClient(conn *grpc.ClientConn) pb.RbacRPCClient {
	return RetryRbacClient(conn)
}
