package runner

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/olive-io/olive/api/types"
)

func isContext(rt reflect.Type) bool {
	return rt.PkgPath()+"."+rt.Name() == "context.Context"
}

func setField(vField reflect.Value, value string) error {
	switch vField.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, e := strconv.ParseInt(value, 10, 64)
		if e != nil {
			return e
		}
		vField.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, e := strconv.ParseUint(value, 10, 64)
		if e != nil {
			return e
		}
		vField.SetUint(v)
	case reflect.String:
		vField.SetString(value)
	case reflect.Ptr:
		v := reflect.New(vField.Type().Elem())
		vv := v.Interface()

		var e error
		e = json.Unmarshal([]byte(value), &vv)
		if e != nil {
			return e
		}
		vField.Set(v)
	case reflect.Slice, reflect.Array:
		v := reflect.New(vField.Type())
		vv := v.Interface()

		var e error
		e = json.Unmarshal([]byte(value), vv)
		if e != nil {
			return e
		}
		vField.Set(v.Elem())

	case reflect.Struct, reflect.Map:
		v := reflect.New(vField.Type())
		vv := v.Interface()

		var e error
		e = json.Unmarshal([]byte(value), &vv)
		if e != nil {
			return e
		}
		vField.Set(v.Elem())
	case reflect.Bool:
		v, _ := strconv.ParseBool(value)
		vField.SetBool(v)
	}

	return nil
}

func extractField(vField reflect.Value) string {
	switch vField.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(vField.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(vField.Uint(), 10)
	case reflect.String:
		return vField.String()
	case reflect.Ptr:
		return extractField(vField.Elem())
	case reflect.Slice, reflect.Array:
		vv := vField.Interface()

		data, _ := json.Marshal(vv)
		return string(data)
	case reflect.Struct, reflect.Map:
		vv := vField.Interface()

		data, _ := json.Marshal(vv)
		return string(data)

	case reflect.Bool:
		return strconv.FormatBool(vField.Bool())
	default:
		return ""
	}
}

func InjectTypeFields(vle reflect.Value, items map[string]string) error {
	typ := vle.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		vle = vle.Elem()
	}

	fmt.Println(typ.Name())
	for i := 0; i < typ.NumField(); i++ {
		tField := typ.Field(i)
		if !tField.IsExported() {
			continue
		}

		text, ok := tField.Tag.Lookup(InjectTag)
		if !ok {
			continue
		}
		if text == "" || text == "-" {
			continue
		}

		vField := vle.Field(i)
		value, ok := items[text]
		if !ok {
			continue
		}
		if err := setField(vField, value); err != nil {
			return fmt.Errorf("inject to context field '%s': %w", tField.Name, err)
		}
	}

	return nil
}

func ExtractTypeFields(t any) map[string]string {
	typ := reflect.TypeOf(t)
	vle := reflect.ValueOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		vle = vle.Elem()
	}

	results := make(map[string]string)
	for i := 0; i < typ.NumField(); i++ {
		tField := typ.Field(i)
		if !tField.IsExported() {
			continue
		}

		text, ok := tField.Tag.Lookup(InjectTag)
		if !ok {
			continue
		}
		if text == "" || text == "-" {
			continue
		}

		vField := vle.Field(i)
		results[text] = extractField(vField)
	}

	return results
}

func GenerateEndpoint(unit WorkUnit, options *WuOptions) *types.Endpoint {
	endpoint := &types.Endpoint{
		Type:    options.Type,
		Kind:    options.Kind,
		Name:    options.Id,
		Headers: map[string]string{},
	}

	endpoint.Parameters = extractValue(options.Request)
	endpoint.Results = extractValue(options.Response)

	return endpoint
}

func extractValue(rt reflect.Type) map[string]*types.Value {
	items := make(map[string]*types.Value)

	for i := 0; i < rt.NumField(); i++ {
		tField := rt.Field(i)
		if !tField.IsExported() {
			continue
		}

		text, ok := tField.Tag.Lookup(InjectTag)
		if !ok {
			continue
		}
		if text == "" || text == "-" {
			continue
		}

		name := text
		items[name] = &types.Value{
			Type: parseValueType(tField.Type),
		}
	}

	return items
}

func parseValueType(rt reflect.Type) types.Value_Type {
	switch rt.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return types.Value_Integer
	case reflect.String:
		return types.Value_String
	case reflect.Ptr:
		return parseValueType(rt.Elem())
	case reflect.Slice, reflect.Array:
		return types.Value_Array
	case reflect.Struct, reflect.Map:
		return types.Value_Object
	case reflect.Bool:
		return types.Value_Boolean
	default:
		return types.Value_String
	}
}
