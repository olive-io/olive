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

package consumer

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	json "github.com/json-iterator/go"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

var (
	ErrNotFound = errors.New("not found")
	ErrBind     = errors.New("binds value")
)

type HandlerFunc func(ctx *Context) (any, error)

// HandlerWrapper wraps the HandlerFunc and returns the equivalent
type HandlerWrapper func(HandlerFunc) HandlerFunc

type IRegularConsumer interface {
	IConsumer
	Identity() *dsypb.Consumer
}

type IConsumer interface {
	Handle(ctx *Context) (any, error)
}

type Context struct {
	context.Context

	Headers     map[string]string
	Properties  map[string]*dsypb.Box
	DataObjects map[string]*dsypb.Box
}

// BindTo binds given value to target from Context.Properties
func (ctx *Context) BindTo(name string, target any) error {
	box, ok := ctx.Properties[name]
	if !ok {
		return ErrNotFound
	}
	if err := box.ValueFor(target); err != nil {
		return errors.CombineErrors(ErrBind, err)
	}
	return nil
}

// BindDataTo binds given value to target from Context.DataObjects
func (ctx *Context) BindDataTo(name string, target any) error {
	box, ok := ctx.Properties[name]
	if !ok {
		return ErrNotFound
	}
	if err := box.ValueFor(target); err != nil {
		return errors.CombineErrors(ErrBind, err)
	}
	return nil
}

func (ctx *Context) MustBind(target any) error {
	vt := reflect.TypeOf(target)
	vv := reflect.ValueOf(target)

	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
		vv = vv.Elem()
	}

	for i := 0; i < vt.NumField(); i++ {
		field := vt.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || strings.HasPrefix(tag, "-") {
			continue
		}
		fname := strings.Split(tag, ",")[0]

		box, ok := ctx.Properties[fname]
		if !ok {
			box, ok = ctx.DataObjects[fname]
			if !ok {
				err := errors.Wrapf(ErrNotFound, "field '%s'", fname)
				return errors.CombineErrors(err, ErrBind)
			}
		}

		vf := vv.Field(i)
		err := setField(vf, []byte(box.Data))
		if err != nil {
			return errors.CombineErrors(ErrBind, err)
		}
	}

	return nil
}

func setField(vField reflect.Value, value []byte) error {
	switch vField.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, e := strconv.ParseInt(string(value), 10, 64)
		if e != nil {
			return e
		}
		vField.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, e := strconv.ParseUint(string(value), 10, 64)
		if e != nil {
			return e
		}
		vField.SetUint(v)
	case reflect.String:
		vField.SetString(string(value))
	case reflect.Ptr:
		v := reflect.New(vField.Type().Elem())
		vv := v.Interface()

		var e error
		e = json.Unmarshal(value, &vv)
		if e != nil {
			return e
		}
		vField.Set(v)
	case reflect.Slice:
		v := reflect.New(vField.Type())
		vv := v.Interface()

		var e error
		e = json.Unmarshal(value, vv)
		if e != nil {
			return e
		}
		vField.Set(v.Elem())

	case reflect.Struct, reflect.Map:
		v := reflect.New(vField.Type())
		vv := v.Interface()

		var e error
		e = json.Unmarshal(value, &vv)
		if e != nil {
			return e
		}
		vField.Set(v)
	default:
		return nil
	}

	return nil
}
