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

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EtcdCluster struct {
	metav1.TypeMeta `json:",inline"`

	Endpoints []string `json:"endpoints" protobuf:"bytes,1,rep,name=endpoints"`
	// dial and request timeout
	Timeout         string `json:"timeout" protobuf:"bytes,2,opt,name=timeout"`
	MaxUnaryRetries int32  `json:"maxUnaryRetries" protobuf:"varint,3,opt,name=maxUnaryRetries"`
}

// RunnerSelector A runner selector represents the union of the results of one or more label queries
// over a set of runners; that is, it represents the OR of the selectors represented
// by the runner selector terms.
// +structType=atomic
type RunnerSelector struct {
	// Required. A list of runner selector terms. The terms are ORed.
	// +listType=atomic
	RunnerSelectorTerms []RunnerSelectorTerm `json:"runnerSelectorTerms" protobuf:"bytes,1,rep,name=runnerSelectorTerms"`
}

// RunnerSelectorTerm A null or empty runner selector term matches no objects. The requirements of
// them are ANDed.
// The TopologySelectorTerm type implements a subset of the RunnerSelectorTerm.
// +structType=atomic
type RunnerSelectorTerm struct {
	// A list of runner selector requirements by runner's labels.
	// +optional
	// +listType=atomic
	MatchExpressions []RunnerSelectorRequirement `json:"matchExpressions,omitempty" protobuf:"bytes,1,rep,name=matchExpressions"`
	// A list of runner selector requirements by runner's fields.
	// +optional
	// +listType=atomic
	MatchFields []RunnerSelectorRequirement `json:"matchFields,omitempty" protobuf:"bytes,2,rep,name=matchFields"`
}

// RunnerSelectorRequirement A runner selector requirement is a selector that contains values, a key, and an operator
// that relates the key and values.
type RunnerSelectorRequirement struct {
	// The label key that the selector applies to.
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
	Operator RunnerSelectorOperator `json:"operator" protobuf:"bytes,2,opt,name=operator,casttype=RunnerSelectorOperator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. If the operator is Gt or Lt, the values
	// array must have a single element, which will be interpreted as an integer.
	// This array is replaced during a strategic merge patch.
	// +optional
	// +listType=atomic
	Values []string `json:"values,omitempty" protobuf:"bytes,3,rep,name=values"`
}

// RunnerSelectorOperator A runner selector operator is the set of operators that can be used in
// a runner selector requirement.
// +enum
type RunnerSelectorOperator string

const (
	RunnerSelectorOpIn           RunnerSelectorOperator = "In"
	RunnerSelectorOpNotIn        RunnerSelectorOperator = "NotIn"
	RunnerSelectorOpExists       RunnerSelectorOperator = "Exists"
	RunnerSelectorOpDoesNotExist RunnerSelectorOperator = "DoesNotExist"
	RunnerSelectorOpGt           RunnerSelectorOperator = "Gt"
	RunnerSelectorOpLt           RunnerSelectorOperator = "Lt"
)

// TopologySelectorTerm A topology selector term represents the result of label queries.
// A null or empty topology selector term matches no objects.
// The requirements of them are ANDed.
// It provides a subset of functionality as RunnerSelectorTerm.
// This is an alpha feature and may change in the future.
// +structType=atomic
type TopologySelectorTerm struct {
	// Usage: Fields of type []TopologySelectorTerm must be listType=atomic.

	// A list of topology selector requirements by labels.
	// +optional
	// +listType=atomic
	MatchLabelExpressions []TopologySelectorLabelRequirement `json:"matchLabelExpressions,omitempty" protobuf:"bytes,1,rep,name=matchLabelExpressions"`
}

// TopologySelectorLabelRequirement A topology selector requirement is a selector that matches given label.
// This is an alpha feature and may change in the future.
type TopologySelectorLabelRequirement struct {
	// The label key that the selector applies to.
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// An array of string values. One value must match the label to be selected.
	// Each entry in Values is ORed.
	// +listType=atomic
	Values []string `json:"values" protobuf:"bytes,2,rep,name=values"`
}

// Affinity is a group of affinity scheduling rules.
type Affinity struct {
	// Describes runner affinity scheduling rules for the region.
	// +optional
	RunnerAffinity *RunnerAffinity `json:"runnerAffinity,omitempty" protobuf:"bytes,1,opt,name=runnerAffinity"`
	// Describes region affinity scheduling rules (e.g. co-locate this region in the same runner, zone, etc. as some other region(s)).
	// +optional
	RegionAffinity *RegionAffinity `json:"regionAffinity,omitempty" protobuf:"bytes,2,opt,name=regionAffinity"`
	// Describes region anti-affinity scheduling rules (e.g. avoid putting this region in the same runner, zone, etc. as some other region(s)).
	// +optional
	RegionAntiAffinity *RegionAntiAffinity `json:"regionAntiAffinity,omitempty" protobuf:"bytes,3,opt,name=regionAntiAffinity"`
}

// RegionAffinity Region affinity is a group of inter region affinity scheduling rules.
type RegionAffinity struct {
	// NOT YET IMPLEMENTED. TODO: Uncomment field once it is implemented.
	// If the affinity requirements specified by this field are not met at
	// scheduling time, the region will not be scheduled onto the runner.
	// If the affinity requirements specified by this field cease to be met
	// at some point during region execution (e.g. due to a region label update), the
	// system will try to eventually evict the region from its runner.
	// When there are multiple elements, the lists of runners corresponding to each
	// regionAffinityTerm are intersected, i.e. all terms must be satisfied.
	// +optional
	// RequiredDuringSchedulingRequiredDuringExecution []RegionAffinityTerm  `json:"requiredDuringSchedulingRequiredDuringExecution,omitempty"`

	// If the affinity requirements specified by this field are not met at
	// scheduling time, the region will not be scheduled onto the runner.
	// If the affinity requirements specified by this field cease to be met
	// at some point during region execution (e.g. due to a region label update), the
	// system may or may not try to eventually evict the region from its runner.
	// When there are multiple elements, the lists of runners corresponding to each
	// regionAffinityTerm are intersected, i.e. all terms must be satisfied.
	// +optional
	// +listType=atomic
	RequiredDuringSchedulingIgnoredDuringExecution []RegionAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" protobuf:"bytes,1,rep,name=requiredDuringSchedulingIgnoredDuringExecution"`
	// The scheduler will prefer to schedule regions to runners that satisfy
	// the affinity expressions specified by this field, but it may choose
	// a runner that violates one or more of the expressions. The runner that is
	// most preferred is the one with the greatest sum of weights, i.e.
	// for each runner that meets all of the scheduling requirements (resource
	// request, requiredDuringScheduling affinity expressions, etc.),
	// compute a sum by iterating through the elements of this field and adding
	// "weight" to the sum if the runner has regions which matches the corresponding regionAffinityTerm; the
	// runner(s) with the highest sum are the most preferred.
	// +optional
	// +listType=atomic
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedRegionAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" protobuf:"bytes,2,rep,name=preferredDuringSchedulingIgnoredDuringExecution"`
}

// RegionAntiAffinity Region anti affinity is a group of inter region anti affinity scheduling rules.
type RegionAntiAffinity struct {
	// NOT YET IMPLEMENTED. TODO: Uncomment field once it is implemented.
	// If the anti-affinity requirements specified by this field are not met at
	// scheduling time, the region will not be scheduled onto the runner.
	// If the anti-affinity requirements specified by this field cease to be met
	// at some point during region execution (e.g. due to a region label update), the
	// system will try to eventually evict the region from its runner.
	// When there are multiple elements, the lists of runners corresponding to each
	// regionAffinityTerm are intersected, i.e. all terms must be satisfied.
	// +optional
	// RequiredDuringSchedulingRequiredDuringExecution []RegionAffinityTerm  `json:"requiredDuringSchedulingRequiredDuringExecution,omitempty"`

	// If the anti-affinity requirements specified by this field are not met at
	// scheduling time, the region will not be scheduled onto the runner.
	// If the anti-affinity requirements specified by this field cease to be met
	// at some point during region execution (e.g. due to a region label update), the
	// system may or may not try to eventually evict the region from its runner.
	// When there are multiple elements, the lists of runners corresponding to each
	// regionAffinityTerm are intersected, i.e. all terms must be satisfied.
	// +optional
	// +listType=atomic
	RequiredDuringSchedulingIgnoredDuringExecution []RegionAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" protobuf:"bytes,1,rep,name=requiredDuringSchedulingIgnoredDuringExecution"`
	// The scheduler will prefer to schedule regions to runners that satisfy
	// the anti-affinity expressions specified by this field, but it may choose
	// a runner that violates one or more of the expressions. The runner that is
	// most preferred is the one with the greatest sum of weights, i.e.
	// for each runner that meets all of the scheduling requirements (resource
	// request, requiredDuringScheduling anti-affinity expressions, etc.),
	// compute a sum by iterating through the elements of this field and adding
	// "weight" to the sum if the runner has regions which matches the corresponding regionAffinityTerm; the
	// runner(s) with the highest sum are the most preferred.
	// +optional
	// +listType=atomic
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedRegionAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" protobuf:"bytes,2,rep,name=preferredDuringSchedulingIgnoredDuringExecution"`
}

// WeightedRegionAffinityTerm The weights of all of the matched WeightedRegionAffinityTerm fields are added per-runner to find the most preferred runner(s)
type WeightedRegionAffinityTerm struct {
	// weight associated with matching the corresponding regionAffinityTerm,
	// in the range 1-100.
	Weight int32 `json:"weight" protobuf:"varint,1,opt,name=weight"`
	// Required. A region affinity term, associated with the corresponding weight.
	RegionAffinityTerm RegionAffinityTerm `json:"regionAffinityTerm" protobuf:"bytes,2,opt,name=regionAffinityTerm"`
}

// RegionAffinityTerm Defines a set of regions (namely those matching the labelSelector
// relative to the given namespace(s)) that this region should be
// co-located (affinity) or not co-located (anti-affinity) with,
// where co-located is defined as running on a runner whose value of
// the label with key <topologyKey> matches that of any runner on which
// a region of the set of regions is running
type RegionAffinityTerm struct {
	// A label query over a set of resources, in this case regions.
	// If it's null, this RegionAffinityTerm matches with no Regions.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" protobuf:"bytes,1,opt,name=labelSelector"`
	// namespaces specifies a static list of namespace names that the term applies to.
	// The term is applied to the union of the namespaces listed in this field
	// and the ones selected by namespaceSelector.
	// null or empty namespaces list and null namespaceSelector means "this region's namespace".
	// +optional
	// +listType=atomic
	Namespaces []string `json:"namespaces,omitempty" protobuf:"bytes,2,rep,name=namespaces"`
	// This region should be co-located (affinity) or not co-located (anti-affinity) with the regions matching
	// the labelSelector in the specified namespaces, where co-located is defined as running on a runner
	// whose value of the label with key topologyKey matches that of any runner on which any of the
	// selected regions is running.
	// Empty topologyKey is not allowed.
	TopologyKey string `json:"topologyKey" protobuf:"bytes,3,opt,name=topologyKey"`
	// A label query over the set of namespaces that the term applies to.
	// The term is applied to the union of the namespaces selected by this field
	// and the ones listed in the namespaces field.
	// null selector and null or empty namespaces list means "this region's namespace".
	// An empty selector ({}) matches all namespaces.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty" protobuf:"bytes,4,opt,name=namespaceSelector"`
	// MatchLabelKeys is a set of region label keys to select which regions will
	// be taken into consideration. The keys are used to lookup values from the
	// incoming region labels, those key-value labels are merged with `labelSelector` as `key in (value)`
	// to select the group of existing regions which regions will be taken into consideration
	// for the incoming region's region (anti) affinity. Keys that don't exist in the incoming
	// region labels will be ignored. The default value is empty.
	// The same key is forbidden to exist in both matchLabelKeys and labelSelector.
	// Also, matchLabelKeys cannot be set when labelSelector isn't set.
	// This is an alpha field and requires enabling MatchLabelKeysInRegionAffinity feature gate.
	// +listType=atomic
	// +optional
	MatchLabelKeys []string `json:"matchLabelKeys,omitempty" protobuf:"bytes,5,rep,name=matchLabelKeys"`
	// MismatchLabelKeys is a set of region label keys to select which regions will
	// be taken into consideration. The keys are used to lookup values from the
	// incoming region labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
	// to select the group of existing regions which regions will be taken into consideration
	// for the incoming region's region (anti) affinity. Keys that don't exist in the incoming
	// region labels will be ignored. The default value is empty.
	// The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
	// Also, mismatchLabelKeys cannot be set when labelSelector isn't set.
	// This is an alpha field and requires enabling MatchLabelKeysInRegionAffinity feature gate.
	// +listType=atomic
	// +optional
	MismatchLabelKeys []string `json:"mismatchLabelKeys,omitempty" protobuf:"bytes,6,rep,name=mismatchLabelKeys"`
}

// RunnerAffinity Runner affinity is a group of runner affinity scheduling rules.
type RunnerAffinity struct {
	// NOT YET IMPLEMENTED. TODO: Uncomment field once it is implemented.
	// If the affinity requirements specified by this field are not met at
	// scheduling time, the region will not be scheduled onto the runner.
	// If the affinity requirements specified by this field cease to be met
	// at some point during region execution (e.g. due to an update), the system
	// will try to eventually evict the region from its runner.
	// +optional
	// RequiredDuringSchedulingRequiredDuringExecution *RunnerSelector `json:"requiredDuringSchedulingRequiredDuringExecution,omitempty"`

	// If the affinity requirements specified by this field are not met at
	// scheduling time, the region will not be scheduled onto the runner.
	// If the affinity requirements specified by this field cease to be met
	// at some point during region execution (e.g. due to an update), the system
	// may or may not try to eventually evict the region from its runner.
	// +optional
	RequiredDuringSchedulingIgnoredDuringExecution *RunnerSelector `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" protobuf:"bytes,1,opt,name=requiredDuringSchedulingIgnoredDuringExecution"`
	// The scheduler will prefer to schedule regions to runners that satisfy
	// the affinity expressions specified by this field, but it may choose
	// a runner that violates one or more of the expressions. The runner that is
	// most preferred is the one with the greatest sum of weights, i.e.
	// for each runner that meets all of the scheduling requirements (resource
	// request, requiredDuringScheduling affinity expressions, etc.),
	// compute a sum by iterating through the elements of this field and adding
	// "weight" to the sum if the runner matches the corresponding matchExpressions; the
	// runner(s) with the highest sum are the most preferred.
	// +optional
	// +listType=atomic
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" protobuf:"bytes,2,rep,name=preferredDuringSchedulingIgnoredDuringExecution"`
}

// PreferredSchedulingTerm An empty preferred scheduling term matches all objects with implicit weight 0
// (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
type PreferredSchedulingTerm struct {
	// Weight associated with matching the corresponding runnerSelectorTerm, in the range 1-100.
	Weight int32 `json:"weight" protobuf:"varint,1,opt,name=weight"`
	// A runner selector term, associated with the corresponding weight.
	Preference RunnerSelectorTerm `json:"preference" protobuf:"bytes,2,opt,name=preference"`
}

// Taint The runner this Taint is attached to has the "effect" on
// any region that does not tolerate the Taint.
type Taint struct {
	// Required. The taint key to be applied to a runner.
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// The taint value corresponding to the taint key.
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	// Required. The effect of the taint on regions
	// that do not tolerate the taint.
	// Valid effects are NoSchedule, PreferNoSchedule and NoExecute.
	Effect TaintEffect `json:"effect" protobuf:"bytes,3,opt,name=effect,casttype=TaintEffect"`
	// TimeAdded represents the time at which the taint was added.
	// It is only written for NoExecute taints.
	// +optional
	TimeAdded *metav1.Time `json:"timeAdded,omitempty" protobuf:"bytes,4,opt,name=timeAdded"`
}

// +enum
type TaintEffect string

const (
	// TaintEffectNoSchedule do not allow new regions to schedule onto the runner unless they tolerate the taint,
	// but allow all regions submitted to Kubelet without going through the scheduler
	// to start, and allow all already-running regions to continue running.
	// Enforced by the scheduler.
	TaintEffectNoSchedule TaintEffect = "NoSchedule"
	// TaintEffectPreferNoSchedule likes TaintEffectNoSchedule, but the scheduler tries not to schedule
	// new regions onto the runner, rather than prohibiting new regions from scheduling
	// onto the runner entirely. Enforced by the scheduler.
	TaintEffectPreferNoSchedule TaintEffect = "PreferNoSchedule"
	// NOT YET IMPLEMENTED. TODO: Uncomment field once it is implemented.
	// Like TaintEffectNoSchedule, but additionally do not allow regions submitted to
	// Kubelet without going through the scheduler to start.
	// Enforced by Kubelet and the scheduler.
	// TaintEffectNoScheduleNoAdmit TaintEffect = "NoScheduleNoAdmit"

	// TaintEffectNoExecute evicts any already-running regions that do not tolerate the taint.
	// Currently enforced by RunnerController.
	TaintEffectNoExecute TaintEffect = "NoExecute"
)

// Toleration The region this Toleration is attached to tolerates any taint that matches
// the triple <key,value,effect> using the matching operator <operator>.
type Toleration struct {
	// Key is the taint key that the toleration applies to. Empty means match all taint keys.
	// If the key is empty, operator must be Exists; this combination means to match all values and all keys.
	// +optional
	Key string `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	// Operator represents a key's relationship to the value.
	// Valid operators are Exists and Equal. Defaults to Equal.
	// Exists is equivalent to wildcard for value, so that a region can
	// tolerate all taints of a particular category.
	// +optional
	Operator TolerationOperator `json:"operator,omitempty" protobuf:"bytes,2,opt,name=operator,casttype=TolerationOperator"`
	// Value is the taint value the toleration matches to.
	// If the operator is Exists, the value should be empty, otherwise just a regular string.
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,3,opt,name=value"`
	// Effect indicates the taint effect to match. Empty means match all taint effects.
	// When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
	// +optional
	Effect TaintEffect `json:"effect,omitempty" protobuf:"bytes,4,opt,name=effect,casttype=TaintEffect"`
	// TolerationSeconds represents the period of time the toleration (which must be
	// of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
	// it is not set, which means tolerate the taint forever (do not evict). Zero and
	// negative values will be treated as 0 (evict immediately) by the system.
	// +optional
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty" protobuf:"varint,5,opt,name=tolerationSeconds"`
}

// TolerationOperator a toleration operator is the set of operators that can be used in a toleration.
// +enum
type TolerationOperator string

const (
	TolerationOpExists TolerationOperator = "Exists"
	TolerationOpEqual  TolerationOperator = "Equal"
)

// AvoidRegions describes regions that should avoid this runner. This is the value for a
// Runner annotation with key scheduler.alpha.olive.io/preferAvoidRegions and
// will eventually become a field of RunnerStatus.
type AvoidRegions struct {
	// Bounded-sized list of signatures of regions that should avoid this runner, sorted
	// in timestamp order from oldest to newest. Size of the slice is unspecified.
	// +optional
	// +listType=atomic
	PreferAvoidRegions []PreferAvoidRegionsEntry `json:"preferAvoidRegions,omitempty" protobuf:"bytes,1,rep,name=preferAvoidRegions"`
}

// PreferAvoidRegionsEntry describes a class of regions that should avoid this runner.
type PreferAvoidRegionsEntry struct {
	// The class of regions.
	RegionSignature RegionSignature `json:"regionSignature" protobuf:"bytes,1,opt,name=regionSignature"`
	// Time at which this entry was added to the list.
	// +optional
	EvictionTime metav1.Time `json:"evictionTime,omitempty" protobuf:"bytes,2,opt,name=evictionTime"`
	// (brief) reason why this entry was added to the list.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`
	// Human readable message indicating why this entry was added to the list.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
}

// RegionSignature describes the class of regions that should avoid this runner.
// Exactly one field should be set.
type RegionSignature struct {
	// Reference to controller whose regions should avoid this runner.
	// +optional
	RegionController *metav1.OwnerReference `json:"regionController,omitempty" protobuf:"bytes,1,opt,name=regionController"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Binding ties one object to another; for example, a region is bound to a runner by a scheduler.
type Binding struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// The target object that you want to bind to the standard object.
	Target ObjectReference `json:"target" protobuf:"bytes,2,opt,name=target"`
}

// ObjectReference contains enough information to let you inspect or modify the referred object.
// ---
// New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.
//  1. Ignored fields.  It includes many fields which are not generally honored.  For instance, ResourceVersion and FieldPath are both very rarely valid in actual usage.
//  2. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular
//     restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".
//     Those cannot be well described when embedded.
//  3. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.
//  4. The fields are both imprecise and overly precise.  Kind is not a precise mapping to a URL. This can produce ambiguity
//     during interpretation and require a REST mapping.  In most cases, the dependency is on the group,resource tuple
//     and the version of the actual struct is irrelevant.
//  5. We cannot easily change it.  Because this type is embedded in many locations, updates to this type
//     will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.
//
// Instead of using this type, create a locally provided and used type that is well-focused on your reference.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +structType=atomic
type ObjectReference struct {
	// Kind of the referent.
	// +optional
	Kind string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	// Namespace of the referent.
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
	// Name of the referent.
	// +optional
	Names []string `json:"name,omitempty" protobuf:"bytes,3,rep,name=name"`
	// UID of the referent.
	// +optional
	UID types.UID `json:"uid,omitempty" protobuf:"bytes,4,opt,name=uid,casttype=k8s.io/apimachinery/pkg/types.UID"`
	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,5,opt,name=apiVersion"`
	// Specific resourceVersion to which this reference is made, if any.
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,6,opt,name=resourceVersion"`

	// If referring to a piece of an object instead of an entire object, this string
	// should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2].
	// For example, if the object reference is to a container within a region, this would take on a value like:
	// "spec.containers{name}" (where "name" refers to the name of the container that triggered
	// the event) or if no container name is specified "spec.containers[2]" (container with
	// index 2 in this region). This syntax is chosen only to have some well-defined way of
	// referencing a part of an object.
	// TODO: this design is not final and this field is subject to change in the future.
	// +optional
	FieldPath string `json:"fieldPath,omitempty" protobuf:"bytes,7,opt,name=fieldPath"`
}

// RegionSchedulingGate is associated to a Region to guard its scheduling.
type RegionSchedulingGate struct {
	// Name of the scheduling gate.
	// Each scheduling gate must have a unique name field.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
}

// +enum
type UnsatisfiableConstraintAction string

const (
	// DoNotSchedule instructs the scheduler not to schedule the region
	// when constraints are not satisfied.
	DoNotSchedule UnsatisfiableConstraintAction = "DoNotSchedule"
	// ScheduleAnyway instructs the scheduler to schedule the region
	// even if constraints are not satisfied.
	ScheduleAnyway UnsatisfiableConstraintAction = "ScheduleAnyway"
)

// RunnerInclusionPolicy defines the type of runner inclusion policy
// +enum
type RunnerInclusionPolicy string

const (
	// RunnerInclusionPolicyIgnore means ignore this scheduling directive when calculating region topology spread skew.
	RunnerInclusionPolicyIgnore RunnerInclusionPolicy = "Ignore"
	// RunnerInclusionPolicyHonor means use this scheduling directive when calculating region topology spread skew.
	RunnerInclusionPolicyHonor RunnerInclusionPolicy = "Honor"
)

// TopologySpreadConstraint specifies how to spread matching regions among the given topology.
type TopologySpreadConstraint struct {
	// MaxSkew describes the degree to which regions may be unevenly distributed.
	// When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference
	// between the number of matching regions in the target topology and the global minimum.
	// The global minimum is the minimum number of matching regions in an eligible domain
	// or zero if the number of eligible domains is less than MinDomains.
	// For example, in a 3-zone cluster, MaxSkew is set to 1, and regions with the same
	// labelSelector spread as 2/2/1:
	// In this case, the global minimum is 1.
	// +-------+-------+-------+
	// | zone1 | zone2 | zone3 |
	// +-------+-------+-------+
	// |  R R  |  R R  |   R   |
	// +-------+-------+-------+
	// - if MaxSkew is 1, incoming region can only be scheduled to zone3 to become 2/2/2;
	// scheduling it onto zone1(zone2) would make the ActualSkew(3-1) on zone1(zone2)
	// violate MaxSkew(1).
	// - if MaxSkew is 2, incoming region can be scheduled onto any zone.
	// When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence
	// to topologies that satisfy it.
	// It's a required field. Default value is 1 and 0 is not allowed.
	MaxSkew int32 `json:"maxSkew" protobuf:"varint,1,opt,name=maxSkew"`
	// TopologyKey is the key of runner labels. Runners that have a label with this key
	// and identical values are considered to be in the same topology.
	// We consider each <key, value> as a "bucket", and try to put balanced number
	// of regions into each bucket.
	// We define a domain as a particular instance of a topology.
	// Also, we define an eligible domain as a domain whose runners meet the requirements of
	// runnerAffinityPolicy and runnerTaintsPolicy.
	// e.g. If TopologyKey is "kubernetes.io/hostname", each Runner is a domain of that topology.
	// And, if TopologyKey is "topology.kubernetes.io/zone", each zone is a domain of that topology.
	// It's a required field.
	TopologyKey string `json:"topologyKey" protobuf:"bytes,2,opt,name=topologyKey"`
	// WhenUnsatisfiable indicates how to deal with a region if it doesn't satisfy
	// the spread constraint.
	// - DoNotSchedule (default) tells the scheduler not to schedule it.
	// - ScheduleAnyway tells the scheduler to schedule the region in any location,
	//   but giving higher precedence to topologies that would help reduce the
	//   skew.
	// A constraint is considered "Unsatisfiable" for an incoming region
	// if and only if every possible runner assignment for that region would violate
	// "MaxSkew" on some topology.
	// For example, in a 3-zone cluster, MaxSkew is set to 1, and regions with the same
	// labelSelector spread as 3/1/1:
	// +-------+-------+-------+
	// | zone1 | zone2 | zone3 |
	// +-------+-------+-------+
	// | R R R |   R   |   R   |
	// +-------+-------+-------+
	// If WhenUnsatisfiable is set to DoNotSchedule, incoming region can only be scheduled
	// to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies
	// MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler
	// won't make it *more* imbalanced.
	// It's a required field.
	WhenUnsatisfiable UnsatisfiableConstraintAction `json:"whenUnsatisfiable" protobuf:"bytes,3,opt,name=whenUnsatisfiable,casttype=UnsatisfiableConstraintAction"`
	// LabelSelector is used to find matching regions.
	// Regions that match this label selector are counted to determine the number of regions
	// in their corresponding topology domain.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" protobuf:"bytes,4,opt,name=labelSelector"`
	// MinDomains indicates a minimum number of eligible domains.
	// When the number of eligible domains with matching topology keys is less than minDomains,
	// Region Topology Spread treats "global minimum" as 0, and then the calculation of Skew is performed.
	// And when the number of eligible domains with matching topology keys equals or greater than minDomains,
	// this value has no effect on scheduling.
	// As a result, when the number of eligible domains is less than minDomains,
	// scheduler won't schedule more than maxSkew Regions to those domains.
	// If value is nil, the constraint behaves as if MinDomains is equal to 1.
	// Valid values are integers greater than 0.
	// When value is not nil, WhenUnsatisfiable must be DoNotSchedule.
	//
	// For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and regions with the same
	// labelSelector spread as 2/2/2:
	// +-------+-------+-------+
	// | zone1 | zone2 | zone3 |
	// +-------+-------+-------+
	// |  R R  |  R R  |  R R  |
	// +-------+-------+-------+
	// The number of domains is less than 5(MinDomains), so "global minimum" is treated as 0.
	// In this situation, new region with the same labelSelector cannot be scheduled,
	// because computed skew will be 3(3 - 0) if new Region is scheduled to any of the three zones,
	// it will violate MaxSkew.
	// +optional
	MinDomains *int32 `json:"minDomains,omitempty" protobuf:"varint,5,opt,name=minDomains"`
	// RunnerAffinityPolicy indicates how we will treat Region's runnerAffinity/runnerSelector
	// when calculating region topology spread skew. Options are:
	// - Honor: only runners matching runnerAffinity/runnerSelector are included in the calculations.
	// - Ignore: runnerAffinity/runnerSelector are ignored. All runners are included in the calculations.
	//
	// If this value is nil, the behavior is equivalent to the Honor policy.
	// This is a beta-level feature default enabled by the RunnerInclusionPolicyInRegionTopologySpread feature flag.
	// +optional
	RunnerAffinityPolicy *RunnerInclusionPolicy `json:"runnerAffinityPolicy,omitempty" protobuf:"bytes,6,opt,name=runnerAffinityPolicy"`
	// RunnerTaintsPolicy indicates how we will treat runner taints when calculating
	// region topology spread skew. Options are:
	// - Honor: runners without taints, along with tainted runners for which the incoming region
	// has a toleration, are included.
	// - Ignore: runner taints are ignored. All runners are included.
	//
	// If this value is nil, the behavior is equivalent to the Ignore policy.
	// This is a beta-level feature default enabled by the RunnerInclusionPolicyInRegionTopologySpread feature flag.
	// +optional
	RunnerTaintsPolicy *RunnerInclusionPolicy `json:"runnerTaintsPolicy,omitempty" protobuf:"bytes,7,opt,name=runnerTaintsPolicy"`
	// MatchLabelKeys is a set of region label keys to select the regions over which
	// spreading will be calculated. The keys are used to lookup values from the
	// incoming region labels, those key-value labels are ANDed with labelSelector
	// to select the group of existing regions over which spreading will be calculated
	// for the incoming region. The same key is forbidden to exist in both MatchLabelKeys and LabelSelector.
	// MatchLabelKeys cannot be set when LabelSelector isn't set.
	// Keys that don't exist in the incoming region labels will
	// be ignored. A null or empty list means only match against labelSelector.
	//
	// This is a beta field and requires the MatchLabelKeysInRegionTopologySpread feature gate to be enabled (enabled by default).
	// +listType=atomic
	// +optional
	MatchLabelKeys []string `json:"matchLabelKeys,omitempty" protobuf:"bytes,8,opt,name=matchLabelKeys"`
}

type RunnerConditionType string

// These are valid but not exhaustive conditions of runner. A cloud provider may set a condition not listed here.
// Relevant events contain "RunnerReady", "RunnerNotReady", "RunnerSchedulable", and "RunnerNotSchedulable".
const (
	// RunnerReady means olive-runner is healthy and ready to accept regions.
	RunnerReady RunnerConditionType = "Ready"
	// RunnerMemoryPressure means the olive-runner is under pressure due to insufficient available memory.
	RunnerMemoryPressure RunnerConditionType = "MemoryPressure"
	// RunnerDiskPressure means the olive-runner is under pressure due to insufficient available disk.
	RunnerDiskPressure RunnerConditionType = "DiskPressure"
	// RunnerPIDPressure means the olive-runner is under pressure due to insufficient available PID.
	RunnerPIDPressure RunnerConditionType = "PIDPressure"
	// RunnerNetworkUnavailable means that network for the runner is not correctly configured.
	RunnerNetworkUnavailable RunnerConditionType = "NetworkUnavailable"
)

// RunnerCondition contains condition information for a runner.
type RunnerCondition struct {
	// Type of runner condition.
	Type RunnerConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=RunnerConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty" protobuf:"bytes,3,opt,name=lastHeartbeatTime"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// RunnerPhase defines the phase in which a runner is in
type RunnerPhase string

// These are the valid phases of runner.
const (
	// RunnerPending means the runner has been created/added by the system, but not configured.
	RunnerPending RunnerPhase = "Pending"
	// RunnerRunning means the runner has been configured and has Olive components running.
	RunnerRunning RunnerPhase = "Running"
	// RunnerTerminated means the runner has been removed from the cluster.
	RunnerTerminated RunnerPhase = "Terminated"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Runner the olive runner
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RunnerSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status RunnerStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// RunnerSpec is the specification of a Runner.
type RunnerSpec struct {
	// ID is the member ID for this member.
	ID int64 `json:"id" protobuf:"varint,1,opt,name=id"`
	// name is the human-readable name of the member. If the member is not started, the name will be an empty string.
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
	// peerURLs is the list of URLs the member exposes to the cluster for communication.
	PeerURL string `json:"peerURL" protobuf:"bytes,3,opt,name=peerURL"`
	// clientURLs is the list of URLs the member exposes to clients for communication. If the member is not started, clientURLs will be empty.
	ClientURL string `json:"clientURL" protobuf:"bytes,4,opt,name=clientURL"`
	// isLearner indicates if the member is raft learner.
	IsLearner *bool `json:"isLearner" protobuf:"varint,5,opt,name=isLearner"`
	// version information, generated during build
	VersionRef string `json:"versionRef" protobuf:"bytes,6,opt,name=versionRef"`

	// Unschedulable controls node schedulability of new regions. By default, runner is schedulable.
	// +optional
	Unschedulable bool `json:"unschedulable,omitempty" protobuf:"varint,7,opt,name=unschedulable"`

	// If specified, the runner's taints.
	// +optional
	// +listType=atomic
	Taints []Taint `json:"taints,omitempty" protobuf:"bytes,8,rep,name=taints"`
}

type RunnerStatus struct {
	// Capacity represents the total resources of a runner.
	// +optional
	Capacity ResourceList `json:"capacity,omitempty" protobuf:"bytes,1,rep,name=capacity,casttype=ResourceList,castkey=ResourceName"`
	// Allocatable represents the resources of a runner that are available for scheduling.
	// Defaults to Capacity.
	// +optional
	Allocatable ResourceList `json:"allocatable,omitempty" protobuf:"bytes,2,rep,name=allocatable,casttype=ResourceList,castkey=ResourceName"`

	// Conditions is an array of current observed runner conditions.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []RunnerCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,3,rep,name=conditions"`

	Phase   RunnerPhase `json:"phase" protobuf:"bytes,4,opt,name=phase,casttype=RunnerPhase"`
	Message string      `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`

	Stat *RunnerStat `json:"stat,omitempty" protobuf:"bytes,6,opt,name=stat"`
}

// +k8s:deepcopy-gen:true

// RunnerStat is the stat information of Runner
type RunnerStat struct {
	CpuTotal    float64  `json:"cpuTotal" protobuf:"fixed64,1,opt,name=cpuTotal"`
	MemoryTotal float64  `json:"memoryTotal" protobuf:"fixed64,2,opt,name=memoryTotal"`
	Regions     []int64  `json:"regions" protobuf:"varint,3,rep,name=regions"`
	Leaders     []string `json:"leaders" protobuf:"bytes,4,rep,name=leaders"`
	Definitions int64    `json:"definitions" protobuf:"varint,5,opt,name=definitions"`
	// dynamic statistic, don't save to storage
	Dynamic *RunnerDynamicStat `json:"dynamic" protobuf:"bytes,6,opt,name=dynamic"`
}

type RunnerDynamicStat struct {
	BpmnProcesses int64   `json:"bpmnProcesses" protobuf:"varint,1,opt,name=bpmnProcesses"`
	BpmnEvents    int64   `json:"bpmnEvents" protobuf:"varint,2,opt,name=bpmnEvents"`
	BpmnTasks     int64   `json:"bpmnTasks" protobuf:"varint,3,opt,name=bpmnTasks"`
	CpuUsed       float64 `json:"cpuUsed,omitempty" protobuf:"fixed64,4,opt,name=cpuUsed"`
	MemoryUsed    float64 `json:"memoryUsed,omitempty" protobuf:"fixed64,5,opt,name=memoryUsed"`
	Timestamp     int64   `json:"timestamp" protobuf:"varint,6,opt,name=timestamp"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerList is a list of Runner objects.
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Runner
	Items []Runner `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// RegionConditionType is a valid value for RegionCondition.Type
type RegionConditionType string

// These are built-in conditions of region. An application may use a custom condition not listed here.
const (
	// RegionInitialized means that all init containers in the region have started successfully.
	RegionInitialized RegionConditionType = "Initialized"
	// RegionReady means the region is able to service requests and should be added to the
	// load balancing pools of all matching services.
	RegionReady RegionConditionType = "Ready"
	// RegionScheduled represents status of the scheduling process for this region.
	RegionScheduled RegionConditionType = "RegionScheduled"
	// DisruptionTarget indicates the region is about to be terminated due to a
	// disruption (such as preemption, eviction API or garbage-collection).
	DisruptionTarget RegionConditionType = "DisruptionTarget"
	// RegionReadyToStartContainers region sandbox is successfully configured and
	// the region is ready to launch containers.
	RegionReadyToStartContainers RegionConditionType = "RegionReadyToStartContainers"
)

// These are reasons for a region's transition to a condition.
const (
	// RegionReasonUnschedulable reason in RegionScheduled RegionCondition means that the scheduler
	// can't schedule the region right now, for example due to insufficient resources in the cluster.
	RegionReasonUnschedulable = "Unschedulable"

	// RegionReasonSchedulingGated reason in RegionScheduled RegionCondition means that the scheduler
	// skips scheduling the region because one or more scheduling gates are still present.
	RegionReasonSchedulingGated = "SchedulingGated"

	// RegionReasonSchedulerError reason in RegionScheduled RegionCondition means that some internal error happens
	// during scheduling, for example due to nodeAffinity parsing errors.
	RegionReasonSchedulerError = "SchedulerError"

	// TerminationByKubelet reason in DisruptionTarget region condition indicates that the termination
	// is initiated by kubelet
	RegionReasonTerminationByKubelet = "TerminationByKubelet"

	// RegionReasonPreemptionByScheduler reason in DisruptionTarget region condition indicates that the
	// disruption was initiated by scheduler's preemption.
	RegionReasonPreemptionByScheduler = "PreemptionByScheduler"
)

// RegionCondition contains details for the current condition of this region.
type RegionCondition struct {
	// Type is the type of the condition.
	Type RegionConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=RegionConditionType"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// Last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty" protobuf:"bytes,3,opt,name=lastProbeTime"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// RegionPhase defines the phase in which a region is in
type RegionPhase string

// These are the valid phases of runner.
const (
	// RegionPending means the runner has been created/added by the system, but not configured.
	RegionPending RegionPhase = "Pending"
	// RegionTerminated means the region has been removed from the cluster.
	RegionTerminated RegionPhase = "Terminated"
	RegionSucceeded  RegionPhase = "Succeeded"
	RegionFailed     RegionPhase = "Failed"
	RegionUnknown    RegionPhase = "Unknown"
	RegionPeering    RegionPhase = "Peering"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Region the olive runner
type Region struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RegionSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status RegionStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// RegionSpec is the specification of a Region.
type RegionSpec struct {
	Id               int64           `json:"id" protobuf:"varint,1,opt,name=id"`
	DeploymentId     int64           `json:"deploymentId" protobuf:"varint,2,opt,name=deploymentId"`
	Replicas         []RegionReplica `json:"replicas" protobuf:"bytes,3,rep,name=replicas"`
	ElectionRTT      int64           `json:"electionRTT" protobuf:"varint,4,opt,name=electionRTT"`
	HeartbeatRTT     int64           `json:"heartbeatRTT" protobuf:"varint,5,opt,name=heartbeatRTT"`
	Priority         *int32          `json:"priority" protobuf:"varint,6,opt,name=priority"`
	SchedulerName    string          `json:"schedulerName" protobuf:"bytes,7,opt,name=schedulerName"`
	DefinitionsLimit int64           `json:"definitionsLimit" protobuf:"varint,8,opt,name=definitionsLimit"`

	// RunnerSelector is a selector which must be true for the region to fit on a runner.
	// Selector which must match a runner's labels for the region to be scheduled on that runner.
	// +optional
	// +mapType=atomic
	RunnerSelector map[string]string `json:"runnerSelector,omitempty" protobuf:"bytes,9,rep,name=runnerSelector"`

	// If specified, the region's scheduling constraints
	// +optional
	Affinity *Affinity `json:"affinity,omitempty" protobuf:"bytes,10,opt,name=affinity"`

	RunnerNames []string `json:"regionNames" protobuf:"bytes,11,rep,name=regionNames"`

	// If specified, the region's tolerations.
	// +optional
	// +listType=atomic
	Tolerations []Toleration `json:"tolerations,omitempty" protobuf:"bytes,12,rep,name=tolerations"`

	// SchedulingGates is an opaque list of values that if specified will block scheduling the region.
	// If schedulingGates is not empty, the region will stay in the SchedulingGated state and the
	// scheduler will not attempt to schedule the region.
	//
	// SchedulingGates can only be set at region creation time, and be removed only afterwards.
	//
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	// +optional
	SchedulingGates []RegionSchedulingGate `json:"schedulingGates,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,13,rep,name=schedulingGates"`
}

type RegionReplica struct {
	Id          int64  `json:"id" protobuf:"varint,1,opt,name=id"`
	Runner      int64  `json:"runner" protobuf:"varint,2,opt,name=runner"`
	Region      int64  `json:"region" protobuf:"varint,3,opt,name=region"`
	RaftAddress string `json:"raftAddress" protobuf:"bytes,4,opt,name=raftAddress"`
	IsNonVoting bool   `json:"isNonVoting" protobuf:"varint,5,opt,name=isNonVoting"`
	IsWitness   bool   `json:"isWitness" protobuf:"varint,6,opt,name=isWitness"`
	IsJoin      bool   `json:"isJoin" protobuf:"varint,7,opt,name=isJoin"`
}

type RegionStatus struct {
	Phase   RegionPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=RegionPhase"`
	Message string      `json:"message" protobuf:"bytes,2,opt,name=message"`

	Leader      int64 `json:"leader" protobuf:"varint,3,opt,name=leader"`
	Definitions int64 `json:"definitions" protobuf:"varint,4,opt,name=definitions"`

	// Current service state of region.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []RegionCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,5,rep,name=conditions"`

	Stat *RegionStat `json:"stat" protobuf:"bytes,6,opt,name=stat"`
}

// +k8s:deepcopy-gen:true

// RegionStat is the stat information of Region
type RegionStat struct {
	Leader             int64  `json:"leader" protobuf:"varint,1,opt,name=leader"`
	Term               int64  `json:"term" protobuf:"varint,2,opt,name=term"`
	Replicas           int32  `json:"replicas" protobuf:"varint,3,opt,name=replicas"`
	Definitions        int64  `json:"definitions" protobuf:"varint,4,opt,name=definitions"`
	RunningDefinitions int64  `json:"runningDefinitions" protobuf:"varint,5,opt,name=runningDefinitions"`
	BpmnProcesses      int64  `json:"bpmnProcesses" protobuf:"varint,6,opt,name=bpmnProcesses"`
	BpmnEvents         int64  `json:"bpmnEvents" protobuf:"varint,7,opt,name=bpmnEvents"`
	BpmnTasks          int64  `json:"bpmnTasks" protobuf:"varint,8,opt,name=bpmnTasks"`
	Message            string `json:"message" protobuf:"bytes,9,opt,name=message"`
	Timestamp          int64  `json:"timestamp" protobuf:"varint,10,opt,name=timestamp"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionList is a list of Region objects.
type RegionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Region
	Items []Region `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// The below types are used by olive_client and api_server.

// ConditionStatus defines conditions of resources
type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition;
// "ConditionFalse" means a resource is not in the condition; "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// NamespaceSpec describes the attributes on a Namespace
type NamespaceSpec struct {
	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage
	Finalizers []FinalizerName `json:"finalizers" protobuf:"bytes,1,rep,name=finalizers,casttype=FinalizerName"`
}

// FinalizerName is the name identifying a finalizer during namespace lifecycle.
type FinalizerName string

// FinalizerOlive there are internal finalizer values to Olive, must be qualified name unless defined here or
// in metav1.
const (
	FinalizerOlive FinalizerName = "Olive"
)

// NamespaceStatus is information about the current status of a Namespace.
type NamespaceStatus struct {
	// Phase is the current lifecycle phase of the namespace.
	// +optional
	Phase NamespacePhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=NamespacePhase"`
	// +optional
	Conditions []NamespaceCondition `json:"conditions" protobuf:"bytes,2,rep,name=conditions"`
}

// NamespacePhase defines the phase in which the namespace is
type NamespacePhase string

// These are the valid phases of a namespace.
const (
	// NamespaceActive means the namespace is available for use in the system
	NamespaceActive NamespacePhase = "Active"
	// NamespaceTerminating means the namespace is undergoing graceful termination
	NamespaceTerminating NamespacePhase = "Terminating"
)

// NamespaceConditionType defines constants reporting on status during namespace lifetime and deletion progress
type NamespaceConditionType string

// These are valid conditions of a namespace.
const (
	NamespaceDeletionDiscoveryFailure NamespaceConditionType = "NamespaceDeletionDiscoveryFailure"
	NamespaceDeletionContentFailure   NamespaceConditionType = "NamespaceDeletionContentFailure"
	NamespaceDeletionGVParsingFailure NamespaceConditionType = "NamespaceDeletionGroupVersionParsingFailure"
)

// NamespaceCondition contains details about state of namespace.
type NamespaceCondition struct {
	// Type of namespace controller condition.
	Type NamespaceConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=NamespaceConditionType"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// +optional
	Reason string `json:"reason" protobuf:"bytes,4,opt,name=reason"`
	// +optional
	Message string `json:"message" protobuf:"bytes,5,opt,name=message"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Namespace provides a scope for Names.
// Use of multiple namespaces is optional
type Namespace struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the Namespace.
	// +optional
	Spec NamespaceSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status describes the current status of a Namespace
	// +optional
	Status NamespaceStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NamespaceList is a list of Namespaces.
type NamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Items []Namespace `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ResourceName is the name identifying various resources in a ResourceList.
type ResourceName string

// Resource names must be not more than 63 characters, consisting of upper- or lower-case alphanumeric characters,
// with the -, _, and . characters allowed anywhere, except the first or last character.
// The default convention, matching that for annotations, is to use lower-case names, with dashes, rather than
// camel case, separating compound words.
// Fully-qualified resource typenames are constructed from a DNS-style subdomain, followed by a slash `/` and a name.
const (
	// CPU, in cores. (500m = .5 cores)
	ResourceCPU ResourceName = "cpu"
	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	ResourceMemory ResourceName = "memory"
	// Volume size, in bytes (e,g. 5Gi = 5GiB = 5 * 1024 * 1024 * 1024)
	ResourceStorage ResourceName = "storage"
	// Local ephemeral storage, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	ResourceEphemeralStorage ResourceName = "ephemeral-storage"
)

// The following identify resource prefix for olive object types
const (
	// ResourceHugePagesPrefix HugePages, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	// As burst is not supported for HugePages, we would only quota its request, and ignore the limit.
	ResourceHugePagesPrefix = "requests.hugepages-"
	// DefaultResourcePrefix Default resource prefix
	DefaultResourcePrefix = "requests."

	// ResourceRequestsHugePagesPrefix HugePages request, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	// As burst is not supported for HugePages, we would only quota its request, and ignore the limit.
	ResourceRequestsHugePagesPrefix = "requests.hugepages-"
	// DefaultResourceRequestsPrefix Default resource requests prefix
	DefaultResourceRequestsPrefix = "requests."
)

// The following identify resource constants for olive object types
const (
	// ResourceRegions regions number
	ResourceRegions ResourceName = "Regions"
)

// ResourceList is a set of (resource name, quantity) pairs.
type ResourceList map[ResourceName]resource.Quantity

// EventSource contains information for an event.
type EventSource struct {
	// Component from which the event is generated.
	// +optional
	Component string `json:"component,omitempty" protobuf:"bytes,1,opt,name=component"`
	// Node name on which the event is generated.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,2,opt,name=host"`
}

// Valid values for event types (new types could be added in future)
const (
	// Information only and will not cause any problems
	EventTypeNormal string = "Normal"
	// These events are to warn that something might go wrong
	EventTypeWarning string = "Warning"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Event is a report of an event somewhere in the cluster.  Events
// have a limited retention time and triggers and messages may evolve
// with time.  Event consumers should not rely on the timing of an event
// with a given Reason reflecting a consistent underlying trigger, or the
// continued existence of events with that Reason.  Events should be
// treated as informative, best-effort, supplemental data.
type Event struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// The object that this event is about.
	InvolvedObject ObjectReference `json:"involvedObject" protobuf:"bytes,2,opt,name=involvedObject"`

	// This should be a short, machine understandable string that gives the reason
	// for the transition into the object's current status.
	// TODO: provide exact specification for format.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`

	// A human-readable description of the status of this operation.
	// TODO: decide on maximum length.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// The component reporting this event. Should be a short machine understandable string.
	// +optional
	Source EventSource `json:"source,omitempty" protobuf:"bytes,5,opt,name=source"`

	// The time at which the event was first recorded. (Time of server receipt is in TypeMeta.)
	// +optional
	FirstTimestamp metav1.Time `json:"firstTimestamp,omitempty" protobuf:"bytes,6,opt,name=firstTimestamp"`

	// The time at which the most recent occurrence of this event was recorded.
	// +optional
	LastTimestamp metav1.Time `json:"lastTimestamp,omitempty" protobuf:"bytes,7,opt,name=lastTimestamp"`

	// The number of times this event has occurred.
	// +optional
	Count int32 `json:"count,omitempty" protobuf:"varint,8,opt,name=count"`

	// Type of this event (Normal, Warning), new types could be added in the future
	// +optional
	Type string `json:"type,omitempty" protobuf:"bytes,9,opt,name=type"`

	// Time when this Event was first observed.
	// +optional
	EventTime metav1.MicroTime `json:"eventTime,omitempty" protobuf:"bytes,10,opt,name=eventTime"`

	// Data about the Event series this event represents or nil if it's a singleton Event.
	// +optional
	Series *EventSeries `json:"series,omitempty" protobuf:"bytes,11,opt,name=series"`

	// What action was taken/failed regarding to the Regarding object.
	// +optional
	Action string `json:"action,omitempty" protobuf:"bytes,12,opt,name=action"`

	// Optional secondary object for more complex actions.
	// +optional
	Related *ObjectReference `json:"related,omitempty" protobuf:"bytes,13,opt,name=related"`

	// Name of the controller that emitted this Event, e.g. `kubernetes.io/kubelet`.
	// +optional
	ReportingController string `json:"reportingComponent" protobuf:"bytes,14,opt,name=reportingComponent"`

	// ID of the controller instance, e.g. `olive-runner-xyzf`.
	// +optional
	ReportingInstance string `json:"reportingInstance" protobuf:"bytes,15,opt,name=reportingInstance"`
}

// EventSeries contain information on series of events, i.e. thing that was/is happening
// continuously for some time.
type EventSeries struct {
	// Number of occurrences in this series up to the last heartbeat time
	Count int32 `json:"count,omitempty" protobuf:"varint,1,name=count"`
	// Time of the last occurrence observed
	LastObservedTime metav1.MicroTime `json:"lastObservedTime,omitempty" protobuf:"bytes,2,name=lastObservedTime"`

	// +k8s:deprecated=state,protobuf=3
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventList is a list of events.
type EventList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of events
	Items []Event `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// DefPhase defines the phase in which a definition is in
type DefPhase string

// These are the valid phases of runner.
const (
	// DefPending means the runner has been created/added by the system, but not configured.
	DefPending DefPhase = "Pending"
	// DefTerminated means the runner has been removed from the cluster.
	DefTerminated DefPhase = "Terminated"
	// DefSucceeded means definition binding with region
	DefSucceeded DefPhase = "Succeeded"
	// DefFailed means that region is not ok
	DefFailed DefPhase = "Failed"
)

// +genclient
// +k8s:protobuf-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Definition is bpmn definitions
type Definition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   DefinitionSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status DefinitionStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type DefinitionSpec struct {
	Content string `json:"content,omitempty" protobuf:"bytes,1,opt,name=content"`
	Version int64  `json:"version,omitempty" protobuf:"varint,2,opt,name=version"`
	// the name of olive region
	RegionName    string `json:"regionName,omitempty" protobuf:"bytes,3,opt,name=regionName"`
	Priority      *int64 `json:"priority,omitempty" protobuf:"varint,4,opt,name=priority"`
	SchedulerName string `json:"schedulerName" protobuf:"bytes,5,opt,name=schedulerName"`
}

type DefinitionStatus struct {
	Phase DefPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=DefPhase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DefinitionList is a list of Definition objects.
type DefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Definition
	Items []Definition `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ProcessPhase defines the phase in which a bpmn process instance is in
type ProcessPhase string

// These are the valid phases of runner.
const (
	// ProcessPending means the runner has been created/added by the system, but not configured.
	ProcessPending ProcessPhase = "Pending"
	// ProcessTerminated means the runner has been removed from the cluster.
	ProcessTerminated ProcessPhase = "Terminated"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Process is bpmn process instance
type Process struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ProcessSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status ProcessStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

type ProcessSpec struct {
	// the id of Definition
	Definition string `json:"definition" protobuf:"bytes,1,opt,name=definition"`
	// the version if Definition
	Version int64 `json:"version" protobuf:"varint,2,opt,name=version"`
	// the process id of bpmn Process Element in Definition
	BpmnProcess string                        `json:"bpmnProcess" protobuf:"bytes,3,opt,name=bpmnProcess"`
	Headers     map[string]string             `json:"headers" protobuf:"bytes,4,rep,name=headers"`
	Properties  map[string]apidiscoveryv1.Box `json:"properties" protobuf:"bytes,5,rep,name=properties"`
	DataObjects map[string]string             `json:"dataObjects" protobuf:"bytes,6,rep,name=dataObjects"`
}

type ProcessStatus struct {
	Phase   ProcessPhase `json:"phase" protobuf:"bytes,1,opt,name=phase,casttype=ProcessPhase"`
	Message string       `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	// the id of olive region
	Region int64 `json:"region" protobuf:"varint,3,opt,name=region"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProcessList is a list of Process objects.
type ProcessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of Process
	Items []Process `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProcessStat is stat information of Process
type ProcessStat struct {
	metav1.TypeMeta `json:",inline"`

	// The id of Process
	Id                string `json:"id" protobuf:"bytes,1,opt,name=id"`
	DefinitionContent string `json:"definitionContent" protobuf:"bytes,2,opt,name=definitionContent"`

	State ProcessRunningState `json:"processState" protobuf:"bytes,3,opt,name=processState"`

	Attempts    int64                     `json:"attempts" protobuf:"varint,4,opt,name=attempts"`
	FlowRunners map[string]FlowRunnerStat `json:"flowRunners" protobuf:"bytes,5,rep,name=flowRunners"`
	StartTime   int64                     `json:"startTime" protobuf:"varint,6,opt,name=startTime"`
	EndTime     int64                     `json:"endTime" protobuf:"varint,7,opt,name=endTime"`
}

type ProcessRunningState struct {
	Properties  map[string]string `json:"properties" protobuf:"bytes,1,rep,name=properties"`
	DataObjects map[string]string `json:"dataObjects" protobuf:"bytes,2,rep,name=dataObjects"`
	Variables   map[string]string `json:"variables" protobuf:"bytes,3,rep,name=variables"`
}

type FlowRunnerStat struct {
	Id          string            `json:"id" protobuf:"bytes,1,opt,name=id"`
	Name        string            `json:"name" protobuf:"bytes,2,opt,name=name"`
	Headers     map[string]string `json:"headers" protobuf:"bytes,3,rep,name=headers"`
	Properties  map[string]string `json:"properties" protobuf:"bytes,4,rep,name=properties"`
	DataObjects map[string]string `json:"dataObjects" protobuf:"bytes,5,rep,name=dataObjects"`
	StartTime   int64             `json:"startTime" protobuf:"varint,6,opt,name=startTime"`
	EndTime     int64             `json:"endTime" protobuf:"varint,7,opt,name=endTime"`
}
