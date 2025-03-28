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

package types

import (
	"time"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/protobuf/proto"
)

func PbToHeader(header *etcdserverpb.ResponseHeader) *ResponseHeader {
	return &ResponseHeader{
		ClusterID: header.ClusterId,
		MemberID:  header.MemberId,
		Revision:  header.Revision,
		RaftTerm:  header.RaftTerm,
	}
}

func PbToMember(member *etcdserverpb.Member) *Member {
	return &Member{
		Id:         member.ID,
		Name:       member.Name,
		PeerUrls:   member.PeerURLs,
		ClientUrls: member.ClientURLs,
		IsLeader:   member.IsLearner,
	}
}

func (x *Process) ToSnapshot() *ProcessSnapshot {
	snapshot := &ProcessSnapshot{
		Id:       x.Id,
		Priority: x.Priority,
		Status:   x.Status,
	}
	return snapshot
}

// Finished returns true if Status equals ProcessStatus_Ok, ProcessStatus_Failed.
func (x *Process) Finished() bool {
	switch x.Status {
	case ProcessStatus_Ok,
		ProcessStatus_Failed:
		return true
	default:
		return false
	}
}

// Executed returns true if Status equals ProcessStatus_Running, ProcessStatus_Ok, ProcessStatus_Failed.
func (x *Process) Executed() bool {
	switch x.Status {
	case ProcessStatus_Running,
		ProcessStatus_Ok,
		ProcessStatus_Failed:
		return true
	default:
		return false
	}
}

func (x *ProcessSnapshot) ExecuteExpired(interval time.Duration) bool {
	if x.ReadyAt == 0 {
		return false
	}
	deadline := x.ReadyAt + interval.Nanoseconds()
	return deadline < time.Now().UnixNano()
}

func EventFromKV(kv *mvccpb.KeyValue) (*RunnerEvent, error) {
	var x RunnerEvent
	if err := proto.Unmarshal(kv.Value, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func DefinitionFromKV(kv *mvccpb.KeyValue) (*Definition, error) {
	var x Definition
	if err := proto.Unmarshal(kv.Value, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func ProcessFromKV(kv *mvccpb.KeyValue) (*Process, error) {
	var x Process
	if err := proto.Unmarshal(kv.Value, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func ProcessSnapshotFromKV(kv *mvccpb.KeyValue) (*ProcessSnapshot, error) {
	var x ProcessSnapshot
	if err := proto.Unmarshal(kv.Value, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func RunnerFromKV(kv *mvccpb.KeyValue) (*Runner, error) {
	var x Runner
	if err := proto.Unmarshal(kv.Value, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func StatFromKV(kv *mvccpb.KeyValue) (*RunnerStat, error) {
	var x RunnerStat
	if err := proto.Unmarshal(kv.Value, &x); err != nil {
		return nil, err
	}
	return &x, nil
}
