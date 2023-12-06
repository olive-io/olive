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

package olivepb

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
)

// InternalRaftStringer implements custom proto Stringer:
// redact password, replace value fields with value_size fields.
type InternalRaftStringer struct {
	Request *RaftInternalRequest
}

func (as *InternalRaftStringer) String() string {
	switch {
	case as.Request.Put != nil:
		return fmt.Sprintf("put:<%s>",
			NewLoggablePutRequest(as.Request.Put).String(),
		)
	default:
		// nothing to redact
	}
	return as.Request.String()
}

// loggablePutRequest implements a custom proto String to replace value bytes field with a value
// size field.
// To preserve proto encoding of the key bytes, a faked out proto type is used here.
type loggablePutRequest struct {
	Key       []byte `protobuf:"bytes,1,opt,name=key,proto3"`
	ValueSize int64  `protobuf:"varint,2,opt,name=value_size,proto3"`
}

func NewLoggablePutRequest(request *RegionPutRequest) *loggablePutRequest {
	return &loggablePutRequest{
		request.Key,
		int64(len(request.Value)),
	}
}

func (m *loggablePutRequest) Reset()         { *m = loggablePutRequest{} }
func (m *loggablePutRequest) String() string { return proto.CompactTextString(m) }
func (*loggablePutRequest) ProtoMessage()    {}
