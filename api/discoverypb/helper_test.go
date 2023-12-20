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

package discoverypb

import (
	"reflect"
	"testing"

	"github.com/olive-io/bpmn/schema"
)

type TestData struct {
	Name string `json:"name"`
}

func TestBoxFromT(t *testing.T) {
	type args struct {
		value any
	}
	arr := []int32{1, 2}
	t1 := TestData{Name: "t1"}
	tests := []struct {
		name string
		args args
		want *Box
	}{
		{"BoxFromT_string", args{value: "a"}, &Box{Type: Box_String, Data: []byte("a")}},
		{"BoxFromT_string_ptr", args{value: schema.NewStringP("a")}, &Box{Type: Box_String, Data: []byte("a")}},
		{"BoxFromT_int", args{value: 1}, &Box{Type: Box_Integer, Data: []byte("1")}},
		{"BoxFromT_int_ptr", args{value: schema.NewIntegerP[int](1)}, &Box{Type: Box_Integer, Data: []byte("1")}},
		{"BoxFromT_float", args{value: 1.1}, &Box{Type: Box_Float, Data: []byte("1.100000")}},
		{"BoxFromT_float_ptr", args{value: schema.NewFloatP[float32](1.1)}, &Box{Type: Box_Float, Data: []byte("1.100000")}},
		{"BoxFromT_boolean", args{value: true}, &Box{Type: Box_Boolean, Data: []byte("true")}},
		{"BoxFromT_boolean_ptr", args{value: schema.NewBoolP(true)}, &Box{Type: Box_Boolean, Data: []byte("true")}},
		{"BoxFromT_array", args{value: []int32{1, 2}}, &Box{Type: Box_Array, Data: []byte(`[1,2]`)}},
		{"BoxFromT_array_ptr", args{value: &arr}, &Box{Type: Box_Array, Data: []byte(`[1,2]`)}},
		{"BoxFromT_struct", args{value: t1}, &Box{Type: Box_Object, Data: []byte(`{"name":"t1"}`)}},
		{"BoxFromT_struct_ptr", args{value: &t1}, &Box{Type: Box_Object, Data: []byte(`{"name":"t1"}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BoxFromT(tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BoxFromT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBox_ValueFor(t *testing.T) {
	box := &Box{
		Type: Box_Integer,
		Data: []byte(`{"name": "t1"}`),
	}

	t1 := new(TestData)
	type fields struct {
		Type Box_BoxType
		Data []byte
	}
	type args struct {
		target any
	}
	integer := new(int32)
	boolean := new(bool)
	floatT := new(float32)
	arr := make([]int32, 0)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"value_for_integer", fields{Type: Box_Integer, Data: []byte("1")}, args{target: integer}, false},
		{"value_for_bool", fields{Type: Box_Boolean, Data: []byte("true")}, args{target: boolean}, false},
		{"value_for_float", fields{Type: Box_Float, Data: []byte("1.1")}, args{target: floatT}, false},
		{"value_for_array", fields{Type: Box_Array, Data: []byte(`[1,2,3]`)}, args{target: arr}, false},
		{"value_for_struct", fields{Type: Box_Object, Data: box.Data}, args{target: t1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Box{
				Type: tt.fields.Type,
				Data: tt.fields.Data,
			}
			if err := m.ValueFor(tt.args.target); (err != nil) != tt.wantErr {
				t.Errorf("ValueFor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
