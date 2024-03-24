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

func NewLoggablePutRequest(request *RegionPutRequest) *LoggablePutRequest {
	return &LoggablePutRequest{
		Key:       request.Key,
		ValueSize: int64(len(request.Value)),
	}
}
