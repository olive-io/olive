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
	config "github.com/olive-io/olive/apis/config/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// mostResourceScorer favors runners with most requested resources.
// It calculates the percentage of memory and CPU requested by regions scheduled on the runner, and prioritizes
// based on the maximum of the average of the fraction of requested to capacity.
//
// Details:
// (cpu(MaxRunnerScore * requested * cpuWeight / capacity) + memory(MaxRunnerScore * requested * memoryWeight / capacity) + ...) / weightSum
func mostResourceScorer(resources []config.ResourceSpec) func(requested, allocable []int64) int64 {
	return func(requested, allocable []int64) int64 {
		var runnerScore, weightSum int64
		for i := range requested {
			if allocable[i] == 0 {
				continue
			}
			weight := resources[i].Weight
			resourceScore := mostRequestedScore(requested[i], allocable[i])
			runnerScore += resourceScore * weight
			weightSum += weight
		}
		if weightSum == 0 {
			return 0
		}
		return runnerScore / weightSum
	}
}

// The used capacity is calculated on a scale of 0-MaxRunnerScore (MaxRunnerScore is
// constant with value set to 100).
// 0 being the lowest priority and 100 being the highest.
// The more resources are used the higher the score is. This function
// is almost a reversed version of runnerresources.leastRequestedScore.
func mostRequestedScore(requested, capacity int64) int64 {
	if capacity == 0 {
		return 0
	}
	if requested > capacity {
		// `requested` might be greater than `capacity` because regions with no
		// requests get minimum values.
		requested = capacity
	}

	return (requested * framework.MaxRunnerScore) / capacity
}
