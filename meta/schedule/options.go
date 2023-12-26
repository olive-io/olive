// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
