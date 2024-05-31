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

package queue

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/olive-io/olive/client-go/generated/clientset/versioned/fake"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// NewTestQueue creates a priority queue with an empty informer factory.
func NewTestQueue(ctx context.Context, lessFn framework.LessFunc, opts ...Option) *PriorityQueue {
	return NewTestQueueWithObjects(ctx, lessFn, nil, opts...)
}

// NewTestQueueWithObjects creates a priority queue with an informer factory
// populated with the provided objects.
func NewTestQueueWithObjects(
	ctx context.Context,
	lessFn framework.LessFunc,
	objs []runtime.Object,
	opts ...Option,
) *PriorityQueue {
	informerFactory := informers.NewSharedInformerFactory(fake.NewSimpleClientset(objs...), 0)
	return NewTestQueueWithInformerFactory(ctx, lessFn, informerFactory, opts...)
}

func NewTestQueueWithInformerFactory(
	ctx context.Context,
	lessFn framework.LessFunc,
	informerFactory informers.SharedInformerFactory,
	opts ...Option,
) *PriorityQueue {
	pq := NewPriorityQueue(lessFn, informerFactory, opts...)
	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())
	return pq
}
