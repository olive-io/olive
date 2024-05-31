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

package helper

import (
	"testing"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

func TestDoNotScheduleTaintsFilterFunc(t *testing.T) {
	tests := []struct {
		name     string
		taint    *corev1.Taint
		expected bool
	}{
		{
			name: "should include the taints with NoSchedule effect",
			taint: &corev1.Taint{
				Effect: corev1.TaintEffectNoSchedule,
			},
			expected: true,
		},
		{
			name: "should include the taints with NoExecute effect",
			taint: &corev1.Taint{
				Effect: corev1.TaintEffectNoExecute,
			},
			expected: true,
		},
		{
			name: "should not include the taints with PreferNoSchedule effect",
			taint: &corev1.Taint{
				Effect: corev1.TaintEffectPreferNoSchedule,
			},
			expected: false,
		},
	}

	filterPredicate := DoNotScheduleTaintsFilterFunc()

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			if got := filterPredicate(test.taint); got != test.expected {
				t.Errorf("unexpected result, expected %v but got %v", test.expected, got)
			}
		})
	}
}
