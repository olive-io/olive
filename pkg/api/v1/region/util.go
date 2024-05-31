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

package region

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// Visitor is called with each object name, and returns true if visiting should continue
type Visitor func(name string) (shouldContinue bool)

func skipEmptyNames(visitor Visitor) Visitor {
	return func(name string) bool {
		if len(name) == 0 {
			// continue visiting
			return true
		}
		// delegate to visitor
		return visitor(name)
	}
}

// IsRegionAvailable returns true if a region is available; false otherwise.
// Precondition for an available region is that it must be ready. On top
// of that, there are two cases when a region can be considered available:
// 1. minReadySeconds == 0, or
// 2. LastTransitionTime (is set) + minReadySeconds < current time
func IsRegionAvailable(region *corev1.Region, minReadySeconds int32, now metav1.Time) bool {
	if !IsRegionReady(region) {
		return false
	}

	c := GetRegionReadyCondition(region.Status)
	minReadySecondsDuration := time.Duration(minReadySeconds) * time.Second
	if minReadySeconds == 0 || (!c.LastTransitionTime.IsZero() && c.LastTransitionTime.Add(minReadySecondsDuration).Before(now.Time)) {
		return true
	}
	return false
}

// IsRegionReady returns true if a region is ready; false otherwise.
func IsRegionReady(region *corev1.Region) bool {
	return IsRegionReadyConditionTrue(region.Status)
}

// IsRegionTerminal returns true if a region is terminal, all containers are stopped and cannot ever regress.
func IsRegionTerminal(region *corev1.Region) bool {
	return IsRegionPhaseTerminal(region.Status.Phase)
}

// IsRegionPhaseTerminal returns true if the region's phase is terminal.
func IsRegionPhaseTerminal(phase corev1.RegionPhase) bool {
	return phase == corev1.RegionFailed || phase == corev1.RegionSucceeded
}

// IsRegionReadyConditionTrue returns true if a region is ready; false otherwise.
func IsRegionReadyConditionTrue(status corev1.RegionStatus) bool {
	condition := GetRegionReadyCondition(status)
	return condition != nil && condition.Status == corev1.ConditionTrue
}

// GetRegionReadyCondition extracts the region ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetRegionReadyCondition(status corev1.RegionStatus) *corev1.RegionCondition {
	_, condition := GetRegionCondition(&status, corev1.RegionReady)
	return condition
}

// GetRegionCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetRegionCondition(status *corev1.RegionStatus, conditionType corev1.RegionConditionType) (int, *corev1.RegionCondition) {
	if status == nil {
		return -1, nil
	}
	return GetRegionConditionFromList(status.Conditions, conditionType)
}

// GetRegionConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func GetRegionConditionFromList(conditions []corev1.RegionCondition, conditionType corev1.RegionConditionType) (int, *corev1.RegionCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

// UpdateRegionCondition updates existing region condition or creates a new one. Sets LastTransitionTime to now if the
// status has changed.
// Returns true if region condition has changed or has been added.
func UpdateRegionCondition(status *corev1.RegionStatus, condition *corev1.RegionCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this region condition.
	conditionIndex, oldCondition := GetRegionCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new region condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastProbeTime.Equal(&oldCondition.LastProbeTime) &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}
