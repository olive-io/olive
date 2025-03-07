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
	"go.etcd.io/etcd/api/v3/etcdserverpb"
)

func PbToHeader(header *etcdserverpb.ResponseHeader) *ResponseHeader {
	return &ResponseHeader{
		ClusterID: header.ClusterId,
		MemberID:  header.MemberId,
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

func (x *ProcessInstance) ToSnapshot() *ProcessSnapshot {
	snapshot := &ProcessSnapshot{
		Id:       x.Id,
		Priority: x.Priority,
		Status:   x.Status,
	}
	return snapshot
}
