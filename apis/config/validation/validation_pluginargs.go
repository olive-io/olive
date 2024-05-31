/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"fmt"
	"strings"

	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	config "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// supportedScoringStrategyTypes has to be a set of strings for use with field.Unsupported
var supportedScoringStrategyTypes = sets.New(
	string(config.LeastAllocated),
	string(config.MostAllocated),
	string(config.RequestedToCapacityRatio),
)

// ValidateDefaultPreemptionArgs validates that DefaultPreemptionArgs are correct.
func ValidateDefaultPreemptionArgs(path *field.Path, args *config.DefaultPreemptionArgs) error {
	var allErrs field.ErrorList
	percentagePath := path.Child("minCandidateRunnersPercentage")
	absolutePath := path.Child("minCandidateRunnersAbsolute")
	if err := validateMinCandidateRunnersPercentage(args.MinCandidateRunnersPercentage, percentagePath); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateMinCandidateRunnersAbsolute(args.MinCandidateRunnersAbsolute, absolutePath); err != nil {
		allErrs = append(allErrs, err)
	}
	if args.MinCandidateRunnersPercentage == 0 && args.MinCandidateRunnersAbsolute == 0 {
		allErrs = append(allErrs,
			field.Invalid(percentagePath, args.MinCandidateRunnersPercentage, "cannot be zero at the same time as minCandidateRunnersAbsolute"),
			field.Invalid(absolutePath, args.MinCandidateRunnersAbsolute, "cannot be zero at the same time as minCandidateRunnersPercentage"))
	}
	return allErrs.ToAggregate()
}

// validateMinCandidateRunnersPercentage validates that
// minCandidateRunnersPercentage is within the allowed range.
func validateMinCandidateRunnersPercentage(minCandidateRunnersPercentage int32, p *field.Path) *field.Error {
	if minCandidateRunnersPercentage < 0 || minCandidateRunnersPercentage > 100 {
		return field.Invalid(p, minCandidateRunnersPercentage, "not in valid range [0, 100]")
	}
	return nil
}

// validateMinCandidateRunnersAbsolute validates that minCandidateRunnersAbsolute
// is within the allowed range.
func validateMinCandidateRunnersAbsolute(minCandidateRunnersAbsolute int32, p *field.Path) *field.Error {
	if minCandidateRunnersAbsolute < 0 {
		return field.Invalid(p, minCandidateRunnersAbsolute, "not in valid range [0, inf)")
	}
	return nil
}

// ValidateInterRegionAffinityArgs validates that InterRegionAffinityArgs are correct.
func ValidateInterRegionAffinityArgs(path *field.Path, args *config.InterRegionAffinityArgs) error {
	return validateHardRegionAffinityWeight(path.Child("hardRegionAffinityWeight"), args.HardRegionAffinityWeight)
}

// validateHardRegionAffinityWeight validates that weight is within allowed range.
func validateHardRegionAffinityWeight(path *field.Path, w int32) error {
	const (
		minHardRegionAffinityWeight = 0
		maxHardRegionAffinityWeight = 100
	)

	if w < minHardRegionAffinityWeight || w > maxHardRegionAffinityWeight {
		msg := fmt.Sprintf("not in valid range [%d, %d]", minHardRegionAffinityWeight, maxHardRegionAffinityWeight)
		return field.Invalid(path, w, msg)
	}
	return nil
}

// ValidateRegionTopologySpreadArgs validates that RegionTopologySpreadArgs are correct.
// It replicates the validation from pkg/apis/core/validation.validateTopologySpreadConstraints
// with an additional check for .labelSelector to be nil.
func ValidateRegionTopologySpreadArgs(path *field.Path, args *config.RegionTopologySpreadArgs) error {
	var allErrs field.ErrorList
	if err := validateDefaultingType(path.Child("defaultingType"), args.DefaultingType, args.DefaultConstraints); err != nil {
		allErrs = append(allErrs, err)
	}

	defaultConstraintsPath := path.Child("defaultConstraints")
	for i, c := range args.DefaultConstraints {
		p := defaultConstraintsPath.Index(i)
		if c.MaxSkew <= 0 {
			f := p.Child("maxSkew")
			allErrs = append(allErrs, field.Invalid(f, c.MaxSkew, "not in valid range (0, inf)"))
		}
		allErrs = append(allErrs, validateTopologyKey(p.Child("topologyKey"), c.TopologyKey)...)
		if err := validateWhenUnsatisfiable(p.Child("whenUnsatisfiable"), c.WhenUnsatisfiable); err != nil {
			allErrs = append(allErrs, err)
		}
		if c.LabelSelector != nil {
			f := field.Forbidden(p.Child("labelSelector"), "constraint must not define a selector, as they deduced for each pod")
			allErrs = append(allErrs, f)
		}
		if err := validateConstraintNotRepeat(defaultConstraintsPath, args.DefaultConstraints, i); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs.ToAggregate()
}

func validateDefaultingType(p *field.Path, v config.RegionTopologySpreadConstraintsDefaulting, constraints []corev1.TopologySpreadConstraint) *field.Error {
	if v != config.SystemDefaulting && v != config.ListDefaulting {
		return field.NotSupported(p, v, []string{string(config.SystemDefaulting), string(config.ListDefaulting)})
	}
	if v == config.SystemDefaulting && len(constraints) > 0 {
		return field.Invalid(p, v, "when .defaultConstraints are not empty")
	}
	return nil
}

func validateTopologyKey(p *field.Path, v string) field.ErrorList {
	var allErrs field.ErrorList
	if len(v) == 0 {
		allErrs = append(allErrs, field.Required(p, "can not be empty"))
	} else {
		allErrs = append(allErrs, metav1validation.ValidateLabelName(v, p)...)
	}
	return allErrs
}

func validateWhenUnsatisfiable(p *field.Path, v corev1.UnsatisfiableConstraintAction) *field.Error {
	supportedScheduleActions := sets.New(string(corev1.DoNotSchedule), string(corev1.ScheduleAnyway))

	if len(v) == 0 {
		return field.Required(p, "can not be empty")
	}
	if !supportedScheduleActions.Has(string(v)) {
		return field.NotSupported(p, v, sets.List(supportedScheduleActions))
	}
	return nil
}

func validateConstraintNotRepeat(path *field.Path, constraints []corev1.TopologySpreadConstraint, idx int) *field.Error {
	c := &constraints[idx]
	for i := range constraints[:idx] {
		other := &constraints[i]
		if c.TopologyKey == other.TopologyKey && c.WhenUnsatisfiable == other.WhenUnsatisfiable {
			return field.Duplicate(path.Index(idx), fmt.Sprintf("{%v, %v}", c.TopologyKey, c.WhenUnsatisfiable))
		}
	}
	return nil
}

func validateFunctionShape(shape []config.UtilizationShapePoint, path *field.Path) field.ErrorList {
	const (
		minUtilization = 0
		maxUtilization = 100
		minScore       = 0
		maxScore       = int32(config.MaxCustomPriorityScore)
	)

	var allErrs field.ErrorList

	if len(shape) == 0 {
		allErrs = append(allErrs, field.Required(path, "at least one point must be specified"))
		return allErrs
	}

	for i := 1; i < len(shape); i++ {
		if shape[i-1].Utilization >= shape[i].Utilization {
			allErrs = append(allErrs, field.Invalid(path.Index(i).Child("utilization"), shape[i].Utilization, "utilization values must be sorted in increasing order"))
			break
		}
	}

	for i, point := range shape {
		if point.Utilization < minUtilization || point.Utilization > maxUtilization {
			msg := fmt.Sprintf("not in valid range [%d, %d]", minUtilization, maxUtilization)
			allErrs = append(allErrs, field.Invalid(path.Index(i).Child("utilization"), point.Utilization, msg))
		}

		if point.Score < minScore || point.Score > maxScore {
			msg := fmt.Sprintf("not in valid range [%d, %d]", minScore, maxScore)
			allErrs = append(allErrs, field.Invalid(path.Index(i).Child("score"), point.Score, msg))
		}
	}

	return allErrs
}

func validateResources(resources []config.ResourceSpec, p *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	for i, resource := range resources {
		if resource.Weight <= 0 || resource.Weight > 100 {
			msg := fmt.Sprintf("resource weight of %v not in valid range (0, 100]", resource.Name)
			allErrs = append(allErrs, field.Invalid(p.Index(i).Child("weight"), resource.Weight, msg))
		}
	}
	return allErrs
}

// ValidateRunnerResourcesBalancedAllocationArgs validates that RunnerResourcesBalancedAllocationArgs are set correctly.
func ValidateRunnerResourcesBalancedAllocationArgs(path *field.Path, args *config.RunnerResourcesBalancedAllocationArgs) error {
	var allErrs field.ErrorList
	seenResources := sets.New[string]()
	for i, resource := range args.Resources {
		if seenResources.Has(resource.Name) {
			allErrs = append(allErrs, field.Duplicate(path.Child("resources").Index(i).Child("name"), resource.Name))
		} else {
			seenResources.Insert(resource.Name)
		}
		if resource.Weight != 1 {
			allErrs = append(allErrs, field.Invalid(path.Child("resources").Index(i).Child("weight"), resource.Weight, "must be 1"))
		}
	}
	return allErrs.ToAggregate()
}

func ValidateRunnerResourcesFitArgs(path *field.Path, args *config.RunnerResourcesFitArgs) error {
	var allErrs field.ErrorList
	resPath := path.Child("ignoredResources")
	for i, res := range args.IgnoredResources {
		path := resPath.Index(i)
		if errs := metav1validation.ValidateLabelName(res, path); len(errs) != 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	groupPath := path.Child("ignoredResourceGroups")
	for i, group := range args.IgnoredResourceGroups {
		path := groupPath.Index(i)
		if strings.Contains(group, "/") {
			allErrs = append(allErrs, field.Invalid(path, group, "resource group name can't contain '/'"))
		}
		if errs := metav1validation.ValidateLabelName(group, path); len(errs) != 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	strategyPath := path.Child("scoringStrategy")
	if args.ScoringStrategy != nil {
		if !supportedScoringStrategyTypes.Has(string(args.ScoringStrategy.Type)) {
			allErrs = append(allErrs, field.NotSupported(strategyPath.Child("type"), args.ScoringStrategy.Type, sets.List(supportedScoringStrategyTypes)))
		}
		allErrs = append(allErrs, validateResources(args.ScoringStrategy.Resources, strategyPath.Child("resources"))...)
		if args.ScoringStrategy.RequestedToCapacityRatio != nil {
			allErrs = append(allErrs, validateFunctionShape(args.ScoringStrategy.RequestedToCapacityRatio.Shape, strategyPath.Child("shape"))...)
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return allErrs.ToAggregate()
}
