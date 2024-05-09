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

type RunnerGroup struct {
	lg  *zap.Logger
	oct *client.Client
}

func (tree *RouteTree) registerRunnerGroup() error {
	rg := &RunnerGroup{lg: tree.lg, oct: tree.oct}
	summary := rg.Summary()

	group := tree.root.Group("/meta/runner", summary.Name, summary.Description, rg.HandlerChains()...)
	group.GET("/list", []fizz.OperationOption{
		fizz.Summary("List all runners in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(rg.runnerList, 200))

	group.POST("/get", []fizz.OperationOption{
		fizz.Summary("Get the olive runner in olive system by id and version."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(rg.runnerGet, 200))

	return tree.Group(rg)
}

func (rg *RunnerGroup) Summary() RouteGroupSummary {
	return RouteGroupSummary{
		Name:        "olive.Runner",
		Description: "the documents of olive runner component",
	}
}

func (rg *RunnerGroup) HandlerChains() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}

type RunnerListRequest struct {
	Limit    int64  `query:"limit"`
	Continue string `query:"continue"`
}

type RunnerListResponse = client.ListRunnerResponse

func (rg *RunnerGroup) runnerList(ctx *gin.Context, in *RunnerListRequest) (*RunnerListResponse, error) {
	var opts []client.PageOption
	if in.Limit > 0 {
		opts = append(opts, client.WithLimit(in.Limit))
	}
	if in.Continue != "" {
		opts = append(opts, client.WithToken(in.Continue))
	}

	resp, err := rg.oct.ListRunner(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type RunnerGetRequest struct {
	Id uint64 `json:"id" validate:"required"`
}

type RunnerGetResponse = client.GetRunnerResponse

func (rg *RunnerGroup) runnerGet(ctx *gin.Context, in *RunnerGetRequest) (*RunnerGetResponse, error) {
	resp, err := rg.oct.GetRunner(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
