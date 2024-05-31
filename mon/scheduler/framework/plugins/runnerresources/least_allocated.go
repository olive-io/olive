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

// leastResourceScorer favors runners with fewer requested resources.
// It calculates the percentage of memory, CPU and other resources requested by regions scheduled on the runner, and
// prioritizes based on the minimum of the average of the fraction of requested to capacity.
//
// Details:
// (cpu((capacity-requested)*MaxRunnerScore*cpuWeight/capacity) + memory((capacity-requested)*MaxRunnerScore*memoryWeight/capacity) + ...)/weightSum
func leastResourceScorer(resources []config.ResourceSpec) func([]int64, []int64) int64 {
	return func(requested, allocable []int64) int64 {
		var runnerScore, weightSum int64
		for i := range requested {
			if allocable[i] == 0 {
				continue
			}
			weight := resources[i].Weight
			resourceScore := leastRequestedScore(requested[i], allocable[i])
			runnerScore += resourceScore * weight
			weightSum += weight
		}
		if weightSum == 0 {
			return 0
		}
		return runnerScore / weightSum
	}
}

// The unused capacity is calculated on a scale of 0-MaxRunnerScore
// 0 being the lowest priority and `MaxRunnerScore` being the highest.
// The more unused resources the higher the score is.
func leastRequestedScore(requested, capacity int64) int64 {
	if capacity == 0 {
		return 0
	}
	if requested > capacity {
		return 0
	}

	return ((capacity - requested) * framework.MaxRunnerScore) / capacity
}
