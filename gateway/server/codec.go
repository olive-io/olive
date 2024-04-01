// Copyright 2024 The olive Authors
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

package server

import (
	gbytes "bytes"
	"encoding/json"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/encoding"
)

func init() {
	encoding.RegisterCodec(newJSONCodec())
}

type jsonCodec struct {
	jsonpb.Marshaler
	jsonpb.Unmarshaler
}

func newJSONCodec() *jsonCodec {
	return &jsonCodec{
		Marshaler: jsonpb.Marshaler{
			EmitDefaults: true,
			OrigName:     true,
		},
		Unmarshaler: jsonpb.Unmarshaler{},
	}
}

func (c jsonCodec) Name() string {
	return "json"
}

func (c jsonCodec) Marshal(v interface{}) (out []byte, err error) {
	if pm, ok := v.(proto.Message); ok {
		b := new(gbytes.Buffer)
		err := c.Marshaler.Marshal(b, pm)
		if err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}
	return json.Marshal(v)
}

func (c jsonCodec) Unmarshal(data []byte, v interface{}) (err error) {
	if pm, ok := v.(proto.Message); ok {
		b := gbytes.NewBuffer(data)
		return c.Unmarshaler.Unmarshal(b, pm)
	}
	return json.Unmarshal(data, v)
}
