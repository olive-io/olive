/*
Copyright 2025 The olive Authors

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

package server

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/olive-io/bpmn/schema"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/server/dao"
	"github.com/olive-io/olive/server/scheduler"
)

var _ pb.BpmnRPCServer = (*bpmnGRPCServer)(nil)

type bpmnGRPCServer struct {
	pb.UnimplementedBpmnRPCServer

	ctx context.Context
	lg  *zap.Logger

	sch *scheduler.Scheduler

	definitionsDao *dao.DefinitionsDao
	processDao     *dao.ProcessDao
}

func newBpmnServer(ctx context.Context, lg *zap.Logger, sch *scheduler.Scheduler, definitionsDao *dao.DefinitionsDao, processDao *dao.ProcessDao) *bpmnGRPCServer {
	wch := sch.Watch(ctx, "system")
	server := &bpmnGRPCServer{
		ctx:            ctx,
		lg:             lg,
		sch:            sch,
		definitionsDao: definitionsDao,
		processDao:     processDao,
	}
	go server.process(wch)
	return server
}

func (bgs *bpmnGRPCServer) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionsRequest) (*pb.DeployDefinitionsResponse, error) {
	bpmnDef, err := schema.Parse(req.Content)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	idPtr, _ := bpmnDef.Id()
	if idPtr == nil || *idPtr == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Definitions id is required"))
	}
	uid := *idPtr

	executed := false
	for i := range *bpmnDef.Processes() {
		process := (*bpmnDef.Processes())[i]
		executePtr, _ := process.IsExecutable()
		executed = executePtr
		if executed {
			break
		}
	}
	if !executed {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("no executable process found"))
	}

	definitions, err := bgs.definitionsDao.GetDefinitions(ctx, 0, uid)
	if err != nil {
		if !dao.IsNotFound(err) {
			return nil, status.Error(codes.Internal, err.Error())
		}
		definitions = &types.Definitions{
			Uid:         uid,
			Description: req.Description,
			Metadata:    req.Metadata,
			Content:     string(req.Content),
			Version:     1,
			IsExecute:   true,
		}
		id, err := bgs.definitionsDao.CreateDefinitions(ctx, definitions)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if definitions.Id == 0 {
			definitions.Id = id
		}

		bgs.lg.Info("deploy definition",
			zap.String("uid", uid),
			zap.Uint64("version", definitions.Version),
		)

		return &pb.DeployDefinitionsResponse{Definitions: definitions}, nil
	}

	if req.Metadata != nil {
		definitions.Metadata = req.Metadata
	}
	if req.Description != "" {
		definitions.Description = req.Description
	}
	definitions.Content = string(req.Content)

	bgs.lg.Info("add definitions snapshot",
		zap.String("uid", uid),
		zap.Uint64("version", definitions.Version))

	if err = bgs.definitionsDao.AddSnapshots(ctx, definitions); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	bgs.lg.Info("deploy definition",
		zap.String("uid", uid),
		zap.Uint64("version", definitions.Version),
	)

	rsp := &pb.DeployDefinitionsResponse{Definitions: definitions}
	return rsp, nil
}

func (bgs *bpmnGRPCServer) ListDefinitions(ctx context.Context, req *pb.ListDefinitionsRequest) (*pb.ListDefinitionsResponse, error) {
	page, size := req.Page, req.Size
	list, total, err := bgs.definitionsDao.ListDefinitions(ctx, page, size)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rsp := &pb.ListDefinitionsResponse{
		DefinitionsList: list,
		Total:           total,
	}
	return rsp, nil
}

func (bgs *bpmnGRPCServer) GetDefinitions(ctx context.Context, req *pb.GetDefinitionsRequest) (*pb.GetDefinitionsResponse, error) {
	definitions, err := bgs.getDefinitionsWithVersion(ctx, 0, req.Uid, req.Version)
	if err != nil {
		return nil, err
	}

	rsp := &pb.GetDefinitionsResponse{
		Definitions: definitions,
	}
	return rsp, nil
}

func (bgs *bpmnGRPCServer) getDefinitionsWithVersion(ctx context.Context, id int64, uid string, version uint64) (*types.Definitions, error) {
	definitions, err := bgs.definitionsDao.GetDefinitions(ctx, id, uid)
	if err != nil {
		if dao.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if version != 0 && definitions.Version != version {
		snapshot, err := bgs.definitionsDao.GetSnapshot(ctx, definitions.Uid, version)
		if err != nil {
			if dao.IsNotFound(err) {
				return nil, status.Error(codes.NotFound, err.Error())
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		definitions.UpdateAt = snapshot.CreateAt
		definitions.Metadata = snapshot.Metadata
		definitions.Description = snapshot.Description
		definitions.Content = snapshot.Content
		definitions.Version = snapshot.Version
	}
	return definitions, nil
}

func (bgs *bpmnGRPCServer) GetDefinitionsSnapshots(ctx context.Context, req *pb.GetDefinitionsSnapshotsRequest) (*pb.GetDefinitionsSnapshotsResponse, error) {
	snapshots, total, err := bgs.definitionsDao.GetDefinitionsSnapshots(ctx, req.Uid, req.Page, req.Size)
	if err != nil {
		if dao.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	rsp := &pb.GetDefinitionsSnapshotsResponse{
		Snapshots: snapshots,
		Total:     total,
	}
	return rsp, nil
}

func (bgs *bpmnGRPCServer) RemoveDefinitions(ctx context.Context, req *pb.RemoveDefinitionsRequest) (*pb.RemoveDefinitionsResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (bgs *bpmnGRPCServer) ExecuteProcess(ctx context.Context, req *pb.ExecuteProcessRequest) (*pb.ExecuteProcessResponse, error) {
	definitionId := req.DefinitionsId
	version := req.DefinitionsVersion
	definitions, err := bgs.getDefinitionsWithVersion(ctx, definitionId, "", version)
	if err != nil {
		return nil, err
	}

	process := &types.Process{
		Name:     req.Name,
		Uid:      uuid.New().String(),
		Metadata: map[string]string{},
		Priority: req.Priority,
		Args: &types.BpmnArgs{
			Headers:     req.Headers,
			Properties:  req.Properties,
			DataObjects: req.DataObjects,
		},
		DefinitionsId:      definitionId,
		DefinitionsVersion: version,
		Context: &types.ProcessContext{
			Variables:   map[string]string{},
			DataObjects: map[string]string{},
		},
	}

	if err = bgs.processDao.CreateProcess(ctx, process); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	processStat := &scheduler.ProcessStat{
		Process:     process,
		Definitions: definitions.Content,
		FlowNodes:   make([]*types.FlowNode, 0),
	}

	err = bgs.sch.Execute(processStat)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ExecuteProcessResponse{Process: process}, nil
}

func (bgs *bpmnGRPCServer) ListProcess(ctx context.Context, req *pb.ListProcessRequest) (*pb.ListProcessResponse, error) {
	page, size := req.Page, req.Size
	options := dao.NewListProcessOptions(req.DefinitionsId, req.DefinitionsVersion)
	processes, total, err := bgs.processDao.ListProcesses(ctx, page, size, options)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rsp := &pb.ListProcessResponse{
		Processes: processes,
		Total:     total,
	}

	return rsp, nil
}

func (bgs *bpmnGRPCServer) GetProcess(ctx context.Context, req *pb.GetProcessRequest) (*pb.GetProcessResponse, error) {
	process, err := bgs.processDao.GetProcess(ctx, req.Id)
	if err != nil {
		if dao.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	nodes, err := bgs.processDao.ListFlowNodes(ctx, process.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rsp := &pb.GetProcessResponse{
		Process:    process,
		Activities: nodes,
	}
	return rsp, nil
}

func (bgs *bpmnGRPCServer) process(wch *scheduler.WatchChan) {
	lg := bgs.lg
	ctx := bgs.ctx
	for {
		select {
		case <-ctx.Done():
			wch.Close()
			return
		default:
		}

		rsp := wch.Next()
		if rsp.Err != nil {

		}
		if p := rsp.Process; p != nil {
			err := bgs.processDao.UpdateProcess(ctx, p)
			if err != nil {
				lg.Error("update process",
					zap.Int64("id", p.Id),
					zap.Int64("definitions", p.DefinitionsId),
					zap.Uint64("version", p.DefinitionsVersion),
					zap.Error(err))
			}
		}
		if node := rsp.FlowNode; node != nil {
			err := bgs.processDao.SaveFlowNode(ctx, node)
			if err != nil {
				lg.Error("save flow node",
					zap.Int64("process", node.ProcessId),
					zap.String("flow", node.Name),
					zap.Error(err))
			}
		}
	}
}
