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

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/pkg/tree"
)

var router = tree.New[WorkUnit]()

func RegisterWorkUnit(workUnit WorkUnit, opts ...WuOption) {
	options := NewWuOptions(opts...)
	url := options.String()
	unit := &workUnitImpl{
		options: options,
		inner:   workUnit,
	}
	router.Insert(url, unit)
}

func GetWorkUnit(opts ...WuOption) (WorkUnit, bool) {
	options := NewWuOptions(opts...)

	url := path.Join(options.Type.String())
	if options.Kind != "" {
		url = path.Join(options.Kind, url)
	}
	if options.Id != "" {
		url = options.Id
	}
	_, unit, ok := router.LongestPrefix(url)
	return unit.(*workUnitImpl).inner, ok
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
	Commit(ctx context.Context, request any) (any, error)
	Rollback(ctx context.Context) error
	Destroy(ctx context.Context) error
}

var _ WorkUnit = (*workUnitImpl)(nil)

type workUnitImpl struct {
	options *WuOptions
	inner   WorkUnit
}

func (w *workUnitImpl) Commit(ctx context.Context, request any) (rsp any, err error) {
	stepCounter.Inc()
	stepCommitCounter.Inc()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[%s] commit panic recovered: %s", w.options.Id, r)
		}
	}()

	rsp, err = w.inner.Commit(ctx, request)

	stepCommitCounter.Desc()
	stepCounter.Desc()
	return rsp, err
}

func (w *workUnitImpl) Rollback(ctx context.Context) (err error) {
	stepCounter.Inc()
	stepRollbackCounter.Inc()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[%s] rollback panic recovered: %s", w.options.Id, r)
		}
	}()

	err = w.inner.Rollback(ctx)

	stepRollbackCounter.Desc()
	stepCounter.Desc()
	return err
}

func (w *workUnitImpl) Destroy(ctx context.Context) (err error) {
	stepCounter.Inc()
	stepDestroyCounter.Inc()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[%s] destroy panic recovered: %s", w.options.Id, r)
		}
	}()
	err = w.inner.Destroy(ctx)

	stepDestroyCounter.Desc()
	stepCounter.Desc()

	return err
}
