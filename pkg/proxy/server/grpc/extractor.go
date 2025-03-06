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

package grpc

import (
	"fmt"
	"reflect"
	"strings"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

func extractGRPCEndpoint(method reflect.Method) *dsypb.Endpoint {
	if method.PkgPath != "" {
		return nil
	}

	var respType, reqType reflect.Type
	var stream bool
	mt := method.Type

	in, out := mt.NumIn(), mt.NumOut()
	if in == 3 && out == 2 {
		reqType = mt.In(2)
		respType = mt.Out(0)
	} else if in == 3 && out == 1 {
		reqType = mt.In(1)
		respType = mt.In(2)
		stream = true
	} else if in == 2 && out == 1 {
		reqType = mt.In(1)
		respType = mt.In(1)
		stream = true
	} else {
		panic("invalid grpc endpoint")
	}

	// are we dealing with a stream?
	switch respType.Kind() {
	case reflect.Func, reflect.Interface:
		stream = true
	default:
	}

	request := extractGRPCBox(reqType, 0)
	response := extractGRPCBox(respType, 0)

	ep := &dsypb.Endpoint{
		Name:     method.Name,
		Request:  request,
		Response: response,
		Metadata: make(map[string]string),
	}

	if stream {
		if _, exists := ep.Metadata["stream"]; !exists {
			ep.Metadata["stream"] = fmt.Sprintf("%v", stream)
		}
	}

	return ep
}

func extractGRPCBox(v reflect.Type, d int) *dsypb.Box {
	if d == 5 {
		return nil
	}
	if v == nil {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	box := &dsypb.Box{}

	switch v.Kind() {
	case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64:
		box.Type = dsypb.BoxType_integer
	case reflect.Float32, reflect.Float64:
		box.Type = dsypb.BoxType_float
	case reflect.String:
		box.Type = dsypb.BoxType_string
	case reflect.Bool:
		box.Type = dsypb.BoxType_boolean
	case reflect.Struct:
		box.Type = dsypb.BoxType_object
		parameters := map[string]*dsypb.Box{}
		for i := 0; i < v.NumField(); i++ {
			ftp := v.Field(i)
			field := extractGRPCBox(ftp.Type, d+1)
			if field == nil {
				continue
			}

			var name string
			// if we can find a json tag use it
			if tags := ftp.Tag.Get("json"); len(tags) > 0 {
				parts := strings.Split(tags, ",")
				if parts[0] == "-" || parts[0] == "omitempty" {
					continue
				}
				name = parts[0]
			}

			// if there's no name default it
			if len(name) == 0 {
				continue
			}

			parameters[name] = field
		}
		box.Parameters = parameters
		box.Ref = boxRefPath(v)
	case reflect.Slice:
		p := v.Elem()
		if p.Kind() == reflect.Ptr {
			p = p.Elem()
		}

		box.Type = dsypb.BoxType_array
		if p.Kind() == reflect.Uint8 {
			box.Type = dsypb.BoxType_string
		} else {
			if ref := extractGRPCBox(p, d+1); ref != nil {
				box.Ref = ref.Ref
				if ref.Ref == "" {
					box.Ref = ref.Type.String()
				}
			}
		}
	case reflect.Map:
		// arg.Type = fmt.Sprintf(`map %s:%s`, v.Key().String(), v.Elem().String())
		box.Type = dsypb.BoxType_map
		if ref := extractGRPCBox(v.Elem(), d+1); ref != nil {
			box.Ref = ref.Ref
			if ref.Ref == "" {
				box.Ref = ref.Type.String()
			}
		}
	default:
	}

	return box
}

func boxRefPath(v reflect.Type) string {
	return strings.ReplaceAll(v.PkgPath(), "/", ".") + "." + v.Name()
}
