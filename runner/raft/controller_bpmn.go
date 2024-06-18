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

	definition := obj.(*corev1.Definition)
	if definition.Spec.Region == 0 ||
		definition.Status.Phase == corev1.DefPending {
		return
	}

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
	//logger := klog.FromContext(ctx)

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
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncDefinitionHandler(ctx context.Context, key string) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "Definition", key)

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
	logger.Info("Successfully deployed", "Definition", key)

	definition.Status.Phase = corev1.DefActive
	definition, err = c.client.CoreV1().Definitions(namespace).UpdateStatus(ctx, definition, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	logger.Info("Successfully synced", "Definition", key)

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
	req := &pb.ShardDeployDefinitionRequest{Definition: definition}
	if _, err := shard.DeployDefinition(ctx, req); err != nil {
		return err
	}

	return nil
}

func (c *Controller) enqueueProcess(obj interface{}) {
	var key string
	var err error

	process := obj.(*corev1.Process)
	if process.Status.Region == 0 {
		return
	}

	if !process.NeedScheduler() {
		return
	}

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.processQ.Add(key)
}

func (c *Controller) runProcessWorker(ctx context.Context) {
	for c.processNextBpmnProcessWorkItem(ctx) {
	}
}

func (c *Controller) processNextBpmnProcessWorkItem(ctx context.Context) bool {
	obj, shutdown := c.processQ.Get()
	//logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.processQ.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.processQ.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in defintionQ but got %#v", obj))
			return nil
		}
		if err := c.syncProcessHandler(ctx, key); err != nil {
			c.processQ.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.processQ.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncProcessHandler(ctx context.Context, key string) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "Process", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	process, err := c.client.CoreV1().Processes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	if process.Spec.BpmnProcess == "" {
		return fmt.Errorf("no bpmn process found for '%s'", key)
	}

	if err = c.runBpmnProcess(ctx, process); err != nil {
		return err
	}
	logger.Info("Successfully deployed", "Process", key)

	process.Status.Phase = corev1.ProcessRunning
	process, err = c.client.CoreV1().Processes(namespace).UpdateStatus(ctx, process, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	logger.Info("Successfully synced", "Process", key)

	return nil
}

func (c *Controller) runBpmnProcess(ctx context.Context, process *corev1.Process) error {
	lg := c.cfg.Logger
	if process.Spec.Definition == "" || process.Status.Region == 0 {
		lg.Warn("invalid process instance")
		return nil
	}

	shardId := process.Status.Region
	shard, ok := c.getShard(uint64(shardId))
	if !ok {
		lg.Info("shard running others", zap.Int64("id", shardId))
		return nil
	}

	lg.Info("definition executed",
		zap.String("definition", process.Spec.Definition),
		zap.Int64("version", process.Spec.Version))

	req := &pb.ShardRunBpmnProcessRequest{Process: process}
	resp, err := shard.RunBpmnProcess(ctx, req)
	if err != nil {
		return err
	}
	process.Status.Phase = resp.Stat.Status.Phase

	return nil
}
