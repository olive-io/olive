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

package queuesort

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
	corev1helpers "github.com/olive-io/olive/pkg/scheduling/corev1"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = names.PrioritySort

// PrioritySort is a plugin that implements Priority based sorting.
type PrioritySort struct{}

var _ framework.QueueSortPlugin = &PrioritySort{}

// Name returns name of the plugin.
func (pl *PrioritySort) Name() string {
	return Name
}

// Less is the function used by the activeQ heap algorithm to sort pods.
// It sorts pods based on their priority. When priorities are equal, it uses
// RegionQueueInfo.timestamp.
func (pl *PrioritySort) Less(rInfo1, rInfo2 *framework.QueuedRegionInfo) bool {
	p1 := corev1helpers.RegionPriority(rInfo1.Region)
	p2 := corev1helpers.RegionPriority(rInfo2.Region)
	return (p1 > p2) || (p1 == p2 && rInfo1.Timestamp.Before(rInfo2.Timestamp))
}

// New initializes a new plugin and returns it.
func New(_ context.Context, _ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &PrioritySort{}, nil
}
