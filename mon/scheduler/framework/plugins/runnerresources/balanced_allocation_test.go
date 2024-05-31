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

import (
	"context"
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/ktesting"

	config "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
	"github.com/olive-io/olive/mon/scheduler/internal/cache"
	st "github.com/olive-io/olive/mon/scheduler/testing"
	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
)

func TestRunnerResourcesBalancedAllocation(t *testing.T) {
	cpuAndMemoryAndGPU := corev1.RegionSpec{
		RunnerNames: []string{"runner1"},
	}
	labels1 := map[string]string{
		"foo": "bar",
		"baz": "blah",
	}
	labels2 := map[string]string{
		"bar": "foo",
		"baz": "blah",
	}
	cpuOnly := corev1.RegionSpec{
		RunnerNames: []string{"runner1"},
	}
	cpuOnly2 := cpuOnly
	cpuOnly2.RunnerNames = []string{"runner2"}
	cpuAndMemory := corev1.RegionSpec{
		RunnerNames: []string{"runner2"},
	}

	defaultResourceBalancedAllocationSet := []config.ResourceSpec{
		{Name: string(v1.ResourceCPU), Weight: 1},
		{Name: string(v1.ResourceMemory), Weight: 1},
	}
	scalarResource := map[string]int64{
		"nvidia.com/gpu": 8,
	}

	tests := []struct {
		region       *corev1.Region
		regions      []*corev1.Region
		runners      []*corev1.Runner
		expectedList framework.RunnerScoreList
		name         string
		args         config.RunnerResourcesBalancedAllocationArgs
		runPreScore  bool
	}{
		{
			// Runner1 scores (remaining resources) on 0-MaxRunnerScore scale
			// CPU Fraction: 0 / 4000 = 0%
			// Memory Fraction: 0 / 10000 = 0%
			// Runner1 Score: (1-0) * MaxRunnerScore = MaxRunnerScore
			// Runner2 scores (remaining resources) on 0-MaxRunnerScore scale
			// CPU Fraction: 0 / 4000 = 0 %
			// Memory Fraction: 0 / 10000 = 0%
			// Runner2 Score: (1-0) * MaxRunnerScore = MaxRunnerScore
			region:       st.MakeRegion().Obj(),
			runners:      []*corev1.Runner{makeRunner("runner1", 4000, 10000, nil), makeRunner("runner2", 4000, 10000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: framework.MaxRunnerScore}},
			name:         "nothing scheduled, nothing requested",
			args:         config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore:  true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 3000 / 4000= 75%
			// Memory Fraction: 5000 / 10000 = 50%
			// Runner1 std: (0.75 - 0.5) / 2 = 0.125
			// Runner1 Score: (1 - 0.125)*MaxRunnerScore = 87
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 3000 / 6000= 50%
			// Memory Fraction: 5000/10000 = 50%
			// Runner2 std: 0
			// Runner2 Score: (1-0) * MaxRunnerScore = MaxRunnerScore
			region:       &corev1.Region{Spec: cpuAndMemory},
			runners:      []*corev1.Runner{makeRunner("runner1", 4000, 10000, nil), makeRunner("runner2", 6000, 10000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 87}, {Name: "runner2", Score: framework.MaxRunnerScore}},
			name:         "nothing scheduled, resources requested, differently sized runners",
			args:         config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore:  true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 0 / 4000= 0%
			// Memory Fraction: 0 / 10000 = 0%
			// Runner1 std: 0
			// Runner1 Score: (1-0) * MaxRunnerScore = MaxRunnerScore
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 0 / 4000= 0%
			// Memory Fraction: 0 / 10000 = 0%
			// Runner2 std: 0
			// Runner2 Score: (1-0) * MaxRunnerScore = MaxRunnerScore
			region:       st.MakeRegion().Obj(),
			runners:      []*corev1.Runner{makeRunner("runner1", 4000, 10000, nil), makeRunner("runner2", 4000, 10000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner2", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: framework.MaxRunnerScore}},
			name:         "no resources requested, regions without container scheduled",
			regions: []*corev1.Region{
				st.MakeRegion().Runners("runner1").Labels(labels2).Obj(),
				st.MakeRegion().Runners("runner1").Labels(labels1).Obj(),
				st.MakeRegion().Runners("runner2").Labels(labels1).Obj(),
				st.MakeRegion().Runners("runner2").Labels(labels1).Obj(),
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 0 / 250 = 0%
			// Memory Fraction: 0 / 1000 = 0%
			// Runner1 std: (0 - 0) / 2 = 0
			// Runner1 Score: (1 - 0)*MaxRunnerScore = 100
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 0 / 250 = 0%
			// Memory Fraction: 0 / 1000 = 0%
			// Runner2 std: (0 - 0) / 2 = 0
			// Runner2 Score: (1 - 0)*MaxRunnerScore = 100
			region:       st.MakeRegion().Obj(),
			runners:      []*corev1.Runner{makeRunner("runner1", 250, 1000*1024*1024, nil), makeRunner("runner2", 250, 1000*1024*1024, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 100}, {Name: "runner2", Score: 100}},
			name:         "no resources requested, regions with container scheduled",
			regions: []*corev1.Region{
				st.MakeRegion().Runners("runner1").Obj(),
				st.MakeRegion().Runners("runner1").Obj(),
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 0 / 20000 = 0%
			// Runner1 std: (0.6 - 0) / 2 = 0.3
			// Runner1 Score: (1 - 0.3)*MaxRunnerScore = 70
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 5000 / 20000 = 25%
			// Runner2 std: (0.6 - 0.25) / 2 = 0.175
			// Runner2 Score: (1 - 0.175)*MaxRunnerScore = 82
			region:       st.MakeRegion().Obj(),
			runners:      []*corev1.Runner{makeRunner("runner1", 10000, 20000, nil), makeRunner("runner2", 10000, 20000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 70}, {Name: "runner2", Score: 82}},
			name:         "no resources requested, regions scheduled with resources",
			regions: []*corev1.Region{
				{Spec: cpuOnly, ObjectMeta: metav1.ObjectMeta{Labels: labels2}},
				{Spec: cpuOnly, ObjectMeta: metav1.ObjectMeta{Labels: labels1}},
				{Spec: cpuOnly2, ObjectMeta: metav1.ObjectMeta{Labels: labels1}},
				{Spec: cpuAndMemory, ObjectMeta: metav1.ObjectMeta{Labels: labels1}},
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 5000 / 20000 = 25%
			// Runner1 std: (0.6 - 0.25) / 2 = 0.175
			// Runner1 Score: (1 - 0.175)*MaxRunnerScore = 82
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 10000 / 20000 = 50%
			// Runner2 std: (0.6 - 0.5) / 2 = 0.05
			// Runner2 Score: (1 - 0.05)*MaxRunnerScore = 95
			region:       &corev1.Region{Spec: cpuAndMemory},
			runners:      []*corev1.Runner{makeRunner("runner1", 10000, 20000, nil), makeRunner("runner2", 10000, 20000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 82}, {Name: "runner2", Score: 95}},
			name:         "resources requested, regions scheduled with resources",
			regions: []*corev1.Region{
				{Spec: cpuOnly},
				{Spec: cpuAndMemory},
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 5000 / 20000 = 25%
			// Runner1 std: (0.6 - 0.25) / 2 = 0.175
			// Runner1 Score: (1 - 0.175)*MaxRunnerScore = 82
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 10000 / 50000 = 20%
			// Runner2 std: (0.6 - 0.2) / 2 = 0.2
			// Runner2 Score: (1 - 0.2)*MaxRunnerScore = 80
			region:       &corev1.Region{Spec: cpuAndMemory},
			runners:      []*corev1.Runner{makeRunner("runner1", 10000, 20000, nil), makeRunner("runner2", 10000, 50000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 82}, {Name: "runner2", Score: 80}},
			name:         "resources requested, regions scheduled with resources, differently sized runners",
			regions: []*corev1.Region{
				{Spec: cpuOnly},
				{Spec: cpuAndMemory},
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 6000 = 1
			// Memory Fraction: 0 / 10000 = 0
			// Runner1 std: (1 - 0) / 2 = 0.5
			// Runner1 Score: (1 - 0.5)*MaxRunnerScore = 50
			// Runner1 Score: MaxRunnerScore - (1 - 0) * MaxRunnerScore = 0
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 6000 = 1
			// Memory Fraction 5000 / 10000 = 50%
			// Runner2 std: (1 - 0.5) / 2 = 0.25
			// Runner2 Score: (1 - 0.25)*MaxRunnerScore = 75
			region:       &corev1.Region{Spec: cpuOnly},
			runners:      []*corev1.Runner{makeRunner("runner1", 6000, 10000, nil), makeRunner("runner2", 6000, 10000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 50}, {Name: "runner2", Score: 75}},
			name:         "requested resources at runner capacity",
			regions: []*corev1.Region{
				{Spec: cpuOnly},
				{Spec: cpuAndMemory},
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: true,
		},
		{
			region:       st.MakeRegion().Obj(),
			runners:      []*corev1.Runner{makeRunner("runner1", 0, 0, nil), makeRunner("runner2", 0, 0, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 100}, {Name: "runner2", Score: 100}},
			name:         "zero runner resources, regions scheduled with resources",
			regions: []*corev1.Region{
				{Spec: cpuOnly},
				{Spec: cpuAndMemory},
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: true,
		},
		// Runner1 scores on 0-MaxRunnerScore scale
		// CPU Fraction: 3000 / 3500 = 85.71%
		// Memory Fraction: 5000 / 40000 = 12.5%
		// GPU Fraction: 4 / 8 = 0.5%
		// Runner1 std: sqrt(((0.8571 - 0.503) *  (0.8571 - 0.503) + (0.503 - 0.125) * (0.503 - 0.125) + (0.503 - 0.5) * (0.503 - 0.5)) / 3) = 0.3002
		// Runner1 Score: (1 - 0.3002)*MaxRunnerScore = 70
		// Runner2 scores on 0-MaxRunnerScore scale
		// CPU Fraction: 3000 / 3500 = 85.71%
		// Memory Fraction: 5000 / 40000 = 12.5%
		// GPU Fraction: 1 / 8 = 12.5%
		// Runner2 std: sqrt(((0.8571 - 0.378) *  (0.8571 - 0.378) + (0.378 - 0.125) * (0.378 - 0.125)) + (0.378 - 0.125) * (0.378 - 0.125)) / 3) = 0.345
		// Runner2 Score: (1 - 0.358)*MaxRunnerScore = 65
		{
			region:       st.MakeRegion().Obj(),
			runners:      []*corev1.Runner{makeRunner("runner1", 3500, 40000, scalarResource), makeRunner("runner2", 3500, 40000, scalarResource)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 70}, {Name: "runner2", Score: 65}},
			name:         "include scalar resource on a runner for balanced resource allocation",
			regions: []*corev1.Region{
				{Spec: cpuAndMemory},
				{Spec: cpuAndMemoryAndGPU},
			},
			args: config.RunnerResourcesBalancedAllocationArgs{Resources: []config.ResourceSpec{
				{Name: string(v1.ResourceCPU), Weight: 1},
				{Name: string(v1.ResourceMemory), Weight: 1},
				{Name: "nvidia.com/gpu", Weight: 1},
			}},
			runPreScore: true,
		},
		// Only one runner (runner1) has the scalar resource, region doesn't request the scalar resource and the scalar resource should be skipped for consideration.
		// Runner1: std = 0, score = 100
		// Runner2: std = 0, score = 100
		{
			region:       st.MakeRegion().Obj(),
			runners:      []*corev1.Runner{makeRunner("runner1", 3500, 40000, scalarResource), makeRunner("runner2", 3500, 40000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 100}, {Name: "runner2", Score: 100}},
			name:         "runner without the scalar resource results to a higher score",
			regions: []*corev1.Region{
				{Spec: cpuOnly},
				{Spec: cpuOnly2},
			},
			args: config.RunnerResourcesBalancedAllocationArgs{Resources: []config.ResourceSpec{
				{Name: string(v1.ResourceCPU), Weight: 1},
				{Name: "nvidia.com/gpu", Weight: 1},
			}},
			runPreScore: true,
		},
		{
			// Runner1 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 5000 / 20000 = 25%
			// Runner1 std: (0.6 - 0.25) / 2 = 0.175
			// Runner1 Score: (1 - 0.175)*MaxRunnerScore = 82
			// Runner2 scores on 0-MaxRunnerScore scale
			// CPU Fraction: 6000 / 10000 = 60%
			// Memory Fraction: 10000 / 20000 = 50%
			// Runner2 std: (0.6 - 0.5) / 2 = 0.05
			// Runner2 Score: (1 - 0.05)*MaxRunnerScore = 95
			region:       &corev1.Region{Spec: cpuAndMemory},
			runners:      []*corev1.Runner{makeRunner("runner1", 10000, 20000, nil), makeRunner("runner2", 10000, 20000, nil)},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 82}, {Name: "runner2", Score: 95}},
			name:         "resources requested, regions scheduled with resources if PreScore not called",
			regions: []*corev1.Region{
				{Spec: cpuOnly},
				{Spec: cpuAndMemory},
			},
			args:        config.RunnerResourcesBalancedAllocationArgs{Resources: defaultResourceBalancedAllocationSet},
			runPreScore: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := cache.NewSnapshot(test.regions, test.runners)
			_, ctx := ktesting.NewTestContext(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			fh, _ := runtime.NewFramework(ctx, nil, nil, runtime.WithSnapshotSharedLister(snapshot))
			p, _ := NewBalancedAllocation(ctx, &test.args, fh, feature.Features{})
			state := framework.NewCycleState()
			for i := range test.runners {
				if test.runPreScore {
					status := p.(framework.PreScorePlugin).PreScore(ctx, state, test.region, tf.BuildRunnerInfos(test.runners))
					if !status.IsSuccess() {
						t.Errorf("PreScore is expected to return success, but didn't. Got status: %v", status)
					}
				}
				hostResult, status := p.(framework.ScorePlugin).Score(ctx, state, test.region, test.runners[i].Name)
				if !status.IsSuccess() {
					t.Errorf("Score is expected to return success, but didn't. Got status: %v", status)
				}
				if !reflect.DeepEqual(test.expectedList[i].Score, hostResult) {
					t.Errorf("got score %v for host %v, expected %v", hostResult, test.runners[i].Name, test.expectedList[i].Score)
				}
			}
		})
	}
}
