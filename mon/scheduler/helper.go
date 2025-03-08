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

package scheduler

import (
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api/types"
)

func parseRunnerKV(kv *mvccpb.KeyValue) (*types.Runner, error) {
	var runner types.Runner
	err := proto.Unmarshal(kv.Value, &runner)
	return &runner, err
}

func parseStatKV(kv *mvccpb.KeyValue) (*types.RunnerStat, error) {
	var stat types.RunnerStat
	err := proto.Unmarshal(kv.Value, &stat)
	return &stat, err
}

func parsePSnapKV(kv *mvccpb.KeyValue) (*types.ProcessSnapshot, error) {
	var ps types.ProcessSnapshot
	err := proto.Unmarshal(kv.Value, &ps)
	return &ps, err
}

func parseProcessKV(kv *mvccpb.KeyValue) (*types.ProcessInstance, error) {
	var pi types.ProcessInstance
	err := proto.Unmarshal(kv.Value, &pi)
	return &pi, err
}
