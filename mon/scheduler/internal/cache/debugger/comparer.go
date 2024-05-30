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
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	corelisters "github.com/olive-io/olive/client/generated/listers/core/v1"
	monlisters "github.com/olive-io/olive/client/generated/listers/mon/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
	internalqueue "github.com/olive-io/olive/mon/scheduler/internal/queue"
)

// CacheComparer is an implementation of the Scheduler's cache comparer.
type CacheComparer struct {
	RunnerLister     monlisters.RunnerLister
	DefinitionLister corelisters.DefinitionLister
	Cache            internalcache.Cache
	DefinitionQueue  internalqueue.SchedulingQueue
}

// Compare compares the runners and pods of RunnerLister with Cache.Snapshot.
func (c *CacheComparer) Compare(logger klog.Logger) error {
	logger.V(3).Info("Cache comparer started")
	defer logger.V(3).Info("Cache comparer finished")

	runners, err := c.RunnerLister.List(labels.Everything())
	if err != nil {
		return err
	}

	pods, err := c.DefinitionLister.List(labels.Everything())
	if err != nil {
		return err
	}

	dump := c.Cache.Dump()

	pendingDefinitions, _ := c.DefinitionQueue.PendingDefinitions()

	if missed, redundant := c.CompareRunners(runners, dump.Runners); len(missed)+len(redundant) != 0 {
		logger.Info("Cache mismatch", "missedRunners", missed, "redundantRunners", redundant)
	}

	if missed, redundant := c.CompareDefinitions(pods, pendingDefinitions, dump.Runners); len(missed)+len(redundant) != 0 {
		logger.Info("Cache mismatch", "missedDefinitions", missed, "redundantDefinitions", redundant)
	}

	return nil
}

// CompareRunners compares actual runners with cached runners.
func (c *CacheComparer) CompareRunners(runners []*monv1.Runner, runnerinfos map[string]*framework.RunnerInfo) (missed, redundant []string) {
	actual := []string{}
	for _, runner := range runners {
		actual = append(actual, runner.Name)
	}

	cached := []string{}
	for runnerName := range runnerinfos {
		cached = append(cached, runnerName)
	}

	return compareStrings(actual, cached)
}

// CompareDefinitions compares actual pods with cached pods.
func (c *CacheComparer) CompareDefinitions(pods, waitingDefinitions []*corev1.Definition, runnerinfos map[string]*framework.RunnerInfo) (missed, redundant []string) {
	actual := []string{}
	for _, pod := range pods {
		actual = append(actual, string(pod.UID))
	}

	cached := []string{}
	for _, runnerinfo := range runnerinfos {
		for _, p := range runnerinfo.Definitions {
			cached = append(cached, string(p.Definition.UID))
		}
	}
	for _, pod := range waitingDefinitions {
		cached = append(cached, string(pod.UID))
	}

	return compareStrings(actual, cached)
}

func compareStrings(actual, cached []string) (missed, redundant []string) {
	missed, redundant = []string{}, []string{}

	sort.Strings(actual)
	sort.Strings(cached)

	compare := func(i, j int) int {
		if i == len(actual) {
			return 1
		} else if j == len(cached) {
			return -1
		}
		return strings.Compare(actual[i], cached[j])
	}

	for i, j := 0, 0; i < len(actual) || j < len(cached); {
		switch compare(i, j) {
		case 0:
			i++
			j++
		case -1:
			missed = append(missed, actual[i])
			i++
		case 1:
			redundant = append(redundant, cached[j])
			j++
		}
	}

	return
}
