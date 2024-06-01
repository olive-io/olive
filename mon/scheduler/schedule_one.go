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

package scheduler

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	utiltrace "k8s.io/utils/trace"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/apis/core/validation"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/parallelize"
	internalqueue "github.com/olive-io/olive/mon/scheduler/internal/queue"
	"github.com/olive-io/olive/mon/scheduler/metrics"
	"github.com/olive-io/olive/mon/scheduler/util"
	regionutil "github.com/olive-io/olive/pkg/api/v1/region"
)

const (
	// Percentage of plugin metrics to be sampled.
	pluginMetricsSamplePercent = 10
	// minFeasibleRunnersToFind is the minimum number of runners that would be scored
	// in each scheduling cycle. This is a semi-arbitrary value to ensure that a
	// certain minimum of runners are checked for feasibility. This in turn helps
	// ensure a minimum level of spreading.
	minFeasibleRunnersToFind = 100
	// minFeasibleRunnersPercentageToFind is the minimum percentage of runners that
	// would be scored in each scheduling cycle. This is a semi-arbitrary value
	// to ensure that a certain minimum of runners are checked for feasibility.
	// This in turn helps ensure a minimum level of spreading.
	minFeasibleRunnersPercentageToFind = 5
	// numberOfHighestScoredRunnersToReport is the number of runner scores
	// to be included in ScheduleResult.
	numberOfHighestScoredRunnersToReport = 3
)

// ScheduleOne does the entire scheduling workflow for a single region. It is serialized on the scheduling algorithm's host fitting.
func (sched *Scheduler) ScheduleOne(ctx context.Context) {
	logger := klog.FromContext(ctx)
	regionInfo, err := sched.NextRegion(logger)
	if err != nil {
		logger.Error(err, "Error while retrieving next region from scheduling queue")
		return
	}
	// region could be nil when schedulerQueue is closed
	if regionInfo == nil || regionInfo.Region == nil {
		return
	}

	region := regionInfo.Region
	logger = klog.LoggerWithValues(logger, "region", klog.KObj(region))
	ctx = klog.NewContext(ctx, logger)
	logger.V(4).Info("About to try and schedule region", "region", klog.KObj(region))

	fwk, err := sched.frameworkForRegion(region)
	if err != nil {
		// This shouldn't happen, because we only accept for scheduling the regions
		// which specify a scheduler name that matches one of the profiles.
		logger.Error(err, "Error occurred")
		return
	}
	if sched.skipRegionSchedule(ctx, fwk, region) {
		return
	}

	logger.V(3).Info("Attempting to schedule region", "region", klog.KObj(region))

	// Synchronously attempt to find a fit for the region.
	start := time.Now()
	state := framework.NewCycleState()
	state.SetRecordPluginMetrics(rand.Intn(100) < pluginMetricsSamplePercent)

	// Initialize an empty regionsToActivate struct, which will be filled up by plugins or stay empty.
	regionsToActivate := framework.NewRegionsToActivate()
	state.Write(framework.RegionsToActivateKey, regionsToActivate)

	schedulingCycleCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	scheduleResult, assumedRegionInfo, status := sched.schedulingCycle(schedulingCycleCtx, state, fwk, regionInfo, start, regionsToActivate)
	if !status.IsSuccess() {
		sched.FailureHandler(schedulingCycleCtx, fwk, assumedRegionInfo, status, scheduleResult.nominatingInfo, start)
		return
	}

	// bind the region to its host asynchronously (we can do this b/c of the assumption step above).
	go func() {
		bindingCycleCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		metrics.Goroutines.WithLabelValues(metrics.Binding).Inc()
		defer metrics.Goroutines.WithLabelValues(metrics.Binding).Dec()

		status := sched.bindingCycle(bindingCycleCtx, state, fwk, scheduleResult, assumedRegionInfo, start, regionsToActivate)
		if !status.IsSuccess() {
			sched.handleBindingCycleError(bindingCycleCtx, state, fwk, assumedRegionInfo, start, scheduleResult, status)
			return
		}
		// Usually, DoneRegion is called inside the scheduling queue,
		// but in this case, we need to call it here because this Region won't go back to the scheduling queue.
		sched.SchedulingQueue.Done(assumedRegionInfo.Region.UID)
	}()
}

var clearNominatedRunner = &framework.NominatingInfo{NominatingMode: framework.ModeOverride, NominatedRunnerName: ""}

// schedulingCycle tries to schedule a single Region.
func (sched *Scheduler) schedulingCycle(
	ctx context.Context,
	state *framework.CycleState,
	fwk framework.Framework,
	regionInfo *framework.QueuedRegionInfo,
	start time.Time,
	regionsToActivate *framework.RegionsToActivate,
) (ScheduleResult, *framework.QueuedRegionInfo, *framework.Status) {
	logger := klog.FromContext(ctx)
	region := regionInfo.Region
	scheduleResult, err := sched.ScheduleRegion(ctx, fwk, state, region)
	if err != nil {
		defer func() {
			metrics.SchedulingAlgorithmLatency.Observe(metrics.SinceInSeconds(start))
		}()
		if err == ErrNoRunnersAvailable {
			status := framework.NewStatus(framework.UnschedulableAndUnresolvable).WithError(err)
			return ScheduleResult{nominatingInfo: clearNominatedRunner}, regionInfo, status
		}

		fitError, ok := err.(*framework.FitError)
		if !ok {
			logger.Error(err, "Error selecting runner for region", "region", klog.KObj(region))
			return ScheduleResult{nominatingInfo: clearNominatedRunner}, regionInfo, framework.AsStatus(err)
		}

		// ScheduleRegion() may have failed because the region would not fit on any host, so we try to
		// preempt, with the expectation that the next time the region is tried for scheduling it
		// will fit due to the preemption. It is also possible that a different region will schedule
		// into the resources that were preempted, but this is harmless.

		if !fwk.HasPostFilterPlugins() {
			logger.V(3).Info("No PostFilter plugins are registered, so no preemption will be performed")
			return ScheduleResult{}, regionInfo, framework.NewStatus(framework.Unschedulable).WithError(err)
		}

		// Run PostFilter plugins to attempt to make the region schedulable in a future scheduling cycle.
		result, status := fwk.RunPostFilterPlugins(ctx, state, region, fitError.Diagnosis.RunnerToStatusMap)
		msg := status.Message()
		fitError.Diagnosis.PostFilterMsg = msg
		if status.Code() == framework.Error {
			logger.Error(nil, "Status after running PostFilter plugins for region", "region", klog.KObj(region), "status", msg)
		} else {
			logger.V(5).Info("Status after running PostFilter plugins for region", "region", klog.KObj(region), "status", msg)
		}

		var nominatingInfo *framework.NominatingInfo
		if result != nil {
			nominatingInfo = result.NominatingInfo
		}
		return ScheduleResult{nominatingInfo: nominatingInfo}, regionInfo, framework.NewStatus(framework.Unschedulable).WithError(err)
	}

	metrics.SchedulingAlgorithmLatency.Observe(metrics.SinceInSeconds(start))
	// Tell the cache to assume that a region now is running on a given runner, even though it hasn't been bound yet.
	// This allows us to keep scheduling without waiting on binding to occur.
	assumedRegionInfo := regionInfo.DeepCopy()
	assumedRegion := assumedRegionInfo.Region
	// assume modifies `assumedRegion` by setting RunnerName=scheduleResult.SuggestedHost
	err = sched.assume(logger, assumedRegion, scheduleResult.SuggestedHost)
	if err != nil {
		// This is most probably result of a BUG in retrying logic.
		// We report an error here so that region scheduling can be retried.
		// This relies on the fact that Error will check if the region has been bound
		// to a runner and if so will not add it back to the unscheduled regions queue
		// (otherwise this would cause an infinite loop).
		return ScheduleResult{nominatingInfo: clearNominatedRunner}, assumedRegionInfo, framework.AsStatus(err)
	}

	// Run the Reserve method of reserve plugins.
	if sts := fwk.RunReservePluginsReserve(ctx, state, assumedRegion, scheduleResult.SuggestedHost); !sts.IsSuccess() {
		// trigger un-reserve to clean up state associated with the reserved Region
		fwk.RunReservePluginsUnreserve(ctx, state, assumedRegion, scheduleResult.SuggestedHost)
		if forgetErr := sched.Cache.ForgetRegion(logger, assumedRegion); forgetErr != nil {
			logger.Error(forgetErr, "Scheduler cache ForgetRegion failed")
		}

		if sts.IsRejected() {
			fitErr := &framework.FitError{
				NumAllRunners: 1,
				Region:        region,
				Diagnosis: framework.Diagnosis{
					RunnerToStatusMap: framework.RunnerToStatusMap{scheduleResult.SuggestedHost: sts},
				},
			}
			fitErr.Diagnosis.AddPluginStatus(sts)
			return ScheduleResult{nominatingInfo: clearNominatedRunner}, assumedRegionInfo, framework.NewStatus(sts.Code()).WithError(fitErr)
		}
		return ScheduleResult{nominatingInfo: clearNominatedRunner}, assumedRegionInfo, sts
	}

	// Run "permit" plugins.
	runPermitStatus := fwk.RunPermitPlugins(ctx, state, assumedRegion, scheduleResult.SuggestedHost)
	if !runPermitStatus.IsWait() && !runPermitStatus.IsSuccess() {
		// trigger un-reserve to clean up state associated with the reserved Region
		fwk.RunReservePluginsUnreserve(ctx, state, assumedRegion, scheduleResult.SuggestedHost)
		if forgetErr := sched.Cache.ForgetRegion(logger, assumedRegion); forgetErr != nil {
			logger.Error(forgetErr, "Scheduler cache ForgetRegion failed")
		}

		if runPermitStatus.IsRejected() {
			fitErr := &framework.FitError{
				NumAllRunners: 1,
				Region:        region,
				Diagnosis: framework.Diagnosis{
					RunnerToStatusMap: framework.RunnerToStatusMap{scheduleResult.SuggestedHost: runPermitStatus},
				},
			}
			fitErr.Diagnosis.AddPluginStatus(runPermitStatus)
			return ScheduleResult{nominatingInfo: clearNominatedRunner}, assumedRegionInfo, framework.NewStatus(runPermitStatus.Code()).WithError(fitErr)
		}

		return ScheduleResult{nominatingInfo: clearNominatedRunner}, assumedRegionInfo, runPermitStatus
	}

	// At the end of a successful scheduling cycle, pop and move up Regions if needed.
	if len(regionsToActivate.Map) != 0 {
		sched.SchedulingQueue.Activate(logger, regionsToActivate.Map)
		// Clear the entries after activation.
		regionsToActivate.Map = make(map[string]*corev1.Region)
	}

	return scheduleResult, assumedRegionInfo, nil
}

// bindingCycle tries to bind an assumed Region.
func (sched *Scheduler) bindingCycle(
	ctx context.Context,
	state *framework.CycleState,
	fwk framework.Framework,
	scheduleResult ScheduleResult,
	assumedRegionInfo *framework.QueuedRegionInfo,
	start time.Time,
	regionsToActivate *framework.RegionsToActivate) *framework.Status {
	logger := klog.FromContext(ctx)

	assumedRegion := assumedRegionInfo.Region

	// Run "permit" plugins.
	if status := fwk.WaitOnPermit(ctx, assumedRegion); !status.IsSuccess() {
		if status.IsRejected() {
			fitErr := &framework.FitError{
				NumAllRunners: 1,
				Region:        assumedRegionInfo.Region,
				Diagnosis: framework.Diagnosis{
					RunnerToStatusMap:    framework.RunnerToStatusMap{scheduleResult.SuggestedHost: status},
					UnschedulablePlugins: sets.New(status.Plugin()),
				},
			}
			return framework.NewStatus(status.Code()).WithError(fitErr)
		}
		return status
	}

	// Run "prebind" plugins.
	if status := fwk.RunPreBindPlugins(ctx, state, assumedRegion, scheduleResult.SuggestedHost); !status.IsSuccess() {
		return status
	}

	// Run "bind" plugins.
	if status := sched.bind(ctx, fwk, assumedRegion, scheduleResult.SuggestedHost, state); !status.IsSuccess() {
		return status
	}

	// Calculating runnerResourceString can be heavy. Avoid it if klog verbosity is below 2.
	logger.V(2).Info("Successfully bound region to runner", "region", klog.KObj(assumedRegion), "runner", scheduleResult.SuggestedHost, "evaluatedRunners", scheduleResult.EvaluatedRunners, "feasibleRunners", scheduleResult.FeasibleRunners)
	metrics.RegionScheduled(fwk.ProfileName(), metrics.SinceInSeconds(start))
	metrics.RegionSchedulingAttempts.Observe(float64(assumedRegionInfo.Attempts))
	if assumedRegionInfo.InitialAttemptTimestamp != nil {
		metrics.RegionSchedulingSLIDuration.WithLabelValues(getAttemptsLabel(assumedRegionInfo)).Observe(metrics.SinceInSeconds(*assumedRegionInfo.InitialAttemptTimestamp))
	}
	// Run "postbind" plugins.
	fwk.RunPostBindPlugins(ctx, state, assumedRegion, scheduleResult.SuggestedHost)

	// At the end of a successful binding cycle, move up Regions if needed.
	if len(regionsToActivate.Map) != 0 {
		sched.SchedulingQueue.Activate(logger, regionsToActivate.Map)
		// Unlike the logic in schedulingCycle(), we don't bother deleting the entries
		// as `regionsToActivate.Map` is no longer consumed.
	}

	return nil
}

func (sched *Scheduler) handleBindingCycleError(
	ctx context.Context,
	state *framework.CycleState,
	fwk framework.Framework,
	regionInfo *framework.QueuedRegionInfo,
	start time.Time,
	scheduleResult ScheduleResult,
	status *framework.Status) {
	logger := klog.FromContext(ctx)

	assumedRegion := regionInfo.Region
	// trigger un-reserve plugins to clean up state associated with the reserved Region
	fwk.RunReservePluginsUnreserve(ctx, state, assumedRegion, scheduleResult.SuggestedHost)
	if forgetErr := sched.Cache.ForgetRegion(logger, assumedRegion); forgetErr != nil {
		logger.Error(forgetErr, "scheduler cache ForgetRegion failed")
	} else {
		// "Forget"ing an assumed Region in binding cycle should be treated as a RegionDelete event,
		// as the assumed Region had occupied a certain amount of resources in scheduler cache.
		//
		// Avoid moving the assumed Region itself as it's always Unschedulable.
		// It's intentional to "defer" this operation; otherwise MoveAllToActiveOrBackoffQueue() would
		// add this event to in-flight events and thus move the assumed region to backoffQ anyways if the plugins don't have appropriate QueueingHint.
		if status.IsRejected() {
			defer sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, internalqueue.AssignedRegionDelete, assumedRegion, nil, func(region *corev1.Region) bool {
				return assumedRegion.UID != region.UID
			})
		} else {
			sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, internalqueue.AssignedRegionDelete, assumedRegion, nil, nil)
		}
	}

	sched.FailureHandler(ctx, fwk, regionInfo, status, clearNominatedRunner, start)
}

func (sched *Scheduler) frameworkForRegion(region *corev1.Region) (framework.Framework, error) {
	fwk, ok := sched.Profiles[region.Spec.SchedulerName]
	if !ok {
		return nil, fmt.Errorf("profile not found for scheduler name %q", region.Spec.SchedulerName)
	}
	return fwk, nil
}

// skipRegionSchedule returns true if we could skip scheduling the region for specified cases.
func (sched *Scheduler) skipRegionSchedule(ctx context.Context, fwk framework.Framework, region *corev1.Region) bool {
	// Case 1: region is being deleted.
	if region.DeletionTimestamp != nil {
		fwk.EventRecorder().Eventf(region, nil, corev1.EventTypeWarning, "FailedScheduling", "Scheduling", "skip schedule deleting region: %v/%v", region.Namespace, region.Name)
		klog.FromContext(ctx).V(3).Info("Skip schedule deleting region", "region", klog.KObj(region))
		return true
	}

	// Case 2: region that has been assumed could be skipped.
	// An assumed region can be added again to the scheduling queue if it got an update event
	// during its previous scheduling cycle but before getting assumed.
	isAssumed, err := sched.Cache.IsAssumedRegion(region)
	if err != nil {
		// TODO(91633): pass ctx into a revised HandleError
		utilruntime.HandleError(fmt.Errorf("failed to check whether region %s/%s is assumed: %v", region.Namespace, region.Name, err))
		return false
	}
	return isAssumed
}

// scheduleRegion tries to schedule the given region to one of the runners in the runner list.
// If it succeeds, it will return the name of the runner.
// If it fails, it will return a FitError with reasons.
func (sched *Scheduler) scheduleRegion(ctx context.Context, fwk framework.Framework, state *framework.CycleState, region *corev1.Region) (result ScheduleResult, err error) {
	trace := utiltrace.New("Scheduling", utiltrace.Field{Key: "namespace", Value: region.Namespace}, utiltrace.Field{Key: "name", Value: region.Name})
	defer trace.LogIfLong(100 * time.Millisecond)
	if err := sched.Cache.UpdateSnapshot(klog.FromContext(ctx), sched.runnerInfoSnapshot); err != nil {
		return result, err
	}
	trace.Step("Snapshotting scheduler cache and runner infos done")

	if sched.runnerInfoSnapshot.NumRunners() == 0 {
		return result, ErrNoRunnersAvailable
	}

	feasibleRunners, diagnosis, err := sched.findRunnersThatFitRegion(ctx, fwk, state, region)
	if err != nil {
		return result, err
	}
	trace.Step("Computing predicates done")

	if len(feasibleRunners) == 0 {
		return result, &framework.FitError{
			Region:        region,
			NumAllRunners: sched.runnerInfoSnapshot.NumRunners(),
			Diagnosis:     diagnosis,
		}
	}

	// When only one runner after predicate, just use it.
	if len(feasibleRunners) == 1 {
		return ScheduleResult{
			SuggestedHost:    feasibleRunners[0].Runner().Name,
			EvaluatedRunners: 1 + len(diagnosis.RunnerToStatusMap),
			FeasibleRunners:  1,
		}, nil
	}

	priorityList, err := prioritizeRunners(ctx, sched.Extenders, fwk, state, region, feasibleRunners)
	if err != nil {
		return result, err
	}

	host, _, err := selectHost(priorityList, numberOfHighestScoredRunnersToReport)
	trace.Step("Prioritizing done")

	return ScheduleResult{
		SuggestedHost:    host,
		EvaluatedRunners: len(feasibleRunners) + len(diagnosis.RunnerToStatusMap),
		FeasibleRunners:  len(feasibleRunners),
	}, err
}

// Filters the runners to find the ones that fit the region based on the framework
// filter plugins and filter extenders.
func (sched *Scheduler) findRunnersThatFitRegion(ctx context.Context, fwk framework.Framework, state *framework.CycleState, region *corev1.Region) ([]*framework.RunnerInfo, framework.Diagnosis, error) {
	logger := klog.FromContext(ctx)

	allRunners, err := sched.runnerInfoSnapshot.RunnerInfos().List()
	if err != nil {
		return nil, framework.Diagnosis{
			RunnerToStatusMap: make(framework.RunnerToStatusMap),
		}, err
	}

	diagnosis := framework.Diagnosis{
		RunnerToStatusMap: make(framework.RunnerToStatusMap, len(allRunners)),
	}
	// Run "prefilter" plugins.
	preRes, s := fwk.RunPreFilterPlugins(ctx, state, region)
	if !s.IsSuccess() {
		if !s.IsRejected() {
			return nil, diagnosis, s.AsError()
		}
		// All runners in RunnerToStatusMap will have the same status so that they can be handled in the preemption.
		// Some non trivial refactoring is needed to avoid this copy.
		for _, n := range allRunners {
			diagnosis.RunnerToStatusMap[n.Runner().Name] = s
		}

		// Record the messages from PreFilter in Diagnosis.PreFilterMsg.
		msg := s.Message()
		diagnosis.PreFilterMsg = msg
		logger.V(5).Info("Status after running PreFilter plugins for region", "region", klog.KObj(region), "status", msg)
		diagnosis.AddPluginStatus(s)
		return nil, diagnosis, nil
	}

	// "NominatedRunnerName" can potentially be set in a previous scheduling cycle as a result of preemption.
	// This runner is likely the only candidate that will fit the region, and hence we try it first before iterating over all runners.
	//if len(region.Status.NominatedRunnerName) > 0 {
	//	feasibleRunners, err := sched.evaluateNominatedRunner(ctx, region, fwk, state, diagnosis)
	//	if err != nil {
	//		logger.Error(err, "Evaluation failed on nominated runner", "region", klog.KObj(region), "runner", region.Status.NominatedRunnerName)
	//	}
	//	// Nominated runner passes all the filters, scheduler is good to assign this runner to the region.
	//	if len(feasibleRunners) != 0 {
	//		return feasibleRunners, diagnosis, nil
	//	}
	//}

	runners := allRunners
	if !preRes.AllRunners() {
		runners = make([]*framework.RunnerInfo, 0, len(preRes.RunnerNames))
		for _, n := range allRunners {
			if !preRes.RunnerNames.Has(n.Runner().Name) {
				// We consider Runners that are filtered out by PreFilterResult as rejected via UnschedulableAndUnresolvable.
				// We have to record them in RunnerToStatusMap so that they won't be considered as candidates in the preemption.
				diagnosis.RunnerToStatusMap[n.Runner().Name] = framework.NewStatus(framework.UnschedulableAndUnresolvable, "runner is filtered out by the prefilter result")
				continue
			}
			runners = append(runners, n)
		}
	}
	feasibleRunners, err := sched.findRunnersThatPassFilters(ctx, fwk, state, region, &diagnosis, runners)
	// always try to update the sched.nextStartRunnerIndex regardless of whether an error has occurred
	// this is helpful to make sure that all the runners have a chance to be searched
	processedRunners := len(feasibleRunners) + len(diagnosis.RunnerToStatusMap)
	sched.nextStartRunnerIndex = (sched.nextStartRunnerIndex + processedRunners) % len(runners)
	if err != nil {
		return nil, diagnosis, err
	}

	feasibleRunnersAfterExtender, err := findRunnersThatPassExtenders(ctx, sched.Extenders, region, feasibleRunners, diagnosis.RunnerToStatusMap)
	if err != nil {
		return nil, diagnosis, err
	}
	if len(feasibleRunnersAfterExtender) != len(feasibleRunners) {
		// Extenders filtered out some runners.
		//
		// Extender doesn't support any kind of requeueing feature like EnqueueExtensions in the scheduling framework.
		// When Extenders reject some Runners and the region ends up being unschedulable,
		// we put framework.ExtenderName to pInfo.UnschedulablePlugins.
		// This Region will be requeued from unschedulable region pool to activeQ/backoffQ
		// by any kind of cluster events.
		// https://github.com/kubernetes/kubernetes/issues/122019
		if diagnosis.UnschedulablePlugins == nil {
			diagnosis.UnschedulablePlugins = sets.New[string]()
		}
		diagnosis.UnschedulablePlugins.Insert(framework.ExtenderName)
	}

	return feasibleRunnersAfterExtender, diagnosis, nil
}

//func (sched *Scheduler) evaluateNominatedRunner(ctx context.Context, region *corev1.Region, fwk framework.Framework, state *framework.CycleState, diagnosis framework.Diagnosis) ([]*framework.RunnerInfo, error) {
//	nnn := region.Status.NominatedRunnerName
//	runnerInfo, err := sched.runnerInfoSnapshot.Get(nnn)
//	if err != nil {
//		return nil, err
//	}
//	runner := []*framework.RunnerInfo{runnerInfo}
//	feasibleRunners, err := sched.findRunnersThatPassFilters(ctx, fwk, state, region, &diagnosis, runner)
//	if err != nil {
//		return nil, err
//	}
//
//	feasibleRunners, err = findRunnersThatPassExtenders(ctx, sched.Extenders, region, feasibleRunners, diagnosis.RunnerToStatusMap)
//	if err != nil {
//		return nil, err
//	}
//
//	return feasibleRunners, nil
//}

// hasScoring checks if scoring runners is configured.
func (sched *Scheduler) hasScoring(fwk framework.Framework) bool {
	if fwk.HasScorePlugins() {
		return true
	}
	for _, extender := range sched.Extenders {
		if extender.IsPrioritizer() {
			return true
		}
	}
	return false
}

// hasExtenderFilters checks if any extenders filter runners.
func (sched *Scheduler) hasExtenderFilters() bool {
	for _, extender := range sched.Extenders {
		if extender.IsFilter() {
			return true
		}
	}
	return false
}

// findRunnersThatPassFilters finds the runners that fit the filter plugins.
func (sched *Scheduler) findRunnersThatPassFilters(
	ctx context.Context,
	fwk framework.Framework,
	state *framework.CycleState,
	region *corev1.Region,
	diagnosis *framework.Diagnosis,
	runners []*framework.RunnerInfo) ([]*framework.RunnerInfo, error) {
	numAllRunners := len(runners)
	numRunnersToFind := sched.numFeasibleRunnersToFind(fwk.PercentageOfRunnersToScore(), int32(numAllRunners))
	if !sched.hasExtenderFilters() && !sched.hasScoring(fwk) {
		numRunnersToFind = 1
	}

	// Create feasible list with enough space to avoid growing it
	// and allow assigning.
	feasibleRunners := make([]*framework.RunnerInfo, numRunnersToFind)

	if !fwk.HasFilterPlugins() {
		for i := range feasibleRunners {
			feasibleRunners[i] = runners[(sched.nextStartRunnerIndex+i)%numAllRunners]
		}
		return feasibleRunners, nil
	}

	errCh := parallelize.NewErrorChannel()
	var feasibleRunnersLen int32
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type runnerStatus struct {
		runner string
		status *framework.Status
	}
	result := make([]*runnerStatus, numAllRunners)
	checkRunner := func(i int) {
		// We check the runners starting from where we left off in the previous scheduling cycle,
		// this is to make sure all runners have the same chance of being examined across regions.
		runnerInfo := runners[(sched.nextStartRunnerIndex+i)%numAllRunners]
		status := fwk.RunFilterPluginsWithNominatedRegions(ctx, state, region, runnerInfo)
		if status.Code() == framework.Error {
			errCh.SendErrorWithCancel(status.AsError(), cancel)
			return
		}
		if status.IsSuccess() {
			length := atomic.AddInt32(&feasibleRunnersLen, 1)
			if length > numRunnersToFind {
				cancel()
				atomic.AddInt32(&feasibleRunnersLen, -1)
			} else {
				feasibleRunners[length-1] = runnerInfo
			}
		} else {
			result[i] = &runnerStatus{runner: runnerInfo.Runner().Name, status: status}
		}
	}

	beginCheckRunner := time.Now()
	statusCode := framework.Success
	defer func() {
		// We record Filter extension point latency here instead of in framework.go because framework.RunFilterPlugins
		// function is called for each runner, whereas we want to have an overall latency for all runners per scheduling cycle.
		// Note that this latency also includes latency for `addNominatedRegions`, which calls framework.RunPreFilterAddRegion.
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.Filter, statusCode.String(), fwk.ProfileName()).Observe(metrics.SinceInSeconds(beginCheckRunner))
	}()

	// Stops searching for more runners once the configured number of feasible runners
	// are found.
	fwk.Parallelizer().Until(ctx, numAllRunners, checkRunner, metrics.Filter)
	feasibleRunners = feasibleRunners[:feasibleRunnersLen]
	for _, item := range result {
		if item == nil {
			continue
		}
		diagnosis.RunnerToStatusMap[item.runner] = item.status
		diagnosis.AddPluginStatus(item.status)
	}
	if err := errCh.ReceiveError(); err != nil {
		statusCode = framework.Error
		return feasibleRunners, err
	}
	return feasibleRunners, nil
}

// numFeasibleRunnersToFind returns the number of feasible runners that once found, the scheduler stops
// its search for more feasible runners.
func (sched *Scheduler) numFeasibleRunnersToFind(percentageOfRunnersToScore *int32, numAllRunners int32) (numRunners int32) {
	if numAllRunners < minFeasibleRunnersToFind {
		return numAllRunners
	}

	// Use profile percentageOfRunnersToScore if it's set. Otherwise, use global percentageOfRunnersToScore.
	var percentage int32
	if percentageOfRunnersToScore != nil {
		percentage = *percentageOfRunnersToScore
	} else {
		percentage = sched.percentageOfRunnersToScore
	}

	if percentage == 0 {
		percentage = int32(50) - numAllRunners/125
		if percentage < minFeasibleRunnersPercentageToFind {
			percentage = minFeasibleRunnersPercentageToFind
		}
	}

	numRunners = numAllRunners * percentage / 100
	if numRunners < minFeasibleRunnersToFind {
		return minFeasibleRunnersToFind
	}

	return numRunners
}

func findRunnersThatPassExtenders(ctx context.Context, extenders []framework.Extender, region *corev1.Region, feasibleRunners []*framework.RunnerInfo, statuses framework.RunnerToStatusMap) ([]*framework.RunnerInfo, error) {
	logger := klog.FromContext(ctx)
	// Extenders are called sequentially.
	// Runners in original feasibleRunners can be excluded in one extender, and pass on to the next
	// extender in a decreasing manner.
	for _, extender := range extenders {
		if len(feasibleRunners) == 0 {
			break
		}
		if !extender.IsInterested(region) {
			continue
		}

		// Status of failed runners in failedAndUnresolvableMap will be added or overwritten in <statuses>,
		// so that the scheduler framework can respect the UnschedulableAndUnresolvable status for
		// particular runners, and this may eventually improve preemption efficiency.
		// Note: users are recommended to configure the extenders that may return UnschedulableAndUnresolvable
		// status ahead of others.
		feasibleList, failedMap, failedAndUnresolvableMap, err := extender.Filter(region, feasibleRunners)
		if err != nil {
			if extender.IsIgnorable() {
				logger.Info("Skipping extender as it returned error and has ignorable flag set", "extender", extender, "err", err)
				continue
			}
			return nil, err
		}

		for failedRunnerName, failedMsg := range failedAndUnresolvableMap {
			var aggregatedReasons []string
			if _, found := statuses[failedRunnerName]; found {
				aggregatedReasons = statuses[failedRunnerName].Reasons()
			}
			aggregatedReasons = append(aggregatedReasons, failedMsg)
			statuses[failedRunnerName] = framework.NewStatus(framework.UnschedulableAndUnresolvable, aggregatedReasons...)
		}

		for failedRunnerName, failedMsg := range failedMap {
			if _, found := failedAndUnresolvableMap[failedRunnerName]; found {
				// failedAndUnresolvableMap takes precedence over failedMap
				// note that this only happens if the extender returns the runner in both maps
				continue
			}
			if _, found := statuses[failedRunnerName]; !found {
				statuses[failedRunnerName] = framework.NewStatus(framework.Unschedulable, failedMsg)
			} else {
				statuses[failedRunnerName].AppendReason(failedMsg)
			}
		}

		feasibleRunners = feasibleList
	}
	return feasibleRunners, nil
}

// prioritizeRunners prioritizes the runners by running the score plugins,
// which return a score for each runner from the call to RunScorePlugins().
// The scores from each plugin are added together to make the score for that runner, then
// any extenders are run as well.
// All scores are finally combined (added) to get the total weighted scores of all runners
func prioritizeRunners(
	ctx context.Context,
	extenders []framework.Extender,
	fwk framework.Framework,
	state *framework.CycleState,
	region *corev1.Region,
	runners []*framework.RunnerInfo,
) ([]framework.RunnerPluginScores, error) {
	logger := klog.FromContext(ctx)
	// If no priority configs are provided, then all runners will have a score of one.
	// This is required to generate the priority list in the required format
	if len(extenders) == 0 && !fwk.HasScorePlugins() {
		result := make([]framework.RunnerPluginScores, 0, len(runners))
		for i := range runners {
			result = append(result, framework.RunnerPluginScores{
				Name:       runners[i].Runner().Name,
				TotalScore: 1,
			})
		}
		return result, nil
	}

	// Run PreScore plugins.
	preScoreStatus := fwk.RunPreScorePlugins(ctx, state, region, runners)
	if !preScoreStatus.IsSuccess() {
		return nil, preScoreStatus.AsError()
	}

	// Run the Score plugins.
	runnersScores, scoreStatus := fwk.RunScorePlugins(ctx, state, region, runners)
	if !scoreStatus.IsSuccess() {
		return nil, scoreStatus.AsError()
	}

	// Additional details logged at level 10 if enabled.
	loggerVTen := logger.V(10)
	if loggerVTen.Enabled() {
		for _, runnerScore := range runnersScores {
			for _, pluginScore := range runnerScore.Scores {
				loggerVTen.Info("Plugin scored runner for region", "region", klog.KObj(region), "plugin", pluginScore.Name, "runner", runnerScore.Name, "score", pluginScore.Score)
			}
		}
	}

	if len(extenders) != 0 && runners != nil {
		// allRunnerExtendersScores has all extenders scores for all runners.
		// It is keyed with runner name.
		allRunnerExtendersScores := make(map[string]*framework.RunnerPluginScores, len(runners))
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i := range extenders {
			if !extenders[i].IsInterested(region) {
				continue
			}
			wg.Add(1)
			go func(extIndex int) {
				metrics.Goroutines.WithLabelValues(metrics.PrioritizingExtender).Inc()
				defer func() {
					metrics.Goroutines.WithLabelValues(metrics.PrioritizingExtender).Dec()
					wg.Done()
				}()
				prioritizedList, weight, err := extenders[extIndex].Prioritize(region, runners)
				if err != nil {
					// Prioritization errors from extender can be ignored, let k8s/other extenders determine the priorities
					logger.V(5).Info("Failed to run extender's priority function. No score given by this extender.", "error", err, "region", klog.KObj(region), "extender", extenders[extIndex].Name())
					return
				}
				mu.Lock()
				defer mu.Unlock()
				for i := range *prioritizedList {
					runnername := (*prioritizedList)[i].Host
					score := (*prioritizedList)[i].Score
					if loggerVTen.Enabled() {
						loggerVTen.Info("Extender scored runner for region", "region", klog.KObj(region), "extender", extenders[extIndex].Name(), "runner", runnername, "score", score)
					}

					// MaxExtenderPriority may diverge from the max priority used in the scheduler and defined by MaxRunnerScore,
					// therefore we need to scale the score returned by extenders to the score range used by the scheduler.
					finalscore := score * weight * (framework.MaxRunnerScore / extenderv1.MaxExtenderPriority)

					if allRunnerExtendersScores[runnername] == nil {
						allRunnerExtendersScores[runnername] = &framework.RunnerPluginScores{
							Name:   runnername,
							Scores: make([]framework.PluginScore, 0, len(extenders)),
						}
					}
					allRunnerExtendersScores[runnername].Scores = append(allRunnerExtendersScores[runnername].Scores, framework.PluginScore{
						Name:  extenders[extIndex].Name(),
						Score: finalscore,
					})
					allRunnerExtendersScores[runnername].TotalScore += finalscore
				}
			}(i)
		}
		// wait for all go routines to finish
		wg.Wait()
		for i := range runnersScores {
			if score, ok := allRunnerExtendersScores[runners[i].Runner().Name]; ok {
				runnersScores[i].Scores = append(runnersScores[i].Scores, score.Scores...)
				runnersScores[i].TotalScore += score.TotalScore
			}
		}
	}

	if loggerVTen.Enabled() {
		for i := range runnersScores {
			loggerVTen.Info("Calculated runner's final score for region", "region", klog.KObj(region), "runner", runnersScores[i].Name, "score", runnersScores[i].TotalScore)
		}
	}
	return runnersScores, nil
}

var errEmptyPriorityList = errors.New("empty priorityList")

// selectHost takes a prioritized list of runners and then picks one
// in a reservoir sampling manner from the runners that had the highest score.
// It also returns the top {count} Runners,
// and the top of the list will be always the selected host.
func selectHost(runnerScoreList []framework.RunnerPluginScores, count int) (string, []framework.RunnerPluginScores, error) {
	if len(runnerScoreList) == 0 {
		return "", nil, errEmptyPriorityList
	}

	var h runnerScoreHeap = runnerScoreList
	heap.Init(&h)
	cntOfMaxScore := 1
	selectedIndex := 0
	// The top of the heap is the RunnerScoreResult with the highest score.
	sortedRunnerScoreList := make([]framework.RunnerPluginScores, 0, count)
	sortedRunnerScoreList = append(sortedRunnerScoreList, heap.Pop(&h).(framework.RunnerPluginScores))

	// This for-loop will continue until all Runners with the highest scores get checked for a reservoir sampling,
	// and sortedRunnerScoreList gets (count - 1) elements.
	for ns := heap.Pop(&h).(framework.RunnerPluginScores); ; ns = heap.Pop(&h).(framework.RunnerPluginScores) {
		if ns.TotalScore != sortedRunnerScoreList[0].TotalScore && len(sortedRunnerScoreList) == count {
			break
		}

		if ns.TotalScore == sortedRunnerScoreList[0].TotalScore {
			cntOfMaxScore++
			if rand.Intn(cntOfMaxScore) == 0 {
				// Replace the candidate with probability of 1/cntOfMaxScore
				selectedIndex = cntOfMaxScore - 1
			}
		}

		sortedRunnerScoreList = append(sortedRunnerScoreList, ns)

		if h.Len() == 0 {
			break
		}
	}

	if selectedIndex != 0 {
		// replace the first one with selected one
		previous := sortedRunnerScoreList[0]
		sortedRunnerScoreList[0] = sortedRunnerScoreList[selectedIndex]
		sortedRunnerScoreList[selectedIndex] = previous
	}

	if len(sortedRunnerScoreList) > count {
		sortedRunnerScoreList = sortedRunnerScoreList[:count]
	}

	return sortedRunnerScoreList[0].Name, sortedRunnerScoreList, nil
}

// runnerScoreHeap is a heap of framework.RunnerPluginScores.
type runnerScoreHeap []framework.RunnerPluginScores

// runnerScoreHeap implements heap.Interface.
var _ heap.Interface = &runnerScoreHeap{}

func (h runnerScoreHeap) Len() int           { return len(h) }
func (h runnerScoreHeap) Less(i, j int) bool { return h[i].TotalScore > h[j].TotalScore }
func (h runnerScoreHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *runnerScoreHeap) Push(x interface{}) {
	*h = append(*h, x.(framework.RunnerPluginScores))
}

func (h *runnerScoreHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// assume signals to the cache that a region is already in the cache, so that binding can be asynchronous.
// assume modifies `assumed`.
func (sched *Scheduler) assume(logger klog.Logger, assumed *corev1.Region, host ...string) error {
	// Optimistically assume that the binding will succeed and send it to apiserver
	// in the background.
	// If the binding fails, scheduler will release resources allocated to assumed region
	// immediately.
	assumed.Spec.RunnerNames = host

	if err := sched.Cache.AssumeRegion(logger, assumed); err != nil {
		logger.Error(err, "Scheduler cache AssumeRegion failed")
		return err
	}
	// if "assumed" is a nominated region, we should remove it from internal cache
	if sched.SchedulingQueue != nil {
		sched.SchedulingQueue.DeleteNominatedRegionIfExists(assumed)
	}

	return nil
}

// bind binds a region to a given runner defined in a binding object.
// The precedence for binding is: (1) extenders and (2) framework plugins.
// We expect this to run asynchronously, so we handle binding metrics internally.
func (sched *Scheduler) bind(ctx context.Context, fwk framework.Framework, assumed *corev1.Region, targetRunner string, state *framework.CycleState) (status *framework.Status) {
	logger := klog.FromContext(ctx)
	defer func() {
		sched.finishBinding(logger, fwk, assumed, targetRunner, status)
	}()

	bound, err := sched.extendersBinding(logger, assumed, targetRunner)
	if bound {
		return framework.AsStatus(err)
	}
	return fwk.RunBindPlugins(ctx, state, assumed, targetRunner)
}

// TODO(#87159): Move this to a Plugin.
func (sched *Scheduler) extendersBinding(logger klog.Logger, region *corev1.Region, runner string) (bool, error) {
	for _, extender := range sched.Extenders {
		if !extender.IsBinder() || !extender.IsInterested(region) {
			continue
		}
		err := extender.Bind(&corev1.Binding{
			ObjectMeta: metav1.ObjectMeta{Namespace: region.Namespace, Name: region.Name, UID: region.UID},
			Target:     corev1.ObjectReference{Kind: "Runner", Names: []string{runner}},
		})
		if err != nil && extender.IsIgnorable() {
			logger.Info("Skipping extender in bind as it returned error and has ignorable flag set", "extender", extender, "err", err)
			continue
		}
		return true, err
	}
	return false, nil
}

func (sched *Scheduler) finishBinding(logger klog.Logger, fwk framework.Framework, assumed *corev1.Region, targetRunner string, status *framework.Status) {
	if finErr := sched.Cache.FinishBinding(logger, assumed); finErr != nil {
		logger.Error(finErr, "Scheduler cache FinishBinding failed")
	}
	if !status.IsSuccess() {
		logger.V(1).Info("Failed to bind region", "region", klog.KObj(assumed))
		return
	}

	fwk.EventRecorder().Eventf(assumed, nil, corev1.EventTypeNormal, "Scheduled", "Binding", "Successfully assigned %v/%v to %v", assumed.Namespace, assumed.Name, targetRunner)
}

func getAttemptsLabel(p *framework.QueuedRegionInfo) string {
	// We breakdown the region scheduling duration by attempts capped to a limit
	// to avoid ending up with a high cardinality metric.
	if p.Attempts >= 15 {
		return "15+"
	}
	return strconv.Itoa(p.Attempts)
}

// handleSchedulingFailure records an event for the region that indicates the
// region has failed to schedule. Also, update the region condition and nominated runner name if set.
func (sched *Scheduler) handleSchedulingFailure(ctx context.Context, fwk framework.Framework, regionInfo *framework.QueuedRegionInfo, status *framework.Status, nominatingInfo *framework.NominatingInfo, start time.Time) {
	calledDone := false
	defer func() {
		if !calledDone {
			// Basically, AddUnschedulableIfNotPresent calls DoneRegion internally.
			// But, AddUnschedulableIfNotPresent isn't called in some corner cases.
			// Here, we call DoneRegion explicitly to avoid leaking the region.
			sched.SchedulingQueue.Done(regionInfo.Region.UID)
		}
	}()

	logger := klog.FromContext(ctx)
	reason := corev1.RegionReasonSchedulerError
	if status.IsRejected() {
		reason = corev1.RegionReasonUnschedulable
	}

	switch reason {
	case corev1.RegionReasonUnschedulable:
		metrics.RegionUnschedulable(fwk.ProfileName(), metrics.SinceInSeconds(start))
	case corev1.RegionReasonSchedulerError:
		metrics.RegionScheduleError(fwk.ProfileName(), metrics.SinceInSeconds(start))
	}

	region := regionInfo.Region
	err := status.AsError()
	errMsg := status.Message()

	if err == ErrNoRunnersAvailable {
		logger.V(2).Info("Unable to schedule region; no runners are registered to the cluster; waiting", "region", klog.KObj(region))
	} else if fitError, ok := err.(*framework.FitError); ok { // Inject UnschedulablePlugins to RegionInfo, which will be used later for moving Regions between queues efficiently.
		regionInfo.UnschedulablePlugins = fitError.Diagnosis.UnschedulablePlugins
		regionInfo.PendingPlugins = fitError.Diagnosis.PendingPlugins
		logger.V(2).Info("Unable to schedule region; no fit; waiting", "region", klog.KObj(region), "err", errMsg)
	} else {
		logger.Error(err, "Error scheduling region; retrying", "region", klog.KObj(region))
	}

	// Check if the Region exists in informer cache.
	regionLister := fwk.SharedInformerFactory().Core().V1().Regions().Lister()
	cachedRegion, e := regionLister.Get(region.Name)
	if e != nil {
		logger.Info("Region doesn't exist in informer cache", "region", klog.KObj(region), "err", e)
		// We need to call DoneRegion here because we don't call AddUnschedulableIfNotPresent in this case.
	} else {
		// In the case of extender, the region may have been bound successfully, but timed out returning its response to the scheduler.
		// It could result in the live version to carry .spec.runnerName, and that's inconsistent with the internal-queued version.

		// As <cachedRegion> is from SharedInformer, we need to do a DeepCopy() here.
		// ignore this err since apiserver doesn't properly validate affinity terms
		// and we can't fix the validation for backwards compatibility.
		regionInfo.RegionInfo, _ = framework.NewRegionInfo(cachedRegion.DeepCopy())
		if err := sched.SchedulingQueue.AddUnschedulableIfNotPresent(logger, regionInfo, sched.SchedulingQueue.SchedulingCycle()); err != nil {
			logger.Error(err, "Error occurred")
		}
		calledDone = true
	}

	// Update the scheduling queue with the nominated region information. Without
	// this, there would be a race condition between the next scheduling cycle
	// and the time the scheduler receives a Region Update for the nominated region.
	// Here we check for nil only for tests.
	if sched.SchedulingQueue != nil {
		logger := klog.FromContext(ctx)
		sched.SchedulingQueue.AddNominatedRegion(logger, regionInfo.RegionInfo, nominatingInfo)
	}

	if err == nil {
		// Only tests can reach here.
		return
	}

	msg := truncateMessage(errMsg)
	fwk.EventRecorder().Eventf(region, nil, corev1.EventTypeWarning, "FailedScheduling", "Scheduling", msg)
	if err := updateRegion(ctx, sched.client, region, &corev1.RegionCondition{
		Type:    corev1.RegionScheduled,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: errMsg,
	}, nominatingInfo); err != nil {
		klog.FromContext(ctx).Error(err, "Error updating region", "region", klog.KObj(region))
	}
}

// truncateMessage truncates a message if it hits the NoteLengthLimit.
func truncateMessage(message string) string {
	maxLimit := validation.NoteLengthLimit
	if len(message) <= maxLimit {
		return message
	}
	suffix := " ..."
	return message[:maxLimit-len(suffix)] + suffix
}

func updateRegion(ctx context.Context, client clientset.Interface, region *corev1.Region, condition *corev1.RegionCondition, nominatingInfo *framework.NominatingInfo) error {
	logger := klog.FromContext(ctx)
	logger.V(3).Info("Updating region condition", "region", klog.KObj(region), "conditionType", condition.Type, "conditionStatus", condition.Status, "conditionReason", condition.Reason)
	regionStatusCopy := region.Status.DeepCopy()
	// NominatedRunnerName is updated only if we are trying to set it, and the value is
	// different from the existing one.
	//nnnNeedsUpdate := nominatingInfo.Mode() == framework.ModeOverride && region.Status.NominatedRunnerName != nominatingInfo.NominatedRunnerName
	nnnNeedsUpdate := nominatingInfo.Mode() == framework.ModeOverride
	if !regionutil.UpdateRegionCondition(regionStatusCopy, condition) && !nnnNeedsUpdate {
		return nil
	}
	if nnnNeedsUpdate {
		//regionStatusCopy.NominatedRunnerName = nominatingInfo.NominatedRunnerName
	}
	return util.PatchRegionStatus(ctx, client, region, regionStatusCopy)
}
