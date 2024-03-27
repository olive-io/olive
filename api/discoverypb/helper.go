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
	"fmt"
	"reflect"
	"strconv"

	json "github.com/json-iterator/go"
)

func (m *Box) WithRef(ref string) *Box {
	m.Ref = ref
	return m
}

func BoxFromAny(value any) *Box {
	return boxFromAny(reflect.TypeOf(value), value)
}

func boxFromAny(vt reflect.Type, v any) *Box {
	if box, ok := v.(*Box); ok {
		return box
	}

	box := &Box{}
	if vv, ok := v.([]byte); ok {
		if err := json.Unmarshal(vv, &box); err == nil {
			return box
		}
	}

	if vt.Kind() == reflect.Pointer {
		return boxFromAny(vt.Elem(), v)
	}

	vv := reflect.ValueOf(v)
	if vv.Type().Kind() == reflect.Pointer {
		vv = vv.Elem()
	}

	bt := parseBoxType(vt)
	box.Type = bt
	switch bt {
	case BoxType_integer:
		box.Data = []byte(strconv.FormatInt(vv.Int(), 10))
	case BoxType_float:
		box.Data = []byte(fmt.Sprintf("%f", vv.Float()))
	case BoxType_boolean:
		box.Data = []byte("false")
		if vv.Bool() {
			box.Data = []byte("true")
		}
	case BoxType_string:
		box.Data = []byte(vv.String())
	case BoxType_array:
		rt := parseBoxType(vt.Elem())
		box.Ref = rt.String()
		box.Data, _ = json.Marshal(v)
	case BoxType_map:
		box.Parameters = map[string]*Box{
			"key":   &Box{Type: parseBoxType(vt.Key())},
			"value": &Box{Type: parseBoxType(vt.Elem())},
		}
		box.Data, _ = json.Marshal(v)
	default:
		box.Data, _ = json.Marshal(v)
	}
	return box
}

func parseBoxType(vf reflect.Type) BoxType {
	if vf.Kind() == reflect.Pointer {
		return parseBoxType(vf.Elem())
	}

	switch vf.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return BoxType_integer
	case reflect.Float32, reflect.Float64:
		return BoxType_float
	case reflect.Bool:
		return BoxType_boolean
	case reflect.String:
		return BoxType_string
	case reflect.Slice:
		return BoxType_array
	case reflect.Struct:
		return BoxType_object
	case reflect.Map:
		return BoxType_map
	default:
		return BoxType_object
	}
}

func (m *Box) Value() any {
	var value any

	switch m.Type {
	case BoxType_boolean:
		value = false
		if string(m.Data) == "true" {
			value = true
		}
	case BoxType_integer:
		n, _ := strconv.ParseInt(string(m.Data), 10, 64)
		value = n
	case BoxType_float:
		f, _ := strconv.ParseFloat(string(m.Data), 64)
		value = f
	case BoxType_string:
		value = string(m.Data)
	case BoxType_array:
		switch BoxType(BoxType_value[m.Ref]) {
		case BoxType_boolean:
			target := make([]bool, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_integer:
			target := make([]int64, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_float:
			target := make([]float64, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_string:
			target := make([]string, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_object, BoxType_map:
			target := make([]map[string]any, 0)
			_ = m.ValueFor(&target)
			value = target
		default:
			target := make([]any, 0)
			_ = m.ValueFor(&target)
			value = target
		}
	default:
		target := map[string]any{}
		_ = json.Unmarshal(m.Data, &target)
		value = target
	}

	return value
}

func (m *Box) ValueFor(target any) error {
	return json.Unmarshal(m.Data, &target)
}
