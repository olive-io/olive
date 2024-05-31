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

package helper

import (
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// DefaultNormalizeScore generates a Normalize Score function that can normalize the
// scores from [0, max(scores)] to [0, maxPriority]. If reverse is set to true, it
// reverses the scores by subtracting it from maxPriority.
// Note: The input scores are always assumed to be non-negative integers.
func DefaultNormalizeScore(maxPriority int64, reverse bool, scores framework.RunnerScoreList) *framework.Status {
	var maxCount int64
	for i := range scores {
		if scores[i].Score > maxCount {
			maxCount = scores[i].Score
		}
	}

	if maxCount == 0 {
		if reverse {
			for i := range scores {
				scores[i].Score = maxPriority
			}
		}
		return nil
	}

	for i := range scores {
		score := scores[i].Score

		score = maxPriority * score / maxCount
		if reverse {
			score = maxPriority - score
		}

		scores[i].Score = score
	}
	return nil
}
