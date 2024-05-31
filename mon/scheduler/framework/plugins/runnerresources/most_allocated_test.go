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
//func TestMostAllocatedScoringStrategy(t *testing.T) {
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
//			// Runner1 scores (used resources) on 0-MaxRunnerScore scale
//			// CPU Score: (0 * MaxRunnerScore)  / 4000 = 0
//			// Memory Score: (0 * MaxRunnerScore) / 10000 = 0
//			// Runner1 Score: (0 + 0) / 2 = 0
//			// Runner2 scores (used resources) on 0-MaxRunnerScore scale
//			// CPU Score: (0 * MaxRunnerScore) / 4000 = 0
//			// Memory Score: (0 * MaxRunnerScore) / 10000 = 0
//			// Runner2 Score: (0 + 0) / 2 = 0
//			name:            "nothing scheduled, nothing requested",
//			requestedRegion: st.MakeRegion().Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: framework.MinRunnerScore}, {Name: "runner2", Score: framework.MinRunnerScore}},
//			resources:       defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: (3000 * MaxRunnerScore) / 4000 = 75
//			// Memory Score: (5000 * MaxRunnerScore) / 10000 = 50
//			// Runner1 Score: (75 + 50) / 2 = 62
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: (3000 * MaxRunnerScore) / 6000 = 50
//			// Memory Score: (5000 * MaxRunnerScore) / 10000 = 50
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
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 62}, {Name: "runner2", Score: 50}},
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
//			// CPU Score: (6000 * MaxRunnerScore) / 10000 = 60
//			// Memory Score: (0 * MaxRunnerScore) / 20000 = 0
//			// Runner1 Score: (60 + 0) / 2 = 30
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: (6000 * MaxRunnerScore) / 10000 = 60
//			// Memory Score: (5000 * MaxRunnerScore) / 20000 = 25
//			// Runner2 Score: (60 + 25) / 2 = 42
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
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 30}, {Name: "runner2", Score: 42}},
//			resources:      defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: (6000 * MaxRunnerScore) / 10000 = 60
//			// Memory Score: (5000 * MaxRunnerScore) / 20000 = 25
//			// Runner1 Score: (60 + 25) / 2 = 42
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: (6000 * MaxRunnerScore) / 10000 = 60
//			// Memory Score: (10000 * MaxRunnerScore) / 20000 = 50
//			// Runner2 Score: (60 + 50) / 2 = 55
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
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 42}, {Name: "runner2", Score: 55}},
//			resources:      defaultResources,
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Score: 5000 * MaxRunnerScore / 5000 return 100
//			// Memory Score: (9000 * MaxRunnerScore) / 10000 = 90
//			// Runner1 Score: (100 + 90) / 2 = 95
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Score: (5000 * MaxRunnerScore) / 10000 = 50
//			// Memory Score: 9000 * MaxRunnerScore / 9000 return 100
//			// Runner2 Score: (50 + 100) / 2 = 75
//			name: "resources requested equal runner capacity",
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "4000"}).
//				Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "5000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "10000", "memory": "9000"}).Obj(),
//			},
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 95}, {Name: "runner2", Score: 75}},
//			resources:       defaultResources,
//		},
//		{
//			// CPU Score: (3000 *100) / 4000 = 75
//			// Memory Score: (5000 *100) / 10000 = 50
//			// Runner1 Score: (75 * 1 + 50 * 2) / (1 + 2) = 58
//			// CPU Score: (3000 *100) / 6000 = 50
//			// Memory Score: (5000 *100) / 10000 = 50
//			// Runner2 Score: (50 * 1 + 50 * 2) / (1 + 2) = 50
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
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 58}, {Name: "runner2", Score: 50}},
//			resources: []config.ResourceSpec{
//				{Name: "memory", Weight: 2},
//				{Name: "cpu", Weight: 1},
//			},
//		},
//		{
//			// Runner1 scores on 0-MaxRunnerScore scale
//			// CPU Fraction: 300 / 250 = 100%
//			// Memory Fraction: 600 / 1000 = 60%
//			// Runner1 Score: (100 + 60) / 2 = 80
//			// Runner2 scores on 0-MaxRunnerScore scale
//			// CPU Fraction: 100 / 250 = 40%
//			// Memory Fraction: 200 / 1000 = 20%
//			// Runner2 Score: (20 + 40) / 2 = 30
//			name:            "no resources requested, regions scheduled, nonzero request for resource",
//			requestedRegion: st.MakeRegion().Container("container").Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "250m", "memory": "1000Mi"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "250m", "memory": "1000Mi"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Container("container").Obj(),
//				st.MakeRegion().Runner("runner1").Container("container").Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 80}, {Name: "runner2", Score: 30}},
//			resources:      defaultResources,
//		},
//		{
//			// resource with negative weight is not allowed
//			name: "resource with negative weight",
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
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
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
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
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			resources: []config.ResourceSpec{
//				{Name: "memory", Weight: 101},
//			},
//			wantErrs: field.ErrorList{
//				&field.Error{
//					Type:  field.ErrorTypeInvalid,
//					Field: "scoringStrategy.resources[0].weight",
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
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "4"}).Obj(),
//			},
//			resources:       extendedResourceSet,
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 50}, {Name: "runner2", Score: 50}},
//		},
//		{
//			// Honor extended resource if the region requests.
//			// For both runners: cpuScore and memScore are 50.
//			// In terms of extended resource score:
//			// - runner1 get: 2 / 4 * 100 = 50
//			// - runner2 get: 2 / 10 * 100 = 20
//			// So the final scores are:
//			// - runner1: (50 + 50 + 50) / 3 = 50
//			// - runner2: (50 + 50 + 20) / 3 = 40
//			name: "honor extended resource if the region request",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000", v1.ResourceName(extendedRes): "2"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "4"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "10"}).Obj(),
//			},
//			resources:       extendedResourceSet,
//			existingRegions: nil,
//			expectedScores:  []framework.RunnerScore{{Name: "runner1", Score: 50}, {Name: "runner2", Score: 40}},
//		},
//		{
//			// If the runner doesn't have a resource
//			// CPU Score: (3000 * 100) / 6000 = 50
//			// Memory Score: (4000 * 100) / 10000 = 40
//			// Runner1 Score: (50 * 1 + 40 * 1) / (1 + 1) = 45
//			// Runner2 Score: (50 * 1 + 40 * 1) / (1 + 1) = 45
//			name: "if the runner doesn't have a resource",
//			requestedRegion: st.MakeRegion().Runner("runner1").
//				Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "4000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000", v1.ResourceName(extendedRes): "4"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 45}, {Name: "runner2", Score: 45}},
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
//			p, err := NewFit(ctx,
//				&config.RunnerResourcesFitArgs{
//					ScoringStrategy: &config.ScoringStrategy{
//						Type:      config.MostAllocated,
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
