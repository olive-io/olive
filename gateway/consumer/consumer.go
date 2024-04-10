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

	"github.com/cockroachdb/errors"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

var (
	ErrNotFound = errors.New("not found")
	ErrBind     = errors.New("binds value")
)

type HandlerFunc func(ctx *Context) (any, error)

// HandlerWrapper wraps the HandlerFunc and returns the equivalent
type HandlerWrapper func(HandlerFunc) HandlerFunc

type IKnownConsumer interface {
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
