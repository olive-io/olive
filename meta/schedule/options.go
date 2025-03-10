/*
   Copyright 2023 The olive Authors

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

package schedule

import (
	"math"

	pb "github.com/olive-io/olive/api/olivepb"
)

type runnerMatch func(runner *pb.Runner) bool

func allRunnerMatch() runnerMatch {
	return func(runner *pb.Runner) bool {
		return true
	}
}

func runnerMatchWithout(id uint64) runnerMatch {
	return func(runner *pb.Runner) bool {
		return runner.Id != id
	}
}

type runnerOptions struct {
	matches []runnerMatch
	count   int
}

func newRunnerOptions() *runnerOptions {
	return &runnerOptions{
		matches: []runnerMatch{},
		count:   math.MaxInt,
	}
}

type runnerOption func(options *runnerOptions)

func runnerWithMatch(match runnerMatch) runnerOption {
	return func(options *runnerOptions) {
		if options.matches == nil {
			options.matches = make([]runnerMatch, 0)
		}
		options.matches = append(options.matches, match)
	}
}

func runnerWithCount(count int) runnerOption {
	return func(options *runnerOptions) {
		options.count = count
	}
}
