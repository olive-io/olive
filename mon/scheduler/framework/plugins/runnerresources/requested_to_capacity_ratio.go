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
	"math"

	config "github.com/olive-io/olive/apis/config/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/helper"
)

const maxUtilization = 100

// buildRequestedToCapacityRatioScorerFunction allows users to apply bin packing
// on core resources like CPU, Memory as well as extended resources like accelerators.
func buildRequestedToCapacityRatioScorerFunction(scoringFunctionShape helper.FunctionShape, resources []config.ResourceSpec) func([]int64, []int64) int64 {
	rawScoringFunction := helper.BuildBrokenLinearFunction(scoringFunctionShape)
	resourceScoringFunction := func(requested, capacity int64) int64 {
		if capacity == 0 || requested > capacity {
			return rawScoringFunction(maxUtilization)
		}

		return rawScoringFunction(requested * maxUtilization / capacity)
	}
	return func(requested, allocable []int64) int64 {
		var runnerScore, weightSum int64
		for i := range requested {
			if allocable[i] == 0 {
				continue
			}
			weight := resources[i].Weight
			resourceScore := resourceScoringFunction(requested[i], allocable[i])
			if resourceScore > 0 {
				runnerScore += resourceScore * weight
				weightSum += weight
			}
		}
		if weightSum == 0 {
			return 0
		}
		return int64(math.Round(float64(runnerScore) / float64(weightSum)))
	}
}

func requestedToCapacityRatioScorer(resources []config.ResourceSpec, shape []config.UtilizationShapePoint) func([]int64, []int64) int64 {
	shapes := make([]helper.FunctionShapePoint, 0, len(shape))
	for _, point := range shape {
		shapes = append(shapes, helper.FunctionShapePoint{
			Utilization: int64(point.Utilization),
			// MaxCustomPriorityScore may diverge from the max score used in the scheduler and defined by MaxRunnerScore,
			// therefore we need to scale the score returned by requested to capacity ratio to the score range
			// used by the scheduler.
			Score: int64(point.Score) * (framework.MaxRunnerScore / config.MaxCustomPriorityScore),
		})
	}

	return buildRequestedToCapacityRatioScorerFunction(shapes, resources)
}
