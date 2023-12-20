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

func BoxFromT(value any) *Box {
	return boxFromT(reflect.ValueOf(value), value)
}

func boxFromT(vf reflect.Value, v any) *Box {
	if vf.Type().Kind() == reflect.Pointer {
		return boxFromT(vf.Elem(), v)
	}

	box := &Box{}
	switch vf.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		box.Type = Box_Integer
		box.Data = []byte(fmt.Sprintf("%d", vf.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		box.Type = Box_Integer
		box.Data = []byte(fmt.Sprintf("%d", vf.Uint()))
	case reflect.Float32, reflect.Float64:
		box.Type = Box_Float
		box.Data = []byte(fmt.Sprintf("%f", vf.Float()))
	case reflect.Bool:
		box.Type = Box_Boolean
		box.Data = []byte("false")
		if vf.Bool() {
			box.Data = []byte("true")
		}
	case reflect.String:
		box.Type = Box_String
		box.Data = []byte(vf.String())
	case reflect.Slice:
		box.Type = Box_Array
		box.Data, _ = json.Marshal(v)

	case reflect.Struct:
		box.Type = Box_Object
		box.Data, _ = json.Marshal(v)

	default:
	}

	return box
}

func (m *Box) Value() any {
	var value any
	switch m.Type {
	case Box_Boolean:
		value = false
		if string(m.Data) == "true" {
			value = true
		}
	case Box_Integer:
		n, _ := strconv.ParseInt(string(m.Data), 10, 64)
		value = n
	case Box_Float:
		f, _ := strconv.ParseFloat(string(m.Data), 64)
		value = f
	case Box_String:
		value = string(m.Data)
	case Box_Array:
		target := make([]any, 0)
		_ = json.Unmarshal(m.Data, &target)
		value = target
	case Box_Object:
		target := map[string]any{}
		_ = json.Unmarshal(m.Data, &target)
		value = target
	}

	return value
}

func (m *Box) ValueFor(target any) error {
	return json.Unmarshal(m.Data, &target)
}
