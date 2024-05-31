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

package runneraffinity

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// RunnerSelector is a runtime representation of v1.RunnerSelector.
type RunnerSelector struct {
	lazy LazyErrorRunnerSelector
}

// LazyErrorRunnerSelector is a runtime representation of v1.RunnerSelector that
// only reports parse errors when no terms match.
type LazyErrorRunnerSelector struct {
	terms []nodeSelectorTerm
}

// NewRunnerSelector returns a RunnerSelector or aggregate parsing errors found.
func NewRunnerSelector(ns *corev1.RunnerSelector, opts ...field.PathOption) (*RunnerSelector, error) {
	lazy := NewLazyErrorRunnerSelector(ns, opts...)
	var errs []error
	for _, term := range lazy.terms {
		if len(term.parseErrs) > 0 {
			errs = append(errs, term.parseErrs...)
		}
	}
	if len(errs) != 0 {
		return nil, errors.Flatten(errors.NewAggregate(errs))
	}
	return &RunnerSelector{lazy: *lazy}, nil
}

// NewLazyErrorRunnerSelector creates a RunnerSelector that only reports parse
// errors when no terms match.
func NewLazyErrorRunnerSelector(ns *corev1.RunnerSelector, opts ...field.PathOption) *LazyErrorRunnerSelector {
	p := field.ToPath(opts...)
	parsedTerms := make([]nodeSelectorTerm, 0, len(ns.RunnerSelectorTerms))
	path := p.Child("nodeSelectorTerms")
	for i, term := range ns.RunnerSelectorTerms {
		// nil or empty term selects no objects
		if isEmptyRunnerSelectorTerm(&term) {
			continue
		}
		p := path.Index(i)
		parsedTerms = append(parsedTerms, newRunnerSelectorTerm(&term, p))
	}
	return &LazyErrorRunnerSelector{
		terms: parsedTerms,
	}
}

// Match checks whether the node labels and fields match the selector terms, ORed;
// nil or empty term matches no objects.
func (ns *RunnerSelector) Match(node *corev1.Runner) bool {
	// parse errors are reported in NewRunnerSelector.
	match, _ := ns.lazy.Match(node)
	return match
}

// Match checks whether the node labels and fields match the selector terms, ORed;
// nil or empty term matches no objects.
// Parse errors are only returned if no terms matched.
func (ns *LazyErrorRunnerSelector) Match(node *corev1.Runner) (bool, error) {
	if node == nil {
		return false, nil
	}
	nodeLabels := labels.Set(node.Labels)
	nodeFields := extractRunnerFields(node)

	var errs []error
	for _, term := range ns.terms {
		match, tErrs := term.match(nodeLabels, nodeFields)
		if len(tErrs) > 0 {
			errs = append(errs, tErrs...)
			continue
		}
		if match {
			return true, nil
		}
	}
	return false, errors.Flatten(errors.NewAggregate(errs))
}

// PreferredSchedulingTerms is a runtime representation of []v1.PreferredSchedulingTerms.
type PreferredSchedulingTerms struct {
	terms []preferredSchedulingTerm
}

// NewPreferredSchedulingTerms returns a PreferredSchedulingTerms or all the parsing errors found.
// If a v1.PreferredSchedulingTerm has a 0 weight, its parsing is skipped.
func NewPreferredSchedulingTerms(terms []corev1.PreferredSchedulingTerm, opts ...field.PathOption) (*PreferredSchedulingTerms, error) {
	p := field.ToPath(opts...)
	var errs []error
	parsedTerms := make([]preferredSchedulingTerm, 0, len(terms))
	for i, term := range terms {
		path := p.Index(i)
		if term.Weight == 0 || isEmptyRunnerSelectorTerm(&term.Preference) {
			continue
		}
		parsedTerm := preferredSchedulingTerm{
			nodeSelectorTerm: newRunnerSelectorTerm(&term.Preference, path),
			weight:           int(term.Weight),
		}
		if len(parsedTerm.parseErrs) > 0 {
			errs = append(errs, parsedTerm.parseErrs...)
		} else {
			parsedTerms = append(parsedTerms, parsedTerm)
		}
	}
	if len(errs) != 0 {
		return nil, errors.Flatten(errors.NewAggregate(errs))
	}
	return &PreferredSchedulingTerms{terms: parsedTerms}, nil
}

// Score returns a score for a Runner: the sum of the weights of the terms that
// match the Runner.
func (t *PreferredSchedulingTerms) Score(node *corev1.Runner) int64 {
	var score int64
	nodeLabels := labels.Set(node.Labels)
	nodeFields := extractRunnerFields(node)
	for _, term := range t.terms {
		// parse errors are reported in NewPreferredSchedulingTerms.
		if ok, _ := term.match(nodeLabels, nodeFields); ok {
			score += int64(term.weight)
		}
	}
	return score
}

func isEmptyRunnerSelectorTerm(term *corev1.RunnerSelectorTerm) bool {
	return len(term.MatchExpressions) == 0 && len(term.MatchFields) == 0
}

func extractRunnerFields(n *corev1.Runner) fields.Set {
	f := make(fields.Set)
	if len(n.Name) > 0 {
		f["metadata.name"] = n.Name
	}
	return f
}

type nodeSelectorTerm struct {
	matchLabels labels.Selector
	matchFields fields.Selector
	parseErrs   []error
}

func newRunnerSelectorTerm(term *corev1.RunnerSelectorTerm, path *field.Path) nodeSelectorTerm {
	var parsedTerm nodeSelectorTerm
	var errs []error
	if len(term.MatchExpressions) != 0 {
		p := path.Child("matchExpressions")
		parsedTerm.matchLabels, errs = nodeSelectorRequirementsAsSelector(term.MatchExpressions, p)
		if errs != nil {
			parsedTerm.parseErrs = append(parsedTerm.parseErrs, errs...)
		}
	}
	if len(term.MatchFields) != 0 {
		p := path.Child("matchFields")
		parsedTerm.matchFields, errs = nodeSelectorRequirementsAsFieldSelector(term.MatchFields, p)
		if errs != nil {
			parsedTerm.parseErrs = append(parsedTerm.parseErrs, errs...)
		}
	}
	return parsedTerm
}

func (t *nodeSelectorTerm) match(nodeLabels labels.Set, nodeFields fields.Set) (bool, []error) {
	if t.parseErrs != nil {
		return false, t.parseErrs
	}
	if t.matchLabels != nil && !t.matchLabels.Matches(nodeLabels) {
		return false, nil
	}
	if t.matchFields != nil && len(nodeFields) > 0 && !t.matchFields.Matches(nodeFields) {
		return false, nil
	}
	return true, nil
}

var validSelectorOperators = []string{
	string(corev1.RunnerSelectorOpIn),
	string(corev1.RunnerSelectorOpNotIn),
	string(corev1.RunnerSelectorOpExists),
	string(corev1.RunnerSelectorOpDoesNotExist),
	string(corev1.RunnerSelectorOpGt),
	string(corev1.RunnerSelectorOpLt),
}

// nodeSelectorRequirementsAsSelector converts the []RunnerSelectorRequirement api type into a struct that implements
// labels.Selector.
func nodeSelectorRequirementsAsSelector(nsm []corev1.RunnerSelectorRequirement, path *field.Path) (labels.Selector, []error) {
	if len(nsm) == 0 {
		return labels.Nothing(), nil
	}
	var errs []error
	selector := labels.NewSelector()
	for i, expr := range nsm {
		p := path.Index(i)
		var op selection.Operator
		switch expr.Operator {
		case corev1.RunnerSelectorOpIn:
			op = selection.In
		case corev1.RunnerSelectorOpNotIn:
			op = selection.NotIn
		case corev1.RunnerSelectorOpExists:
			op = selection.Exists
		case corev1.RunnerSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		case corev1.RunnerSelectorOpGt:
			op = selection.GreaterThan
		case corev1.RunnerSelectorOpLt:
			op = selection.LessThan
		default:
			errs = append(errs, field.NotSupported(p.Child("operator"), expr.Operator, validSelectorOperators))
			continue
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values, field.WithPath(p))
		if err != nil {
			errs = append(errs, err)
		} else {
			selector = selector.Add(*r)
		}
	}
	if len(errs) != 0 {
		return nil, errs
	}
	return selector, nil
}

var validFieldSelectorOperators = []string{
	string(corev1.RunnerSelectorOpIn),
	string(corev1.RunnerSelectorOpNotIn),
}

// nodeSelectorRequirementsAsFieldSelector converts the []RunnerSelectorRequirement core type into a struct that implements
// fields.Selector.
func nodeSelectorRequirementsAsFieldSelector(nsr []corev1.RunnerSelectorRequirement, path *field.Path) (fields.Selector, []error) {
	if len(nsr) == 0 {
		return fields.Nothing(), nil
	}
	var errs []error

	var selectors []fields.Selector
	for i, expr := range nsr {
		p := path.Index(i)
		switch expr.Operator {
		case corev1.RunnerSelectorOpIn:
			if len(expr.Values) != 1 {
				errs = append(errs, field.Invalid(p.Child("values"), expr.Values, "must have one element"))
			} else {
				selectors = append(selectors, fields.OneTermEqualSelector(expr.Key, expr.Values[0]))
			}

		case corev1.RunnerSelectorOpNotIn:
			if len(expr.Values) != 1 {
				errs = append(errs, field.Invalid(p.Child("values"), expr.Values, "must have one element"))
			} else {
				selectors = append(selectors, fields.OneTermNotEqualSelector(expr.Key, expr.Values[0]))
			}

		default:
			errs = append(errs, field.NotSupported(p.Child("operator"), expr.Operator, validFieldSelectorOperators))
		}
	}

	if len(errs) != 0 {
		return nil, errs
	}
	return fields.AndSelectors(selectors...), nil
}

type preferredSchedulingTerm struct {
	nodeSelectorTerm
	weight int
}

type RequiredRunnerAffinity struct {
	labelSelector labels.Selector
	nodeSelector  *LazyErrorRunnerSelector
}

// GetRequiredRunnerAffinity returns the parsing result of region's nodeSelector and nodeAffinity.
func GetRequiredRunnerAffinity(region *corev1.Region) RequiredRunnerAffinity {
	var selector labels.Selector
	if len(region.Spec.RunnerSelector) > 0 {
		selector = labels.SelectorFromSet(region.Spec.RunnerSelector)
	}
	// Use LazyErrorRunnerSelector for backwards compatibility of parsing errors.
	var affinity *LazyErrorRunnerSelector
	if region.Spec.Affinity != nil &&
		region.Spec.Affinity.RunnerAffinity != nil &&
		region.Spec.Affinity.RunnerAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		affinity = NewLazyErrorRunnerSelector(region.Spec.Affinity.RunnerAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	}
	return RequiredRunnerAffinity{labelSelector: selector, nodeSelector: affinity}
}

// Match checks whether the region is schedulable onto nodes according to
// the requirements in both nodeSelector and nodeAffinity.
func (s RequiredRunnerAffinity) Match(node *corev1.Runner) (bool, error) {
	if s.labelSelector != nil {
		if !s.labelSelector.Matches(labels.Set(node.Labels)) {
			return false, nil
		}
	}
	if s.nodeSelector != nil {
		return s.nodeSelector.Match(node)
	}
	return true, nil
}
