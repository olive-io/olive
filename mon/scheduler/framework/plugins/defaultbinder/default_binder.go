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

package defaultbinder

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
)

// Name of the plugin used in the plugin registry and configurations.
const Name = names.DefaultBinder

// DefaultBinder binds regions to runners using a k8s client.
type DefaultBinder struct {
	handle framework.Handle
}

var _ framework.BindPlugin = &DefaultBinder{}

// New creates a DefaultBinder.
func New(_ context.Context, _ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &DefaultBinder{handle: handle}, nil
}

// Name returns the name of the plugin.
func (b DefaultBinder) Name() string {
	return Name
}

// Bind binds regions to runners using the k8s client.
func (b DefaultBinder) Bind(ctx context.Context, state *framework.CycleState, p *corev1.Region, runnerNames ...string) *framework.Status {
	logger := klog.FromContext(ctx)
	logger.V(3).Info("Attempting to bind region to runners", "region", klog.KObj(p), "runner", klog.KRef("", strings.Join(runnerNames, ",")))
	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{Namespace: p.Namespace, Name: p.Name, UID: p.UID},
		Target:     corev1.ObjectReference{Kind: "Runner", Names: runnerNames},
	}
	err := b.handle.ClientSet().CoreV1().Regions().Bind(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		return framework.AsStatus(err)
	}
	return nil
}
