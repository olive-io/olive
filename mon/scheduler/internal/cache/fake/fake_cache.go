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

package fake

import (
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
)

// Cache is used for testing
type Cache struct {
	AssumeFunc          func(*corev1.Region)
	ForgetFunc          func(*corev1.Region)
	IsAssumedRegionFunc func(*corev1.Region) bool
	GetRegionFunc       func(*corev1.Region) *corev1.Region
}

// AssumeRegion is a fake method for testing.
func (c *Cache) AssumeRegion(logger klog.Logger, pod *corev1.Region) error {
	c.AssumeFunc(pod)
	return nil
}

// FinishBinding is a fake method for testing.
func (c *Cache) FinishBinding(logger klog.Logger, pod *corev1.Region) error { return nil }

// ForgetRegion is a fake method for testing.
func (c *Cache) ForgetRegion(logger klog.Logger, pod *corev1.Region) error {
	c.ForgetFunc(pod)
	return nil
}

// AddRegion is a fake method for testing.
func (c *Cache) AddRegion(logger klog.Logger, pod *corev1.Region) error { return nil }

// UpdateRegion is a fake method for testing.
func (c *Cache) UpdateRegion(logger klog.Logger, oldRegion, newRegion *corev1.Region) error {
	return nil
}

// RemoveRegion is a fake method for testing.
func (c *Cache) RemoveRegion(logger klog.Logger, pod *corev1.Region) error { return nil }

// IsAssumedRegion is a fake method for testing.
func (c *Cache) IsAssumedRegion(pod *corev1.Region) (bool, error) {
	return c.IsAssumedRegionFunc(pod), nil
}

// GetRegion is a fake method for testing.
func (c *Cache) GetRegion(pod *corev1.Region) (*corev1.Region, error) {
	return c.GetRegionFunc(pod), nil
}

// AddRunner is a fake method for testing.
func (c *Cache) AddRunner(logger klog.Logger, node *corev1.Runner) *framework.RunnerInfo { return nil }

// UpdateRunner is a fake method for testing.
func (c *Cache) UpdateRunner(logger klog.Logger, oldRunner, newRunner *corev1.Runner) *framework.RunnerInfo {
	return nil
}

// RemoveRunner is a fake method for testing.
func (c *Cache) RemoveRunner(logger klog.Logger, node *corev1.Runner) error { return nil }

// UpdateSnapshot is a fake method for testing.
func (c *Cache) UpdateSnapshot(logger klog.Logger, snapshot *internalcache.Snapshot) error {
	return nil
}

// RunnerCount is a fake method for testing.
func (c *Cache) RunnerCount() int { return 0 }

// RegionCount is a fake method for testing.
func (c *Cache) RegionCount() (int, error) { return 0, nil }

// Dump is a fake method for testing.
func (c *Cache) Dump() *internalcache.Dump {
	return &internalcache.Dump{}
}
