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
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/klog/v2/ktesting"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	st "github.com/olive-io/olive/mon/scheduler/testing"
)

func TestPreEnqueue(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Region
		want *framework.Status
	}{
		{
			name: "pod does not carry scheduling gates",
			pod:  st.MakeRegion().Name("p").Obj(),
			want: nil,
		},
		{
			name: "pod carries scheduling gates",
			pod:  st.MakeRegion().Name("p").SchedulingGates([]string{"foo", "bar"}).Obj(),
			want: framework.NewStatus(framework.UnschedulableAndUnresolvable, "waiting for scheduling gates: [foo bar]"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			p, err := New(ctx, nil, nil)
			if err != nil {
				t.Fatalf("Creating plugin: %v", err)
			}

			got := p.(framework.PreEnqueuePlugin).PreEnqueue(ctx, tt.pod)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected status (-want, +got):\n%s", diff)
			}
		})
	}
}
