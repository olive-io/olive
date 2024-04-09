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

package proxy

import (
	"fmt"
	"strconv"

	json "github.com/json-iterator/go"
	"github.com/tidwall/gjson"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

func DecodeBox(data []byte, box *dsypb.Box) error {
	return decodeBox(box, string(data), "")
}

func EncodeBox(box *dsypb.Box) ([]byte, error) {
	v := encodeBox(box)
	return json.Marshal(v)
}

func decodeBox(box *dsypb.Box, data, paths string) error {
	switch box.Type {
	case dsypb.BoxType_object:
		for name := range box.Parameters {
			param := box.Parameters[name]
			root := name
			if len(paths) != 0 {
				root = paths + "." + root
			}
			if err := decodeBox(param, data, root); err != nil {
				return err
			}
		}
	case dsypb.BoxType_map:
		val := gjson.Get(data, paths).Raw
		box.Data = []byte(val)
	case dsypb.BoxType_array:
		val := gjson.Get(data, paths).Raw
		box.Data = []byte(val)
	case dsypb.BoxType_integer:
		val := gjson.Get(data, paths).Int()
		box.Data = []byte(fmt.Sprintf("%d", val))
	case dsypb.BoxType_float:
		val := gjson.Get(data, paths).Float()
		box.Data = []byte(fmt.Sprintf("%f", val))
	case dsypb.BoxType_boolean:
		val := "false"
		if gjson.Get(data, paths).Bool() {
			val = "true"
		}
		box.Data = []byte(val)
	case dsypb.BoxType_string:
		val := gjson.Get(data, paths).String()
		box.Data = []byte(val)
	default:
	}

	return nil
}

func encodeBox(box *dsypb.Box) any {
	switch box.Type {
	case dsypb.BoxType_object:
		out := map[string]any{}
		for name, param := range box.Parameters {
			out[name] = encodeBox(param)
		}
		if len(box.Data) != 0 {
			_ = json.Unmarshal(box.Data, &out)
		}

		return out
	case dsypb.BoxType_map:
		var m map[string]any
		if len(box.Data) != 0 {
			_ = json.Unmarshal(box.Data, &m)
		}
		return m
	case dsypb.BoxType_array:
		var arr []any
		if len(box.Data) != 0 {
			_ = json.Unmarshal(box.Data, &arr)
		}
		return arr
	case dsypb.BoxType_integer:
		var n int64
		if len(box.Data) != 0 {
			n, _ = strconv.ParseInt(string(box.Data), 10, 64)
		}
		return n
	case dsypb.BoxType_float:
		var f float64
		if len(box.Data) != 0 {
			f, _ = strconv.ParseFloat(string(box.Data), 64)
		}
		return f
	case dsypb.BoxType_boolean:
		return false
	case dsypb.BoxType_string:
		var s string
		s = string(box.Data)
		return s
	default:
		return ""
	}
}
