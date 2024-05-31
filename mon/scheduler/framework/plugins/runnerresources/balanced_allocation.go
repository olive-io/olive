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
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/runtime"

	config "github.com/olive-io/olive/apis/config/v1"
	"github.com/olive-io/olive/apis/config/validation"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
)

// BalancedAllocation is a score plugin that calculates the difference between the cpu and memory fraction
// of capacity, and prioritizes the host based on how close the two metrics are to each other.
type BalancedAllocation struct {
	handle framework.Handle
	resourceAllocationScorer
}

var _ framework.PreScorePlugin = &BalancedAllocation{}
var _ framework.ScorePlugin = &BalancedAllocation{}

// BalancedAllocationName is the name of the plugin used in the plugin registry and configurations.
const (
	BalancedAllocationName = names.RunnerResourcesBalancedAllocation

	// balancedAllocationPreScoreStateKey is the key in CycleState to RunnerResourcesBalancedAllocation pre-computed data for Scoring.
	balancedAllocationPreScoreStateKey = "PreScore" + BalancedAllocationName
)

// balancedAllocationPreScoreState computed at PreScore and used at Score.
type balancedAllocationPreScoreState struct {
	// regionRequests have the same order of the resources defined in RunnerResourcesFitArgs.Resources,
	// same for other place we store a list like that.
	regionRequests []int64
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *balancedAllocationPreScoreState) Clone() framework.StateData {
	return s
}

// PreScore calculates incoming region's resource requests and writes them to the cycle state used.
func (ba *BalancedAllocation) PreScore(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, runners []*framework.RunnerInfo) *framework.Status {
	state := &balancedAllocationPreScoreState{
		regionRequests: ba.calculateRegionResourceRequestList(region, ba.resources),
	}
	cycleState.Write(balancedAllocationPreScoreStateKey, state)
	return nil
}

func getBalancedAllocationPreScoreState(cycleState *framework.CycleState) (*balancedAllocationPreScoreState, error) {
	c, err := cycleState.Read(balancedAllocationPreScoreStateKey)
	if err != nil {
		return nil, fmt.Errorf("reading %q from cycleState: %w", balancedAllocationPreScoreStateKey, err)
	}

	s, ok := c.(*balancedAllocationPreScoreState)
	if !ok {
		return nil, fmt.Errorf("invalid PreScore state, got type %T", c)
	}
	return s, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (ba *BalancedAllocation) Name() string {
	return BalancedAllocationName
}

// Score invoked at the score extension point.
func (ba *BalancedAllocation) Score(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (int64, *framework.Status) {
	runnerInfo, err := ba.handle.SnapshotSharedLister().RunnerInfos().Get(runnerName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("getting runner %q from Snapshot: %w", runnerName, err))
	}

	s, err := getBalancedAllocationPreScoreState(state)
	if err != nil {
		s = &balancedAllocationPreScoreState{regionRequests: ba.calculateRegionResourceRequestList(region, ba.resources)}
	}

	// ba.score favors runners with balanced resource usage rate.
	// It calculates the standard deviation for those resources and prioritizes the runner based on how close the usage of those resources is to each other.
	// Detail: score = (1 - std) * MaxRunnerScore, where std is calculated by the root square of Σ((fraction(i)-mean)^2)/len(resources)
	// The algorithm is partly inspired by:
	// "Wei Huang et al. An Energy Efficient Virtual Machine Placement Algorithm with Balanced Resource Utilization"
	return ba.score(ctx, region, runnerInfo, s.regionRequests)
}

// ScoreExtensions of the Score plugin.
func (ba *BalancedAllocation) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// NewBalancedAllocation initializes a new plugin and returns it.
func NewBalancedAllocation(_ context.Context, baArgs runtime.Object, h framework.Handle, fts feature.Features) (framework.Plugin, error) {
	args, ok := baArgs.(*config.RunnerResourcesBalancedAllocationArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type RunnerResourcesBalancedAllocationArgs, got %T", baArgs)
	}

	if err := validation.ValidateRunnerResourcesBalancedAllocationArgs(nil, args); err != nil {
		return nil, err
	}

	return &BalancedAllocation{
		handle: h,
		resourceAllocationScorer: resourceAllocationScorer{
			Name:         BalancedAllocationName,
			scorer:       balancedResourceScorer,
			useRequested: true,
			resources:    args.Resources,
		},
	}, nil
}

func balancedResourceScorer(requested, allocable []int64) int64 {
	var resourceToFractions []float64
	var totalFraction float64
	for i := range requested {
		if allocable[i] == 0 {
			continue
		}
		fraction := float64(requested[i]) / float64(allocable[i])
		if fraction > 1 {
			fraction = 1
		}
		totalFraction += fraction
		resourceToFractions = append(resourceToFractions, fraction)
	}

	std := 0.0

	// For most cases, resources are limited to cpu and memory, the std could be simplified to std := (fraction1-fraction2)/2
	// len(fractions) > 2: calculate std based on the well-known formula - root square of Σ((fraction(i)-mean)^2)/len(fractions)
	// Otherwise, set the std to zero is enough.
	if len(resourceToFractions) == 2 {
		std = math.Abs((resourceToFractions[0] - resourceToFractions[1]) / 2)

	} else if len(resourceToFractions) > 2 {
		mean := totalFraction / float64(len(resourceToFractions))
		var sum float64
		for _, fraction := range resourceToFractions {
			sum = sum + (fraction-mean)*(fraction-mean)
		}
		std = math.Sqrt(sum / float64(len(resourceToFractions)))
	}

	// STD (standard deviation) is always a positive value. 1-deviation lets the score to be higher for runner which has least deviation and
	// multiplying it with `MaxRunnerScore` provides the scaling factor needed.
	return int64((1 - std) * float64(framework.MaxRunnerScore))
}
