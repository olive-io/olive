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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/olive-io/olive/mon/scheduler/framework"
)

func TestDefaultNormalizeScore(t *testing.T) {
	tests := []struct {
		reverse        bool
		scores         []int64
		expectedScores []int64
	}{
		{
			scores:         []int64{1, 2, 3, 4},
			expectedScores: []int64{25, 50, 75, 100},
		},
		{
			reverse:        true,
			scores:         []int64{1, 2, 3, 4},
			expectedScores: []int64{75, 50, 25, 0},
		},
		{
			scores:         []int64{1000, 10, 20, 30},
			expectedScores: []int64{100, 1, 2, 3},
		},
		{
			reverse:        true,
			scores:         []int64{1000, 10, 20, 30},
			expectedScores: []int64{0, 99, 98, 97},
		},
		{
			scores:         []int64{1, 1, 1, 1},
			expectedScores: []int64{100, 100, 100, 100},
		},
		{
			scores:         []int64{1000, 1, 1, 1},
			expectedScores: []int64{100, 0, 0, 0},
		},
		{
			reverse:        true,
			scores:         []int64{0, 1, 1, 1},
			expectedScores: []int64{100, 0, 0, 0},
		},
		{
			scores:         []int64{0, 0, 0, 0},
			expectedScores: []int64{0, 0, 0, 0},
		},
		{
			reverse:        true,
			scores:         []int64{0, 0, 0, 0},
			expectedScores: []int64{100, 100, 100, 100},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			scores := framework.RunnerScoreList{}
			for _, score := range test.scores {
				scores = append(scores, framework.RunnerScore{Score: score})
			}

			expectedScores := framework.RunnerScoreList{}
			for _, score := range test.expectedScores {
				expectedScores = append(expectedScores, framework.RunnerScore{Score: score})
			}

			DefaultNormalizeScore(framework.MaxRunnerScore, test.reverse, scores)
			if diff := cmp.Diff(expectedScores, scores); diff != "" {
				t.Errorf("Unexpected scores (-want, +got):\n%s", diff)
			}
		})
	}
}
