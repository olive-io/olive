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

package runnerunschedulable

import (
	"reflect"
	"testing"

	"k8s.io/klog/v2/ktesting"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

func TestRunnerUnschedulable(t *testing.T) {
	testCases := []struct {
		name       string
		pod        *corev1.Region
		runner     *corev1.Runner
		wantStatus *framework.Status
	}{
		{
			name: "Does not schedule pod to unschedulable runner (runner.Spec.Unschedulable==true)",
			pod:  &corev1.Region{},
			runner: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: true,
				},
			},
			wantStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonUnschedulable),
		},
		{
			name: "Schedule pod to normal runner",
			pod:  &corev1.Region{},
			runner: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: false,
				},
			},
		},
		{
			name: "Schedule pod with toleration to unschedulable runner (runner.Spec.Unschedulable==true)",
			pod: &corev1.Region{
				Spec: corev1.RegionSpec{
					Tolerations: []corev1.Toleration{
						{
							Key:    corev1.TaintRunnerUnschedulable,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
			runner: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: true,
				},
			},
		},
	}

	for _, test := range testCases {
		runnerInfo := framework.NewRunnerInfo()
		runnerInfo.SetRunner(test.runner)
		_, ctx := ktesting.NewTestContext(t)
		p, err := New(ctx, nil, nil)
		if err != nil {
			t.Fatalf("creating plugin: %v", err)
		}
		gotStatus := p.(framework.FilterPlugin).Filter(ctx, nil, test.pod, runnerInfo)
		if !reflect.DeepEqual(gotStatus, test.wantStatus) {
			t.Errorf("status does not match: %v, want: %v", gotStatus, test.wantStatus)
		}
	}
}

func TestIsSchedulableAfterRunnerChange(t *testing.T) {
	testCases := []struct {
		name           string
		pod            *corev1.Region
		oldObj, newObj interface{}
		expectedHint   framework.QueueingHint
		expectedErr    bool
	}{
		{
			name:         "backoff-wrong-new-object",
			pod:          &corev1.Region{},
			newObj:       "not-a-runner",
			expectedHint: framework.Queue,
			expectedErr:  true,
		},
		{
			name: "backoff-wrong-old-object",
			pod:  &corev1.Region{},
			newObj: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: true,
				},
			},
			oldObj:       "not-a-runner",
			expectedHint: framework.Queue,
			expectedErr:  true,
		},
		{
			name: "skip-queue-on-unschedulable-runner-added",
			pod:  &corev1.Region{},
			newObj: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: true,
				},
			},
			expectedHint: framework.QueueSkip,
		},
		{
			name: "queue-on-schedulable-runner-added",
			pod:  &corev1.Region{},
			newObj: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: false,
				},
			},
			expectedHint: framework.Queue,
		},
		{
			name: "skip-unrelated-change",
			pod:  &corev1.Region{},
			newObj: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: true,
					Taints: []corev1.Taint{
						{
							Key:    corev1.TaintRunnerNotReady,
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},
			oldObj: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: true,
				},
			},
			expectedHint: framework.QueueSkip,
		},
		{
			name: "queue-on-unschedulable-field-change",
			pod:  &corev1.Region{},
			newObj: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: false,
				},
			},
			oldObj: &corev1.Runner{
				Spec: corev1.RunnerSpec{
					Unschedulable: true,
				},
			},
			expectedHint: framework.Queue,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			logger, _ := ktesting.NewTestContext(t)
			pl := &RunnerUnschedulable{}
			got, err := pl.isSchedulableAfterRunnerChange(logger, testCase.pod, testCase.oldObj, testCase.newObj)
			if err != nil && !testCase.expectedErr {
				t.Errorf("unexpected error: %v", err)
			}
			if got != testCase.expectedHint {
				t.Errorf("isSchedulableAfterRunnerChange() = %v, want %v", got, testCase.expectedHint)
			}
		})
	}
}
