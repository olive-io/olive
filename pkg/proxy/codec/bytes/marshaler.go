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

package bytes

import (
	"github.com/olive-io/olive/pkg/proxy/codec"
)

type Marshaler struct{}

type Message struct {
	Header map[string]string
	Body   []byte
}

func (n Marshaler) Marshal(v interface{}) ([]byte, error) {
	switch ve := v.(type) {
	case *[]byte:
		return *ve, nil
	case []byte:
		return ve, nil
	case *Message:
		return ve.Body, nil
	}
	return nil, codec.ErrInvalidMessage
}

func (n Marshaler) Unmarshal(d []byte, v interface{}) error {
	switch ve := v.(type) {
	case *[]byte:
		*ve = d
	case *Message:
		ve.Body = d
	}
	return codec.ErrInvalidMessage
}

func (n Marshaler) String() string {
	return "bytes"
}
