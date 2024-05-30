/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
)

// Cache is used for testing
type Cache struct {
	AssumeFunc              func(*corev1.Definition)
	ForgetFunc              func(*corev1.Definition)
	IsAssumedDefinitionFunc func(*corev1.Definition) bool
	GetDefinitionFunc       func(*corev1.Definition) *corev1.Definition
}

// AssumeDefinition is a fake method for testing.
func (c *Cache) AssumeDefinition(logger klog.Logger, pod *corev1.Definition) error {
	c.AssumeFunc(pod)
	return nil
}

// FinishBinding is a fake method for testing.
func (c *Cache) FinishBinding(logger klog.Logger, pod *corev1.Definition) error { return nil }

// ForgetDefinition is a fake method for testing.
func (c *Cache) ForgetDefinition(logger klog.Logger, pod *corev1.Definition) error {
	c.ForgetFunc(pod)
	return nil
}

// AddDefinition is a fake method for testing.
func (c *Cache) AddDefinition(logger klog.Logger, pod *corev1.Definition) error { return nil }

// UpdateDefinition is a fake method for testing.
func (c *Cache) UpdateDefinition(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition) error {
	return nil
}

// RemoveDefinition is a fake method for testing.
func (c *Cache) RemoveDefinition(logger klog.Logger, pod *corev1.Definition) error { return nil }

// IsAssumedDefinition is a fake method for testing.
func (c *Cache) IsAssumedDefinition(pod *corev1.Definition) (bool, error) {
	return c.IsAssumedDefinitionFunc(pod), nil
}

// GetDefinition is a fake method for testing.
func (c *Cache) GetDefinition(pod *corev1.Definition) (*corev1.Definition, error) {
	return c.GetDefinitionFunc(pod), nil
}

// AddRunner is a fake method for testing.
func (c *Cache) AddRunner(logger klog.Logger, node *monv1.Runner) *framework.RunnerInfo { return nil }

// UpdateRunner is a fake method for testing.
func (c *Cache) UpdateRunner(logger klog.Logger, oldRunner, newRunner *monv1.Runner) *framework.RunnerInfo {
	return nil
}

// RemoveRunner is a fake method for testing.
func (c *Cache) RemoveRunner(logger klog.Logger, node *monv1.Runner) error { return nil }

// UpdateSnapshot is a fake method for testing.
func (c *Cache) UpdateSnapshot(logger klog.Logger, snapshot *internalcache.Snapshot) error {
	return nil
}

// RunnerCount is a fake method for testing.
func (c *Cache) RunnerCount() int { return 0 }

// DefinitionCount is a fake method for testing.
func (c *Cache) DefinitionCount() (int, error) { return 0, nil }

// Dump is a fake method for testing.
func (c *Cache) Dump() *internalcache.Dump {
	return &internalcache.Dump{}
}
