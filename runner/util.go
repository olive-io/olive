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

package runner

import (
	"context"
	"fmt"
	"path"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.uber.org/zap"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/runtime"
)

func parseRegionKV(kv *mvccpb.KeyValue, runnerId uint64) (*pb.Region, bool, error) {
	region := new(pb.Region)
	err := region.Unmarshal(kv.Value)
	if err != nil {
		return nil, false, err
	}
	match := false
	for _, replica := range region.Replicas {
		if replica.Runner == runnerId {
			match = true
			break
		}
	}
	return region, match, nil
}

func parseDefinitionKV(kv *mvccpb.KeyValue) (*pb.Definition, bool, error) {
	definition := new(pb.Definition)
	err := definition.Unmarshal(kv.Value)
	if err != nil {
		return nil, false, err
	}
	if definition.Header == nil || definition.Header.Region == 0 {
		return definition, false, nil
	}
	if definition.Header.Rev == 0 {
		definition.Header.Rev = kv.ModRevision
	}
	return definition, true, nil
}

func parseProcessInstanceKV(kv *mvccpb.KeyValue) (*pb.ProcessInstance, bool, error) {
	process := new(pb.ProcessInstance)
	err := process.Unmarshal(kv.Value)
	if err != nil {
		return nil, false, err
	}
	if process.Status != pb.ProcessInstance_Waiting ||
		process.DefinitionId == "" ||
		process.Header == nil ||
		process.Header.Region == 0 {
		return process, false, nil
	}
	if process.Header.Rev == 0 {
		process.Header.Rev = kv.ModRevision
	}
	return process, true, nil
}

func commitProcessInstance(ctx context.Context, lg *zap.Logger, client *client.Client, process *pb.ProcessInstance) {
	if lg == nil {
		lg = zap.NewNop()
	}

	if process.Status == pb.ProcessInstance_Waiting {
		process.Status = pb.ProcessInstance_Prepare
	}
	key := path.Join(runtime.DefaultRunnerProcessInstance,
		process.DefinitionId, fmt.Sprintf("%d", process.DefinitionVersion),
		fmt.Sprintf("%d", process.Id))
	data, _ := process.Marshal()
	_, err := client.Put(ctx, key, string(data))
	if err != nil {
		lg.Error("update process instance",
			zap.String("definition", process.DefinitionId),
			zap.Uint64("version", process.DefinitionVersion),
			zap.Uint64("id", process.Id),
			zap.Error(err))
		return
	}
}
