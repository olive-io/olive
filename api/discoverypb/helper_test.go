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
