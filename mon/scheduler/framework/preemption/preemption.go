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

package preemption

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	corelisters "github.com/olive-io/olive/client/generated/listers/core/v1"
	extendercorev1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/parallelize"
	"github.com/olive-io/olive/mon/scheduler/metrics"
	"github.com/olive-io/olive/mon/scheduler/util"
)

// Candidate represents a nominated runner on which the preemptor can be scheduled,
// along with the list of victims that should be evicted for the preemptor to fit the runner.
type Candidate interface {
	// Victims wraps a list of to-be-preempted Definitions and the number of PDB violation.
	Victims() *extendercorev1.Victims
	// Name returns the target runner name where the preemptor gets nominated to run.
	Name() string
}

type candidate struct {
	victims *extendercorev1.Victims
	name    string
}

// Victims returns s.victims.
func (s *candidate) Victims() *extendercorev1.Victims {
	return s.victims
}

// Name returns s.name.
func (s *candidate) Name() string {
	return s.name
}

type candidateList struct {
	idx   int32
	items []Candidate
}

func newCandidateList(size int32) *candidateList {
	return &candidateList{idx: -1, items: make([]Candidate, size)}
}

// add adds a new candidate to the internal array atomically.
func (cl *candidateList) add(c *candidate) {
	if idx := atomic.AddInt32(&cl.idx, 1); idx < int32(len(cl.items)) {
		cl.items[idx] = c
	}
}

// size returns the number of candidate stored. Note that some add() operations
// might still be executing when this is called, so care must be taken to
// ensure that all add() operations complete before accessing the elements of
// the list.
func (cl *candidateList) size() int32 {
	n := atomic.LoadInt32(&cl.idx) + 1
	if n >= int32(len(cl.items)) {
		n = int32(len(cl.items))
	}
	return n
}

// get returns the internal candidate array. This function is NOT atomic and
// assumes that all add() operations have been completed.
func (cl *candidateList) get() []Candidate {
	return cl.items[:cl.size()]
}

// Interface is expected to be implemented by different preemption plugins as all those member
// methods might have different behavior compared with the default preemption.
type Interface interface {
	// GetOffsetAndNumCandidates chooses a random offset and calculates the number of candidates that should be
	// shortlisted for dry running preemption.
	GetOffsetAndNumCandidates(runners int32) (int32, int32)
	// CandidatesToVictimsMap builds a map from the target runner to a list of to-be-preempted Definitions and the number of PDB violation.
	CandidatesToVictimsMap(candidates []Candidate) map[string]*extendercorev1.Victims
	// DefinitionEligibleToPreemptOthers returns one bool and one string. The bool indicates whether this definition should be considered for
	// preempting other definitions or not. The string includes the reason if this definition isn't eligible.
	DefinitionEligibleToPreemptOthers(definition *corev1.Definition, nominatedRunnerStatus *framework.Status) (bool, string)
	// SelectVictimsOnRunner finds minimum set of definitions on the given runner that should be preempted in order to make enough room
	// for "definition" to be scheduled.
	// Note that both `state` and `runnerInfo` are deep copied.
	SelectVictimsOnRunner(ctx context.Context, state *framework.CycleState,
		definition *corev1.Definition, runnerInfo *framework.RunnerInfo) ([]*corev1.Definition, int, *framework.Status)
	// OrderedScoreFuncs returns a list of ordered score functions to select preferable runner where victims will be preempted.
	// The ordered score functions will be processed one by one iff we find more than one runner with the highest score.
	// Default score functions will be processed if nil returned here for backwards-compatibility.
	OrderedScoreFuncs(ctx context.Context, runnersToVictims map[string]*extendercorev1.Victims) []func(runner string) int64
}

type Evaluator struct {
	PluginName       string
	Handler          framework.Handle
	DefinitionLister corelisters.DefinitionLister
	State            *framework.CycleState
	Interface
}

// Preempt returns a PostFilterResult carrying suggested nominatedRunnerName, along with a Status.
// The semantics of returned <PostFilterResult, Status> varies on different scenarios:
//
//   - <nil, Error>. This denotes it's a transient/rare error that may be self-healed in future cycles.
//
//   - <nil, Unschedulable>. This status is mostly as expected like the preemptor is waiting for the
//     victims to be fully terminated.
//
//   - In both cases above, a nil PostFilterResult is returned to keep the definition's nominatedRunnerName unchanged.
//
//   - <non-nil PostFilterResult, Unschedulable>. It indicates the definition cannot be scheduled even with preemption.
//     In this case, a non-nil PostFilterResult is returned and result.NominatingMode instructs how to deal with
//     the nominatedRunnerName.
//
//   - <non-nil PostFilterResult, Success>. It's the regular happy path
//     and the non-empty nominatedRunnerName will be applied to the preemptor definition.
func (ev *Evaluator) Preempt(ctx context.Context, definition *corev1.Definition, m framework.RunnerToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	logger := klog.FromContext(ctx)

	// 0) Fetch the latest version of <definition>.
	// It's safe to directly fetch definition here. Because the informer cache has already been
	// initialized when creating the Scheduler obj.
	// However, tests may need to manually initialize the shared definition informer.
	definitionNamespace, definitionName := definition.Namespace, definition.Name
	definition, err := ev.DefinitionLister.Definitions(definition.Namespace).Get(definition.Name)
	if err != nil {
		logger.Error(err, "Could not get the updated preemptor definition object", "definition", klog.KRef(definitionNamespace, definitionName))
		return nil, framework.AsStatus(err)
	}

	// 1) Ensure the preemptor is eligible to preempt other definitions.
	//if ok, msg := ev.DefinitionEligibleToPreemptOthers(definition, m[definition.Status.NominatedRunnerName]); !ok {
	//	logger.V(5).Info("Definition is not eligible for preemption", "definition", klog.KObj(definition), "reason", msg)
	//	return nil, framework.NewStatus(framework.Unschedulable, msg)
	//}

	// 2) Find all preemption candidates.
	candidates, runnerToStatusMap, err := ev.findCandidates(ctx, definition, m)
	if err != nil && len(candidates) == 0 {
		return nil, framework.AsStatus(err)
	}

	// Return a FitError only when there are no candidates that fit the definition.
	if len(candidates) == 0 {
		fitError := &framework.FitError{
			Definition:    definition,
			NumAllRunners: len(runnerToStatusMap),
			Diagnosis: framework.Diagnosis{
				RunnerToStatusMap: runnerToStatusMap,
				// Leave UnschedulablePlugins or PendingPlugins as nil as it won't be used on moving Definitions.
			},
		}
		// Specify nominatedRunnerName to clear the definition's nominatedRunnerName status, if applicable.
		return framework.NewPostFilterResultWithNominatedRunner(""), framework.NewStatus(framework.Unschedulable, fitError.Error())
	}

	// 3) Interact with registered Extenders to filter out some candidates if needed.
	candidates, status := ev.callExtenders(logger, definition, candidates)
	if !status.IsSuccess() {
		return nil, status
	}

	// 4) Find the best candidate.
	bestCandidate := ev.SelectCandidate(ctx, candidates)
	if bestCandidate == nil || len(bestCandidate.Name()) == 0 {
		return nil, framework.NewStatus(framework.Unschedulable, "no candidate runner for preemption")
	}

	// 5) Perform preparation work before nominating the selected candidate.
	if status := ev.prepareCandidate(ctx, bestCandidate, definition, ev.PluginName); !status.IsSuccess() {
		return nil, status
	}

	return framework.NewPostFilterResultWithNominatedRunner(bestCandidate.Name()), framework.NewStatus(framework.Success)
}

// FindCandidates calculates a slice of preemption candidates.
// Each candidate is executable to make the given <definition> schedulable.
func (ev *Evaluator) findCandidates(ctx context.Context, definition *corev1.Definition, m framework.RunnerToStatusMap) ([]Candidate, framework.RunnerToStatusMap, error) {
	allRunners, err := ev.Handler.SnapshotSharedLister().RunnerInfos().List()
	if err != nil {
		return nil, nil, err
	}
	if len(allRunners) == 0 {
		return nil, nil, errors.New("no runners available")
	}
	logger := klog.FromContext(ctx)
	potentialRunners, unschedulableRunnerStatus := runnersWherePreemptionMightHelp(allRunners, m)
	if len(potentialRunners) == 0 {
		logger.V(3).Info("Preemption will not help schedule definition on any runner", "definition", klog.KObj(definition))
		// In this case, we should clean-up any existing nominated runner name of the definition.
		//if err := util.ClearNominatedRunnerName(ctx, ev.Handler.ClientSet(), definition); err != nil {
		//	logger.Error(err, "Could not clear the nominatedRunnerName field of definition", "definition", klog.KObj(definition))
		//	// We do not return as this error is not critical.
		//}
		return nil, unschedulableRunnerStatus, nil
	}

	//pdbs, err := getDefinitionDisruptionBudgets(ev.PdbLister)
	//if err != nil {
	//	return nil, nil, err
	//}

	offset, numCandidates := ev.GetOffsetAndNumCandidates(int32(len(potentialRunners)))
	if loggerV := logger.V(5); logger.Enabled() {
		var sample []string
		for i := offset; i < offset+10 && i < int32(len(potentialRunners)); i++ {
			sample = append(sample, potentialRunners[i].Runner().Name)
		}
		loggerV.Info("Selected candidates from a pool of runners", "potentialRunnersCount", len(potentialRunners), "offset", offset, "sampleLength", len(sample), "sample", sample, "candidates", numCandidates)
	}
	candidates, runnerStatuses, err := ev.DryRunPreemption(ctx, definition, potentialRunners, offset, numCandidates)
	for runner, runnerStatus := range unschedulableRunnerStatus {
		runnerStatuses[runner] = runnerStatus
	}
	return candidates, runnerStatuses, err
}

// callExtenders calls given <extenders> to select the list of feasible candidates.
// We will only check <candidates> with extenders that support preemption.
// Extenders which do not support preemption may later prevent preemptor from being scheduled on the nominated
// runner. In that case, scheduler will find a different host for the preemptor in subsequent scheduling cycles.
func (ev *Evaluator) callExtenders(logger klog.Logger, definition *corev1.Definition, candidates []Candidate) ([]Candidate, *framework.Status) {
	extenders := ev.Handler.Extenders()
	runnerLister := ev.Handler.SnapshotSharedLister().RunnerInfos()
	if len(extenders) == 0 {
		return candidates, nil
	}

	// Migrate candidate slice to victimsMap to adapt to the Extender interface.
	// It's only applicable for candidate slice that have unique nominated runner name.
	victimsMap := ev.CandidatesToVictimsMap(candidates)
	if len(victimsMap) == 0 {
		return candidates, nil
	}
	for _, extender := range extenders {
		if !extender.SupportsPreemption() || !extender.IsInterested(definition) {
			continue
		}
		runnerNameToVictims, err := extender.ProcessPreemption(definition, victimsMap, runnerLister)
		if err != nil {
			if extender.IsIgnorable() {
				logger.Info("Skipped extender as it returned error and has ignorable flag set",
					"extender", extender.Name(), "err", err)
				continue
			}
			return nil, framework.AsStatus(err)
		}
		// Check if the returned victims are valid.
		for runnerName, victims := range runnerNameToVictims {
			if victims == nil || len(victims.Definitions) == 0 {
				if extender.IsIgnorable() {
					delete(runnerNameToVictims, runnerName)
					logger.Info("Ignored runner for which the extender didn't report victims", "runner", klog.KRef("", runnerName), "extender", extender.Name())
					continue
				}
				return nil, framework.AsStatus(fmt.Errorf("expected at least one victim definition on runner %q", runnerName))
			}
		}

		// Replace victimsMap with new result after preemption. So the
		// rest of extenders can continue use it as parameter.
		victimsMap = runnerNameToVictims

		// If runner list becomes empty, no preemption can happen regardless of other extenders.
		if len(victimsMap) == 0 {
			break
		}
	}

	var newCandidates []Candidate
	for runnerName := range victimsMap {
		newCandidates = append(newCandidates, &candidate{
			victims: victimsMap[runnerName],
			name:    runnerName,
		})
	}
	return newCandidates, nil
}

// SelectCandidate chooses the best-fit candidate from given <candidates> and return it.
// NOTE: This method is exported for easier testing in default preemption.
func (ev *Evaluator) SelectCandidate(ctx context.Context, candidates []Candidate) Candidate {
	logger := klog.FromContext(ctx)

	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	victimsMap := ev.CandidatesToVictimsMap(candidates)
	scoreFuncs := ev.OrderedScoreFuncs(ctx, victimsMap)
	candidateRunner := pickOneRunnerForPreemption(logger, victimsMap, scoreFuncs)

	// Same as candidatesToVictimsMap, this logic is not applicable for out-of-tree
	// preemption plugins that exercise different candidates on the same nominated runner.
	if victims := victimsMap[candidateRunner]; victims != nil {
		return &candidate{
			victims: victims,
			name:    candidateRunner,
		}
	}

	// We shouldn't reach here.
	logger.Error(errors.New("no candidate selected"), "Should not reach here", "candidates", candidates)
	// To not break the whole flow, return the first candidate.
	return candidates[0]
}

// prepareCandidate does some preparation work before nominating the selected candidate:
// - Evict the victim definitions
// - Reject the victim definitions if they are in waitingDefinition map
// - Clear the low-priority definitions' nominatedRunnerName status if needed
func (ev *Evaluator) prepareCandidate(ctx context.Context, c Candidate, definition *corev1.Definition, pluginName string) *framework.Status {
	fh := ev.Handler
	//cs := ev.Handler.ClientSet()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	logger := klog.FromContext(ctx)
	errCh := parallelize.NewErrorChannel()
	preemptDefinition := func(index int) {
		victim := c.Victims().Definitions[index]
		// If the victim is a WaitingDefinition, send a reject message to the PermitPlugin.
		// Otherwise we should delete the victim.
		if waitingDefinition := fh.GetWaitingDefinition(victim.UID); waitingDefinition != nil {
			waitingDefinition.Reject(pluginName, "preempted")
			logger.V(2).Info("Preemptor definition rejected a waiting definition", "preemptor", klog.KObj(definition), "waitingDefinition", klog.KObj(victim), "runner", c.Name())
		} else {
			//if feature.DefaultFeatureGate.Enabled(features.DefinitionDisruptionConditions) {
			//	condition := &corev1.DefinitionCondition{
			//		Type:    corev1.DisruptionTarget,
			//		Status:  corev1.ConditionTrue,
			//		Reason:  corev1.DefinitionReasonPreemptionByScheduler,
			//		Message: fmt.Sprintf("%s: preempting to accommodate a higher priority definition", definition.Spec.SchedulerName),
			//	}
			//	newStatus := definition.Status.DeepCopy()
			//	updated := apidefinition.UpdateDefinitionCondition(newStatus, condition)
			//	if updated {
			//		if err := util.PatchDefinitionStatus(ctx, cs, victim, newStatus); err != nil {
			//			logger.Error(err, "Could not add DisruptionTarget condition due to preemption", "definition", klog.KObj(victim), "preemptor", klog.KObj(definition))
			//			errCh.SendErrorWithCancel(err, cancel)
			//			return
			//		}
			//	}
			//}
			//if err := util.DeleteDefinition(ctx, cs, victim); err != nil {
			//	logger.Error(err, "Preempted definition", "definition", klog.KObj(victim), "preemptor", klog.KObj(definition))
			//	errCh.SendErrorWithCancel(err, cancel)
			//	return
			//}
			//logger.V(2).Info("Preemptor Definition preempted victim Definition", "preemptor", klog.KObj(definition), "victim", klog.KObj(victim), "runner", c.Name())
		}

		// corev1.EventTypeNormal
		fh.EventRecorder().Eventf(victim, definition, "", "Preempted", "Preempting", "Preempted by definition %v on runner %v", definition.UID, c.Name())
	}

	fh.Parallelizer().Until(ctx, len(c.Victims().Definitions), preemptDefinition, ev.PluginName)
	if err := errCh.ReceiveError(); err != nil {
		return framework.AsStatus(err)
	}

	metrics.PreemptionVictims.Observe(float64(len(c.Victims().Definitions)))

	// Lower priority definitions nominated to run on this runner, may no longer fit on
	// this runner. So, we should remove their nomination. Removing their
	// nomination updates these definitions and moves them to the active queue. It
	// lets scheduler find another place for them.
	//nominatedDefinitions := getLowerPriorityNominatedDefinitions(logger, fh, definition, c.Name())
	//if err := util.ClearNominatedRunnerName(ctx, cs, nominatedDefinitions...); err != nil {
	//	logger.Error(err, "Cannot clear 'NominatedRunnerName' field")
	//	// We do not return as this error is not critical.
	//}

	return nil
}

// runnersWherePreemptionMightHelp returns a list of runners with failed predicates
// that may be satisfied by removing definitions from the runner.
func runnersWherePreemptionMightHelp(runners []*framework.RunnerInfo, m framework.RunnerToStatusMap) ([]*framework.RunnerInfo, framework.RunnerToStatusMap) {
	var potentialRunners []*framework.RunnerInfo
	runnerStatuses := make(framework.RunnerToStatusMap)
	for _, runner := range runners {
		name := runner.Runner().Name
		// We rely on the status by each plugin - 'Unschedulable' or 'UnschedulableAndUnresolvable'
		// to determine whether preemption may help or not on the runner.
		if m[name].Code() == framework.UnschedulableAndUnresolvable {
			runnerStatuses[runner.Runner().Name] = framework.NewStatus(framework.UnschedulableAndUnresolvable, "Preemption is not helpful for scheduling")
			continue
		}
		potentialRunners = append(potentialRunners, runner)
	}
	return potentialRunners, runnerStatuses
}

//func getDefinitionDisruptionBudgets(pdbLister policylisters.DefinitionDisruptionBudgetLister) ([]*policy.DefinitionDisruptionBudget, error) {
//	if pdbLister != nil {
//		return pdbLister.List(labels.Everything())
//	}
//	return nil, nil
//}

// pickOneRunnerForPreemption chooses one runner among the given runners.
// It assumes definitions in each map entry are ordered by decreasing priority.
// If the scoreFuns is not empty, It picks a runner based on score scoreFuns returns.
// If the scoreFuns is empty,
// It picks a runner based on the following criteria:
// 1. A runner with minimum number of PDB violations.
// 2. A runner with minimum highest priority victim is picked.
// 3. Ties are broken by sum of priorities of all victims.
// 4. If there are still ties, runner with the minimum number of victims is picked.
// 5. If there are still ties, runner with the latest start time of all highest priority victims is picked.
// 6. If there are still ties, the first such runner is picked (sort of randomly).
// The 'minRunners1' and 'minRunners2' are being reused here to save the memory
// allocation and garbage collection time.
func pickOneRunnerForPreemption(logger klog.Logger, runnersToVictims map[string]*extendercorev1.Victims, scoreFuncs []func(runner string) int64) string {
	if len(runnersToVictims) == 0 {
		return ""
	}

	allCandidates := make([]string, 0, len(runnersToVictims))
	for runner := range runnersToVictims {
		allCandidates = append(allCandidates, runner)
	}

	if len(scoreFuncs) == 0 {
		minNumPDBViolatingScoreFunc := func(runner string) int64 {
			// The smaller the NumPDBViolations, the higher the score.
			return -runnersToVictims[runner].NumPDBViolations
		}
		minHighestPriorityScoreFunc := func(runner string) int64 {
			// highestDefinitionPriority is the highest priority among the victims on this runner.
			//highestDefinitionPriority := corev1helpers.DefinitionPriority(runnersToVictims[runner].Definitions[0])
			highestDefinitionPriority := 100
			// The smaller the highestDefinitionPriority, the higher the score.
			return -int64(highestDefinitionPriority)
		}
		minSumPrioritiesScoreFunc := func(runner string) int64 {
			var sumPriorities int64
			for _, definition := range runnersToVictims[runner].Definitions {
				// We add MaxInt32+1 to all priorities to make all of them >= 0. This is
				// needed so that a runner with a few definitions with negative priority is not
				// picked over a runner with a smaller number of definitions with the same negative
				// priority (and similar scenarios).
				//sumPriorities += int64(corev1helpers.DefinitionPriority(definition)) + int64(math.MaxInt32+1)
				_ = definition
				sumPriorities += int64(math.MaxInt32 + 1)
			}
			// The smaller the sumPriorities, the higher the score.
			return -sumPriorities
		}
		minNumDefinitionsScoreFunc := func(runner string) int64 {
			// The smaller the length of definitions, the higher the score.
			return -int64(len(runnersToVictims[runner].Definitions))
		}
		latestStartTimeScoreFunc := func(runner string) int64 {
			// Get the earliest start time of all definitions on the current runner.
			earliestStartTimeOnRunner := util.GetEarliestDefinitionStartTime(runnersToVictims[runner])
			if earliestStartTimeOnRunner == nil {
				logger.Error(errors.New("earliestStartTime is nil for runner"), "Should not reach here", "runner", runner)
				return int64(math.MinInt64)
			}
			// The bigger the earliestStartTimeOnRunner, the higher the score.
			return earliestStartTimeOnRunner.UnixNano()
		}

		// Each scoreFunc scores the runners according to specific rules and keeps the name of the runner
		// with the highest score. If and only if the scoreFunc has more than one runner with the highest
		// score, we will execute the other scoreFunc in order of precedence.
		scoreFuncs = []func(string) int64{
			// A runner with a minimum number of PDB is preferable.
			minNumPDBViolatingScoreFunc,
			// A runner with a minimum highest priority victim is preferable.
			minHighestPriorityScoreFunc,
			// A runner with the smallest sum of priorities is preferable.
			minSumPrioritiesScoreFunc,
			// A runner with the minimum number of definitions is preferable.
			minNumDefinitionsScoreFunc,
			// A runner with the latest start time of all highest priority victims is preferable.
			latestStartTimeScoreFunc,
			// If there are still ties, then the first Runner in the list is selected.
		}
	}

	for _, f := range scoreFuncs {
		selectedRunners := []string{}
		maxScore := int64(math.MinInt64)
		for _, runner := range allCandidates {
			score := f(runner)
			if score > maxScore {
				maxScore = score
				selectedRunners = []string{}
			}
			if score == maxScore {
				selectedRunners = append(selectedRunners, runner)
			}
		}
		if len(selectedRunners) == 1 {
			return selectedRunners[0]
		}
		allCandidates = selectedRunners
	}

	return allCandidates[0]
}

// getLowerPriorityNominatedDefinitions returns definitions whose priority is smaller than the
// priority of the given "definition" and are nominated to run on the given runner.
// Note: We could possibly check if the nominated lower priority definitions still fit
// and return those that no longer fit, but that would require lots of
// manipulation of RunnerInfo and PreFilter state per nominated definition. It may not be
// worth the complexity, especially because we generally expect to have a very
// small number of nominated definitions per runner.
func getLowerPriorityNominatedDefinitions(logger klog.Logger, pn framework.DefinitionNominator, definition *corev1.Definition, runnerName string) []*corev1.Definition {
	definitionInfos := pn.NominatedDefinitionsForRunner(runnerName)

	if len(definitionInfos) == 0 {
		return nil
	}

	var lowerPriorityDefinitions []*corev1.Definition
	//definitionPriority := corecorev1helpers.DefinitionPriority(definition)
	for _, pi := range definitionInfos {
		//if corecorev1helpers.DefinitionPriority(pi.Definition) < definitionPriority {
		lowerPriorityDefinitions = append(lowerPriorityDefinitions, pi.Definition)
		//}
	}
	return lowerPriorityDefinitions
}

// DryRunPreemption simulates Preemption logic on <potentialRunners> in parallel,
// returns preemption candidates and a map indicating filtered runners statuses.
// The number of candidates depends on the constraints defined in the plugin's args. In the returned list of
// candidates, ones that do not violate PDB are preferred over ones that do.
// NOTE: This method is exported for easier testing in default preemption.
func (ev *Evaluator) DryRunPreemption(ctx context.Context, definition *corev1.Definition, potentialRunners []*framework.RunnerInfo,
	offset int32, numCandidates int32) ([]Candidate, framework.RunnerToStatusMap, error) {
	fh := ev.Handler
	nonViolatingCandidates := newCandidateList(numCandidates)
	violatingCandidates := newCandidateList(numCandidates)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	runnerStatuses := make(framework.RunnerToStatusMap)
	var statusesLock sync.Mutex
	var errs []error
	checkRunner := func(i int) {
		runnerInfoCopy := potentialRunners[(int(offset)+i)%len(potentialRunners)].Snapshot()
		stateCopy := ev.State.Clone()
		definitions, numPDBViolations, status := ev.SelectVictimsOnRunner(ctx, stateCopy, definition, runnerInfoCopy)
		if status.IsSuccess() && len(definitions) != 0 {
			victims := extendercorev1.Victims{
				Definitions:      definitions,
				NumPDBViolations: int64(numPDBViolations),
			}
			c := &candidate{
				victims: &victims,
				name:    runnerInfoCopy.Runner().Name,
			}
			if numPDBViolations == 0 {
				nonViolatingCandidates.add(c)
			} else {
				violatingCandidates.add(c)
			}
			nvcSize, vcSize := nonViolatingCandidates.size(), violatingCandidates.size()
			if nvcSize > 0 && nvcSize+vcSize >= numCandidates {
				cancel()
			}
			return
		}
		if status.IsSuccess() && len(definitions) == 0 {
			status = framework.AsStatus(fmt.Errorf("expected at least one victim definition on runner %q", runnerInfoCopy.Runner().Name))
		}
		statusesLock.Lock()
		if status.Code() == framework.Error {
			errs = append(errs, status.AsError())
		}
		runnerStatuses[runnerInfoCopy.Runner().Name] = status
		statusesLock.Unlock()
	}
	fh.Parallelizer().Until(ctx, len(potentialRunners), checkRunner, ev.PluginName)
	return append(nonViolatingCandidates.get(), violatingCandidates.get()...), runnerStatuses, utilerrors.NewAggregate(errs)
}
