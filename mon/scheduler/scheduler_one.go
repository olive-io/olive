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

package scheduler

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

func (s *Scheduler) enqueueDefinition(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	s.definitionQ.Add(key)
}

func (s *Scheduler) definitionWorker(ctx context.Context) {
	for s.processNextDefinitionWorkItem(ctx) {
	}
}

func (s *Scheduler) processNextDefinitionWorkItem(ctx context.Context) bool {
	obj, shutdown := s.definitionQ.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	if !s.leaderReady() {
		return false
	}

	err := func(obj interface{}) error {
		defer s.definitionQ.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			s.definitionQ.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := s.syncDefinitionHandler(ctx, key); err != nil {
			s.definitionQ.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		s.definitionQ.Forget(obj)
		logger.V(4).Info("Successfully synced", "Definition", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (s *Scheduler) syncDefinitionHandler(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "resourceName", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Definition resource with this namespace/name
	def, err := s.clientSet.CoreV1().Definitions(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("definition '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	if def.Status.Phase != corev1.DefPending {
		return nil
	}

	if def.Spec.Region == 0 {
		snapshot, ok := s.regionQ.Pop()
		if !ok {
			return nil
		}
		region := snapshot.Get()
		def.Spec.Region = region.Spec.Id
		_, err = s.clientSet.CoreV1().Definitions(namespace).Update(ctx, def, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		logger.Info(fmt.Sprintf("binding Definition %s to Region %s", def.Name, region.Name))
	}

	return nil
}
