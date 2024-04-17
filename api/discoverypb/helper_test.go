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

package discoverypb

type TestData struct {
	Name string `json:"name"`
}

//func TestBoxFromAny(t *testing.T) {
//	type args struct {
//		value any
//	}
//	arr := []int32{1, 2}
//	t1 := TestData{Name: "t1"}
//	t2 := map[string]any{"c": map[string]string{"name": "cc"}}
//	tests := []struct {
//		name string
//		args args
//		want *Box
//	}{
//		{"BoxFromAny_string", args{value: "a"}, &Box{Type: BoxType_string, Data: []byte("a")}},
//		{"BoxFromAny_string_ptr", args{value: schema.NewStringP("a")}, &Box{Type: BoxType_string, Data: []byte("a")}},
//		{"BoxFromAny_int", args{value: 1}, &Box{Type: BoxType_integer, Data: []byte("1")}},
//		{"BoxFromAny_int_ptr", args{value: schema.NewIntegerP[int](1)}, &Box{Type: BoxType_integer, Data: []byte("1")}},
//		{"BoxFromAny_float", args{value: 1.1}, &Box{Type: BoxType_float, Data: []byte("1.100000")}},
//		{"BoxFromAny_float_ptr", args{value: schema.NewFloatP[float32](1.1)}, &Box{Type: BoxType_float, Data: []byte("1.100000")}},
//		{"BoxFromAny_boolean", args{value: true}, &Box{Type: BoxType_boolean, Data: []byte("true")}},
//		{"BoxFromAny_boolean_ptr", args{value: schema.NewBoolP(true)}, &Box{Type: BoxType_boolean, Data: []byte("true")}},
//		{"BoxFromAny_array", args{value: []int32{1, 2}}, &Box{Type: BoxType_array, Ref: "integer", Data: []byte(`[1,2]`)}},
//		{"BoxFromAny_array_ptr", args{value: &arr}, &Box{Type: BoxType_array, Ref: "integer", Data: []byte(`[1,2]`)}},
//		{"BoxFromAny_struct", args{value: t1}, &Box{Type: BoxType_object, Data: []byte(`{"name":"t1"}`)}},
//		{"BoxFromAny_struct_ptr", args{value: &t1}, &Box{Type: BoxType_object, Data: []byte(`{"name":"t1"}`)}},
//		{"BoxFromAny_map", args{value: map[string]string{"name": "t1"}}, &Box{Type: BoxType_map, Data: []byte(`{"name":"t1"}`)}},
//		{"BoxFromAny_map2", args{value: t2}, &Box{Type: BoxType_map, Data: []byte(`{"c":{"name":"cc"}}`)}},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := BoxFromAny(tt.args.value); !(reflect.DeepEqual(got.Data, tt.want.Data) && reflect.DeepEqual(got.Type, tt.want.Type)) {
//				t.Errorf("BoxFromAny() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestBox_Value(t *testing.T) {
//	box := &Box{
//		Type: BoxType_integer,
//		Data: []byte(`{"name": "t1"}`),
//	}
//
//	type fields struct {
//		Type BoxType
//		Ref  string
//		Data []byte
//	}
//	type args struct {
//		target any
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		{"value_integer", fields{Type: BoxType_integer, Data: []byte("1")}, args{target: 1}},
//		{"value_bool", fields{Type: BoxType_boolean, Data: []byte("true")}, args{target: true}},
//		{"value_float", fields{Type: BoxType_float, Data: []byte("1.1")}, args{target: 1.1}},
//		{"value_for_array", fields{Type: BoxType_array, Ref: "integer", Data: []byte(`[1,2,3]`)}, args{target: []int64{1, 2, 3}}},
//		{"value_map", fields{Type: BoxType_object, Data: box.Data}, args{target: map[string]any{"name": "t1"}}},
//		{"value_struct", fields{Type: BoxType_object, Data: box.Data}, args{target: map[string]any{"name": "t1"}}},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			m := &Box{
//				Type: tt.fields.Type,
//				Ref:  tt.fields.Ref,
//				Data: tt.fields.Data,
//			}
//			value := m.Value()
//			a, _ := json.Marshal(value)
//			b, _ := json.Marshal(tt.args.target)
//			if string(a) != string(b) {
//				t.Errorf("Value() expected = %v, want %v", string(a), string(b))
//			}
//		})
//	}
//}
//
//func TestBox_ValueFor(t *testing.T) {
//	box := &Box{
//		Type: BoxType_integer,
//		Data: []byte(`{"name": "t1"}`),
//	}
//
//	t1 := new(TestData)
//	type fields struct {
//		Type    BoxType
//		IsArray bool
//		Data    []byte
//	}
//	type args struct {
//		target any
//	}
//	integer := new(int32)
//	boolean := new(bool)
//	floatT := new(float32)
//	arr := make([]int32, 0)
//	m := make(map[string]string)
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{"value_for_integer", fields{Type: BoxType_integer, Data: []byte("1")}, args{target: integer}, false},
//		{"value_for_bool", fields{Type: BoxType_boolean, Data: []byte("true")}, args{target: boolean}, false},
//		{"value_for_float", fields{Type: BoxType_float, Data: []byte("1.1")}, args{target: floatT}, false},
//		{"value_for_array", fields{Type: BoxType_array, Data: []byte(`[1,2,3]`)}, args{target: arr}, false},
//		{"value_for_map", fields{Type: BoxType_object, Data: box.Data}, args{target: m}, false},
//		{"value_for_struct", fields{Type: BoxType_object, Data: box.Data}, args{target: t1}, false},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			m := &Box{
//				Type: tt.fields.Type,
//				Data: tt.fields.Data,
//			}
//			if err := m.ValueFor(tt.args.target); (err != nil) != tt.wantErr {
//				t.Errorf("ValueFor() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
