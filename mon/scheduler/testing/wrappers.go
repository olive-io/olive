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

package testing

import (
	"time"

	v1 "k8s.io/api/core/v1"
	resourcev1alpha2 "k8s.io/api/resource/v1alpha2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

var zero int64

// RunnerSelectorWrapper wraps a RunnerSelector inside.
type RunnerSelectorWrapper struct{ corev1.RunnerSelector }

// MakeRunnerSelector creates a RunnerSelector wrapper.
func MakeRunnerSelector() *RunnerSelectorWrapper {
	return &RunnerSelectorWrapper{corev1.RunnerSelector{}}
}

// In injects a matchExpression (with an operator IN) as a selectorTerm
// to the inner runnerSelector.
// NOTE: appended selecterTerms are ORed.
func (s *RunnerSelectorWrapper) In(key string, vals []string) *RunnerSelectorWrapper {
	expression := corev1.RunnerSelectorRequirement{
		Key:      key,
		Operator: corev1.RunnerSelectorOpIn,
		Values:   vals,
	}
	selectorTerm := corev1.RunnerSelectorTerm{}
	selectorTerm.MatchExpressions = append(selectorTerm.MatchExpressions, expression)
	s.RunnerSelectorTerms = append(s.RunnerSelectorTerms, selectorTerm)
	return s
}

// NotIn injects a matchExpression (with an operator NotIn) as a selectorTerm
// to the inner runnerSelector.
func (s *RunnerSelectorWrapper) NotIn(key string, vals []string) *RunnerSelectorWrapper {
	expression := corev1.RunnerSelectorRequirement{
		Key:      key,
		Operator: corev1.RunnerSelectorOpNotIn,
		Values:   vals,
	}
	selectorTerm := corev1.RunnerSelectorTerm{}
	selectorTerm.MatchExpressions = append(selectorTerm.MatchExpressions, expression)
	s.RunnerSelectorTerms = append(s.RunnerSelectorTerms, selectorTerm)
	return s
}

// Obj returns the inner RunnerSelector.
func (s *RunnerSelectorWrapper) Obj() *corev1.RunnerSelector {
	return &s.RunnerSelector
}

// LabelSelectorWrapper wraps a LabelSelector inside.
type LabelSelectorWrapper struct{ metav1.LabelSelector }

// MakeLabelSelector creates a LabelSelector wrapper.
func MakeLabelSelector() *LabelSelectorWrapper {
	return &LabelSelectorWrapper{metav1.LabelSelector{}}
}

// Label applies a {k,v} pair to the inner LabelSelector.
func (s *LabelSelectorWrapper) Label(k, v string) *LabelSelectorWrapper {
	if s.MatchLabels == nil {
		s.MatchLabels = make(map[string]string)
	}
	s.MatchLabels[k] = v
	return s
}

// In injects a matchExpression (with an operator In) to the inner labelSelector.
func (s *LabelSelectorWrapper) In(key string, vals []string) *LabelSelectorWrapper {
	expression := metav1.LabelSelectorRequirement{
		Key:      key,
		Operator: metav1.LabelSelectorOpIn,
		Values:   vals,
	}
	s.MatchExpressions = append(s.MatchExpressions, expression)
	return s
}

// NotIn injects a matchExpression (with an operator NotIn) to the inner labelSelector.
func (s *LabelSelectorWrapper) NotIn(key string, vals []string) *LabelSelectorWrapper {
	expression := metav1.LabelSelectorRequirement{
		Key:      key,
		Operator: metav1.LabelSelectorOpNotIn,
		Values:   vals,
	}
	s.MatchExpressions = append(s.MatchExpressions, expression)
	return s
}

// Exists injects a matchExpression (with an operator Exists) to the inner labelSelector.
func (s *LabelSelectorWrapper) Exists(k string) *LabelSelectorWrapper {
	expression := metav1.LabelSelectorRequirement{
		Key:      k,
		Operator: metav1.LabelSelectorOpExists,
	}
	s.MatchExpressions = append(s.MatchExpressions, expression)
	return s
}

// NotExist injects a matchExpression (with an operator NotExist) to the inner labelSelector.
func (s *LabelSelectorWrapper) NotExist(k string) *LabelSelectorWrapper {
	expression := metav1.LabelSelectorRequirement{
		Key:      k,
		Operator: metav1.LabelSelectorOpDoesNotExist,
	}
	s.MatchExpressions = append(s.MatchExpressions, expression)
	return s
}

// Obj returns the inner LabelSelector.
func (s *LabelSelectorWrapper) Obj() *metav1.LabelSelector {
	return &s.LabelSelector
}

// RegionWrapper wraps a Region inside.
type RegionWrapper struct{ corev1.Region }

// MakeRegion creates a Region wrapper.
func MakeRegion() *RegionWrapper {
	return &RegionWrapper{corev1.Region{}}
}

// Obj returns the inner Region.
func (p *RegionWrapper) Obj() *corev1.Region {
	return &p.Region
}

// Name sets `s` as the name of the inner region.
func (p *RegionWrapper) Name(s string) *RegionWrapper {
	p.SetName(s)
	return p
}

// UID sets `s` as the UID of the inner region.
func (p *RegionWrapper) UID(s string) *RegionWrapper {
	p.SetUID(types.UID(s))
	return p
}

// SchedulerName sets `s` as the scheduler name of the inner region.
func (p *RegionWrapper) SchedulerName(s string) *RegionWrapper {
	p.Spec.SchedulerName = s
	return p
}

// Namespace sets `s` as the namespace of the inner region.
func (p *RegionWrapper) Namespace(s string) *RegionWrapper {
	p.SetNamespace(s)
	return p
}

// OwnerReference updates the owning controller of the region.
func (p *RegionWrapper) OwnerReference(name string, gvk schema.GroupVersionKind) *RegionWrapper {
	p.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			Name:       name,
			Controller: ptr.To(true),
		},
	}
	return p
}

// Priority sets a priority value into RegionSpec of the inner region.
func (p *RegionWrapper) Priority(val int32) *RegionWrapper {
	p.Spec.Priority = &val
	return p
}

// CreationTimestamp sets the inner region's CreationTimestamp.
func (p *RegionWrapper) CreationTimestamp(t metav1.Time) *RegionWrapper {
	p.ObjectMeta.CreationTimestamp = t
	return p
}

// Terminating sets the inner region's deletionTimestamp to current timestamp.
func (p *RegionWrapper) Terminating() *RegionWrapper {
	now := metav1.Now()
	p.DeletionTimestamp = &now
	return p
}

// Runners sets `s` as the runnerNames of the inner region.
func (p *RegionWrapper) Runners(s ...string) *RegionWrapper {
	p.Spec.RunnerNames = s
	return p
}

// RunnerSelector sets `m` as the runnerSelector of the inner region.
func (p *RegionWrapper) RunnerSelector(m map[string]string) *RegionWrapper {
	p.Spec.RunnerSelector = m
	return p
}

// RunnerAffinityIn creates a HARD runner affinity (with the operator In)
// and injects into the inner region.
func (p *RegionWrapper) RunnerAffinityIn(key string, vals []string) *RegionWrapper {
	if p.Spec.Affinity == nil {
		p.Spec.Affinity = &corev1.Affinity{}
	}
	if p.Spec.Affinity.RunnerAffinity == nil {
		p.Spec.Affinity.RunnerAffinity = &corev1.RunnerAffinity{}
	}
	runnerSelector := MakeRunnerSelector().In(key, vals).Obj()
	p.Spec.Affinity.RunnerAffinity.RequiredDuringSchedulingIgnoredDuringExecution = runnerSelector
	return p
}

// RunnerAffinityNotIn creates a HARD runner affinity (with the operator NotIn)
// and injects into the inner region.
func (p *RegionWrapper) RunnerAffinityNotIn(key string, vals []string) *RegionWrapper {
	if p.Spec.Affinity == nil {
		p.Spec.Affinity = &corev1.Affinity{}
	}
	if p.Spec.Affinity.RunnerAffinity == nil {
		p.Spec.Affinity.RunnerAffinity = &corev1.RunnerAffinity{}
	}
	runnerSelector := MakeRunnerSelector().NotIn(key, vals).Obj()
	p.Spec.Affinity.RunnerAffinity.RequiredDuringSchedulingIgnoredDuringExecution = runnerSelector
	return p
}

// Condition adds a `condition(Type, Status, Reason)` to .Status.Conditions.
func (p *RegionWrapper) Condition(t corev1.RegionConditionType, s corev1.ConditionStatus, r string) *RegionWrapper {
	p.Status.Conditions = append(p.Status.Conditions, corev1.RegionCondition{Type: t, Status: s, Reason: r})
	return p
}

// Conditions sets `conditions` as .status.Conditions of the inner region.
func (p *RegionWrapper) Conditions(conditions []corev1.RegionCondition) *RegionWrapper {
	p.Status.Conditions = append(p.Status.Conditions, conditions...)
	return p
}

// Toleration creates a toleration (with the operator Exists)
// and injects into the inner region.
func (p *RegionWrapper) Toleration(key string) *RegionWrapper {
	p.Spec.Tolerations = append(p.Spec.Tolerations, corev1.Toleration{
		Key:      key,
		Operator: corev1.TolerationOpExists,
	})
	return p
}

// SchedulingGates sets `gates` as additional SchedulerGates of the inner region.
func (p *RegionWrapper) SchedulingGates(gates []string) *RegionWrapper {
	for _, gate := range gates {
		p.Spec.SchedulingGates = append(p.Spec.SchedulingGates, corev1.RegionSchedulingGate{Name: gate})
	}
	return p
}

// RegionAffinityKind represents different kinds of RegionAffinity.
type RegionAffinityKind int

const (
	// NilRegionAffinity is a no-op which doesn't apply any RegionAffinity.
	NilRegionAffinity RegionAffinityKind = iota
	// RegionAffinityWithRequiredReq applies a HARD requirement to region.spec.affinity.RegionAffinity.
	RegionAffinityWithRequiredReq
	// RegionAffinityWithPreferredReq applies a SOFT requirement to region.spec.affinity.RegionAffinity.
	RegionAffinityWithPreferredReq
	// RegionAffinityWithRequiredPreferredReq applies HARD and SOFT requirements to region.spec.affinity.RegionAffinity.
	RegionAffinityWithRequiredPreferredReq
	// RegionAntiAffinityWithRequiredReq applies a HARD requirement to region.spec.affinity.RegionAntiAffinity.
	RegionAntiAffinityWithRequiredReq
	// RegionAntiAffinityWithPreferredReq applies a SOFT requirement to region.spec.affinity.RegionAntiAffinity.
	RegionAntiAffinityWithPreferredReq
	// RegionAntiAffinityWithRequiredPreferredReq applies HARD and SOFT requirements to region.spec.affinity.RegionAntiAffinity.
	RegionAntiAffinityWithRequiredPreferredReq
)

// RegionAffinity creates a RegionAffinity with topology key and label selector
// and injects into the inner region.
func (p *RegionWrapper) RegionAffinity(topologyKey string, labelSelector *metav1.LabelSelector, kind RegionAffinityKind) *RegionWrapper {
	if kind == NilRegionAffinity {
		return p
	}

	if p.Spec.Affinity == nil {
		p.Spec.Affinity = &corev1.Affinity{}
	}
	if p.Spec.Affinity.RegionAffinity == nil {
		p.Spec.Affinity.RegionAffinity = &corev1.RegionAffinity{}
	}
	term := corev1.RegionAffinityTerm{LabelSelector: labelSelector, TopologyKey: topologyKey}
	switch kind {
	case RegionAffinityWithRequiredReq:
		p.Spec.Affinity.RegionAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			term,
		)
	case RegionAffinityWithPreferredReq:
		p.Spec.Affinity.RegionAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			corev1.WeightedRegionAffinityTerm{Weight: 1, RegionAffinityTerm: term},
		)
	case RegionAffinityWithRequiredPreferredReq:
		p.Spec.Affinity.RegionAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			term,
		)
		p.Spec.Affinity.RegionAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			corev1.WeightedRegionAffinityTerm{Weight: 1, RegionAffinityTerm: term},
		)
	}
	return p
}

// RegionAntiAffinity creates a RegionAntiAffinity with topology key and label selector
// and injects into the inner region.
func (p *RegionWrapper) RegionAntiAffinity(topologyKey string, labelSelector *metav1.LabelSelector, kind RegionAffinityKind) *RegionWrapper {
	if kind == NilRegionAffinity {
		return p
	}

	if p.Spec.Affinity == nil {
		p.Spec.Affinity = &corev1.Affinity{}
	}
	if p.Spec.Affinity.RegionAntiAffinity == nil {
		p.Spec.Affinity.RegionAntiAffinity = &corev1.RegionAntiAffinity{}
	}
	term := corev1.RegionAffinityTerm{LabelSelector: labelSelector, TopologyKey: topologyKey}
	switch kind {
	case RegionAntiAffinityWithRequiredReq:
		p.Spec.Affinity.RegionAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			term,
		)
	case RegionAntiAffinityWithPreferredReq:
		p.Spec.Affinity.RegionAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			corev1.WeightedRegionAffinityTerm{Weight: 1, RegionAffinityTerm: term},
		)
	case RegionAntiAffinityWithRequiredPreferredReq:
		p.Spec.Affinity.RegionAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			term,
		)
		p.Spec.Affinity.RegionAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
			p.Spec.Affinity.RegionAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			corev1.WeightedRegionAffinityTerm{Weight: 1, RegionAffinityTerm: term},
		)
	}
	return p
}

// RegionAffinityExists creates a RegionAffinity with the operator "Exists"
// and injects into the inner region.
func (p *RegionWrapper) RegionAffinityExists(labelKey, topologyKey string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().Exists(labelKey).Obj()
	p.RegionAffinity(topologyKey, labelSelector, kind)
	return p
}

// RegionAntiAffinityExists creates a RegionAntiAffinity with the operator "Exists"
// and injects into the inner region.
func (p *RegionWrapper) RegionAntiAffinityExists(labelKey, topologyKey string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().Exists(labelKey).Obj()
	p.RegionAntiAffinity(topologyKey, labelSelector, kind)
	return p
}

// RegionAffinityNotExists creates a RegionAffinity with the operator "NotExists"
// and injects into the inner region.
func (p *RegionWrapper) RegionAffinityNotExists(labelKey, topologyKey string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().NotExist(labelKey).Obj()
	p.RegionAffinity(topologyKey, labelSelector, kind)
	return p
}

// RegionAntiAffinityNotExists creates a RegionAntiAffinity with the operator "NotExists"
// and injects into the inner region.
func (p *RegionWrapper) RegionAntiAffinityNotExists(labelKey, topologyKey string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().NotExist(labelKey).Obj()
	p.RegionAntiAffinity(topologyKey, labelSelector, kind)
	return p
}

// RegionAffinityIn creates a RegionAffinity with the operator "In"
// and injects into the inner region.
func (p *RegionWrapper) RegionAffinityIn(labelKey, topologyKey string, vals []string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().In(labelKey, vals).Obj()
	p.RegionAffinity(topologyKey, labelSelector, kind)
	return p
}

// RegionAntiAffinityIn creates a RegionAntiAffinity with the operator "In"
// and injects into the inner region.
func (p *RegionWrapper) RegionAntiAffinityIn(labelKey, topologyKey string, vals []string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().In(labelKey, vals).Obj()
	p.RegionAntiAffinity(topologyKey, labelSelector, kind)
	return p
}

// RegionAffinityNotIn creates a RegionAffinity with the operator "NotIn"
// and injects into the inner region.
func (p *RegionWrapper) RegionAffinityNotIn(labelKey, topologyKey string, vals []string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().NotIn(labelKey, vals).Obj()
	p.RegionAffinity(topologyKey, labelSelector, kind)
	return p
}

// RegionAntiAffinityNotIn creates a RegionAntiAffinity with the operator "NotIn"
// and injects into the inner region.
func (p *RegionWrapper) RegionAntiAffinityNotIn(labelKey, topologyKey string, vals []string, kind RegionAffinityKind) *RegionWrapper {
	labelSelector := MakeLabelSelector().NotIn(labelKey, vals).Obj()
	p.RegionAntiAffinity(topologyKey, labelSelector, kind)
	return p
}

// SpreadConstraint constructs a TopologySpreadConstraint object and injects
// into the inner region.
func (p *RegionWrapper) SpreadConstraint(maxSkew int, tpKey string, mode corev1.UnsatisfiableConstraintAction, selector *metav1.LabelSelector, minDomains *int32, runnerAffinityPolicy, runnerTaintsPolicy *corev1.RunnerInclusionPolicy, matchLabelKeys []string) *RegionWrapper {
	c := corev1.TopologySpreadConstraint{
		MaxSkew:              int32(maxSkew),
		TopologyKey:          tpKey,
		WhenUnsatisfiable:    mode,
		LabelSelector:        selector,
		MinDomains:           minDomains,
		RunnerAffinityPolicy: runnerAffinityPolicy,
		RunnerTaintsPolicy:   runnerTaintsPolicy,
		MatchLabelKeys:       matchLabelKeys,
	}
	p.Spec.TopologySpreadConstraints = append(p.Spec.TopologySpreadConstraints, c)
	return p
}

// Label sets a {k,v} pair to the inner region label.
func (p *RegionWrapper) Label(k, v string) *RegionWrapper {
	if p.ObjectMeta.Labels == nil {
		p.ObjectMeta.Labels = make(map[string]string)
	}
	p.ObjectMeta.Labels[k] = v
	return p
}

// Labels sets all {k,v} pair provided by `labels` to the inner region labels.
func (p *RegionWrapper) Labels(labels map[string]string) *RegionWrapper {
	for k, v := range labels {
		p.Label(k, v)
	}
	return p
}

// Annotation sets a {k,v} pair to the inner region annotation.
func (p *RegionWrapper) Annotation(key, value string) *RegionWrapper {
	metav1.SetMetaDataAnnotation(&p.ObjectMeta, key, value)
	return p
}

// Annotations sets all {k,v} pair provided by `annotations` to the inner region annotations.
func (p *RegionWrapper) Annotations(annotations map[string]string) *RegionWrapper {
	for k, v := range annotations {
		p.Annotation(k, v)
	}
	return p
}

// PreemptionPolicy sets the give preemption policy to the inner region.
func (p *RegionWrapper) PreemptionPolicy(policy v1.PreemptionPolicy) *RegionWrapper {
	p.Spec.PreemptionPolicy = &policy
	return p
}

// Overhead sets the give ResourceList to the inner region
func (p *RegionWrapper) Overhead(rl v1.ResourceList) *RegionWrapper {
	p.Spec.Overhead = rl
	return p
}

// RunnerWrapper wraps a Runner inside.
type RunnerWrapper struct{ corev1.Runner }

// MakeRunner creates a Runner wrapper.
func MakeRunner() *RunnerWrapper {
	w := &RunnerWrapper{corev1.Runner{}}
	return w.Capacity(nil)
}

// Obj returns the inner Runner.
func (n *RunnerWrapper) Obj() *corev1.Runner {
	return &n.Runner
}

// Name sets `s` as the name of the inner region.
func (n *RunnerWrapper) Name(s string) *RunnerWrapper {
	n.SetName(s)
	return n
}

// UID sets `s` as the UID of the inner region.
func (n *RunnerWrapper) UID(s string) *RunnerWrapper {
	n.SetUID(types.UID(s))
	return n
}

// Label applies a {k,v} label pair to the inner runner.
func (n *RunnerWrapper) Label(k, v string) *RunnerWrapper {
	if n.Labels == nil {
		n.Labels = make(map[string]string)
	}
	n.Labels[k] = v
	return n
}

// Annotation applies a {k,v} annotation pair to the inner runner.
func (n *RunnerWrapper) Annotation(k, v string) *RunnerWrapper {
	if n.Annotations == nil {
		n.Annotations = make(map[string]string)
	}
	metav1.SetMetaDataAnnotation(&n.ObjectMeta, k, v)
	return n
}

// Capacity sets the capacity and the allocatable resources of the inner runner.
// Each entry in `resources` corresponds to a resource name and its quantity.
// By default, the capacity and allocatable number of regions are set to 32.
func (n *RunnerWrapper) Capacity(resources map[corev1.ResourceName]string) *RunnerWrapper {
	res := corev1.ResourceList{
		corev1.ResourceRegions: resource.MustParse("32"),
	}
	for name, value := range resources {
		res[name] = resource.MustParse(value)
	}
	n.Status.Capacity, n.Status.Allocatable = res, res
	return n
}

// Taints applies taints to the inner runner.
func (n *RunnerWrapper) Taints(taints []corev1.Taint) *RunnerWrapper {
	n.Spec.Taints = taints
	return n
}

// Unschedulable applies the unschedulable field.
func (n *RunnerWrapper) Unschedulable(unschedulable bool) *RunnerWrapper {
	n.Spec.Unschedulable = unschedulable
	return n
}

// Condition applies the runner condition.
func (n *RunnerWrapper) Condition(typ corev1.RunnerConditionType, status corev1.ConditionStatus, message, reason string) *RunnerWrapper {
	n.Status.Conditions = []corev1.RunnerCondition{
		{
			Type:               typ,
			Status:             status,
			Message:            message,
			Reason:             reason,
			LastHeartbeatTime:  metav1.Time{Time: time.Now()},
			LastTransitionTime: metav1.Time{Time: time.Now()},
		},
	}
	return n
}

// ResourceClaimWrapper wraps a ResourceClaim inside.
type ResourceClaimWrapper struct{ resourcev1alpha2.ResourceClaim }

// MakeResourceClaim creates a ResourceClaim wrapper.
func MakeResourceClaim() *ResourceClaimWrapper {
	return &ResourceClaimWrapper{resourcev1alpha2.ResourceClaim{}}
}

// FromResourceClaim creates a ResourceClaim wrapper from some existing object.
func FromResourceClaim(other *resourcev1alpha2.ResourceClaim) *ResourceClaimWrapper {
	return &ResourceClaimWrapper{*other.DeepCopy()}
}

// Obj returns the inner ResourceClaim.
func (wrapper *ResourceClaimWrapper) Obj() *resourcev1alpha2.ResourceClaim {
	return &wrapper.ResourceClaim
}

// Name sets `s` as the name of the inner object.
func (wrapper *ResourceClaimWrapper) Name(s string) *ResourceClaimWrapper {
	wrapper.SetName(s)
	return wrapper
}

// UID sets `s` as the UID of the inner object.
func (wrapper *ResourceClaimWrapper) UID(s string) *ResourceClaimWrapper {
	wrapper.SetUID(types.UID(s))
	return wrapper
}

// Namespace sets `s` as the namespace of the inner object.
func (wrapper *ResourceClaimWrapper) Namespace(s string) *ResourceClaimWrapper {
	wrapper.SetNamespace(s)
	return wrapper
}

// OwnerReference updates the owning controller of the object.
func (wrapper *ResourceClaimWrapper) OwnerReference(name, uid string, gvk schema.GroupVersionKind) *ResourceClaimWrapper {
	wrapper.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			Name:       name,
			UID:        types.UID(uid),
			Controller: ptr.To(true),
		},
	}
	return wrapper
}

// AllocationMode sets the allocation mode of the inner object.
func (wrapper *ResourceClaimWrapper) AllocationMode(a resourcev1alpha2.AllocationMode) *ResourceClaimWrapper {
	wrapper.ResourceClaim.Spec.AllocationMode = a
	return wrapper
}

// ResourceClassName sets the resource class name of the inner object.
func (wrapper *ResourceClaimWrapper) ResourceClassName(name string) *ResourceClaimWrapper {
	wrapper.ResourceClaim.Spec.ResourceClassName = name
	return wrapper
}

// Allocation sets the allocation of the inner object.
func (wrapper *ResourceClaimWrapper) Allocation(allocation *resourcev1alpha2.AllocationResult) *ResourceClaimWrapper {
	wrapper.ResourceClaim.Status.Allocation = allocation
	return wrapper
}

// DeallocationRequested sets that field of the inner object.
func (wrapper *ResourceClaimWrapper) DeallocationRequested(deallocationRequested bool) *ResourceClaimWrapper {
	wrapper.ResourceClaim.Status.DeallocationRequested = deallocationRequested
	return wrapper
}

// ReservedFor sets that field of the inner object.
func (wrapper *ResourceClaimWrapper) ReservedFor(consumers ...resourcev1alpha2.ResourceClaimConsumerReference) *ResourceClaimWrapper {
	wrapper.ResourceClaim.Status.ReservedFor = consumers
	return wrapper
}

// RegionSchedulingWrapper wraps a RegionSchedulingContext inside.
type RegionSchedulingWrapper struct {
	resourcev1alpha2.RegionSchedulingContext
}

// MakeRegionSchedulingContexts creates a RegionSchedulingContext wrapper.
func MakeRegionSchedulingContexts() *RegionSchedulingWrapper {
	return &RegionSchedulingWrapper{resourcev1alpha2.RegionSchedulingContext{}}
}

// FromRegionSchedulingContexts creates a RegionSchedulingContext wrapper from an existing object.
func FromRegionSchedulingContexts(other *resourcev1alpha2.RegionSchedulingContext) *RegionSchedulingWrapper {
	return &RegionSchedulingWrapper{*other.DeepCopy()}
}

// Obj returns the inner object.
func (wrapper *RegionSchedulingWrapper) Obj() *resourcev1alpha2.RegionSchedulingContext {
	return &wrapper.RegionSchedulingContext
}

// Name sets `s` as the name of the inner object.
func (wrapper *RegionSchedulingWrapper) Name(s string) *RegionSchedulingWrapper {
	wrapper.SetName(s)
	return wrapper
}

// UID sets `s` as the UID of the inner object.
func (wrapper *RegionSchedulingWrapper) UID(s string) *RegionSchedulingWrapper {
	wrapper.SetUID(types.UID(s))
	return wrapper
}

// Namespace sets `s` as the namespace of the inner object.
func (wrapper *RegionSchedulingWrapper) Namespace(s string) *RegionSchedulingWrapper {
	wrapper.SetNamespace(s)
	return wrapper
}

// OwnerReference updates the owning controller of the inner object.
func (wrapper *RegionSchedulingWrapper) OwnerReference(name, uid string, gvk schema.GroupVersionKind) *RegionSchedulingWrapper {
	wrapper.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         gvk.GroupVersion().String(),
			Kind:               gvk.Kind,
			Name:               name,
			UID:                types.UID(uid),
			Controller:         ptr.To(true),
			BlockOwnerDeletion: ptr.To(true),
		},
	}
	return wrapper
}

// Label applies a {k,v} label pair to the inner object
func (wrapper *RegionSchedulingWrapper) Label(k, v string) *RegionSchedulingWrapper {
	if wrapper.Labels == nil {
		wrapper.Labels = make(map[string]string)
	}
	wrapper.Labels[k] = v
	return wrapper
}

// SelectedRunner sets that field of the inner object.
func (wrapper *RegionSchedulingWrapper) SelectedRunner(s string) *RegionSchedulingWrapper {
	wrapper.Spec.SelectedRunner = s
	return wrapper
}

// PotentialRunners sets that field of the inner object.
func (wrapper *RegionSchedulingWrapper) PotentialRunners(runners ...string) *RegionSchedulingWrapper {
	wrapper.Spec.PotentialRunners = runners
	return wrapper
}

// ResourceClaims sets that field of the inner object.
func (wrapper *RegionSchedulingWrapper) ResourceClaims(statuses ...resourcev1alpha2.ResourceClaimSchedulingStatus) *RegionSchedulingWrapper {
	wrapper.Status.ResourceClaims = statuses
	return wrapper
}
