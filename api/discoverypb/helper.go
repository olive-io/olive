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

func (m *Box) Split() map[string]*Box {
	params := make(map[string]*Box)
	for name, param := range m.Parameters {
		params[name] = param
	}
	return params
}

func JoinBoxes(boxes map[string]*Box) *Box {
	b := &Box{
		Type:       BoxType_object,
		Parameters: boxes,
	}
	return b
}

func BoxFromAny(value any) *Box {
	return boxFromAny(reflect.TypeOf(value), value)
}

func boxFromAny(vt reflect.Type, v any) *Box {
	if box, ok := v.(*Box); ok {
		return box
	}

	box := &Box{}
	switch vv := v.(type) {
	case []byte:
		if err := json.Unmarshal(vv, &box); err == nil {
			return box
		}
	case *Box:
		return vv
	default:
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
