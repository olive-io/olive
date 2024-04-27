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
	"github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/tonic"
	"github.com/olive-io/olive/pkg/tonic/fizz"
	"go.uber.org/zap"
)

type DefinitionsGroup struct {
	lg  *zap.Logger
	oct *client.Client
}

func (tree *RouteTree) registerDefinitionsGroup() error {
	dg := &DefinitionsGroup{lg: tree.lg, oct: tree.oct}
	summary := dg.Summary()

	group := tree.root.Group("/definitions", summary.Name, summary.Description, dg.HandlerChains()...)
	group.GET("", []fizz.OperationOption{
		fizz.Summary("List all the definitions in olive cluster."),
	}, tonic.Handler(dg.definitionsList, 200))

	return tree.Group(dg)
}

func (dg *DefinitionsGroup) Summary() RouteGroupSummary {
	return RouteGroupSummary{
		Name:        "olive.Definitions",
		Description: "the documents of olive bpmn definitions",
	}
}

func (dg *DefinitionsGroup) HandlerChains() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}

type DefinitionsListRequest struct {
	Limit    int64  `query:"limit"`
	Continue string `query:"continue"`
}

type DefinitionsListResponse = olivepb.ListDefinitionResponse

func (dg *DefinitionsGroup) definitionsList(ctx *gin.Context, in *DefinitionsListRequest) (*DefinitionsListResponse, error) {
	options := make([]client.ListDefinitionOption, 0)
	if in.Limit != 0 {
		options = append(options, client.WithLimit(in.Limit))
	}
	if in.Continue != "" {
		options = append(options, client.WithContinue(in.Continue))
	}
	definitions, token, err := dg.oct.ListDefinitions(ctx, options...)
	if err != nil {
		return nil, err
	}
	resp := &DefinitionsListResponse{
		Header:        nil,
		Definitions:   definitions,
		ContinueToken: token,
	}
	return resp, nil
}
