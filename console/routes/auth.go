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

	authv1 "github.com/olive-io/olive/apis/pb/auth"
	"github.com/olive-io/olive/pkg/tonic"
	"github.com/olive-io/olive/pkg/tonic/fizz"
	"github.com/olive-io/olive/pkg/tonic/openapi"

	"github.com/olive-io/olive/client"
)

type AuthGroup struct {
	lg  *zap.Logger
	oct *client.Client
}

func (tree *RouteTree) registerAuthGroup() error {
	ag := &AuthGroup{lg: tree.lg, oct: tree.oct}
	summary := ag.Summary()

	group := tree.root.Group("/auth", summary.Name, summary.Description, ag.HandlerChains()...)
	group.GET("/role/list", []fizz.OperationOption{
		fizz.Summary("List all roles in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.roleList, 200))
	group.POST("/role/get", []fizz.OperationOption{
		fizz.Summary("Get the role by name in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.roleGet, 200))
	group.POST("/role/add", []fizz.OperationOption{
		fizz.Summary("Adds the role to olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.roleAdd, 200))
	group.POST("/role/update", []fizz.OperationOption{
		fizz.Summary("Updates the role in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.roleUpdate, 200))
	group.POST("/role/remove", []fizz.OperationOption{
		fizz.Summary("Removes the role in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.roleRemove, 200))
	group.GET("/user/list", []fizz.OperationOption{
		fizz.Summary("List all users in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.userList, 200))
	group.POST("/user/get", []fizz.OperationOption{
		fizz.Summary("Get the user by name in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.userGet, 200))
	group.POST("/user/add", []fizz.OperationOption{
		fizz.Summary("Adds the user to olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.userAdd, 200))
	group.POST("/user/update", []fizz.OperationOption{
		fizz.Summary("Updates the user in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.userUpdate, 200))
	group.POST("/user/remove", []fizz.OperationOption{
		fizz.Summary("Removes the user in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(ag.userRemove, 200))

	return tree.Group(ag)
}

func (ag *AuthGroup) Summary() RouteGroupSummary {
	return RouteGroupSummary{
		Name:        "olive.Auth",
		Description: "the documents of olive authentication.",
	}
}

func (ag *AuthGroup) HandlerChains() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}

type RoleListRequest struct {
	Limit    int64  `query:"limit"`
	Continue string `query:"continue"`
}

type RoleListResponse = client.ListRoleResponse

func (ag *AuthGroup) roleList(ctx *gin.Context, in *RoleListRequest) (*RoleListResponse, error) {
	var opts []client.PageOption
	if in.Limit > 0 {
		opts = append(opts, client.WithLimit(in.Limit))
	}
	if in.Continue != "" {
		opts = append(opts, client.WithToken(in.Continue))
	}

	resp, err := ag.oct.ListRole(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type RoleGetRequest struct {
	Name string `json:"name" binding:"required"`
}

type RoleGetResponse = client.GetRoleResponse

func (ag *AuthGroup) roleGet(ctx *gin.Context, in *RoleGetRequest) (*RoleGetResponse, error) {
	resp, err := ag.oct.GetRole(ctx, in.Name)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type RoleAddRequest struct {
	Role *authv1.Role `json:"role" binding:"required"`
}

type RoleAddResponse = client.CreateRoleResponse

func (ag *AuthGroup) roleAdd(ctx *gin.Context, in *RoleAddRequest) (*RoleAddResponse, error) {
	resp, err := ag.oct.CreateRole(ctx, in.Role)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type RoleUpdateRequest struct {
	Name string `json:"name" binding:"required"`

	*authv1.RolePatcher `json:",inline"`
}

type RoleUpdateResponse = client.UpdateRoleResponse

func (ag *AuthGroup) roleUpdate(ctx *gin.Context, in *RoleUpdateRequest) (*RoleUpdateResponse, error) {
	resp, err := ag.oct.UpdateRole(ctx, in.Name, in.RolePatcher)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type RoleRemoveRequest struct {
	Name string `json:"name" binding:"required"`
}

type RoleRemoveResponse = client.RemoveRoleResponse

func (ag *AuthGroup) roleRemove(ctx *gin.Context, in *RoleUpdateRequest) (*RoleRemoveResponse, error) {
	resp, err := ag.oct.RemoveRole(ctx, in.Name)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type UserListRequest struct {
	Limit    int64  `query:"limit"`
	Continue string `query:"continue"`
}

type UserListResponse = client.ListUserResponse

func (ag *AuthGroup) userList(ctx *gin.Context, in *RoleListRequest) (*UserListResponse, error) {
	var opts []client.PageOption
	if in.Limit > 0 {
		opts = append(opts, client.WithLimit(in.Limit))
	}
	if in.Continue != "" {
		opts = append(opts, client.WithToken(in.Continue))
	}

	resp, err := ag.oct.ListUser(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type UserGetRequest struct {
	Name string `json:"name" binding:"required"`
}

type UserGetResponse = client.GetUserResponse

func (ag *AuthGroup) userGet(ctx *gin.Context, in *UserGetRequest) (*UserGetResponse, error) {
	resp, err := ag.oct.GetUser(ctx, in.Name)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type UserAddRequest struct {
	User *authv1.User `json:"user" binding:"required"`
}

type UserAddResponse = client.CreateUserResponse

func (ag *AuthGroup) userAdd(ctx *gin.Context, in *UserAddRequest) (*UserAddResponse, error) {
	resp, err := ag.oct.CreateUser(ctx, in.User)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type UserUpdateRequest struct {
	Name string `json:"name" binding:"required"`

	*authv1.UserPatcher `json:",inline"`
}

type UserUpdateResponse = client.UpdateUserResponse

func (ag *AuthGroup) userUpdate(ctx *gin.Context, in *UserUpdateRequest) (*UserUpdateResponse, error) {
	resp, err := ag.oct.UpdateUser(ctx, in.Name, in.UserPatcher)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type UserRemoveRequest struct {
	Name string `json:"name" binding:"required"`
}

type UserRemoveResponse = client.RemoveUserResponse

func (ag *AuthGroup) userRemove(ctx *gin.Context, in *UserUpdateRequest) (*UserRemoveResponse, error) {
	resp, err := ag.oct.RemoveUser(ctx, in.Name)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
