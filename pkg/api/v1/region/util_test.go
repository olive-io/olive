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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// collectResourcePaths traverses the object, computing all the struct paths that lead to fields with resourcename in the name.
func collectResourcePaths(t *testing.T, resourcename string, path *field.Path, name string, tp reflect.Type) sets.Set[string] {
	resourcename = strings.ToLower(resourcename)
	resourcePaths := sets.New[string]()

	if tp.Kind() == reflect.Pointer {
		resourcePaths.Insert(sets.List[string](collectResourcePaths(t, resourcename, path, name, tp.Elem()))...)
		return resourcePaths
	}

	if strings.Contains(strings.ToLower(name), resourcename) {
		resourcePaths.Insert(path.String())
	}

	switch tp.Kind() {
	case reflect.Pointer:
		resourcePaths.Insert(sets.List[string](collectResourcePaths(t, resourcename, path, name, tp.Elem()))...)
	case reflect.Struct:
		// ObjectMeta is generic and therefore should never have a field with a specific resource's name;
		// it contains cycles so it's easiest to just skip it.
		if name == "ObjectMeta" {
			break
		}
		for i := 0; i < tp.NumField(); i++ {
			field := tp.Field(i)
			resourcePaths.Insert(sets.List[string](collectResourcePaths(t, resourcename, path.Child(field.Name), field.Name, field.Type))...)
		}
	case reflect.Interface:
		t.Errorf("cannot find %s fields in interface{} field %s", resourcename, path.String())
	case reflect.Map:
		resourcePaths.Insert(sets.List[string](collectResourcePaths(t, resourcename, path.Key("*"), "", tp.Elem()))...)
	case reflect.Slice:
		resourcePaths.Insert(sets.List[string](collectResourcePaths(t, resourcename, path.Key("*"), "", tp.Elem()))...)
	default:
		// all primitive types
	}

	return resourcePaths
}

func newRegion(now metav1.Time, ready bool, beforeSec int) *corev1.Region {
	conditionStatus := corev1.ConditionFalse
	if ready {
		conditionStatus = corev1.ConditionTrue
	}
	return &corev1.Region{
		Status: corev1.RegionStatus{
			Conditions: []corev1.RegionCondition{
				{
					Type:               corev1.RegionReady,
					LastTransitionTime: metav1.NewTime(now.Time.Add(-1 * time.Duration(beforeSec) * time.Second)),
					Status:             conditionStatus,
				},
			},
		},
	}
}

func TestIsRegionAvailable(t *testing.T) {
	now := metav1.Now()
	tests := []struct {
		region          *corev1.Region
		minReadySeconds int32
		expected        bool
	}{
		{
			region:          newRegion(now, false, 0),
			minReadySeconds: 0,
			expected:        false,
		},
		{
			region:          newRegion(now, true, 0),
			minReadySeconds: 1,
			expected:        false,
		},
		{
			region:          newRegion(now, true, 0),
			minReadySeconds: 0,
			expected:        true,
		},
		{
			region:          newRegion(now, true, 51),
			minReadySeconds: 50,
			expected:        true,
		},
	}

	for i, test := range tests {
		isAvailable := IsRegionAvailable(test.region, test.minReadySeconds, now)
		if isAvailable != test.expected {
			t.Errorf("[tc #%d] expected available region: %t, got: %t", i, test.expected, isAvailable)
		}
	}
}

func TestIsRegionTerminal(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		regionPhase corev1.RegionPhase
		expected    bool
	}{
		{
			regionPhase: corev1.RegionFailed,
			expected:    true,
		},
		{
			regionPhase: corev1.RegionSucceeded,
			expected:    true,
		},
		{
			regionPhase: corev1.RegionUnknown,
			expected:    false,
		},
		{
			regionPhase: corev1.RegionPending,
			expected:    false,
		},
		{
			regionPhase: corev1.RegionPeering,
			expected:    false,
		},
		{
			expected: false,
		},
	}

	for i, test := range tests {
		region := newRegion(now, true, 0)
		region.Status.Phase = test.regionPhase
		isTerminal := IsRegionTerminal(region)
		if isTerminal != test.expected {
			t.Errorf("[tc #%d] expected terminal region: %t, got: %t", i, test.expected, isTerminal)
		}
	}
}

func TestUpdateRegionCondition(t *testing.T) {
	time := metav1.Now()

	regionStatus := corev1.RegionStatus{
		Conditions: []corev1.RegionCondition{
			{
				Type:               corev1.RegionReady,
				Status:             corev1.ConditionTrue,
				Reason:             "successfully",
				Message:            "sync region successfully",
				LastProbeTime:      time,
				LastTransitionTime: metav1.NewTime(time.Add(1000)),
			},
		},
	}
	tests := []struct {
		status     *corev1.RegionStatus
		conditions corev1.RegionCondition
		expected   bool
		desc       string
	}{
		{
			status: &regionStatus,
			conditions: corev1.RegionCondition{
				Type:               corev1.RegionReady,
				Status:             corev1.ConditionTrue,
				Reason:             "successfully",
				Message:            "sync region successfully",
				LastProbeTime:      time,
				LastTransitionTime: metav1.NewTime(time.Add(1000))},
			expected: false,
			desc:     "all equal, no update",
		},
		{
			status: &regionStatus,
			conditions: corev1.RegionCondition{
				Type:               corev1.RegionScheduled,
				Status:             corev1.ConditionTrue,
				Reason:             "successfully",
				Message:            "sync region successfully",
				LastProbeTime:      time,
				LastTransitionTime: metav1.NewTime(time.Add(1000))},
			expected: true,
			desc:     "not equal Type, should get updated",
		},
		{
			status: &regionStatus,
			conditions: corev1.RegionCondition{
				Type:               corev1.RegionReady,
				Status:             corev1.ConditionFalse,
				Reason:             "successfully",
				Message:            "sync region successfully",
				LastProbeTime:      time,
				LastTransitionTime: metav1.NewTime(time.Add(1000))},
			expected: true,
			desc:     "not equal Status, should get updated",
		},
	}

	for _, test := range tests {
		resultStatus := UpdateRegionCondition(test.status, &test.conditions)

		assert.Equal(t, test.expected, resultStatus, test.desc)
	}
}
