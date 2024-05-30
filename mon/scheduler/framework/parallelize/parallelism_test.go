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

package parallelize

import (
	"fmt"
	"testing"
)

func TestChunkSize(t *testing.T) {
	tests := []struct {
		input      int
		wantOutput int
	}{
		{
			input:      32,
			wantOutput: 3,
		},
		{
			input:      16,
			wantOutput: 2,
		},
		{
			input:      1,
			wantOutput: 1,
		},
		{
			input:      0,
			wantOutput: 1,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d", test.input), func(t *testing.T) {
			if chunkSizeFor(test.input, DefaultParallelism) != test.wantOutput {
				t.Errorf("Expected: %d, got: %d", test.wantOutput, chunkSizeFor(test.input, DefaultParallelism))
			}
		})
	}
}
