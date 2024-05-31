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

package corev1

import (
	"encoding/json"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/pkg/scheduling/corev1/runneraffinity"
)

// RegionPriority returns priority of the given region.
func RegionPriority(region *corev1.Region) int32 {
	if region.Spec.Priority != nil {
		return *region.Spec.Priority
	}
	// When priority of a running region is nil, it means it was created at a time
	// that there was no global default priority class and the priority class
	// name of the region was empty. So, we resolve to the static default priority.
	return 0
}

// MatchRunnerSelectorTerms checks whether the runner labels and fields match runner selector terms in ORed;
// nil or empty term matches no objects.
func MatchRunnerSelectorTerms(
	runner *corev1.Runner,
	runnerSelector *corev1.RunnerSelector,
) (bool, error) {
	if runner == nil {
		return false, nil
	}
	return runneraffinity.NewLazyErrorRunnerSelector(runnerSelector).Match(runner)
}

// GetAvoidRegionsFromRunnerAnnotations scans the list of annotations and
// returns the regions that needs to be avoided for this runner from scheduling
func GetAvoidRegionsFromRunnerAnnotations(annotations map[string]string) (corev1.AvoidRegions, error) {
	var avoidRegions corev1.AvoidRegions
	if len(annotations) > 0 && annotations[corev1.PreferAvoidRegionsAnnotationKey] != "" {
		err := json.Unmarshal([]byte(annotations[corev1.PreferAvoidRegionsAnnotationKey]), &avoidRegions)
		if err != nil {
			return avoidRegions, err
		}
	}
	return avoidRegions, nil
}

// TolerationsTolerateTaint checks if taint is tolerated by any of the tolerations.
func TolerationsTolerateTaint(tolerations []corev1.Toleration, taint *corev1.Taint) bool {
	for i := range tolerations {
		if tolerations[i].ToleratesTaint(taint) {
			return true
		}
	}
	return false
}

type taintsFilterFunc func(*corev1.Taint) bool

// FindMatchingUntoleratedTaint checks if the given tolerations tolerates
// all the filtered taints, and returns the first taint without a toleration
// Returns true if there is an untolerated taint
// Returns false if all taints are tolerated
func FindMatchingUntoleratedTaint(taints []corev1.Taint, tolerations []corev1.Toleration, inclusionFilter taintsFilterFunc) (corev1.Taint, bool) {
	filteredTaints := getFilteredTaints(taints, inclusionFilter)
	for _, taint := range filteredTaints {
		if !TolerationsTolerateTaint(tolerations, &taint) {
			return taint, true
		}
	}
	return corev1.Taint{}, false
}

// getFilteredTaints returns a list of taints satisfying the filter predicate
func getFilteredTaints(taints []corev1.Taint, inclusionFilter taintsFilterFunc) []corev1.Taint {
	if inclusionFilter == nil {
		return taints
	}
	filteredTaints := []corev1.Taint{}
	for _, taint := range taints {
		if !inclusionFilter(&taint) {
			continue
		}
		filteredTaints = append(filteredTaints, taint)
	}
	return filteredTaints
}
