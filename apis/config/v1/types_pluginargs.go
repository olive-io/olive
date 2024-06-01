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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DefaultPreemptionArgs holds arguments used to configure the
// DefaultPreemption plugin.
type DefaultPreemptionArgs struct {
	metav1.TypeMeta `json:",inline"`

	// MinCandidateRunnersPercentage is the minimum number of candidates to
	// shortlist when dry running preemption as a percentage of number of runners.
	// Must be in the range [0, 100]. Defaults to 10% of the cluster size if
	// unspecified.
	MinCandidateRunnersPercentage int32 `json:"minCandidateRunnersPercentage,omitempty" protobuf:"varint,1,opt,name=minCandidateRunnersPercentage"`
	// MinCandidateRunnersAbsolute is the absolute minimum number of candidates to
	// shortlist. The likely number of candidates enumerated for dry running
	// preemption is given by the formula:
	// numCandidates = max(numRunners * minCandidateRunnersPercentage, minCandidateRunnersAbsolute)
	// We say "likely" because there are other factors such as PDB violations
	// that play a role in the number of candidates shortlisted. Must be at least
	// 0 runners. Defaults to 100 runners if unspecified.
	MinCandidateRunnersAbsolute int32 `json:"minCandidateRunnersAbsolute,omitempty" protobuf:"varint,2,opt,name=minCandidateRunnersAbsolute"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InterRegionAffinityArgs holds arguments used to configure the InterRegionAffinity plugin.
type InterRegionAffinityArgs struct {
	metav1.TypeMeta `json:",inline"`

	// HardRegionAffinityWeight is the scoring weight for existing regions with a
	// matching hard affinity to the incoming region.
	HardRegionAffinityWeight int32 `json:"hardRegionAffinityWeight,omitempty" protobuf:"varint,1,opt,name=hardRegionAffinityWeight"`

	// IgnorePreferredTermsOfExistingRegions configures the scheduler to ignore existing regions' preferred affinity
	// rules when scoring candidate runners, unless the incoming region has inter-region affinities.
	IgnorePreferredTermsOfExistingRegions bool `json:"ignorePreferredTermsOfExistingRegions" protobuf:"varint,2,opt,name=ignorePreferredTermsOfExistingRegions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerResourcesFitArgs holds arguments used to configure the RunnerResourcesFit plugin.
type RunnerResourcesFitArgs struct {
	metav1.TypeMeta `json:",inline"`

	// IgnoredResources is the list of resources that RunnerResources fit filter
	// should ignore. This doesn't apply to scoring.
	// +listType=atomic
	IgnoredResources []string `json:"ignoredResources,omitempty" protobuf:"bytes,1,rep,name=ignoredResources"`
	// IgnoredResourceGroups defines the list of resource groups that RunnerResources fit filter should ignore.
	// e.g. if group is ["example.com"], it will ignore all resource names that begin
	// with "example.com", such as "example.com/aaa" and "example.com/bbb".
	// A resource group name can't contain '/'. This doesn't apply to scoring.
	// +listType=atomic
	IgnoredResourceGroups []string `json:"ignoredResourceGroups,omitempty" protobuf:"bytes,2,rep,name=ignoredResourceGroups"`

	// ScoringStrategy selects the runner resource scoring strategy.
	// The default strategy is LeastAllocated with an equal "cpu" and "memory" weight.
	ScoringStrategy *ScoringStrategy `json:"scoringStrategy,omitempty" protobuf:"bytes,3,opt,name=scoringStrategy"`
}

// RegionTopologySpreadConstraintsDefaulting defines how to set default constraints
// for the RegionTopologySpread plugin.
type RegionTopologySpreadConstraintsDefaulting string

const (
	// SystemDefaulting instructs to use the kubernetes defined default.
	SystemDefaulting RegionTopologySpreadConstraintsDefaulting = "System"
	// ListDefaulting instructs to use the config provided default.
	ListDefaulting RegionTopologySpreadConstraintsDefaulting = "List"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegionTopologySpreadArgs holds arguments used to configure the RegionTopologySpread plugin.
type RegionTopologySpreadArgs struct {
	metav1.TypeMeta `json:",inline"`

	// DefaultConstraints defines topology spread constraints to be applied to
	// regions that don't define any in `region.spec.topologySpreadConstraints`.
	// `.defaultConstraints[*].labelSelectors` must be empty, as they are
	// deduced from the region's membership to Services, ReplicationControllers,
	// ReplicaSets or StatefulSets.
	// When not empty, .defaultingType must be "List".
	// +optional
	// +listType=atomic
	DefaultConstraints []corev1.TopologySpreadConstraint `json:"defaultConstraints,omitempty" protobuf:"bytes,1,rep,name=defaultConstraints"`

	// DefaultingType determines how .defaultConstraints are deduced. Can be one
	// of "System" or "List".
	//
	// - "System": Use kubernetes defined constraints that spread regions among
	//   Runners and Zones.
	// - "List": Use constraints defined in .defaultConstraints.
	//
	// Defaults to "System".
	// +optional
	DefaultingType RegionTopologySpreadConstraintsDefaulting `json:"defaultingType,omitempty" protobuf:"bytes,2,opt,name=defaultingType,casttype=RegionTopologySpreadConstraintsDefaulting"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerResourcesBalancedAllocationArgs holds arguments used to configure RunnerResourcesBalancedAllocation plugin.
type RunnerResourcesBalancedAllocationArgs struct {
	metav1.TypeMeta `json:",inline"`

	// Resources to be managed, the default is "cpu" and "memory" if not specified.
	// +listType=map
	// +listMapKey=name
	Resources []ResourceSpec `json:"resources,omitempty" protobuf:"bytes,1,rep,name=resources"`
}

// UtilizationShapePoint represents single point of priority function shape.
type UtilizationShapePoint struct {
	// Utilization (x axis). Valid values are 0 to 100. Fully utilized runner maps to 100.
	Utilization int32 `json:"utilization" protobuf:"varint,1,opt,name=utilization"`
	// Score assigned to given utilization (y axis). Valid values are 0 to 10.
	Score int32 `json:"score" protobuf:"varint,2,opt,name=score"`
}

// ResourceSpec represents a single resource.
type ResourceSpec struct {
	// Name of the resource.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Weight of the resource.
	Weight int64 `json:"weight,omitempty" protobuf:"varint,2,opt,name=weight"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RunnerAffinityArgs holds arguments to configure the RunnerAffinity plugin.
type RunnerAffinityArgs struct {
	metav1.TypeMeta `json:",inline"`

	// AddedAffinity is applied to all regions additionally to the RunnerAffinity
	// specified in the regionSpec. That is, Runners need to satisfy AddedAffinity
	// AND .spec.RunnerAffinity. AddedAffinity is empty by default (all Runners
	// match).
	// When AddedAffinity is used, some regions with affinity requirements that match
	// a specific Runner (such as Daemonset regions) might remain unschedulable.
	// +optional
	AddedAffinity *corev1.RunnerAffinity `json:"addedAffinity,omitempty" protobuf:"bytes,1,opt,name=addedAffinity"`
}

// ScoringStrategyType the type of scoring strategy used in RunnerResourcesFit plugin.
type ScoringStrategyType string

const (
	// LeastAllocated strategy prioritizes runners with least allocated resources.
	LeastAllocated ScoringStrategyType = "LeastAllocated"
	// MostAllocated strategy prioritizes runners with most allocated resources.
	MostAllocated ScoringStrategyType = "MostAllocated"
	// RequestedToCapacityRatio strategy allows specifying a custom shape function
	// to score runners based on the request to capacity ratio.
	RequestedToCapacityRatio ScoringStrategyType = "RequestedToCapacityRatio"
)

// ScoringStrategy define ScoringStrategyType for runner resource plugin
type ScoringStrategy struct {
	// Type selects which strategy to run.
	Type ScoringStrategyType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type,casttype=ScoringStrategyType"`

	// Resources to consider when scoring.
	// The default resource set includes "cpu" and "memory" with an equal weight.
	// Allowed weights go from 1 to 100.
	// Weight defaults to 1 if not specified or explicitly set to 0.
	// +listType=map
	// +listMapKey=topologyKey
	Resources []ResourceSpec `json:"resources,omitempty" protobuf:"bytes,2,rep,name=resources"`

	// Arguments specific to RequestedToCapacityRatio strategy.
	RequestedToCapacityRatio *RequestedToCapacityRatioParam `json:"requestedToCapacityRatio,omitempty" protobuf:"bytes,3,opt,name=requestedToCapacityRatio"`
}

// RequestedToCapacityRatioParam define RequestedToCapacityRatio parameters
type RequestedToCapacityRatioParam struct {
	// Shape is a list of points defining the scoring function shape.
	// +listType=atomic
	Shape []UtilizationShapePoint `json:"shape,omitempty" protobuf:"bytes,1,rep,name=shape"`
}
