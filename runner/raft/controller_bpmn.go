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

package raft

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	pb "github.com/olive-io/olive/apis/pb/olive"
)

func (c *Controller) enqueueDef(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.definitionQ.Add(key)
}

func (c *Controller) runDefinitionWorker(ctx context.Context) {
	for c.processNextDefinitionWorkItem(ctx) {
	}
}

func (c *Controller) processNextDefinitionWorkItem(ctx context.Context) bool {
	obj, shutdown := c.definitionQ.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.definitionQ.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.definitionQ.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in defintionQ but got %#v", obj))
			return nil
		}
		if err := c.syncDefinitionHandler(ctx, key); err != nil {
			c.definitionQ.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.definitionQ.Forget(obj)
		logger.Info("Successfully synced", "resourceName", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncDefinitionHandler(ctx context.Context, key string) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "resourceName", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	definition, err := c.client.CoreV1().Definitions(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	if definition.Spec.Region == 0 {
		return nil
	}

	if err = c.deployDefinition(ctx, definition); err != nil {
		return err
	}
	logger.Info("Successfully deployed", "resourceName", key)

	definition.Status.Phase = corev1.DefActive
	definition, err = c.client.CoreV1().Definitions(namespace).UpdateStatus(ctx, definition, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	logger.Info("Successfully synced", "resourceName", key)

	return nil
}

func (c *Controller) deployDefinition(ctx context.Context, definition *corev1.Definition) error {
	lg := c.cfg.Logger
	if definition.Spec.Region == 0 {
		lg.Warn("definition missing region",
			zap.String("name", definition.Name),
			zap.Int64("version", definition.Spec.Version))
		return nil
	}

	shardId := definition.Spec.Region
	shard, ok := c.getShard(uint64(shardId))
	if !ok {
		lg.Info("region running others",
			zap.Int64("id", shardId))
		return nil
	}

	if !shard.isLeader() {
		return nil
	}

	lg.Info("definition deploy",
		zap.String("name", definition.Name),
		zap.Int64("version", definition.Spec.Version))
	defClone := definition.DeepCopy()
	defClone.TypeMeta = metav1.TypeMeta{}
	req := &pb.ShardDeployDefinitionRequest{Definition: defClone}
	if _, err := shard.DeployDefinition(ctx, req); err != nil {
		return err
	}

	return nil
}
