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

// SupportActivity returns true if the given value in Activities
func (m *Node) SupportActivity(act Activity) bool {
	for _, item := range m.Activities {
		if item == act {
			return true
		}
	}
	return false
}

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

	bt, isArray := parseBoxType(vt)
	box.Type = bt
	box.IsArray = isArray

	vv := reflect.ValueOf(v)
	if vv.Type().Kind() == reflect.Pointer {
		vv = vv.Elem()
	}
	if !isArray {
		switch bt {
		case BoxType_Integer:
			box.Data = []byte(strconv.FormatInt(vv.Int(), 10))
		case BoxType_Float:
			box.Data = []byte(fmt.Sprintf("%f", vv.Float()))
		case BoxType_Boolean:
			box.Data = []byte("false")
			if vv.Bool() {
				box.Data = []byte("true")
			}
		case BoxType_String:
			box.Data = []byte(vv.String())
		case BoxType_Object:
			box.Data, _ = json.Marshal(v)
		}
	} else {
		box.Data, _ = json.Marshal(v)
	}
	return box
}

func parseBoxType(vf reflect.Type) (BoxType, bool) {
	if vf.Kind() == reflect.Pointer {
		return parseBoxType(vf.Elem())
	}

	switch vf.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return BoxType_Integer, false
	case reflect.Float32, reflect.Float64:
		return BoxType_Float, false
	case reflect.Bool:
		return BoxType_Boolean, false
	case reflect.String:
		return BoxType_String, false
	case reflect.Slice:
		bt, _ := parseBoxType(vf.Elem())
		return bt, true

	case reflect.Struct:
		return BoxType_Object, false
	case reflect.Map:
		return BoxType_Object, false
	default:
		return BoxType_Object, false
	}
}

func (m *Box) Value() any {
	var value any
	if !m.IsArray {
		switch m.Type {
		case BoxType_Boolean:
			value = false
			if string(m.Data) == "true" {
				value = true
			}
		case BoxType_Integer:
			n, _ := strconv.ParseInt(string(m.Data), 10, 64)
			value = n
		case BoxType_Float:
			f, _ := strconv.ParseFloat(string(m.Data), 64)
			value = f
		case BoxType_String:
			value = string(m.Data)
		case BoxType_Object:
			target := map[string]any{}
			_ = json.Unmarshal(m.Data, &target)
			value = target
		}
	} else {
		switch m.Type {
		case BoxType_Boolean:
			target := make([]bool, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_Integer:
			target := make([]int64, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_Float:
			target := make([]float64, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_String:
			target := make([]string, 0)
			_ = m.ValueFor(&target)
			value = target
		case BoxType_Object:
			target := make([]map[string]any, 0)
			_ = m.ValueFor(&target)
			value = target
		}
	}

	return value
}

func (m *Box) ValueFor(target any) error {
	return json.Unmarshal(m.Data, &target)
}
