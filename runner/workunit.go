/*
Copyright 2025 The olive Authors

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

package runner

import (
	"context"
	"fmt"
	"path"
	"reflect"

	"github.com/olive-io/olive/api/types"
)

// Register registers WorkUnit to Runner
func (r *Runner) Register(workUnit WorkUnit, opts ...WuOption) {
	options := NewWuOptions(opts...)
	url := options.String()
	unit := &workUnitImpl{
		options: options,
		inner:   workUnit,
	}
	r.workUnitTree.Insert(url, unit)
}

func (r *Runner) RegisterFunc(fn any, opts ...WuOption) error {
	rv := reflect.ValueOf(fn)
	rt := rv.Type()
	if rt.Kind() != reflect.Func {
		return fmt.Errorf("fn must be a function")
	}

	var in reflect.Type
	hasCtx := false
	switch rt.NumIn() {
	case 0:
	case 1:
		inType := rt.In(0)
		if isContext(inType) {
			hasCtx = true
		} else {
			in = inType
		}
	case 2:
		inType := rt.In(0)
		if isContext(inType) {
			hasCtx = true
		}
		in = rt.In(1)
	}

	unit := &fnWorkUnit{
		function: rv,
		in:       reflect.New(in),
		hasCtx:   hasCtx,
	}
	r.Register(unit, opts...)
	return nil
}

func (r *Runner) findWorkUnit(opts ...WuOption) (WorkUnit, bool) {
	options := NewWuOptions(opts...)

	url := path.Join(options.Type.String())
	if options.Kind != "" {
		url = path.Join(options.Kind, url)
	}
	if options.Id != "" {
		url = options.Id
	}
	_, unit, ok := r.workUnitTree.LongestPrefix(url)
	if !ok {
		return nil, false
	}
	workUnit := reflect.New(reflect.TypeOf(unit)).Interface().(WorkUnit)
	return workUnit, true
}

type WuOptions struct {
	Type types.FlowNodeType

	Kind string

	Id string
}

func (wo *WuOptions) String() string {
	return path.Join("/", wo.Type.String(), wo.Kind, wo.Id)
}

func NewWuOptions(opts ...WuOption) *WuOptions {
	var options WuOptions

	for _, opt := range opts {
		opt(&options)
	}

	if options.Type == 0 {
		options.Type = types.FlowNodeType_Task
	}

	return &options
}

type WuOption func(*WuOptions)

func WithType(t types.FlowNodeType) WuOption {
	return func(o *WuOptions) {
		o.Type = t
	}
}

func WithKind(k string) WuOption {
	return func(o *WuOptions) {
		o.Kind = k
	}
}

func WithID(url string) WuOption {
	return func(o *WuOptions) {
		o.Id = url
	}
}

type WorkUnit interface {
	Commit(ctx context.Context) (any, error)
	Rollback(ctx context.Context) error
	Destroy(ctx context.Context) error
}

var _ WorkUnit = (*workUnitImpl)(nil)

type workUnitImpl struct {
	options *WuOptions
	inner   WorkUnit
}

func (w *workUnitImpl) Inject(properties map[string]string) error {
	err := InjectTypeFields(reflect.ValueOf(w.inner), properties)
	if err != nil {
		return err
	}
	return nil
}

func (w *workUnitImpl) Commit(ctx context.Context) (rsp any, err error) {
	stepCounter.Inc()
	stepCommitCounter.Inc()

	defer func() {
		stepCommitCounter.Desc()
		stepCounter.Desc()
	}()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[%s] commit panic recovered: %s", w.options.Id, r)
		}
	}()

	rsp, err = w.inner.Commit(ctx)
	return rsp, err
}

func (w *workUnitImpl) Rollback(ctx context.Context) (err error) {
	stepCounter.Inc()
	stepRollbackCounter.Inc()

	defer func() {
		stepRollbackCounter.Desc()
		stepCounter.Desc()
	}()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[%s] rollback panic recovered: %s", w.options.Id, r)
		}
	}()

	err = w.inner.Rollback(ctx)
	return err
}

func (w *workUnitImpl) Destroy(ctx context.Context) (err error) {
	stepCounter.Inc()
	stepDestroyCounter.Inc()

	defer func() {
		stepDestroyCounter.Desc()
		stepCounter.Desc()
	}()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[%s] destroy panic recovered: %s", w.options.Id, r)
		}
	}()
	err = w.inner.Destroy(ctx)
	return err
}

var _ WorkUnit = (*fnWorkUnit)(nil)

type fnWorkUnit struct {
	function reflect.Value
	in       reflect.Value
	hasCtx   bool
}

func (wu *fnWorkUnit) Inject(properties map[string]string) error {
	err := InjectTypeFields(wu.in, properties)
	if err != nil {
		return err
	}
	return nil
}

func (wu *fnWorkUnit) Commit(ctx context.Context) (out any, err error) {
	args := make([]reflect.Value, 0)
	if wu.hasCtx {
		args = append(args, reflect.ValueOf(ctx))
	}
	args = append(args, wu.in)
	returnValues := wu.function.Call(args)

	switch len(returnValues) {
	case 0:
		return map[string]string{}, nil
	case 1:
		out = returnValues[0].Interface()
		if rerr, ok := out.(error); ok {
			return nil, rerr
		} else {
			return out, nil
		}
	case 2:
		out = returnValues[0].Interface()
		rerr, ok := returnValues[1].Interface().(error)
		if ok {
			err = rerr
		}
		return out, err
	}
	return
}

func (wu *fnWorkUnit) Rollback(ctx context.Context) error { return nil }

func (wu *fnWorkUnit) Destroy(ctx context.Context) error { return nil }
