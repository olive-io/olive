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

package runnerresources
//
//import (
//	"context"
//	"testing"
//
//	"github.com/google/go-cmp/cmp"
//	v1 "k8s.io/api/core/v1"
//	"k8s.io/apimachinery/pkg/util/validation/field"
//	"k8s.io/klog/v2/ktesting"
//
//	config "github.com/olive-io/olive/apis/config/v1"
//	corev1 "github.com/olive-io/olive/apis/core/v1"
//	"github.com/olive-io/olive/mon/scheduler/framework"
//	plfeature "github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
//	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
//	"github.com/olive-io/olive/mon/scheduler/internal/cache"
//	st "github.com/olive-io/olive/mon/scheduler/testing"
//	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
//)
//
//func TestLeastAllocatedScoringStrategy(t *testing.T) {
//	tests := []struct {
//		name            string
//		requestedRegion *corev1.Region
//		runners         []*corev1.Runner
//		existingRegions []*corev1.Region
//		expectedScores  framework.RunnerScoreList
//		resources       []config.ResourceSpec
//		wantErrs        field.ErrorList
//		wantStatusCode  framework.Code
//	}{
//		{
//			// Runner1 scores (remaining resources) on 0-MaxRunnerScore scale
//			// CPU Score: ((4000 - 0) * MaxRunnerScore) / 4000 = MaxRunnerScore
//			// Memory Score: ((10000 - 0) * MaxRunnerScore) / 10000 = MaxRunnerScore
//			// Runner1 Score: (100 + 100) / 2 = 100
//			// Runner2 scores (remaining resources) on 0-MaxRunnerScore scale
//			// CPU Score: ((4000 - 0) * MaxRunnerScore) / 4000 = MaxRunnerScore
//			// Memory Score: ((10000 - 0) * MaxRunnerScore) / 10000 = MaxRunnerScore
//			// Runner2 Score: (MaxRunnerScore + MaxRunnerScore) / 2 = MaxRunnerScore
//			name:            "nothing scheduled, nothing requested",
//			requestedRegion: st.MakeRegion().Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: framework.MaxRunnerScore}},
//			resources:       defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((4000 - 3000) * MaxRunnerScore) / 4000 = 25
//			// Memory Score: ((10000 - 5000) * MaxRunnerScore) / 10000 = 50
//			// Runner1 Score: (25 + 50) / 2 = 37
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((6000 - 3000) * MaxRunnerScore) / 6000 = 50
//			// Memory Score: ((10000 - 5000) * MaxRunnerScore) / 10000 = 50
//			// Runner2 Score: (50 + 50) / 2 = 50
//			name: "nothing scheduled, resources requested, differently sized runners",
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 37}, {Name: "runner2", Score: 50}},
//			resources:       defaultResources,
//		},
//		{
//			name: "Resources not set, regions scheduled with error",
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: framework.MinRunnerScore}, {Name: "runner2", Score: framework.MinRunnerScore}},
//			resources:       nil,
//			wantStatusCode:  framework.Error,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((4000 - 0) * MaxRunnerScore) / 4000 = MaxRunnerScore
//			// Memory Score: ((10000 - 0) * MaxRunnerScore) / 10000 = MaxRunnerScore
//			// Runner1 Score: (MaxRunnerScore + MaxRunnerScore) / 2 = MaxRunnerScore
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((4000 - 0) * MaxRunnerScore) / 4000 = MaxRunnerScore
//			// Memory Score: ((10000 - 0) * MaxRunnerScore) / 10000 = MaxRunnerScore
//			// Runner2 Score: (MaxRunnerScore + MaxRunnerScore) / 2 = MaxRunnerScore
//			name:            "no resources requested, regions scheduled",
//			requestedRegion: st.MakeRegion().Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Obj(),
//				st.MakeRegion().Runner("runner1").Obj(),
//				st.MakeRegion().Runner("runner2").Obj(),
//				st.MakeRegion().Runner("runner2").Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: framework.MaxRunnerScore}},
//			resources:      defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((10000 - 6000) * MaxRunnerScore) / 10000 = 40
//			// Memory Score: ((20000 - 0) * MaxRunnerScore) / 20000 = MaxRunnerScore
//			// Runner1 Score: (40 + 100) / 2 = 70
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((10000 - 6000) * MaxRunnerScore) / 10000 = 40
//			// Memory Score: ((20000 - 5000) * MaxRunnerScore) / 20000 = 75
//			// Runner2 Score: (40 + 75) / 2 = 57
//			name:            "no resources requested, regions scheduled with resources",
//			requestedRegion: st.MakeRegion().Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "10000", "memory": "20000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "10000", "memory": "20000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//				st.MakeRegion().Runner("runner1").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 70}, {Name: "runner2", Score: 57}},
//			resources:      defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((10000 - 6000) * MaxRunnerScore) / 10000 = 40
//			// Memory Score: ((20000 - 5000) * MaxRunnerScore) / 20000 = 75
//			// Runner1 Score: (40 + 75) / 2 = 57
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((10000 - 6000) * MaxRunnerScore) / 10000 = 40
//			// Memory Score: ((20000 - 10000) * MaxRunnerScore) / 20000 = 50
//			// Runner2 Score: (40 + 50) / 2 = 45
//			name: "resources requested, regions scheduled with resources",
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "10000", "memory": "20000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "10000", "memory": "20000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 57}, {Name: "runner2", Score: 45}},
//			resources:      defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((10000 - 6000) * MaxRunnerScore) / 10000 = 40
//			// Memory Score: ((20000 - 5000) * MaxRunnerScore) / 20000 = 75
//			// Runner1 Score: (40 + 75) / 2 = 57
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((10000 - 6000) * MaxRunnerScore) / 10000 = 40
//			// Memory Score: ((50000 - 10000) * MaxRunnerScore) / 50000 = 80
//			// Runner2 Score: (40 + 80) / 2 = 60
//			name: "resources requested, regions scheduled with resources, differently sized runners",
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "10000", "memory": "20000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "10000", "memory": "50000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 57}, {Name: "runner2", Score: 60}},
//			resources:      defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((4000 - 6000) * MaxRunnerScore) / 4000 = 0
//			// Memory Score: ((10000 - 0) * MaxRunnerScore) / 10000 = MaxRunnerScore
//			// Runner1 Score: (0 + MaxRunnerScore) / 2 = 50
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: ((4000 - 6000) * MaxRunnerScore) / 4000 = 0
//			// Memory Score: ((10000 - 5000) * MaxRunnerScore) / 10000 = 50
//			// Runner2 Score: (0 + 50) / 2 = 25
//			name:            "requested resources exceed runner capacity",
//			requestedRegion: st.MakeRegion().Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 50}, {Name: "runner2", Score: 25}},
//			resources:      defaultResources,
//		},
//		{
//			name:            "zero runner resources, regions scheduled with resources",
//			requestedRegion: st.MakeRegion().Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Obj(),
//				st.MakeRunner().Name("runner2").Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "0"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: framework.MinRunnerScore}, {Name: "runner2", Score: framework.MinRunnerScore}},
//			resources:      defaultResources,
//		},
//		{
//			// CPU Score: ((4000 - 3000) *100) / 4000 = 25
//			// Memory Score: ((10000 - 5000) *100) / 10000 = 50
//			// Runner1 Score: (25 * 1 + 50 * 2) / (1 + 2) = 41
//			// CPU Score: ((6000 - 3000) *100) / 6000 = 50
//			// Memory Score: ((10000 - 5000) *100) / 10000 = 50
//			// Runner2 Score: (50 * 1 + 50 * 2) / (1 + 2) = 50
//			name: "nothing scheduled, resources requested with different weight on CPU and memory, differently sized runners",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 41}, {Name: "runner2", Score: 50}},
//			resources: []config.ResourceSpec{
//				{Name: "memory", Weight: 2},
//				{Name: "cpu", Weight: 1},
//			},
//		},
//		{
//			// resource with negative weight is not allowed
//			name: "resource with negative weight",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			resources: []config.ResourceSpec{
//				{Name: "memory", Weight: -1},
//				{Name: "cpu", Weight: 1},
//			},
//			wantErrs: field.ErrorList{
//				&field.Error{
//					Type:  field.ErrorTypeInvalid,
//					Field: "scoringStrategy.resources[0].weight",
//				},
//			},
//		},
//		{
//			// resource with zero weight is not allowed
//			name: "resource with zero weight",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 41}, {Name: "runner2", Score: 50}},
//			resources: []config.ResourceSpec{
//				{Name: "memory", Weight: 1},
//				{Name: "cpu", Weight: 0},
//			},
//			wantErrs: field.ErrorList{
//				&field.Error{
//					Type:  field.ErrorTypeInvalid,
//					Field: "scoringStrategy.resources[1].weight",
//				},
//			},
//		},
//		{
//			// resource weight should be less than MaxRunnerScore
//			name: "resource weight larger than MaxRunnerScore",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			resources: []config.ResourceSpec{
//				{Name: "memory", Weight: 1},
//				{Name: "cpu", Weight: 101},
//			},
//			wantErrs: field.ErrorList{
//				&field.Error{
//					Type:  field.ErrorTypeInvalid,
//					Field: "scoringStrategy.resources[1].weight",
//				},
//			},
//		},
//		{
//			// Bypass extended resource if the region does not request.
//			// For both runners: cpuScore and memScore are 50
//			// Given that extended resource score are intentionally bypassed,
//			// the final scores are:
//			// - runner1: (50 + 50) / 2 = 50
//			// - runner2: (50 + 50) / 2 = 50
//			name: "bypass extended resource if the region does not request",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "4"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 50}, {Name: "runner2", Score: 50}},
//			resources:      extendedResourceSet,
//		},
//		{
//			// Honor extended resource if the region requests.
//			// For both runners: cpuScore and memScore are 50.
//			// In terms of extended resource score:
//			// - runner1 get: 2 / 4 * 100 = 50
//			// - runner2 get: (10 - 2) / 10 * 100 = 80
//			// So the final scores are:
//			// - runner1: (50 + 50 + 50) / 3 = 50
//			// - runner2: (50 + 50 + 80) / 3 = 60
//			name: "honor extended resource if the region requests",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000", v1.ResourceName(extendedRes): "2"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "4"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "10"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 50}, {Name: "runner2", Score: 60}},
//			resources:       extendedResourceSet,
//		},
//		{
//			// If the runner doesn't have a resource
//			// CPU Score: ((6000 - 3000) * 100) / 6000 = 50
//			// Memory Score: ((10000 - 4000) * 100) / 10000 = 60
//			// Runner1 Score: (50 * 1 + 60 * 1) / (1 + 1) = 55
//			// Runner2 Score: (50 * 1 + 60 * 1) / (1 + 1) = 55
//			name: "if the runner doesn't have a resource",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "4000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "4"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 55}, {Name: "runner2", Score: 55}},
//			resources: []config.ResourceSpec{
//				{Name: extendedRes, Weight: 2},
//				{Name: string(v1.ResourceCPU), Weight: 1},
//				{Name: string(v1.ResourceMemory), Weight: 1},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//
//			state := framework.NewCycleState()
//			snapshot := cache.NewSnapshot(test.existingRegions, test.runners)
//			fh, _ := runtime.NewFramework(ctx, nil, nil, runtime.WithSnapshotSharedLister(snapshot))
//
//			p, err := NewFit(
//				ctx,
//				&config.RunnerResourcesFitArgs{
//					ScoringStrategy: &config.ScoringStrategy{
//						Type:      config.LeastAllocated,
//						Resources: test.resources,
//					},
//				}, fh, plfeature.Features{})
//
//			if diff := cmp.Diff(test.wantErrs.ToAggregate(), err, ignoreBadValueDetail); diff != "" {
//				t.Fatalf("got err (-want,+got):\n%s", diff)
//			}
//			if err != nil {
//				return
//			}
//
//			status := p.(framework.PreScorePlugin).PreScore(ctx, state, test.requestedRegion, tf.BuildRunnerInfos(test.runners))
//			if !status.IsSuccess() {
//				t.Errorf("PreScore is expected to return success, but didn't. Got status: %v", status)
//			}
//
//			var gotScores framework.RunnerScoreList
//			for _, n := range test.runners {
//				score, status := p.(framework.ScorePlugin).Score(ctx, state, test.requestedRegion, n.Name)
//				if status.Code() != test.wantStatusCode {
//					t.Errorf("unexpected status code, want: %v, got: %v", test.wantStatusCode, status.Code())
//				}
//				gotScores = append(gotScores, framework.RunnerScore{Name: n.Name, Score: score})
//			}
//
//			if diff := cmp.Diff(test.expectedScores, gotScores); diff != "" {
//				t.Errorf("Unexpected scores (-want,+got):\n%s", diff)
//			}
//		})
//	}
//}
