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

package v1

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	json "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
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
		Type:       ObjectType,
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
	case IntegerType:
		box.Data = strconv.FormatInt(vv.Int(), 10)
	case FloatType:
		box.Data = fmt.Sprintf("%f", vv.Float())
	case BooleanType:
		box.Data = "false"
		if vv.Bool() {
			box.Data = "true"
		}
	case StringType:
		box.Data = vv.String()
	case ArrayType:
		rt := parseBoxType(vt.Elem())
		box.Ref = string(rt)
		b, _ := json.Marshal(v)
		box.Data = string(b)
	case MapType:
		box.Parameters = map[string]*Box{
			"key":   &Box{Type: parseBoxType(vt.Key())},
			"value": &Box{Type: parseBoxType(vt.Elem())},
		}
		b, _ := json.Marshal(v)
		box.Data = string(b)
	case ObjectType:
		params := make(map[string]*Box)
		for i := 0; i < vt.NumField(); i++ {
			vf := vt.Field(i)
			tag := vf.Tag.Get("json")
			if tag == "" || strings.HasPrefix(tag, "-") {
				continue
			}
			vfv := reflect.New(vf.Type).Interface()
			fb := boxFromAny(vf.Type, vfv)
			params[tag] = fb
		}
		box.Parameters = params
		b, _ := json.Marshal(v)
		box.Data = string(b)
	default:
		b, _ := json.Marshal(v)
		box.Data = string(b)
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
		return IntegerType
	case reflect.Float32, reflect.Float64:
		return FloatType
	case reflect.Bool:
		return BooleanType
	case reflect.String:
		return StringType
	case reflect.Slice:
		return ArrayType
	case reflect.Struct:
		return ObjectType
	case reflect.Map:
		return MapType
	default:
		return ObjectType
	}
}

func (m *Box) Value() any {
	var value any

	switch m.Type {
	case BooleanType:
		value = false
		if string(m.Data) == "true" {
			value = true
		}
	case IntegerType:
		n, _ := strconv.ParseInt(string(m.Data), 10, 64)
		value = n
	case FloatType:
		f, _ := strconv.ParseFloat(string(m.Data), 64)
		value = f
	case StringType:
		value = string(m.Data)
	case ArrayType:
		switch BoxType(m.Ref) {
		case BooleanType:
			target := make([]bool, 0)
			_ = m.ValueFor(&target)
			value = target
		case IntegerType:
			target := make([]int64, 0)
			_ = m.ValueFor(&target)
			value = target
		case FloatType:
			target := make([]float64, 0)
			_ = m.ValueFor(&target)
			value = target
		case StringType:
			target := make([]string, 0)
			_ = m.ValueFor(&target)
			value = target
		case ObjectType, MapType:
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
		_ = json.Unmarshal([]byte(m.Data), &target)
		value = target
	}

	return value
}

func (m *Box) ValueFor(target any) error {
	return json.Unmarshal([]byte(m.Data), &target)
}

func (m *Box) DecodeJSON(data []byte) error {
	return decodeBox(m, data, "")
}

func (m *Box) EncodeJSON() ([]byte, error) {
	v := encodeBox(m)
	return json.Marshal(v)
}

func decodeBox(box *Box, data []byte, paths string) error {
	if box.Type != ObjectType && paths == "" {
		box.Data = string(data)
		return nil
	}

	switch box.Type {
	case ObjectType:
		for name := range box.Parameters {
			param := box.Parameters[name]
			root := name
			if len(paths) != 0 {
				root = paths + "." + root
			}
			box.Data = string(data)
			if err := decodeBox(param, data, root); err != nil {
				return err
			}
		}
	case MapType:
		val := gjson.GetBytes(data, paths).Raw
		box.Data = val
	case ArrayType:
		val := gjson.GetBytes(data, paths).Raw
		box.Data = val
	case IntegerType:
		val := gjson.GetBytes(data, paths).Int()
		box.Data = fmt.Sprintf("%d", val)
	case FloatType:
		val := gjson.GetBytes(data, paths).Float()
		box.Data = fmt.Sprintf("%f", val)
	case BooleanType:
		val := "false"
		if gjson.GetBytes(data, paths).Bool() {
			val = "true"
		}
		box.Data = val
	case StringType:
		val := gjson.GetBytes(data, paths).String()
		box.Data = val
	default:
	}

	return nil
}

func encodeBox(box *Box) any {
	switch box.Type {
	case ObjectType:
		out := map[string]any{}
		for name, param := range box.Parameters {
			out[name] = encodeBox(param)
		}
		if len(box.Data) != 0 {
			_ = json.Unmarshal([]byte(box.Data), &out)
		}

		return out
	case MapType:
		var m map[string]any
		if len(box.Data) != 0 {
			_ = json.Unmarshal([]byte(box.Data), &m)
		}
		return m
	case ArrayType:
		var arr []any
		if len(box.Data) != 0 {
			_ = json.Unmarshal([]byte(box.Data), &arr)
		}
		return arr
	case IntegerType:
		var n int64
		if len(box.Data) != 0 {
			n, _ = strconv.ParseInt(box.Data, 10, 64)
		}
		return n
	case FloatType:
		var f float64
		if len(box.Data) != 0 {
			f, _ = strconv.ParseFloat(box.Data, 64)
		}
		return f
	case BooleanType:
		return false
	case StringType:
		var s string
		s = box.Data
		return s
	default:
		return ""
	}
}
