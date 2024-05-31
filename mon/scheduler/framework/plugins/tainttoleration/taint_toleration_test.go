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

package tainttoleration

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/ktesting"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
	"github.com/olive-io/olive/mon/scheduler/internal/cache"
	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
)

func runnerWithTaints(runnerName string, taints []corev1.Taint) *corev1.Runner {
	return &corev1.Runner{
		ObjectMeta: metav1.ObjectMeta{
			Name: runnerName,
		},
		Spec: corev1.RunnerSpec{
			Taints: taints,
		},
	}
}

func regionWithTolerations(regionName string, tolerations []corev1.Toleration) *corev1.Region {
	return &corev1.Region{
		ObjectMeta: metav1.ObjectMeta{
			Name: regionName,
		},
		Spec: corev1.RegionSpec{
			Tolerations: tolerations,
		},
	}
}

func TestTaintTolerationScore(t *testing.T) {
	tests := []struct {
		name         string
		region       *corev1.Region
		runners      []*corev1.Runner
		expectedList framework.RunnerScoreList
	}{
		// basic test case
		{
			name: "runner with taints tolerated by the region, gets a higher score than those runner with intolerable taints",
			region: regionWithTolerations("region1", []corev1.Toleration{{
				Key:      "foo",
				Operator: corev1.TolerationOpEqual,
				Value:    "bar",
				Effect:   corev1.TaintEffectPreferNoSchedule,
			}}),
			runners: []*corev1.Runner{
				runnerWithTaints("runnerA", []corev1.Taint{{
					Key:    "foo",
					Value:  "bar",
					Effect: corev1.TaintEffectPreferNoSchedule,
				}}),
				runnerWithTaints("runnerB", []corev1.Taint{{
					Key:    "foo",
					Value:  "blah",
					Effect: corev1.TaintEffectPreferNoSchedule,
				}}),
			},
			expectedList: []framework.RunnerScore{
				{Name: "runnerA", Score: framework.MaxRunnerScore},
				{Name: "runnerB", Score: 0},
			},
		},
		// the count of taints that are tolerated by region, does not matter.
		{
			name: "the runners that all of their taints are tolerated by the region, get the same score, no matter how many tolerable taints a runner has",
			region: regionWithTolerations("region1", []corev1.Toleration{
				{
					Key:      "cpu-type",
					Operator: corev1.TolerationOpEqual,
					Value:    "arm64",
					Effect:   corev1.TaintEffectPreferNoSchedule,
				}, {
					Key:      "disk-type",
					Operator: corev1.TolerationOpEqual,
					Value:    "ssd",
					Effect:   corev1.TaintEffectPreferNoSchedule,
				},
			}),
			runners: []*corev1.Runner{
				runnerWithTaints("runnerA", []corev1.Taint{}),
				runnerWithTaints("runnerB", []corev1.Taint{
					{
						Key:    "cpu-type",
						Value:  "arm64",
						Effect: corev1.TaintEffectPreferNoSchedule,
					},
				}),
				runnerWithTaints("runnerC", []corev1.Taint{
					{
						Key:    "cpu-type",
						Value:  "arm64",
						Effect: corev1.TaintEffectPreferNoSchedule,
					}, {
						Key:    "disk-type",
						Value:  "ssd",
						Effect: corev1.TaintEffectPreferNoSchedule,
					},
				}),
			},
			expectedList: []framework.RunnerScore{
				{Name: "runnerA", Score: framework.MaxRunnerScore},
				{Name: "runnerB", Score: framework.MaxRunnerScore},
				{Name: "runnerC", Score: framework.MaxRunnerScore},
			},
		},
		// the count of taints on a runner that are not tolerated by region, matters.
		{
			name: "the more intolerable taints a runner has, the lower score it gets.",
			region: regionWithTolerations("region1", []corev1.Toleration{{
				Key:      "foo",
				Operator: corev1.TolerationOpEqual,
				Value:    "bar",
				Effect:   corev1.TaintEffectPreferNoSchedule,
			}}),
			runners: []*corev1.Runner{
				runnerWithTaints("runnerA", []corev1.Taint{}),
				runnerWithTaints("runnerB", []corev1.Taint{
					{
						Key:    "cpu-type",
						Value:  "arm64",
						Effect: corev1.TaintEffectPreferNoSchedule,
					},
				}),
				runnerWithTaints("runnerC", []corev1.Taint{
					{
						Key:    "cpu-type",
						Value:  "arm64",
						Effect: corev1.TaintEffectPreferNoSchedule,
					}, {
						Key:    "disk-type",
						Value:  "ssd",
						Effect: corev1.TaintEffectPreferNoSchedule,
					},
				}),
			},
			expectedList: []framework.RunnerScore{
				{Name: "runnerA", Score: framework.MaxRunnerScore},
				{Name: "runnerB", Score: 50},
				{Name: "runnerC", Score: 0},
			},
		},
		// taints-tolerations priority only takes care about the taints and tolerations that have effect PreferNoSchedule
		{
			name: "only taints and tolerations that have effect PreferNoSchedule are checked by taints-tolerations priority function",
			region: regionWithTolerations("region1", []corev1.Toleration{
				{
					Key:      "cpu-type",
					Operator: corev1.TolerationOpEqual,
					Value:    "arm64",
					Effect:   corev1.TaintEffectNoSchedule,
				}, {
					Key:      "disk-type",
					Operator: corev1.TolerationOpEqual,
					Value:    "ssd",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}),
			runners: []*corev1.Runner{
				runnerWithTaints("runnerA", []corev1.Taint{}),
				runnerWithTaints("runnerB", []corev1.Taint{
					{
						Key:    "cpu-type",
						Value:  "arm64",
						Effect: corev1.TaintEffectNoSchedule,
					},
				}),
				runnerWithTaints("runnerC", []corev1.Taint{
					{
						Key:    "cpu-type",
						Value:  "arm64",
						Effect: corev1.TaintEffectPreferNoSchedule,
					}, {
						Key:    "disk-type",
						Value:  "ssd",
						Effect: corev1.TaintEffectPreferNoSchedule,
					},
				}),
			},
			expectedList: []framework.RunnerScore{
				{Name: "runnerA", Score: framework.MaxRunnerScore},
				{Name: "runnerB", Score: framework.MaxRunnerScore},
				{Name: "runnerC", Score: 0},
			},
		},
		{
			name: "Default behaviour No taints and tolerations, lands on runner with no taints",
			//region without tolerations
			region: regionWithTolerations("region1", []corev1.Toleration{}),
			runners: []*corev1.Runner{
				//Runner without taints
				runnerWithTaints("runnerA", []corev1.Taint{}),
				runnerWithTaints("runnerB", []corev1.Taint{
					{
						Key:    "cpu-type",
						Value:  "arm64",
						Effect: corev1.TaintEffectPreferNoSchedule,
					},
				}),
			},
			expectedList: []framework.RunnerScore{
				{Name: "runnerA", Score: framework.MaxRunnerScore},
				{Name: "runnerB", Score: 0},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			state := framework.NewCycleState()
			snapshot := cache.NewSnapshot(nil, test.runners)
			fh, _ := runtime.NewFramework(ctx, nil, nil, runtime.WithSnapshotSharedLister(snapshot))

			p, err := New(ctx, nil, fh)
			if err != nil {
				t.Fatalf("creating plugin: %v", err)
			}
			status := p.(framework.PreScorePlugin).PreScore(ctx, state, test.region, tf.BuildRunnerInfos(test.runners))
			if !status.IsSuccess() {
				t.Errorf("unexpected error: %v", status)
			}
			var gotList framework.RunnerScoreList
			for _, n := range test.runners {
				runnerName := n.ObjectMeta.Name
				score, status := p.(framework.ScorePlugin).Score(ctx, state, test.region, runnerName)
				if !status.IsSuccess() {
					t.Errorf("unexpected error: %v", status)
				}
				gotList = append(gotList, framework.RunnerScore{Name: runnerName, Score: score})
			}

			status = p.(framework.ScorePlugin).ScoreExtensions().NormalizeScore(ctx, state, test.region, gotList)
			if !status.IsSuccess() {
				t.Errorf("unexpected error: %v", status)
			}

			if !reflect.DeepEqual(test.expectedList, gotList) {
				t.Errorf("expected:\n\t%+v,\ngot:\n\t%+v", test.expectedList, gotList)
			}
		})
	}
}

func TestTaintTolerationFilter(t *testing.T) {
	tests := []struct {
		name       string
		region     *corev1.Region
		runner     *corev1.Runner
		wantStatus *framework.Status
	}{
		{
			name:   "A region having no tolerations can't be scheduled onto a runner with nonempty taints",
			region: regionWithTolerations("region1", []corev1.Toleration{}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "dedicated", Value: "user1", Effect: "NoSchedule"}}),
			wantStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable,
				"runner(s) had untolerated taint {dedicated: user1}"),
		},
		{
			name:   "A region which can be scheduled on a dedicated runner assigned to user1 with effect NoSchedule",
			region: regionWithTolerations("region1", []corev1.Toleration{{Key: "dedicated", Value: "user1", Effect: "NoSchedule"}}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "dedicated", Value: "user1", Effect: "NoSchedule"}}),
		},
		{
			name:   "A region which can't be scheduled on a dedicated runner assigned to user2 with effect NoSchedule",
			region: regionWithTolerations("region1", []corev1.Toleration{{Key: "dedicated", Operator: "Equal", Value: "user2", Effect: "NoSchedule"}}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "dedicated", Value: "user1", Effect: "NoSchedule"}}),
			wantStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable,
				"runner(s) had untolerated taint {dedicated: user1}"),
		},
		{
			name:   "A region can be scheduled onto the runner, with a toleration uses operator Exists that tolerates the taints on the runner",
			region: regionWithTolerations("region1", []corev1.Toleration{{Key: "foo", Operator: "Exists", Effect: "NoSchedule"}}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "foo", Value: "bar", Effect: "NoSchedule"}}),
		},
		{
			name: "A region has multiple tolerations, runner has multiple taints, all the taints are tolerated, region can be scheduled onto the runner",
			region: regionWithTolerations("region1", []corev1.Toleration{
				{Key: "dedicated", Operator: "Equal", Value: "user2", Effect: "NoSchedule"},
				{Key: "foo", Operator: "Exists", Effect: "NoSchedule"},
			}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{
				{Key: "dedicated", Value: "user2", Effect: "NoSchedule"},
				{Key: "foo", Value: "bar", Effect: "NoSchedule"},
			}),
		},
		{
			name: "A region has a toleration that keys and values match the taint on the runner, but (non-empty) effect doesn't match, " +
				"can't be scheduled onto the runner",
			region: regionWithTolerations("region1", []corev1.Toleration{{Key: "foo", Operator: "Equal", Value: "bar", Effect: "PreferNoSchedule"}}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "foo", Value: "bar", Effect: "NoSchedule"}}),
			wantStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable,
				"runner(s) had untolerated taint {foo: bar}"),
		},
		{
			name: "The region has a toleration that keys and values match the taint on the runner, the effect of toleration is empty, " +
				"and the effect of taint is NoSchedule. Region can be scheduled onto the runner",
			region: regionWithTolerations("region1", []corev1.Toleration{{Key: "foo", Operator: "Equal", Value: "bar"}}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "foo", Value: "bar", Effect: "NoSchedule"}}),
		},
		{
			name: "The region has a toleration that key and value don't match the taint on the runner, " +
				"but the effect of taint on runner is PreferNoSchedule. Region can be scheduled onto the runner",
			region: regionWithTolerations("region1", []corev1.Toleration{{Key: "dedicated", Operator: "Equal", Value: "user2", Effect: "NoSchedule"}}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "dedicated", Value: "user1", Effect: "PreferNoSchedule"}}),
		},
		{
			name: "The region has no toleration, " +
				"but the effect of taint on runner is PreferNoSchedule. Region can be scheduled onto the runner",
			region: regionWithTolerations("region1", []corev1.Toleration{}),
			runner: runnerWithTaints("runnerA", []corev1.Taint{{Key: "dedicated", Value: "user1", Effect: "PreferNoSchedule"}}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			runnerInfo := framework.NewRunnerInfo()
			runnerInfo.SetRunner(test.runner)
			p, err := New(ctx, nil, nil)
			if err != nil {
				t.Fatalf("creating plugin: %v", err)
			}
			gotStatus := p.(framework.FilterPlugin).Filter(ctx, nil, test.region, runnerInfo)
			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
				t.Errorf("status does not match: %v, want: %v", gotStatus, test.wantStatus)
			}
		})
	}
}
