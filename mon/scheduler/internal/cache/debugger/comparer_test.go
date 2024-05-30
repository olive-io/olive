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

package debugger

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/types"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

func TestCompareRunners(t *testing.T) {
	tests := []struct {
		name      string
		actual    []string
		cached    []string
		missing   []string
		redundant []string
	}{
		{
			name:      "redundant cached value",
			actual:    []string{"foo", "bar"},
			cached:    []string{"bar", "foo", "foobar"},
			missing:   []string{},
			redundant: []string{"foobar"},
		},
		{
			name:      "missing cached value",
			actual:    []string{"foo", "bar", "foobar"},
			cached:    []string{"bar", "foo"},
			missing:   []string{"foobar"},
			redundant: []string{},
		},
		{
			name:      "proper cache set",
			actual:    []string{"foo", "bar", "foobar"},
			cached:    []string{"bar", "foobar", "foo"},
			missing:   []string{},
			redundant: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testCompareRunners(test.actual, test.cached, test.missing, test.redundant, t)
		})
	}
}

func testCompareRunners(actual, cached, missing, redundant []string, t *testing.T) {
	compare := CacheComparer{}
	runners := []*monv1.Runner{}
	for _, runnerName := range actual {
		runner := &monv1.Runner{}
		runner.Name = runnerName
		runners = append(runners, runner)
	}

	runnerInfo := make(map[string]*framework.RunnerInfo)
	for _, runnerName := range cached {
		runnerInfo[runnerName] = &framework.RunnerInfo{}
	}

	m, r := compare.CompareRunners(runners, runnerInfo)

	if diff := cmp.Diff(missing, m); diff != "" {
		t.Errorf("Unexpected missing (-want, +got):\n%s", diff)
	}

	if diff := cmp.Diff(redundant, r); diff != "" {
		t.Errorf("Unexpected redundant (-want, +got):\n%s", diff)
	}
}

func TestCompareDefinitions(t *testing.T) {
	tests := []struct {
		name      string
		actual    []string
		cached    []string
		queued    []string
		missing   []string
		redundant []string
	}{
		{
			name:      "redundant cached value",
			actual:    []string{"foo", "bar"},
			cached:    []string{"bar", "foo", "foobar"},
			queued:    []string{},
			missing:   []string{},
			redundant: []string{"foobar"},
		},
		{
			name:      "redundant and queued values",
			actual:    []string{"foo", "bar"},
			cached:    []string{"foo", "foobar"},
			queued:    []string{"bar"},
			missing:   []string{},
			redundant: []string{"foobar"},
		},
		{
			name:      "missing cached value",
			actual:    []string{"foo", "bar", "foobar"},
			cached:    []string{"bar", "foo"},
			queued:    []string{},
			missing:   []string{"foobar"},
			redundant: []string{},
		},
		{
			name:      "missing and queued values",
			actual:    []string{"foo", "bar", "foobar"},
			cached:    []string{"foo"},
			queued:    []string{"bar"},
			missing:   []string{"foobar"},
			redundant: []string{},
		},
		{
			name:      "correct cache set",
			actual:    []string{"foo", "bar", "foobar"},
			cached:    []string{"bar", "foobar", "foo"},
			queued:    []string{},
			missing:   []string{},
			redundant: []string{},
		},
		{
			name:      "queued cache value",
			actual:    []string{"foo", "bar", "foobar"},
			cached:    []string{"foobar", "foo"},
			queued:    []string{"bar"},
			missing:   []string{},
			redundant: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testCompareDefinitions(test.actual, test.cached, test.queued, test.missing, test.redundant, t)
		})
	}
}

func testCompareDefinitions(actual, cached, queued, missing, redundant []string, t *testing.T) {
	compare := CacheComparer{}
	pods := []*corev1.Definition{}
	for _, uid := range actual {
		pod := &corev1.Definition{}
		pod.UID = types.UID(uid)
		pods = append(pods, pod)
	}

	queuedDefinitions := []*corev1.Definition{}
	for _, uid := range queued {
		pod := &corev1.Definition{}
		pod.UID = types.UID(uid)
		queuedDefinitions = append(queuedDefinitions, pod)
	}

	runnerInfo := make(map[string]*framework.RunnerInfo)
	for _, uid := range cached {
		pod := &corev1.Definition{}
		pod.UID = types.UID(uid)
		pod.Namespace = "ns"
		pod.Name = uid

		runnerInfo[uid] = framework.NewRunnerInfo(pod)
	}

	m, r := compare.CompareDefinitions(pods, queuedDefinitions, runnerInfo)

	if diff := cmp.Diff(missing, m); diff != "" {
		t.Errorf("Unexpected missing (-want, +got):\n%s", diff)
	}

	if diff := cmp.Diff(redundant, r); diff != "" {
		t.Errorf("Unexpected redundant (-want, +got):\n%s", diff)
	}
}
