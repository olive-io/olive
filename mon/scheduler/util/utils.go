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

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client/generated/clientset/versioned"
	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
)

// GetDefinitionFullName returns a name that uniquely identifies a definition.
func GetDefinitionFullName(definition *corev1.Definition) string {
	// Use underscore as the delimiter because it is not allowed in definition name
	// (DNS subdomain format).
	return definition.Name + "_" + definition.Namespace
}

// GetDefinitionStartTime returns start time of the given definition or current timestamp
// if it hasn't started yet.
func GetDefinitionStartTime(definition *corev1.Definition) *metav1.Time {
	// Assumed definitions and bound definitions that haven't started don't have a StartTime yet.
	return &metav1.Time{Time: time.Now()}
}

// GetEarliestDefinitionStartTime returns the earliest start time of all definitions that
// have the highest priority among all victims.
func GetEarliestDefinitionStartTime(victims *extenderv1.Victims) *metav1.Time {
	if len(victims.Definitions) == 0 {
		// should not reach here.
		klog.Background().Error(nil, "victims.Definitions is empty. Should not reach here")
		return nil
	}

	earliestDefinitionStartTime := GetDefinitionStartTime(victims.Definitions[0])
	maxPriority := DefinitionPriority(victims.Definitions[0])

	for _, definition := range victims.Definitions {
		if definitionPriority := DefinitionPriority(definition); definitionPriority == maxPriority {
			if definitionStartTime := GetDefinitionStartTime(definition); definitionStartTime.Before(earliestDefinitionStartTime) {
				earliestDefinitionStartTime = definitionStartTime
			}
		} else if definitionPriority > maxPriority {
			maxPriority = definitionPriority
			earliestDefinitionStartTime = GetDefinitionStartTime(definition)
		}
	}

	return earliestDefinitionStartTime
}

// MoreImportantDefinition return true when priority of the first definition is higher than
// the second one. If two definitions' priorities are equal, compare their StartTime.
// It takes arguments of the type "interface{}" to be used with SortableList,
// but expects those arguments to be *v1.Definition.
func MoreImportantDefinition(definition1, definition2 *corev1.Definition) bool {
	p1 := DefinitionPriority(definition1)
	p2 := DefinitionPriority(definition2)
	if p1 != p2 {
		return p1 > p2
	}
	return GetDefinitionStartTime(definition1).Before(GetDefinitionStartTime(definition2))
}

// Retriable defines the retriable errors during a scheduling cycle.
func Retriable(err error) bool {
	return apierrors.IsInternalError(err) || apierrors.IsServiceUnavailable(err) ||
		net.IsConnectionRefused(err)
}

// PatchDefinitionStatus calculates the delta bytes change from <old.Status> to <newStatus>,
// and then submit a request to API server to patch the definition changes.
func PatchDefinitionStatus(ctx context.Context, cs clientset.Interface, old *corev1.Definition, newStatus *corev1.DefinitionStatus) error {
	if newStatus == nil {
		return nil
	}

	oldData, err := json.Marshal(corev1.Definition{Status: old.Status})
	if err != nil {
		return err
	}

	newData, err := json.Marshal(corev1.Definition{Status: *newStatus})
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, &corev1.Definition{})
	if err != nil {
		return fmt.Errorf("failed to create merge patch for definition %q/%q: %v", old.Namespace, old.Name, err)
	}

	if "{}" == string(patchBytes) {
		return nil
	}

	patchFn := func() error {
		_, err := cs.CoreV1().Definitions(old.Namespace).Patch(ctx, old.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}, "status")
		return err
	}

	return retry.OnError(retry.DefaultBackoff, Retriable, patchFn)
}

// DeleteDefinition deletes the given <definition> from API server
func DeleteDefinition(ctx context.Context, cs clientset.Interface, definition *corev1.Definition) error {
	return cs.CoreV1().Definitions(definition.Namespace).Delete(ctx, definition.Name, metav1.DeleteOptions{})
}

// ClearNominatedNodeName internally submit a patch request to API server
// to set each definitions[*].Status.NominatedNodeName> to "".
func ClearNominatedNodeName(ctx context.Context, cs clientset.Interface, definitions ...*corev1.Definition) utilerrors.Aggregate {
	var errs []error
	for _, p := range definitions {
		definitionStatusCopy := p.Status.DeepCopy()
		if err := PatchDefinitionStatus(ctx, cs, p, definitionStatusCopy); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

// IsScalarResourceName validates the resource for Extended, Hugepages, Native and AttachableVolume resources
func IsScalarResourceName(name v1.ResourceName) bool {
	return true
}

// As converts two objects to the given type.
// Both objects must be of the same type. If not, an error is returned.
// nil objects are allowed and will be converted to nil.
// For oldObj, cache.DeletedFinalStateUnknown is handled and the
// object stored in it will be converted instead.
func As[T any](oldObj, newobj interface{}) (T, T, error) {
	var oldTyped T
	var newTyped T
	var ok bool
	if newobj != nil {
		newTyped, ok = newobj.(T)
		if !ok {
			return oldTyped, newTyped, fmt.Errorf("expected %T, but got %T", newTyped, newobj)
		}
	}

	if oldObj != nil {
		if realOldObj, ok := oldObj.(cache.DeletedFinalStateUnknown); ok {
			oldObj = realOldObj.Obj
		}
		oldTyped, ok = oldObj.(T)
		if !ok {
			return oldTyped, newTyped, fmt.Errorf("expected %T, but got %T", oldTyped, oldObj)
		}
	}
	return oldTyped, newTyped, nil
}

// DefinitionPriority returns priority of the given definition.
func DefinitionPriority(definition *corev1.Definition) int64 {
	if definition.Spec.Priority != nil {
		return *definition.Spec.Priority
	}
	// When priority of a running definition is nil, it means it was created at a time
	// that there was no global default priority class and the priority class
	// name of the definition was empty. So, we resolve to the static default priority.
	return 0
}
