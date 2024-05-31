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

//import (
//	"context"
//	"fmt"
//	"testing"
//
//	"github.com/google/go-cmp/cmp"
//	"github.com/stretchr/testify/assert"
//	v1 "k8s.io/api/core/v1"
//	"k8s.io/apimachinery/pkg/util/validation/field"
//	"k8s.io/klog/v2/ktesting"
//
//	config "github.com/olive-io/olive/apis/config/v1"
//	corev1 "github.com/olive-io/olive/apis/core/v1"
//	"github.com/olive-io/olive/mon/scheduler/framework"
//	plfeature "github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/helper"
//	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
//	"github.com/olive-io/olive/mon/scheduler/internal/cache"
//	st "github.com/olive-io/olive/mon/scheduler/testing"
//	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
//)
//
//func TestRequestedToCapacityRatioScoringStrategy(t *testing.T) {
//	shape := []config.UtilizationShapePoint{
//		{Utilization: 0, Score: 10},
//		{Utilization: 100, Score: 0},
//	}
//
//	tests := []struct {
//		name            string
//		requestedRegion *corev1.Region
//		runners         []*corev1.Runner
//		existingRegions []*corev1.Region
//		expectedScores  framework.RunnerScoreList
//		resources       []config.ResourceSpec
//		shape           []config.UtilizationShapePoint
//		wantErrs        field.ErrorList
//	}{
//		{
//			name:            "nothing scheduled, nothing requested (default - least requested runners have priority)",
//			requestedRegion: st.MakeRegion().Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Obj(),
//				st.MakeRegion().Runner("runner1").Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: framework.MaxRunnerScore}},
//			resources:      defaultResources,
//			shape:          shape,
//		},
//		{
//			name: "nothing scheduled, resources requested, differently sized runners (default - least requested runners have priority)",
//			requestedRegion: st.MakeRegion().
//				Req(map[v1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Req(map[v1.ResourceName]string{"cpu": "2000", "memory": "3000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Obj(),
//				st.MakeRegion().Runner("runner1").Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 38}, {Name: "runner2", Score: 50}},
//			resources:      defaultResources,
//			shape:          shape,
//		},
//		{
//			name:            "no resources requested, regions scheduled with resources (default - least requested runners have priority)",
//			requestedRegion: st.MakeRegion().Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[v1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[v1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[v1.ResourceName]string{"cpu": "3000", "memory": "5000"}).Obj(),
//			},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 38}, {Name: "runner2", Score: 50}},
//			resources:      defaultResources,
//			shape:          shape,
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
//			p, err := NewFit(ctx, &config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.RequestedToCapacityRatio,
//					Resources: test.resources,
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: shape,
//					},
//				},
//			}, fh, plfeature.Features{})
//
//			if diff := cmp.Diff(test.wantErrs.ToAggregate(), err, ignoreBadValueDetail); diff != "" {
//				t.Fatalf("got err (-want,+got):\n%s", diff)
//			}
//			if err != nil {
//				return
//			}
//
//			var gotScores framework.RunnerScoreList
//			for _, n := range test.runners {
//				status := p.(framework.PreScorePlugin).PreScore(ctx, state, test.requestedRegion, tf.BuildRunnerInfos(test.runners))
//				if !status.IsSuccess() {
//					t.Errorf("PreScore is expected to return success, but didn't. Got status: %v", status)
//				}
//				score, status := p.(framework.ScorePlugin).Score(ctx, state, test.requestedRegion, n.Name)
//				if !status.IsSuccess() {
//					t.Errorf("Score is expected to return success, but didn't. Got status: %v", status)
//				}
//				gotScores = append(gotScores, framework.RunnerScore{Name: n.Name, Score: score})
//			}
//
//			if diff := cmp.Diff(test.expectedScores, gotScores); diff != "" {
//				t.Errorf("Unexpected runners (-want,+got):\n%s", diff)
//			}
//		})
//	}
//}
//
//func TestBrokenLinearFunction(t *testing.T) {
//	type Assertion struct {
//		p        int64
//		expected int64
//	}
//	type Test struct {
//		points     []helper.FunctionShapePoint
//		assertions []Assertion
//	}
//
//	tests := []Test{
//		{
//			points: []helper.FunctionShapePoint{{Utilization: 10, Score: 1}, {Utilization: 90, Score: 9}},
//			assertions: []Assertion{
//				{p: -10, expected: 1},
//				{p: 0, expected: 1},
//				{p: 9, expected: 1},
//				{p: 10, expected: 1},
//				{p: 15, expected: 1},
//				{p: 19, expected: 1},
//				{p: 20, expected: 2},
//				{p: 89, expected: 8},
//				{p: 90, expected: 9},
//				{p: 99, expected: 9},
//				{p: 100, expected: 9},
//				{p: 110, expected: 9},
//			},
//		},
//		{
//			points: []helper.FunctionShapePoint{{Utilization: 0, Score: 2}, {Utilization: 40, Score: 10}, {Utilization: 100, Score: 0}},
//			assertions: []Assertion{
//				{p: -10, expected: 2},
//				{p: 0, expected: 2},
//				{p: 20, expected: 6},
//				{p: 30, expected: 8},
//				{p: 40, expected: 10},
//				{p: 70, expected: 5},
//				{p: 100, expected: 0},
//				{p: 110, expected: 0},
//			},
//		},
//		{
//			points: []helper.FunctionShapePoint{{Utilization: 0, Score: 2}, {Utilization: 40, Score: 2}, {Utilization: 100, Score: 2}},
//			assertions: []Assertion{
//				{p: -10, expected: 2},
//				{p: 0, expected: 2},
//				{p: 20, expected: 2},
//				{p: 30, expected: 2},
//				{p: 40, expected: 2},
//				{p: 70, expected: 2},
//				{p: 100, expected: 2},
//				{p: 110, expected: 2},
//			},
//		},
//	}
//
//	for i, test := range tests {
//		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
//			function := helper.BuildBrokenLinearFunction(test.points)
//			for _, assertion := range test.assertions {
//				assert.InDelta(t, assertion.expected, function(assertion.p), 0.1, "points=%v, p=%d", test.points, assertion.p)
//			}
//		})
//	}
//}
//
//func TestResourceBinPackingSingleExtended(t *testing.T) {
//	extendedResource1 := map[string]int64{
//		"intel.com/foo": 4,
//	}
//	extendedResource2 := map[string]int64{
//		"intel.com/foo": 8,
//	}
//	extendedResource3 := map[v1.ResourceName]string{
//		"intel.com/foo": "2",
//	}
//	extendedResource4 := map[v1.ResourceName]string{
//		"intel.com/foo": "4",
//	}
//
//	tests := []struct {
//		region         *corev1.Region
//		regions        []*corev1.Region
//		runners        []*corev1.Runner
//		expectedScores framework.RunnerScoreList
//		name           string
//	}{
//		{
//			//  Runner1 Score = Runner2 Score = 0 as the incoming Region doesn't request extended resource.
//			region:         st.MakeRegion().Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResource2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResource1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 0}, {Name: "runner2", Score: 0}},
//			name:           "nothing scheduled, nothing requested",
//		},
//		{
//			// Runner1 scores (used resources) on 0-MaxRunnerScore scale
//			// Runner1 Score:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),8)
//			//  = 2/8 * maxUtilization = 25 = rawScoringFunction(25)
//			// Runner1 Score: 2
//			// Runner2 scores (used resources) on 0-MaxRunnerScore scale
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),4)
//			//  = 2/4 * maxUtilization = 50 = rawScoringFunction(50)
//			// Runner2 Score: 5
//			region:         st.MakeRegion().Req(extendedResource3).Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResource2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResource1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 2}, {Name: "runner2", Score: 5}},
//			name:           "resources requested, regions scheduled with less resources",
//			regions:        []*corev1.Region{st.MakeRegion().Obj()},
//		},
//		{
//			// Runner1 scores (used resources) on 0-MaxRunnerScore scale
//			// Runner1 Score:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),8)
//			//  = 2/8 * maxUtilization = 25 = rawScoringFunction(25)
//			// Runner1 Score: 2
//			// Runner2 scores (used resources) on 0-MaxRunnerScore scale
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((2+2),4)
//			//  = 4/4 * maxUtilization = maxUtilization = rawScoringFunction(maxUtilization)
//			// Runner2 Score: 10
//			region:         st.MakeRegion().Req(extendedResource3).Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResource2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResource1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 2}, {Name: "runner2", Score: 10}},
//			name:           "resources requested, regions scheduled with resources, on runner with existing region running ",
//			regions:        []*corev1.Region{st.MakeRegion().Req(extendedResource3).Runner("runner2").Obj()},
//		},
//		{
//			// Runner1 scores (used resources) on 0-MaxRunnerScore scale
//			// Runner1 Score:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+4),8)
//			//  = 4/8 * maxUtilization = 50 = rawScoringFunction(50)
//			// Runner1 Score: 5
//			// Runner2 scores (used resources) on 0-MaxRunnerScore scale
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+4),4)
//			//  = 4/4 * maxUtilization = maxUtilization = rawScoringFunction(maxUtilization)
//			// Runner2 Score: 10
//			region:         st.MakeRegion().Req(extendedResource4).Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResource2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResource1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 5}, {Name: "runner2", Score: 10}},
//			name:           "resources requested, regions scheduled with more resources",
//			regions: []*corev1.Region{
//				st.MakeRegion().Obj(),
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			state := framework.NewCycleState()
//			snapshot := cache.NewSnapshot(test.regions, test.runners)
//			_, ctx := ktesting.NewTestContext(t)
//			fh, _ := runtime.NewFramework(ctx, nil, nil, runtime.WithSnapshotSharedLister(snapshot))
//			args := config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type: config.RequestedToCapacityRatio,
//					Resources: []config.ResourceSpec{
//						{Name: "intel.com/foo", Weight: 1},
//					},
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: []config.UtilizationShapePoint{
//							{Utilization: 0, Score: 0},
//							{Utilization: 100, Score: 1},
//						},
//					},
//				},
//			}
//			p, err := NewFit(ctx, &args, fh, plfeature.Features{})
//			if err != nil {
//				t.Fatalf("unexpected error: %v", err)
//			}
//
//			var gotList framework.RunnerScoreList
//			for _, n := range test.runners {
//				status := p.(framework.PreScorePlugin).PreScore(context.Background(), state, test.region, tf.BuildRunnerInfos(test.runners))
//				if !status.IsSuccess() {
//					t.Errorf("PreScore is expected to return success, but didn't. Got status: %v", status)
//				}
//				score, status := p.(framework.ScorePlugin).Score(context.Background(), state, test.region, n.Name)
//				if !status.IsSuccess() {
//					t.Errorf("Score is expected to return success, but didn't. Got status: %v", status)
//				}
//				gotList = append(gotList, framework.RunnerScore{Name: n.Name, Score: score})
//			}
//
//			if diff := cmp.Diff(test.expectedScores, gotList); diff != "" {
//				t.Errorf("Unexpected runnerscore list (-want,+got):\n%s", diff)
//			}
//		})
//	}
//}
//
//func TestResourceBinPackingMultipleExtended(t *testing.T) {
//	extendedResources1 := map[string]int64{
//		"intel.com/foo": 4,
//		"intel.com/bar": 8,
//	}
//	extendedResources2 := map[string]int64{
//		"intel.com/foo": 8,
//		"intel.com/bar": 4,
//	}
//
//	extendedResourceRegion1 := map[v1.ResourceName]string{
//		"intel.com/foo": "2",
//		"intel.com/bar": "2",
//	}
//	extendedResourceRegion2 := map[v1.ResourceName]string{
//		"intel.com/foo": "4",
//		"intel.com/bar": "2",
//	}
//
//	tests := []struct {
//		region         *corev1.Region
//		regions        []*corev1.Region
//		runners        []*corev1.Runner
//		expectedScores framework.RunnerScoreList
//		name           string
//	}{
//		{
//
//			// resources["intel.com/foo"] = 3
//			// resources["intel.com/bar"] = 5
//			// Runner1 scores (used resources) on 0-10 scale
//			// Runner1 Score:
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+0),8)
//			//  = 0/8 * 100 = 0 = rawScoringFunction(0)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+0),4)
//			//  = 0/4 * 100 = 0 = rawScoringFunction(0)
//			// Runner1 Score: (0 * 3) + (0 * 5) / 8 = 0
//
//			// Runner2 scores (used resources) on 0-10 scale
//			// rawScoringFunction(used + requested / available)
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+0),4)
//			//  = 0/4 * 100 = 0 = rawScoringFunction(0)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+0),8)
//			//  = 0/8 * 100 = 0 = rawScoringFunction(0)
//			// Runner2 Score: (0 * 3) + (0 * 5) / 8 = 0
//
//			region:         st.MakeRegion().Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResources2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResources1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 0}, {Name: "runner2", Score: 0}},
//			name:           "nothing scheduled, nothing requested",
//		},
//		{
//
//			// resources["intel.com/foo"] = 3
//			// resources["intel.com/bar"] = 5
//			// Runner1 scores (used resources) on 0-10 scale
//			// Runner1 Score:
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),8)
//			//  = 2/8 * 100 = 25 = rawScoringFunction(25)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),4)
//			//  = 2/4 * 100 = 50 = rawScoringFunction(50)
//			// Runner1 Score: (2 * 3) + (5 * 5) / 8 = 4
//
//			// Runner2 scores (used resources) on 0-10 scale
//			// rawScoringFunction(used + requested / available)
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),4)
//			//  = 2/4 * 100 = 50 = rawScoringFunction(50)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),8)
//			//  = 2/8 * 100 = 25 = rawScoringFunction(25)
//			// Runner2 Score: (5 * 3) + (2 * 5) / 8 = 3
//
//			region:         st.MakeRegion().Req(extendedResourceRegion1).Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResources2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResources1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 4}, {Name: "runner2", Score: 3}},
//			name:           "resources requested, regions scheduled with less resources",
//			regions: []*corev1.Region{
//				st.MakeRegion().Obj(),
//			},
//		},
//		{
//
//			// resources["intel.com/foo"] = 3
//			// resources["intel.com/bar"] = 5
//			// Runner1 scores (used resources) on 0-10 scale
//			// Runner1 Score:
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),8)
//			//  = 2/8 * 100 = 25 = rawScoringFunction(25)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),4)
//			//  = 2/4 * 100 = 50 = rawScoringFunction(50)
//			// Runner1 Score: (2 * 3) + (5 * 5) / 8 = 4
//			// Runner2 scores (used resources) on 0-10 scale
//			// rawScoringFunction(used + requested / available)
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((2+2),4)
//			//  = 4/4 * 100 = 100 = rawScoringFunction(100)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((2+2),8)
//			//  = 4/8 *100 = 50 = rawScoringFunction(50)
//			// Runner2 Score: (10 * 3) + (5 * 5) / 8 = 7
//
//			region:         st.MakeRegion().Req(extendedResourceRegion1).Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResources2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResources1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 4}, {Name: "runner2", Score: 7}},
//			name:           "resources requested, regions scheduled with resources, on runner with existing region running ",
//			regions:        []*corev1.Region{st.MakeRegion().Req(extendedResourceRegion2).Runner("runner2").Obj()},
//		},
//		{
//
//			// resources["intel.com/foo"] = 3
//			// resources["intel.com/bar"] = 5
//			// Runner1 scores (used resources) on 0-10 scale
//			// used + requested / available
//			// intel.com/foo Score: { (0 + 4) / 8 } * 10 = 0
//			// intel.com/bar Score: { (0 + 2) / 4 } * 10 = 0
//			// Runner1 Score: (0.25 * 3) + (0.5 * 5) / 8 = 5
//			// resources["intel.com/foo"] = 3
//			// resources["intel.com/bar"] = 5
//			// Runner2 scores (used resources) on 0-10 scale
//			// used + requested / available
//			// intel.com/foo Score: { (0 + 4) / 4 } * 10 = 0
//			// intel.com/bar Score: { (0 + 2) / 8 } * 10 = 0
//			// Runner2 Score: (1 * 3) + (0.25 * 5) / 8 = 5
//
//			// resources["intel.com/foo"] = 3
//			// resources["intel.com/bar"] = 5
//			// Runner1 scores (used resources) on 0-10 scale
//			// Runner1 Score:
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+4),8)
//			// 4/8 * 100 = 50 = rawScoringFunction(50)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),4)
//			//  = 2/4 * 100 = 50 = rawScoringFunction(50)
//			// Runner1 Score: (5 * 3) + (5 * 5) / 8 = 5
//			// Runner2 scores (used resources) on 0-10 scale
//			// rawScoringFunction(used + requested / available)
//			// intel.com/foo:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+4),4)
//			//  = 4/4 * 100 = 100 = rawScoringFunction(100)
//			// intel.com/bar:
//			// rawScoringFunction(used + requested / available)
//			// resourceScoringFunction((0+2),8)
//			//  = 2/8 * 100 = 25 = rawScoringFunction(25)
//			// Runner2 Score: (10 * 3) + (2 * 5) / 8 = 5
//
//			region:         st.MakeRegion().Req(extendedResourceRegion2).Obj(),
//			runners:        []*corev1.Runner{makeRunner("runner1", 4000, 10000*1024*1024, extendedResources2), makeRunner("runner2", 4000, 10000*1024*1024, extendedResources1)},
//			expectedScores: []framework.RunnerScore{{Name: "runner1", Score: 5}, {Name: "runner2", Score: 5}},
//			name:           "resources requested, regions scheduled with more resources",
//			regions: []*corev1.Region{
//				st.MakeRegion().Obj(),
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			state := framework.NewCycleState()
//			snapshot := cache.NewSnapshot(test.regions, test.runners)
//			_, ctx := ktesting.NewTestContext(t)
//			fh, _ := runtime.NewFramework(ctx, nil, nil, runtime.WithSnapshotSharedLister(snapshot))
//
//			args := config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type: config.RequestedToCapacityRatio,
//					Resources: []config.ResourceSpec{
//						{Name: "intel.com/foo", Weight: 3},
//						{Name: "intel.com/bar", Weight: 5},
//					},
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: []config.UtilizationShapePoint{
//							{Utilization: 0, Score: 0},
//							{Utilization: 100, Score: 1},
//						},
//					},
//				},
//			}
//
//			p, err := NewFit(ctx, &args, fh, plfeature.Features{})
//			if err != nil {
//				t.Fatalf("unexpected error: %v", err)
//			}
//
//			status := p.(framework.PreScorePlugin).PreScore(context.Background(), state, test.region, tf.BuildRunnerInfos(test.runners))
//			if !status.IsSuccess() {
//				t.Errorf("PreScore is expected to return success, but didn't. Got status: %v", status)
//			}
//
//			var gotScores framework.RunnerScoreList
//			for _, n := range test.runners {
//				score, status := p.(framework.ScorePlugin).Score(context.Background(), state, test.region, n.Name)
//				if !status.IsSuccess() {
//					t.Errorf("Score is expected to return success, but didn't. Got status: %v", status)
//				}
//				gotScores = append(gotScores, framework.RunnerScore{Name: n.Name, Score: score})
//			}
//
//			if diff := cmp.Diff(test.expectedScores, gotScores); diff != "" {
//				t.Errorf("Unexpected runnerscore list (-want,+got):\n%s", diff)
//			}
//		})
//	}
//}
