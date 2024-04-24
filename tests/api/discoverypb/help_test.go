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

package discoverypb

import (
	"reflect"
	"testing"

	"github.com/olive-io/bpmn/schema"
	dsypb "github.com/olive-io/olive/api/discoverypb"
)

type TestData struct {
	Name string `json:"name"`
}

func TestBoxFromAny(t *testing.T) {
	type args struct {
		value any
	}
	arr := []int32{1, 2}
	t1 := TestData{Name: "t1"}
	t2 := map[string]any{"c": map[string]string{"name": "cc"}}
	tests := []struct {
		name string
		args args
		want *dsypb.Box
	}{
		{"BoxFromAny_string", args{value: "a"}, &dsypb.Box{Type: dsypb.BoxType_string, Data: []byte("a")}},
		{"BoxFromAny_string_ptr", args{value: schema.NewStringP("a")}, &dsypb.Box{Type: dsypb.BoxType_string, Data: []byte("a")}},
		{"BoxFromAny_int", args{value: 1}, &dsypb.Box{Type: dsypb.BoxType_integer, Data: []byte("1")}},
		{"BoxFromAny_int_ptr", args{value: schema.NewIntegerP[int](1)}, &dsypb.Box{Type: dsypb.BoxType_integer, Data: []byte("1")}},
		{"BoxFromAny_float", args{value: 1.1}, &dsypb.Box{Type: dsypb.BoxType_float, Data: []byte("1.100000")}},
		{"BoxFromAny_float_ptr", args{value: schema.NewFloatP[float32](1.1)}, &dsypb.Box{Type: dsypb.BoxType_float, Data: []byte("1.100000")}},
		{"BoxFromAny_boolean", args{value: true}, &dsypb.Box{Type: dsypb.BoxType_boolean, Data: []byte("true")}},
		{"BoxFromAny_boolean_ptr", args{value: schema.NewBoolP(true)}, &dsypb.Box{Type: dsypb.BoxType_boolean, Data: []byte("true")}},
		{"BoxFromAny_array", args{value: []int32{1, 2}}, &dsypb.Box{Type: dsypb.BoxType_array, Ref: "integer", Data: []byte(`[1,2]`)}},
		{"BoxFromAny_array_ptr", args{value: &arr}, &dsypb.Box{Type: dsypb.BoxType_array, Ref: "integer", Data: []byte(`[1,2]`)}},
		{"BoxFromAny_struct", args{value: t1}, &dsypb.Box{Type: dsypb.BoxType_object, Data: []byte(`{"name":"t1"}`)}},
		{"BoxFromAny_struct_ptr", args{value: &t1}, &dsypb.Box{Type: dsypb.BoxType_object, Data: []byte(`{"name":"t1"}`)}},
		{"BoxFromAny_map", args{value: map[string]string{"name": "t1"}}, &dsypb.Box{Type: dsypb.BoxType_map, Data: []byte(`{"name":"t1"}`)}},
		{"BoxFromAny_map2", args{value: t2}, &dsypb.Box{Type: dsypb.BoxType_map, Data: []byte(`{"c":{"name":"cc"}}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dsypb.BoxFromAny(tt.args.value); !(reflect.DeepEqual(got.Data, tt.want.Data) && reflect.DeepEqual(got.Type, tt.want.Type)) {
				t.Errorf("BoxFromAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
