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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/olive-io/bpmn/schema"
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
		want *Box
	}{
		{"BoxFromAny_string", args{value: "a"}, &Box{Type: BoxType_String, Data: []byte("a")}},
		{"BoxFromAny_string_ptr", args{value: schema.NewStringP("a")}, &Box{Type: BoxType_String, Data: []byte("a")}},
		{"BoxFromAny_int", args{value: 1}, &Box{Type: BoxType_Integer, Data: []byte("1")}},
		{"BoxFromAny_int_ptr", args{value: schema.NewIntegerP[int](1)}, &Box{Type: BoxType_Integer, Data: []byte("1")}},
		{"BoxFromAny_float", args{value: 1.1}, &Box{Type: BoxType_Float, Data: []byte("1.100000")}},
		{"BoxFromAny_float_ptr", args{value: schema.NewFloatP[float32](1.1)}, &Box{Type: BoxType_Float, Data: []byte("1.100000")}},
		{"BoxFromAny_boolean", args{value: true}, &Box{Type: BoxType_Boolean, Data: []byte("true")}},
		{"BoxFromAny_boolean_ptr", args{value: schema.NewBoolP(true)}, &Box{Type: BoxType_Boolean, Data: []byte("true")}},
		{"BoxFromAny_array", args{value: []int32{1, 2}}, &Box{Type: BoxType_Integer, IsArray: true, Data: []byte(`[1,2]`)}},
		{"BoxFromAny_array_ptr", args{value: &arr}, &Box{Type: BoxType_Integer, IsArray: true, Data: []byte(`[1,2]`)}},
		{"BoxFromAny_struct", args{value: t1}, &Box{Type: BoxType_Object, Data: []byte(`{"name":"t1"}`)}},
		{"BoxFromAny_struct_ptr", args{value: &t1}, &Box{Type: BoxType_Object, Data: []byte(`{"name":"t1"}`)}},
		{"BoxFromAny_map", args{value: map[string]string{"name": "t1"}}, &Box{Type: BoxType_Object, Data: []byte(`{"name":"t1"}`)}},
		{"BoxFromAny_map2", args{value: t2}, &Box{Type: BoxType_Object, Data: []byte(`{"c":{"name":"cc"}}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BoxFromAny(tt.args.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BoxFromAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBox_Value(t *testing.T) {
	box := &Box{
		Type: BoxType_Integer,
		Data: []byte(`{"name": "t1"}`),
	}

	type fields struct {
		Type    BoxType
		IsArray bool
		Data    []byte
	}
	type args struct {
		target any
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"value_integer", fields{Type: BoxType_Integer, Data: []byte("1")}, args{target: 1}},
		{"value_bool", fields{Type: BoxType_Boolean, Data: []byte("true")}, args{target: true}},
		{"value_float", fields{Type: BoxType_Float, Data: []byte("1.1")}, args{target: 1.1}},
		{"value_for_array", fields{Type: BoxType_Integer, IsArray: true, Data: []byte(`[1,2,3]`)}, args{target: []int64{1, 2, 3}}},
		{"value_map", fields{Type: BoxType_Object, Data: box.Data}, args{target: map[string]any{"name": "t1"}}},
		{"value_struct", fields{Type: BoxType_Object, Data: box.Data}, args{target: map[string]any{"name": "t1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Box{
				Type:    tt.fields.Type,
				IsArray: tt.fields.IsArray,
				Data:    tt.fields.Data,
			}
			value := m.Value()
			a, _ := json.Marshal(value)
			b, _ := json.Marshal(tt.args.target)
			if string(a) != string(b) {
				t.Errorf("Value() expected = %v, want %v", string(a), string(b))
			}
		})
	}
}

func TestBox_ValueFor(t *testing.T) {
	box := &Box{
		Type: BoxType_Integer,
		Data: []byte(`{"name": "t1"}`),
	}

	t1 := new(TestData)
	type fields struct {
		Type    BoxType
		IsArray bool
		Data    []byte
	}
	type args struct {
		target any
	}
	integer := new(int32)
	boolean := new(bool)
	floatT := new(float32)
	arr := make([]int32, 0)
	m := make(map[string]string)
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"value_for_integer", fields{Type: BoxType_Integer, Data: []byte("1")}, args{target: integer}, false},
		{"value_for_bool", fields{Type: BoxType_Boolean, Data: []byte("true")}, args{target: boolean}, false},
		{"value_for_float", fields{Type: BoxType_Float, Data: []byte("1.1")}, args{target: floatT}, false},
		{"value_for_array", fields{Type: BoxType_Integer, IsArray: true, Data: []byte(`[1,2,3]`)}, args{target: arr}, false},
		{"value_for_map", fields{Type: BoxType_Object, Data: box.Data}, args{target: m}, false},
		{"value_for_struct", fields{Type: BoxType_Object, Data: box.Data}, args{target: t1}, false},
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
