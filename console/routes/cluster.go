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

package routes

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/tonic"
	"github.com/olive-io/olive/pkg/tonic/fizz"
	"github.com/olive-io/olive/pkg/tonic/openapi"
)

type ClusterGroup struct {
	lg  *zap.Logger
	oct *client.Client
}

func (tree *RouteTree) registerCluster() error {
	cg := &ClusterGroup{lg: tree.lg, oct: tree.oct}
	summary := cg.Summary()

	group := tree.root.Group("/cluster", summary.Name, summary.Description, cg.HandlerChains()...)

	group.GET("/members", []fizz.OperationOption{
		fizz.Summary("List the members of the olive cluster meta components."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(cg.memberList, 200))

	group.POST("/member/add", []fizz.OperationOption{
		fizz.Summary("adds a new member into the olive-meta cluster."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(cg.memberAdd, 200))

	group.PATCH("/member/update", []fizz.OperationOption{
		fizz.Summary("updates the peer addresses of the olive-meta member."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(cg.memberUpdate, 200))

	group.POST("/member/remove", []fizz.OperationOption{
		fizz.Summary("removes a member from olive-meta cluster."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(cg.memberRemove, 200))

	return tree.Group(cg)
}

func (c *ClusterGroup) Summary() RouteGroupSummary {
	return RouteGroupSummary{
		Name:        "Olive.MetaCluster",
		Description: "the documents of olive-meta cluster",
	}
}

func (c *ClusterGroup) HandlerChains() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}

type MemberListResponse = client.MemberListResponse

func (c *ClusterGroup) memberList(ctx *gin.Context) (*MemberListResponse, error) {
	resp, err := c.oct.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type MemberAddRequest struct {
	Peers    []string `json:"peers" validate:"required"`
	IsLeader bool     `json:"is_leader"`
}

type MemberAddResponse = client.MemberAddResponse

func (c *ClusterGroup) memberAdd(ctx *gin.Context, in *MemberAddRequest) (*MemberAddResponse, error) {
	var resp *client.MemberAddResponse
	var err error
	if in.IsLeader {
		resp, err = c.oct.MemberAddAsLearner(ctx, in.Peers)
	} else {
		resp, err = c.oct.MemberAdd(ctx, in.Peers)
	}
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type MemberUpdateRequest struct {
	Id    uint64   `json:"id" validate:"required"`
	Peers []string `json:"peers" validate:"required"`
}

type MemberUpdateResponse = client.MemberUpdateResponse

func (c *ClusterGroup) memberUpdate(ctx *gin.Context, in *MemberUpdateRequest) (*MemberUpdateResponse, error) {
	resp, err := c.oct.MemberUpdate(ctx, in.Id, in.Peers)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type RemoveMemberRequest struct {
	Id uint64 `json:"id" validate:"required"`
}

type MemberRemoveResponse = client.MemberRemoveResponse

func (c *ClusterGroup) memberRemove(ctx *gin.Context, in *RemoveMemberRequest) (*MemberRemoveResponse, error) {
	resp, err := c.oct.MemberRemove(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
