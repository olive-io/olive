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

package framework

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	frameworkruntime "github.com/olive-io/olive/mon/scheduler/framework/runtime"
	"github.com/olive-io/olive/mon/scheduler/util"
	corev1helpers "github.com/olive-io/olive/pkg/scheduling/corev1"
)

// FitPredicate is a function type which is used in fake extender.
type FitPredicate func(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status

// PriorityFunc is a function type which is used in fake extender.
type PriorityFunc func(region *corev1.Region, runners []*framework.RunnerInfo) (*framework.RunnerScoreList, error)

// PriorityConfig is used in fake extender to perform Prioritize function.
type PriorityConfig struct {
	Function PriorityFunc
	Weight   int64
}

// ErrorPredicateExtender implements FitPredicate function to always return error status.
func ErrorPredicateExtender(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
	return framework.NewStatus(framework.Error, "some error")
}

// FalsePredicateExtender implements FitPredicate function to always return unschedulable status.
func FalsePredicateExtender(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
	return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("region is unschedulable on the runner %q", runner.Runner().Name))
}

// TruePredicateExtender implements FitPredicate function to always return success status.
func TruePredicateExtender(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
	return framework.NewStatus(framework.Success)
}

// Runner1PredicateExtender implements FitPredicate function to return true
// when the given runner's name is "runner1"; otherwise return false.
func Runner1PredicateExtender(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
	if runner.Runner().Name == "runner1" {
		return framework.NewStatus(framework.Success)
	}
	return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
}

// Runner2PredicateExtender implements FitPredicate function to return true
// when the given runner's name is "runner2"; otherwise return false.
func Runner2PredicateExtender(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
	if runner.Runner().Name == "runner2" {
		return framework.NewStatus(framework.Success)
	}
	return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
}

// ErrorPrioritizerExtender implements PriorityFunc function to always return error.
func ErrorPrioritizerExtender(region *corev1.Region, runners []*framework.RunnerInfo) (*framework.RunnerScoreList, error) {
	return &framework.RunnerScoreList{}, fmt.Errorf("some error")
}

// Runner1PrioritizerExtender implements PriorityFunc function to give score 10
// if the given runner's name is "runner1"; otherwise score 1.
func Runner1PrioritizerExtender(region *corev1.Region, runners []*framework.RunnerInfo) (*framework.RunnerScoreList, error) {
	result := framework.RunnerScoreList{}
	for _, runner := range runners {
		score := 1
		if runner.Runner().Name == "runner1" {
			score = 10
		}
		result = append(result, framework.RunnerScore{Name: runner.Runner().Name, Score: int64(score)})
	}
	return &result, nil
}

// Runner2PrioritizerExtender implements PriorityFunc function to give score 10
// if the given runner's name is "runner2"; otherwise score 1.
func Runner2PrioritizerExtender(region *corev1.Region, runners []*framework.RunnerInfo) (*framework.RunnerScoreList, error) {
	result := framework.RunnerScoreList{}
	for _, runner := range runners {
		score := 1
		if runner.Runner().Name == "runner2" {
			score = 10
		}
		result = append(result, framework.RunnerScore{Name: runner.Runner().Name, Score: int64(score)})
	}
	return &result, nil
}

type runner2PrioritizerPlugin struct{}

// NewRunner2PrioritizerPlugin returns a factory function to build runner2PrioritizerPlugin.
func NewRunner2PrioritizerPlugin() frameworkruntime.PluginFactory {
	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
		return &runner2PrioritizerPlugin{}, nil
	}
}

// Name returns name of the plugin.
func (pl *runner2PrioritizerPlugin) Name() string {
	return "Runner2Prioritizer"
}

// Score return score 100 if the given runnerName is "runner2"; otherwise return score 10.
func (pl *runner2PrioritizerPlugin) Score(_ context.Context, _ *framework.CycleState, _ *corev1.Region, runnerName string) (int64, *framework.Status) {
	score := 10
	if runnerName == "runner2" {
		score = 100
	}
	return int64(score), nil
}

// ScoreExtensions returns nil.
func (pl *runner2PrioritizerPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// FakeExtender is a data struct which implements the Extender interface.
type FakeExtender struct {
	// ExtenderName indicates this fake extender's name.
	// Note that extender name should be unique.
	ExtenderName       string
	Predicates         []FitPredicate
	Prioritizers       []PriorityConfig
	Weight             int64
	RunnerCacheCapable bool
	FilteredRunners    []*framework.RunnerInfo
	UnInterested       bool
	Ignorable          bool
	Binder             func() error

	// Cached runner information for fake extender
	CachedRunnerNameToInfo map[string]*framework.RunnerInfo
}

const defaultFakeExtenderName = "defaultFakeExtender"

// Name returns name of the extender.
func (f *FakeExtender) Name() string {
	if f.ExtenderName == "" {
		// If ExtenderName is unset, use default name.
		return defaultFakeExtenderName
	}
	return f.ExtenderName
}

// IsIgnorable returns a bool value indicating whether internal errors can be ignored.
func (f *FakeExtender) IsIgnorable() bool {
	return f.Ignorable
}

// SupportsPreemption returns true indicating the extender supports preemption.
func (f *FakeExtender) SupportsPreemption() bool {
	// Assume preempt verb is always defined.
	return true
}

// ProcessPreemption implements the extender preempt function.
func (f *FakeExtender) ProcessPreemption(
	region *corev1.Region,
	runnerNameToVictims map[string]*extenderv1.Victims,
	runnerInfos framework.RunnerInfoLister,
) (map[string]*extenderv1.Victims, error) {
	runnerNameToVictimsCopy := map[string]*extenderv1.Victims{}
	// We don't want to change the original runnerNameToVictims
	for k, v := range runnerNameToVictims {
		// In real world implementation, extender's user should have their own way to get runner object
		// by name if needed (e.g. query kube-apiserver etc).
		//
		// For test purpose, we just use runner from parameters directly.
		runnerNameToVictimsCopy[k] = v
	}

	// If Extender.ProcessPreemption ever gets extended with a context parameter, then the logger should be retrieved from that.
	// Now, in order not to modify the Extender interface, we get the logger from klog.TODO()
	logger := klog.TODO()
	for runnerName, victims := range runnerNameToVictimsCopy {
		// Try to do preemption on extender side.
		runnerInfo, _ := runnerInfos.Get(runnerName)
		extenderVictimRegions, extenderPDBViolations, fits, err := f.selectVictimsOnRunnerByExtender(logger, region, runnerInfo)
		if err != nil {
			return nil, err
		}
		// If it's unfit after extender's preemption, this runner is unresolvable by preemption overall,
		// let's remove it from potential preemption runners.
		if !fits {
			delete(runnerNameToVictimsCopy, runnerName)
		} else {
			// Append new victims to original victims
			runnerNameToVictimsCopy[runnerName].Regions = append(victims.Regions, extenderVictimRegions...)
			runnerNameToVictimsCopy[runnerName].NumPDBViolations = victims.NumPDBViolations + int64(extenderPDBViolations)
		}
	}
	return runnerNameToVictimsCopy, nil
}

// selectVictimsOnRunnerByExtender checks the given runners->regions map with predicates on extender's side.
// Returns:
// 1. More victim regions (if any) amended by preemption phase of extender.
// 2. Number of violating victim (used to calculate PDB).
// 3. Fits or not after preemption phase on extender's side.
func (f *FakeExtender) selectVictimsOnRunnerByExtender(logger klog.Logger, region *corev1.Region, runner *framework.RunnerInfo) ([]*corev1.Region, int, bool, error) {
	// If a extender support preemption but have no cached runner info, let's run filter to make sure
	// default scheduler's decision still stand with given region and runner.
	if !f.RunnerCacheCapable {
		err := f.runPredicate(region, runner)
		if err.IsSuccess() {
			return []*corev1.Region{}, 0, true, nil
		} else if err.IsRejected() {
			return nil, 0, false, nil
		} else {
			return nil, 0, false, err.AsError()
		}
	}

	// Otherwise, as a extender support preemption and have cached runner info, we will assume cachedRunnerNameToInfo is available
	// and get cached runner info by given runner name.
	runnerInfoCopy := f.CachedRunnerNameToInfo[runner.Runner().Name].Snapshot()

	var potentialVictims []*corev1.Region

	removeRegion := func(rp *corev1.Region) error {
		return runnerInfoCopy.RemoveRegion(logger, rp)
	}
	addRegion := func(ap *corev1.Region) {
		runnerInfoCopy.AddRegion(ap)
	}
	// As the first step, remove all the lower priority regions from the runner and
	// check if the given region can be scheduled.
	regionPriority := corev1helpers.RegionPriority(region)
	for _, p := range runnerInfoCopy.Regions {
		if corev1helpers.RegionPriority(p.Region) < regionPriority {
			potentialVictims = append(potentialVictims, p.Region)
			if err := removeRegion(p.Region); err != nil {
				return nil, 0, false, err
			}
		}
	}
	sort.Slice(potentialVictims, func(i, j int) bool { return util.MoreImportantRegion(potentialVictims[i], potentialVictims[j]) })

	// If the new region does not fit after removing all the lower priority regions,
	// we are almost done and this runner is not suitable for preemption.
	status := f.runPredicate(region, runnerInfoCopy)
	if status.IsSuccess() {
		// pass
	} else if status.IsRejected() {
		// does not fit
		return nil, 0, false, nil
	} else {
		// internal errors
		return nil, 0, false, status.AsError()
	}

	var victims []*corev1.Region

	// TODO(harry): handle PDBs in the future.
	numViolatingVictim := 0

	reprieveRegion := func(p *corev1.Region) bool {
		addRegion(p)
		status := f.runPredicate(region, runnerInfoCopy)
		if !status.IsSuccess() {
			if err := removeRegion(p); err != nil {
				return false
			}
			victims = append(victims, p)
		}
		return status.IsSuccess()
	}

	// For now, assume all potential victims to be non-violating.
	// Now we try to reprieve non-violating victims.
	for _, p := range potentialVictims {
		reprieveRegion(p)
	}

	return victims, numViolatingVictim, true, nil
}

// runPredicate run predicates of extender one by one for given region and runner.
// Returns: fits or not.
func (f *FakeExtender) runPredicate(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
	for _, predicate := range f.Predicates {
		status := predicate(region, runner)
		if !status.IsSuccess() {
			return status
		}
	}
	return framework.NewStatus(framework.Success)
}

// Filter implements the extender Filter function.
func (f *FakeExtender) Filter(region *corev1.Region, runners []*framework.RunnerInfo) ([]*framework.RunnerInfo, extenderv1.FailedRunnersMap, extenderv1.FailedRunnersMap, error) {
	var filtered []*framework.RunnerInfo
	failedRunnersMap := extenderv1.FailedRunnersMap{}
	failedAndUnresolvableMap := extenderv1.FailedRunnersMap{}
	for _, runner := range runners {
		status := f.runPredicate(region, runner)
		if status.IsSuccess() {
			filtered = append(filtered, runner)
		} else if status.Code() == framework.Unschedulable {
			failedRunnersMap[runner.Runner().Name] = fmt.Sprintf("FakeExtender: runner %q failed", runner.Runner().Name)
		} else if status.Code() == framework.UnschedulableAndUnresolvable {
			failedAndUnresolvableMap[runner.Runner().Name] = fmt.Sprintf("FakeExtender: runner %q failed and unresolvable", runner.Runner().Name)
		} else {
			return nil, nil, nil, status.AsError()
		}
	}

	f.FilteredRunners = filtered
	if f.RunnerCacheCapable {
		return filtered, failedRunnersMap, failedAndUnresolvableMap, nil
	}
	return filtered, failedRunnersMap, failedAndUnresolvableMap, nil
}

// Prioritize implements the extender Prioritize function.
func (f *FakeExtender) Prioritize(region *corev1.Region, runners []*framework.RunnerInfo) (*extenderv1.HostPriorityList, int64, error) {
	result := extenderv1.HostPriorityList{}
	combinedScores := map[string]int64{}
	for _, prioritizer := range f.Prioritizers {
		weight := prioritizer.Weight
		if weight == 0 {
			continue
		}
		priorityFunc := prioritizer.Function
		prioritizedList, err := priorityFunc(region, runners)
		if err != nil {
			return &extenderv1.HostPriorityList{}, 0, err
		}
		for _, hostEntry := range *prioritizedList {
			combinedScores[hostEntry.Name] += hostEntry.Score * weight
		}
	}
	for host, score := range combinedScores {
		result = append(result, extenderv1.HostPriority{Host: host, Score: score})
	}
	return &result, f.Weight, nil
}

// Bind implements the extender Bind function.
func (f *FakeExtender) Bind(binding *v1.Binding) error {
	if f.Binder != nil {
		return f.Binder()
	}
	if len(f.FilteredRunners) != 0 {
		for _, runner := range f.FilteredRunners {
			if runner.Runner().Name == binding.Target.Name {
				f.FilteredRunners = nil
				return nil
			}
		}
		err := fmt.Errorf("Runner %v not in filtered runners %v", binding.Target.Name, f.FilteredRunners)
		f.FilteredRunners = nil
		return err
	}
	return nil
}

// IsBinder returns true indicating the extender implements the Binder function.
func (f *FakeExtender) IsBinder() bool {
	return true
}

// IsPrioritizer returns true if there are any prioritizers.
func (f *FakeExtender) IsPrioritizer() bool {
	return len(f.Prioritizers) > 0
}

// IsFilter returns true if there are any filters.
func (f *FakeExtender) IsFilter() bool {
	return len(f.Predicates) > 0
}

// IsInterested returns a bool indicating whether this extender is interested in this Region.
func (f *FakeExtender) IsInterested(region *corev1.Region) bool {
	return !f.UnInterested
}

var _ framework.Extender = &FakeExtender{}
