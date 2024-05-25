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