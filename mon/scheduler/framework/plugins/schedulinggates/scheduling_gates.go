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

package schedulinggates

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
)

// Name of the plugin used in the plugin registry and configurations.
const Name = names.SchedulingGates

// SchedulingGates checks if a Region carries .spec.schedulingGates.
type SchedulingGates struct{}

var _ framework.PreEnqueuePlugin = &SchedulingGates{}
var _ framework.EnqueueExtensions = &SchedulingGates{}

func (pl *SchedulingGates) Name() string {
	return Name
}

func (pl *SchedulingGates) PreEnqueue(ctx context.Context, p *corev1.Region) *framework.Status {
	if len(p.Spec.SchedulingGates) == 0 {
		return nil
	}
	gates := make([]string, 0, len(p.Spec.SchedulingGates))
	for _, gate := range p.Spec.SchedulingGates {
		gates = append(gates, gate.Name)
	}
	return framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("waiting for scheduling gates: %v", gates))
}

// EventsToRegister returns nil here to indicate that schedulingGates plugin is not
// interested in any event but its own update.
func (pl *SchedulingGates) EventsToRegister() []framework.ClusterEventWithHint {
	return nil
}

// New initializes a new plugin and returns it.
func New(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &SchedulingGates{}, nil
}
