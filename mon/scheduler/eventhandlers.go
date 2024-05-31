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
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runneraffinity"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runnername"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runnerresources"
	"github.com/olive-io/olive/mon/scheduler/internal/queue"
	"github.com/olive-io/olive/mon/scheduler/profile"
	corev1helpers "github.com/olive-io/olive/pkg/scheduling/corev1"
	corev1runneraffinity "github.com/olive-io/olive/pkg/scheduling/corev1/runneraffinity"
)

func (sched *Scheduler) addRunnerToCache(obj interface{}) {
	logger := sched.logger
	runner, ok := obj.(*corev1.Runner)
	if !ok {
		logger.Error(nil, "Cannot convert to *corev1.Runner", "obj", obj)
		return
	}

	logger.V(3).Info("Add event for runner", "runner", klog.KObj(runner))
	runnerInfo := sched.Cache.AddRunner(logger, runner)
	sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, queue.RunnerAdd, nil, runner, preCheckForRunner(runnerInfo))
}

func (sched *Scheduler) updateRunnerInCache(oldObj, newObj interface{}) {
	logger := sched.logger
	oldRunner, ok := oldObj.(*corev1.Runner)
	if !ok {
		logger.Error(nil, "Cannot convert oldObj to *corev1.Runner", "oldObj", oldObj)
		return
	}
	newRunner, ok := newObj.(*corev1.Runner)
	if !ok {
		logger.Error(nil, "Cannot convert newObj to *corev1.Runner", "newObj", newObj)
		return
	}

	logger.V(4).Info("Update event for runner", "runner", klog.KObj(newRunner))
	runnerInfo := sched.Cache.UpdateRunner(logger, oldRunner, newRunner)
	// Only requeue unschedulable regions if the runner became more schedulable.
	for _, evt := range runnerSchedulingPropertiesChange(newRunner, oldRunner) {
		sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, evt, oldRunner, newRunner, preCheckForRunner(runnerInfo))
	}
}

func (sched *Scheduler) deleteRunnerFromCache(obj interface{}) {
	logger := sched.logger
	var runner *corev1.Runner
	switch t := obj.(type) {
	case *corev1.Runner:
		runner = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		runner, ok = t.Obj.(*corev1.Runner)
		if !ok {
			logger.Error(nil, "Cannot convert to *corev1.Runner", "obj", t.Obj)
			return
		}
	default:
		logger.Error(nil, "Cannot convert to *corev1.Runner", "obj", t)
		return
	}

	logger.V(3).Info("Delete event for runner", "runner", klog.KObj(runner))
	if err := sched.Cache.RemoveRunner(logger, runner); err != nil {
		logger.Error(err, "Scheduler cache RemoveRunner failed")
	}
}

func (sched *Scheduler) addRegionToSchedulingQueue(obj interface{}) {
	logger := sched.logger
	region := obj.(*corev1.Region)
	logger.V(3).Info("Add event for unscheduled region", "region", klog.KObj(region))
	if err := sched.SchedulingQueue.Add(logger, region); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to queue %T: %v", obj, err))
	}
}

func (sched *Scheduler) updateRegionInSchedulingQueue(oldObj, newObj interface{}) {
	logger := sched.logger
	oldRegion, newRegion := oldObj.(*corev1.Region), newObj.(*corev1.Region)
	// Bypass update event that carries identical objects; otherwise, a duplicated
	// Region may go through scheduling and cause unexpected behavior (see #96071).
	if oldRegion.ResourceVersion == newRegion.ResourceVersion {
		return
	}

	isAssumed, err := sched.Cache.IsAssumedRegion(newRegion)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to check whether region %s/%s is assumed: %v", newRegion.Namespace, newRegion.Name, err))
	}
	if isAssumed {
		return
	}

	logger.V(4).Info("Update event for unscheduled region", "region", klog.KObj(newRegion))
	if err := sched.SchedulingQueue.Update(logger, oldRegion, newRegion); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to update %T: %v", newObj, err))
	}
}

func (sched *Scheduler) deleteRegionFromSchedulingQueue(obj interface{}) {
	logger := sched.logger
	var region *corev1.Region
	switch t := obj.(type) {
	case *corev1.Region:
		region = obj.(*corev1.Region)
	case cache.DeletedFinalStateUnknown:
		var ok bool
		region, ok = t.Obj.(*corev1.Region)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("unable to convert object %T to *corev1.Region in %T", obj, sched))
			return
		}
	default:
		utilruntime.HandleError(fmt.Errorf("unable to handle object in %T: %T", sched, obj))
		return
	}

	logger.V(3).Info("Delete event for unscheduled region", "region", klog.KObj(region))
	if err := sched.SchedulingQueue.Delete(region); err != nil {
		utilruntime.HandleError(fmt.Errorf("unable to dequeue %T: %v", obj, err))
	}
	fwk, err := sched.frameworkForRegion(region)
	if err != nil {
		// This shouldn't happen, because we only accept for scheduling the regions
		// which specify a scheduler name that matches one of the profiles.
		logger.Error(err, "Unable to get profile", "region", klog.KObj(region))
		return
	}
	// If a waiting region is rejected, it indicates it's previously assumed and we're
	// removing it from the scheduler cache. In this case, signal a AssignedRegionDelete
	// event to immediately retry some unscheduled Regions.
	if fwk.RejectWaitingRegion(region.UID) {
		sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, queue.AssignedRegionDelete, region, nil, nil)
	}
}

func (sched *Scheduler) addRegionToCache(obj interface{}) {
	logger := sched.logger
	region, ok := obj.(*corev1.Region)
	if !ok {
		logger.Error(nil, "Cannot convert to *corev1.Region", "obj", obj)
		return
	}

	logger.V(3).Info("Add event for scheduled region", "region", klog.KObj(region))
	if err := sched.Cache.AddRegion(logger, region); err != nil {
		logger.Error(err, "Scheduler cache AddRegion failed", "region", klog.KObj(region))
	}

	sched.SchedulingQueue.AssignedRegionAdded(logger, region)
}

func (sched *Scheduler) updateRegionInCache(oldObj, newObj interface{}) {
	logger := sched.logger
	oldRegion, ok := oldObj.(*corev1.Region)
	if !ok {
		logger.Error(nil, "Cannot convert oldObj to *corev1.Region", "oldObj", oldObj)
		return
	}
	newRegion, ok := newObj.(*corev1.Region)
	if !ok {
		logger.Error(nil, "Cannot convert newObj to *corev1.Region", "newObj", newObj)
		return
	}

	logger.V(4).Info("Update event for scheduled region", "region", klog.KObj(oldRegion))
	if err := sched.Cache.UpdateRegion(logger, oldRegion, newRegion); err != nil {
		logger.Error(err, "Scheduler cache UpdateRegion failed", "region", klog.KObj(oldRegion))
	}

	sched.SchedulingQueue.AssignedRegionUpdated(logger, oldRegion, newRegion)
}

func (sched *Scheduler) deleteRegionFromCache(obj interface{}) {
	logger := sched.logger
	var region *corev1.Region
	switch t := obj.(type) {
	case *corev1.Region:
		region = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		region, ok = t.Obj.(*corev1.Region)
		if !ok {
			logger.Error(nil, "Cannot convert to *corev1.Region", "obj", t.Obj)
			return
		}
	default:
		logger.Error(nil, "Cannot convert to *corev1.Region", "obj", t)
		return
	}

	logger.V(3).Info("Delete event for scheduled region", "region", klog.KObj(region))
	if err := sched.Cache.RemoveRegion(logger, region); err != nil {
		logger.Error(err, "Scheduler cache RemoveRegion failed", "region", klog.KObj(region))
	}

	sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, queue.AssignedRegionDelete, region, nil, nil)
}

// assignedRegion selects regions that are assigned (scheduled and running).
func assignedRegion(region *corev1.Region) bool {
	return len(region.Spec.RunnerNames) != 0
}

// responsibleForRegion returns true if the region has asked to be scheduled by the given scheduler.
func responsibleForRegion(region *corev1.Region, profiles profile.Map) bool {
	return profiles.HandlesSchedulerName(region.Spec.SchedulerName)
}

const (
	// syncedPollPeriod controls how often you look at the status of your sync funcs
	syncedPollPeriod = 100 * time.Millisecond
)

// WaitForHandlersSync waits for EventHandlers to sync.
// It returns true if it was successful, false if the controller should shut down
func (sched *Scheduler) WaitForHandlersSync(ctx context.Context) error {
	return wait.PollUntilContextCancel(ctx, syncedPollPeriod, true, func(ctx context.Context) (done bool, err error) {
		for _, handler := range sched.registeredHandlers {
			if !handler.HasSynced() {
				return false, nil
			}
		}
		return true, nil
	})
}

// addAllEventHandlers is a helper function used in tests and in Scheduler
// to add event handlers for various informers.
func addAllEventHandlers(
	sched *Scheduler,
	informerFactory informers.SharedInformerFactory,
	dynInformerFactory dynamicinformer.DynamicSharedInformerFactory,
	gvkMap map[framework.GVK]framework.ActionType,
) error {
	var (
		handlerRegistration cache.ResourceEventHandlerRegistration
		err                 error
		handlers            []cache.ResourceEventHandlerRegistration
	)
	// scheduled region cache
	if handlerRegistration, err = informerFactory.Core().V1().Regions().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *corev1.Region:
					return assignedRegion(t)
				case cache.DeletedFinalStateUnknown:
					if _, ok := t.Obj.(*corev1.Region); ok {
						// The carried object may be stale, so we don't use it to check if
						// it's assigned or not. Attempting to cleanup anyways.
						return true
					}
					utilruntime.HandleError(fmt.Errorf("unable to convert object %T to *corev1.Region in %T", obj, sched))
					return false
				default:
					utilruntime.HandleError(fmt.Errorf("unable to handle object in %T: %T", sched, obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    sched.addRegionToCache,
				UpdateFunc: sched.updateRegionInCache,
				DeleteFunc: sched.deleteRegionFromCache,
			},
		},
	); err != nil {
		return err
	}
	handlers = append(handlers, handlerRegistration)

	// unscheduled region queue
	if handlerRegistration, err = informerFactory.Core().V1().Regions().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *corev1.Region:
					return !assignedRegion(t) && responsibleForRegion(t, sched.Profiles)
				case cache.DeletedFinalStateUnknown:
					if region, ok := t.Obj.(*corev1.Region); ok {
						// The carried object may be stale, so we don't use it to check if
						// it's assigned or not.
						return responsibleForRegion(region, sched.Profiles)
					}
					utilruntime.HandleError(fmt.Errorf("unable to convert object %T to *corev1.Region in %T", obj, sched))
					return false
				default:
					utilruntime.HandleError(fmt.Errorf("unable to handle object in %T: %T", sched, obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    sched.addRegionToSchedulingQueue,
				UpdateFunc: sched.updateRegionInSchedulingQueue,
				DeleteFunc: sched.deleteRegionFromSchedulingQueue,
			},
		},
	); err != nil {
		return err
	}
	handlers = append(handlers, handlerRegistration)

	if handlerRegistration, err = informerFactory.Core().V1().Runners().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    sched.addRunnerToCache,
			UpdateFunc: sched.updateRunnerInCache,
			DeleteFunc: sched.deleteRunnerFromCache,
		},
	); err != nil {
		return err
	}
	handlers = append(handlers, handlerRegistration)

	logger := sched.logger
	buildEvtResHandler := func(at framework.ActionType, gvk framework.GVK, shortGVK string) cache.ResourceEventHandlerFuncs {
		funcs := cache.ResourceEventHandlerFuncs{}
		if at&framework.Add != 0 {
			evt := framework.ClusterEvent{Resource: gvk, ActionType: framework.Add, Label: fmt.Sprintf("%vAdd", shortGVK)}
			funcs.AddFunc = func(obj interface{}) {
				sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, evt, nil, obj, nil)
			}
		}
		if at&framework.Update != 0 {
			evt := framework.ClusterEvent{Resource: gvk, ActionType: framework.Update, Label: fmt.Sprintf("%vUpdate", shortGVK)}
			funcs.UpdateFunc = func(old, obj interface{}) {
				sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, evt, old, obj, nil)
			}
		}
		if at&framework.Delete != 0 {
			evt := framework.ClusterEvent{Resource: gvk, ActionType: framework.Delete, Label: fmt.Sprintf("%vDelete", shortGVK)}
			funcs.DeleteFunc = func(obj interface{}) {
				sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(logger, evt, obj, nil, nil)
			}
		}
		return funcs
	}

	for gvk, at := range gvkMap {
		switch gvk {
		case framework.Runner, framework.Region:
			// Do nothing.
		case framework.RegionSchedulingContext:
			//if utilfeature.DefaultFeatureGate.Enabled(features.DynamicResourceAllocation) {
			//	if handlerRegistration, err = informerFactory.Resource().V1alpha2().RegionSchedulingContexts().Informer().AddEventHandler(
			//		buildEvtResHandler(at, framework.RegionSchedulingContext, "RegionSchedulingContext"),
			//	); err != nil {
			//		return err
			//	}
			//	handlers = append(handlers, handlerRegistration)
			//}
		case framework.ResourceClaim:
			//if utilfeature.DefaultFeatureGate.Enabled(features.DynamicResourceAllocation) {
			//	if handlerRegistration, err = informerFactory.Resource().V1alpha2().ResourceClaims().Informer().AddEventHandler(
			//		buildEvtResHandler(at, framework.ResourceClaim, "ResourceClaim"),
			//	); err != nil {
			//		return err
			//	}
			//	handlers = append(handlers, handlerRegistration)
			//}
		case framework.ResourceClass:
			//if utilfeature.DefaultFeatureGate.Enabled(features.DynamicResourceAllocation) {
			//	if handlerRegistration, err = informerFactory.Resource().V1alpha2().ResourceClasses().Informer().AddEventHandler(
			//		buildEvtResHandler(at, framework.ResourceClass, "ResourceClass"),
			//	); err != nil {
			//		return err
			//	}
			//	handlers = append(handlers, handlerRegistration)
			//}
		case framework.ResourceClaimParameters:
			//if utilfeature.DefaultFeatureGate.Enabled(features.DynamicResourceAllocation) {
			//	if handlerRegistration, err = informerFactory.Resource().V1alpha2().ResourceClaimParameters().Informer().AddEventHandler(
			//		buildEvtResHandler(at, framework.ResourceClaimParameters, "ResourceClaimParameters"),
			//	); err != nil {
			//		return err
			//	}
			//	handlers = append(handlers, handlerRegistration)
			//}
		case framework.ResourceClassParameters:
			//if utilfeature.DefaultFeatureGate.Enabled(features.DynamicResourceAllocation) {
			//	if handlerRegistration, err = informerFactory.Resource().V1alpha2().ResourceClassParameters().Informer().AddEventHandler(
			//		buildEvtResHandler(at, framework.ResourceClassParameters, "ResourceClassParameters"),
			//	); err != nil {
			//		return err
			//	}
			//	handlers = append(handlers, handlerRegistration)
			//}
		default:
			// Tests may not instantiate dynInformerFactory.
			if dynInformerFactory == nil {
				continue
			}
			// GVK is expected to be at least 3-folded, separated by dots.
			// <kind in plural>.<version>.<group>
			// Valid examples:
			// - foos.v1.example.com
			// - bars.v1beta1.a.b.c
			// Invalid examples:
			// - foos.v1 (2 sections)
			// - foo.v1.example.com (the first section should be plural)
			if strings.Count(string(gvk), ".") < 2 {
				logger.Error(nil, "incorrect event registration", "gvk", gvk)
				continue
			}
			// Fall back to try dynamic informers.
			gvr, _ := schema.ParseResourceArg(string(gvk))
			dynInformer := dynInformerFactory.ForResource(*gvr).Informer()
			if handlerRegistration, err = dynInformer.AddEventHandler(
				buildEvtResHandler(at, gvk, strings.Title(gvr.Resource)),
			); err != nil {
				return err
			}
			handlers = append(handlers, handlerRegistration)
		}
	}
	sched.registeredHandlers = handlers
	return nil
}

func runnerSchedulingPropertiesChange(newRunner *corev1.Runner, oldRunner *corev1.Runner) []framework.ClusterEvent {
	var events []framework.ClusterEvent

	if runnerSpecUnschedulableChanged(newRunner, oldRunner) {
		events = append(events, queue.RunnerSpecUnschedulableChange)
	}
	if runnerAllocatableChanged(newRunner, oldRunner) {
		events = append(events, queue.RunnerAllocatableChange)
	}
	if runnerLabelsChanged(newRunner, oldRunner) {
		events = append(events, queue.RunnerLabelChange)
	}
	if runnerTaintsChanged(newRunner, oldRunner) {
		events = append(events, queue.RunnerTaintChange)
	}
	if runnerConditionsChanged(newRunner, oldRunner) {
		events = append(events, queue.RunnerConditionChange)
	}
	if runnerAnnotationsChanged(newRunner, oldRunner) {
		events = append(events, queue.RunnerAnnotationChange)
	}

	return events
}

func runnerAllocatableChanged(newRunner *corev1.Runner, oldRunner *corev1.Runner) bool {
	return !equality.Semantic.DeepEqual(oldRunner.Status.Allocatable, newRunner.Status.Allocatable)
}

func runnerLabelsChanged(newRunner *corev1.Runner, oldRunner *corev1.Runner) bool {
	return !equality.Semantic.DeepEqual(oldRunner.GetLabels(), newRunner.GetLabels())
}

func runnerTaintsChanged(newRunner *corev1.Runner, oldRunner *corev1.Runner) bool {
	return !equality.Semantic.DeepEqual(newRunner.Spec.Taints, oldRunner.Spec.Taints)
}

func runnerConditionsChanged(newRunner *corev1.Runner, oldRunner *corev1.Runner) bool {
	strip := func(conditions []corev1.RunnerCondition) map[corev1.RunnerConditionType]corev1.ConditionStatus {
		conditionStatuses := make(map[corev1.RunnerConditionType]corev1.ConditionStatus, len(conditions))
		for i := range conditions {
			conditionStatuses[conditions[i].Type] = conditions[i].Status
		}
		return conditionStatuses
	}
	return !equality.Semantic.DeepEqual(strip(oldRunner.Status.Conditions), strip(newRunner.Status.Conditions))
}

func runnerSpecUnschedulableChanged(newRunner *corev1.Runner, oldRunner *corev1.Runner) bool {
	return newRunner.Spec.Unschedulable != oldRunner.Spec.Unschedulable && !newRunner.Spec.Unschedulable
}

func runnerAnnotationsChanged(newRunner *corev1.Runner, oldRunner *corev1.Runner) bool {
	return !equality.Semantic.DeepEqual(oldRunner.GetAnnotations(), newRunner.GetAnnotations())
}

func preCheckForRunner(runnerInfo *framework.RunnerInfo) queue.PreEnqueueCheck {
	// Note: the following checks doesn't take preemption into considerations, in very rare
	// cases (e.g., runner resizing), "region" may still fail a check but preemption helps. We deliberately
	// chose to ignore those cases as unschedulable regions will be re-queued eventually.
	return func(region *corev1.Region) bool {
		admissionResults := AdmissionCheck(region, runnerInfo, false)
		if len(admissionResults) != 0 {
			return false
		}
		_, isUntolerated := corev1helpers.FindMatchingUntoleratedTaint(runnerInfo.Runner().Spec.Taints, region.Spec.Tolerations, func(t *corev1.Taint) bool {
			return t.Effect == corev1.TaintEffectNoSchedule
		})
		return !isUntolerated
	}
}

// AdmissionCheck calls the filtering logic of runnerresources/runnerport/runnerAffinity/runnername
// and returns the failure reasons. It's used in kubelet(pkg/kubelet/lifecycle/predicate.go) and scheduler.
// It returns the first failure if `includeAllFailures` is set to false; otherwise
// returns all failures.
func AdmissionCheck(region *corev1.Region, runnerInfo *framework.RunnerInfo, includeAllFailures bool) []AdmissionResult {
	var admissionResults []AdmissionResult
	insufficientResources := runnerresources.Fits(region, runnerInfo)
	if len(insufficientResources) != 0 {
		for i := range insufficientResources {
			admissionResults = append(admissionResults, AdmissionResult{InsufficientResource: &insufficientResources[i]})
		}
		if !includeAllFailures {
			return admissionResults
		}
	}

	if matches, _ := corev1runneraffinity.GetRequiredRunnerAffinity(region).Match(runnerInfo.Runner()); !matches {
		admissionResults = append(admissionResults, AdmissionResult{Name: runneraffinity.Name, Reason: runneraffinity.ErrReasonRegion})
		if !includeAllFailures {
			return admissionResults
		}
	}
	if !runnername.Fits(region, runnerInfo) {
		admissionResults = append(admissionResults, AdmissionResult{Name: runnername.Name, Reason: runnername.ErrReason})
		if !includeAllFailures {
			return admissionResults
		}
	}

	return admissionResults
}

// AdmissionResult describes the reason why Scheduler can't admit the region.
// If the reason is a resource fit one, then AdmissionResult.InsufficientResource includes the details.
type AdmissionResult struct {
	Name                 string
	Reason               string
	InsufficientResource *runnerresources.InsufficientResource
}
