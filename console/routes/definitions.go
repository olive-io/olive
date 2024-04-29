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

	"github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/tonic"
	"github.com/olive-io/olive/pkg/tonic/fizz"
)

type DefinitionsGroup struct {
	lg  *zap.Logger
	oct *client.Client
}

func (tree *RouteTree) registerDefinitionsGroup() error {
	dg := &DefinitionsGroup{lg: tree.lg, oct: tree.oct}
	summary := dg.Summary()

	group := tree.root.Group("/bpmn/definitions", summary.Name, summary.Description, dg.HandlerChains()...)
	group.GET("/list", []fizz.OperationOption{
		fizz.Summary("List all definitions in olive system."),
	}, tonic.Handler(dg.definitionsList, 200))

	group.POST("/deploy", []fizz.OperationOption{
		fizz.Summary("Deploy a new definition in olive system."),
	}, tonic.Handler(dg.definitionDeploy, 200))

	group.POST("/get", []fizz.OperationOption{
		fizz.Summary("Get the definition in olive system by id and version."),
	}, tonic.Handler(dg.definitionGet, 200))

	group.POST("/remove", []fizz.OperationOption{
		fizz.Summary("Remove the definition in olive system."),
	}, tonic.Handler(dg.definitionRemove, 200))

	group.POST("/execute", []fizz.OperationOption{
		fizz.Summary("Execute bpmn definitions and starting a new process instance."),
	}, tonic.Handler(dg.definitionExecute, 200))

	group.POST("/process/list", []fizz.OperationOption{
		fizz.Summary("List process instances in the given definition."),
	}, tonic.Handler(dg.processList, 200))

	group.POST("/process/get", []fizz.OperationOption{
		fizz.Summary("Get the process instance information in the given definition."),
	}, tonic.Handler(dg.processGet, 200))

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
	resp, err := dg.oct.ListDefinitions(ctx, in.Limit, in.Continue)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type DefinitionDeployRequest struct {
	Id   string `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
	Body string `json:"body" validate:"required"`
}

type DefinitionDeployResponse = olivepb.DeployDefinitionResponse

func (dg *DefinitionsGroup) definitionDeploy(ctx *gin.Context, in *DefinitionDeployRequest) (*DefinitionDeployResponse, error) {
	resp, err := dg.oct.DeployDefinition(ctx, in.Id, in.Name, []byte(in.Body))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type DefinitionGetRequest struct {
	Id      string `json:"id" validate:"required"`
	Version uint64 `json:"version"`
}

type DefinitionGetResponse = olivepb.GetDefinitionResponse

func (dg *DefinitionsGroup) definitionGet(ctx *gin.Context, in *DefinitionGetRequest) (*DefinitionGetResponse, error) {
	resp, err := dg.oct.GetDefinition(ctx, in.Id, in.Version)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type DefinitionRemoveRequest struct {
	Id      string `json:"id" validate:"required"`
	Version uint64 `json:"version"`
}

type DefinitionRemoveResponse = olivepb.RemoveDefinitionResponse

func (dg *DefinitionsGroup) definitionRemove(ctx *gin.Context, in *DefinitionRemoveRequest) (*DefinitionRemoveResponse, error) {
	resp, err := dg.oct.RemoveDefinition(ctx, in.Id)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type DefinitionExecuteRequest struct {
	Id         string            `json:"id" validate:"required"`
	Version    uint64            `json:"version"`
	Name       string            `json:"name"`
	Headers    map[string]string `json:"headers"`
	Properties map[string]any    `json:"properties"`
}

type DefinitionExecuteResponse = olivepb.ExecuteDefinitionResponse

func (dg *DefinitionsGroup) definitionExecute(ctx *gin.Context, in *DefinitionExecuteRequest) (*DefinitionExecuteResponse, error) {
	options := make([]client.ExecDefinitionOption, 0)
	if in.Version != 0 {
		options = append(options, client.WithVersion(in.Version))
	}
	if in.Name != "" {
		options = append(options, client.WithName(in.Name))
	}
	if in.Headers != nil {
		options = append(options, client.WithHeaders(in.Headers))
	}
	if in.Properties != nil {
		options = append(options, client.WithProperties(in.Properties))
	}

	resp, err := dg.oct.ExecuteDefinition(ctx, in.Id, options...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type ProcessListRequest struct {
	DefinitionId      string `json:"definitionId" validate:"required"`
	DefinitionVersion uint64 `json:"definitionVersion"`
	Id                string `json:"id"`
}

type ProcessListResponse = olivepb.ListProcessInstancesResponse

func (dg *DefinitionsGroup) processList(ctx *gin.Context, in *ProcessListRequest) (*ProcessListResponse, error) {
	resp, err := dg.oct.ListProcessInstances(ctx, in.DefinitionId, in.DefinitionVersion, in.Id)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type ProcessGetRequest struct {
	DefinitionId      string `json:"definitionId" validate:"required"`
	DefinitionVersion uint64 `json:"definitionVersion" validate:"required"`
	Id                string `json:"id" validate:"required"`
}

type ProcessGetResponse = olivepb.GetProcessInstanceResponse

func (dg *DefinitionsGroup) processGet(ctx *gin.Context, in *ProcessGetRequest) (*ProcessGetResponse, error) {
	resp, err := dg.oct.GetProcessInstance(ctx, in.DefinitionId, in.DefinitionVersion, in.Id)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
