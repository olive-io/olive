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
	"testing"

	"github.com/olive-io/olive/gateway/server/testdata"
	"github.com/stretchr/testify/assert"
)

func Test_extractOpenAPIDocs(t *testing.T) {
	docs := testdata.GetOpenAPIDocs(t)
	eps := extractOpenAPIDocs(docs)
	assert.Equal(t, len(eps), 13)
}

func Test_extractHandler(t *testing.T) {
	//handler := &gatewayRpc{}
	//
	//typ := reflect.TypeOf(handler)
	//hdlr := reflect.ValueOf(handler)
	//name := reflect.Indirect(hdlr).Type().Name()
	//
	//var endpoints []*pb.Endpoint
	//
	//for m := 0; m < typ.NumMethod(); m++ {
	//	if e := extractEndpoint(typ.Method(m)); e != nil {
	//		e.Name = name + "." + e.Name
	//
	//		endpoints = append(endpoints, e)
	//	}
	//}
}
