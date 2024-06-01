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

// This file contains structures that implement scheduling queue types.
// Scheduling queues hold regions waiting to be scheduled. This file implements a
// priority queue which has two sub queues and a additional data structure,
// namely: activeQ, backoffQ and unschedulableRegions.
// - activeQ holds regions that are being considered for scheduling.
// - backoffQ holds regions that moved from unschedulableRegions and will move to
//   activeQ when their backoff periods complete.
// - unschedulableRegions holds regions that were already attempted for scheduling and
//   are currently determined to be unschedulable.

package queue

import (
	"container/list"
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	listersv1 "github.com/olive-io/olive/client-go/generated/listers/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/internal/heap"
	"github.com/olive-io/olive/mon/scheduler/metrics"
	"github.com/olive-io/olive/mon/scheduler/util"
)

const (
	// DefaultRegionMaxInUnschedulableRegionsDuration is the default value for the maximum
	// time a region can stay in unschedulableRegions. If a region stays in unschedulableRegions
	// for longer than this value, the region will be moved from unschedulableRegions to
	// backoffQ or activeQ. If this value is empty, the default value (5min)
	// will be used.
	DefaultRegionMaxInUnschedulableRegionsDuration time.Duration = 5 * time.Minute
	// Scheduling queue names
	activeQ              = "Active"
	backoffQ             = "Backoff"
	unschedulableRegions = "Unschedulable"

	preEnqueue = "PreEnqueue"
)

const (
	// DefaultRegionInitialBackoffDuration is the default value for the initial backoff duration
	// for unschedulable regions. To change the default regionInitialBackoffDurationSeconds used by the
	// scheduler, update the ComponentConfig value in defaults.go
	DefaultRegionInitialBackoffDuration time.Duration = 1 * time.Second
	// DefaultRegionMaxBackoffDuration is the default value for the max backoff duration
	// for unschedulable regions. To change the default regionMaxBackoffDurationSeconds used by the
	// scheduler, update the ComponentConfig value in defaults.go
	DefaultRegionMaxBackoffDuration time.Duration = 10 * time.Second
)

// PreEnqueueCheck is a function type. It's used to build functions that
// run against a Region and the caller can choose to enqueue or skip the Region
// by the checking result.
type PreEnqueueCheck func(region *corev1.Region) bool

// SchedulingQueue is an interface for a queue to store regions waiting to be scheduled.
// The interface follows a pattern similar to cache.FIFO and cache.Heap and
// makes it easy to use those data structures as a SchedulingQueue.
type SchedulingQueue interface {
	framework.RegionNominator
	Add(logger klog.Logger, region *corev1.Region) error
	// Activate moves the given regions to activeQ iff they're in unschedulableRegions or backoffQ.
	// The passed-in regions are originally compiled from plugins that want to activate Regions,
	// by injecting the regions through a reserved CycleState struct (RegionsToActivate).
	Activate(logger klog.Logger, regions map[string]*corev1.Region)
	// AddUnschedulableIfNotPresent adds an unschedulable region back to scheduling queue.
	// The regionSchedulingCycle represents the current scheduling cycle number which can be
	// returned by calling SchedulingCycle().
	AddUnschedulableIfNotPresent(logger klog.Logger, region *framework.QueuedRegionInfo, regionSchedulingCycle int64) error
	// SchedulingCycle returns the current number of scheduling cycle which is
	// cached by scheduling queue. Normally, incrementing this number whenever
	// a region is popped (e.g. called Pop()) is enough.
	SchedulingCycle() int64
	// Pop removes the head of the queue and returns it. It blocks if the
	// queue is empty and waits until a new item is added to the queue.
	Pop(logger klog.Logger) (*framework.QueuedRegionInfo, error)
	// Done must be called for region returned by Pop. This allows the queue to
	// keep track of which regions are currently being processed.
	Done(types.UID)
	Update(logger klog.Logger, oldRegion, newRegion *corev1.Region) error
	Delete(region *corev1.Region) error
	// TODO(sanposhiho): move all PreEnqueueCkeck to Requeue and delete it from this parameter eventually.
	// Some PreEnqueueCheck include event filtering logic based on some in-tree plugins
	// and it affect badly to other plugins.
	MoveAllToActiveOrBackoffQueue(logger klog.Logger, event framework.ClusterEvent, oldObj, newObj interface{}, preCheck PreEnqueueCheck)
	AssignedRegionAdded(logger klog.Logger, region *corev1.Region)
	AssignedRegionUpdated(logger klog.Logger, oldRegion, newRegion *corev1.Region)
	PendingRegions() ([]*corev1.Region, string)
	RegionsInActiveQ() []*corev1.Region
	// Close closes the SchedulingQueue so that the goroutine which is
	// waiting to pop items can exit gracefully.
	Close()
	// Run starts the goroutines managing the queue.
	Run(logger klog.Logger)
}

// NewSchedulingQueue initializes a priority queue as a new scheduling queue.
func NewSchedulingQueue(
	lessFn framework.LessFunc,
	informerFactory informers.SharedInformerFactory,
	opts ...Option) SchedulingQueue {
	return NewPriorityQueue(lessFn, informerFactory, opts...)
}

// NominatedRunnerName returns nominated node name of a Region.
func NominatedRunnerName(region *corev1.Region) string {
	//return region.Status.NominatedRunnerName
	return ""
}

// PriorityQueue implements a scheduling queue.
// The head of PriorityQueue is the highest priority pending region. This structure
// has two sub queues and a additional data structure, namely: activeQ,
// backoffQ and unschedulableRegions.
//   - activeQ holds regions that are being considered for scheduling.
//   - backoffQ holds regions that moved from unschedulableRegions and will move to
//     activeQ when their backoff periods complete.
//   - unschedulableRegions holds regions that were already attempted for scheduling and
//     are currently determined to be unschedulable.
type PriorityQueue struct {
	*nominator

	stop  chan struct{}
	clock clock.Clock

	// region initial backoff duration.
	regionInitialBackoffDuration time.Duration
	// region maximum backoff duration.
	regionMaxBackoffDuration time.Duration
	// the maximum time a region can stay in the unschedulableRegions.
	regionMaxInUnschedulableRegionsDuration time.Duration

	cond sync.Cond

	// inFlightRegions holds the UID of all regions which have been popped out for which Done
	// hasn't been called yet - in other words, all regions that are currently being
	// processed (being scheduled, in permit, or in the binding cycle).
	//
	// The values in the map are the entry of each region in the inFlightEvents list.
	// The value of that entry is the *corev1.Region at the time that scheduling of that
	// region started, which can be useful for logging or debugging.
	inFlightRegions map[types.UID]*list.Element

	// inFlightEvents holds the events received by the scheduling queue
	// (entry value is clusterEvent) together with in-flight regions (entry
	// value is *corev1.Region). Entries get added at the end while the mutex is
	// locked, so they get serialized.
	//
	// The region entries are added in Pop and used to track which events
	// occurred after the region scheduling attempt for that region started.
	// They get removed when the scheduling attempt is done, at which
	// point all events that occurred in the meantime are processed.
	//
	// After removal of a region, events at the start of the list are no
	// longer needed because all of the other in-flight regions started
	// later. Those events can be removed.
	inFlightEvents *list.List

	// activeQ is heap structure that scheduler actively looks at to find regions to
	// schedule. Head of heap is the highest priority region.
	activeQ *heap.Heap
	// regionBackoffQ is a heap ordered by backoff expiry. Regions which have completed backoff
	// are popped from this heap before the scheduler looks at activeQ
	regionBackoffQ *heap.Heap
	// unschedulableRegions holds regions that have been tried and determined unschedulable.
	unschedulableRegions *UnschedulableRegions
	// schedulingCycle represents sequence number of scheduling cycle and is incremented
	// when a region is popped.
	schedulingCycle int64
	// moveRequestCycle caches the sequence number of scheduling cycle when we
	// received a move request. Unschedulable regions in and before this scheduling
	// cycle will be put back to activeQueue if we were trying to schedule them
	// when we received move request.
	moveRequestCycle int64

	// preEnqueuePluginMap is keyed with profile name, valued with registered preEnqueue plugins.
	preEnqueuePluginMap map[string][]framework.PreEnqueuePlugin
	// queueingHintMap is keyed with profile name, valued with registered queueing hint functions.
	queueingHintMap QueueingHintMapPerProfile

	// closed indicates that the queue is closed.
	// It is mainly used to let Pop() exit its control loop while waiting for an item.
	closed bool

	metricsRecorder metrics.MetricAsyncRecorder
	// pluginMetricsSamplePercent is the percentage of plugin metrics to be sampled.
	pluginMetricsSamplePercent int

	// isSchedulingQueueHintEnabled indicates whether the feature gate for the scheduling queue is enabled.
	isSchedulingQueueHintEnabled bool
}

// QueueingHintFunction is the wrapper of QueueingHintFn that has PluginName.
type QueueingHintFunction struct {
	PluginName     string
	QueueingHintFn framework.QueueingHintFn
}

// clusterEvent has the event and involved objects.
type clusterEvent struct {
	event framework.ClusterEvent
	// oldObj is the object that involved this event.
	oldObj interface{}
	// newObj is the object that involved this event.
	newObj interface{}
}

type priorityQueueOptions struct {
	clock                                   clock.Clock
	regionInitialBackoffDuration            time.Duration
	regionMaxBackoffDuration                time.Duration
	regionMaxInUnschedulableRegionsDuration time.Duration
	regionLister                            listersv1.RegionLister
	metricsRecorder                         metrics.MetricAsyncRecorder
	pluginMetricsSamplePercent              int
	preEnqueuePluginMap                     map[string][]framework.PreEnqueuePlugin
	queueingHintMap                         QueueingHintMapPerProfile
}

// Option configures a PriorityQueue
type Option func(*priorityQueueOptions)

// WithClock sets clock for PriorityQueue, the default clock is clock.RealClock.
func WithClock(clock clock.Clock) Option {
	return func(o *priorityQueueOptions) {
		o.clock = clock
	}
}

// WithRegionInitialBackoffDuration sets region initial backoff duration for PriorityQueue.
func WithRegionInitialBackoffDuration(duration time.Duration) Option {
	return func(o *priorityQueueOptions) {
		o.regionInitialBackoffDuration = duration
	}
}

// WithRegionMaxBackoffDuration sets region max backoff duration for PriorityQueue.
func WithRegionMaxBackoffDuration(duration time.Duration) Option {
	return func(o *priorityQueueOptions) {
		o.regionMaxBackoffDuration = duration
	}
}

// WithRegionLister sets region lister for PriorityQueue.
func WithRegionLister(pl listersv1.RegionLister) Option {
	return func(o *priorityQueueOptions) {
		o.regionLister = pl
	}
}

// WithRegionMaxInUnschedulableRegionsDuration sets regionMaxInUnschedulableRegionsDuration for PriorityQueue.
func WithRegionMaxInUnschedulableRegionsDuration(duration time.Duration) Option {
	return func(o *priorityQueueOptions) {
		o.regionMaxInUnschedulableRegionsDuration = duration
	}
}

// QueueingHintMapPerProfile is keyed with profile name, valued with queueing hint map registered for the profile.
type QueueingHintMapPerProfile map[string]QueueingHintMap

// QueueingHintMap is keyed with ClusterEvent, valued with queueing hint functions registered for the event.
type QueueingHintMap map[framework.ClusterEvent][]*QueueingHintFunction

// WithQueueingHintMapPerProfile sets queueingHintMap for PriorityQueue.
func WithQueueingHintMapPerProfile(m QueueingHintMapPerProfile) Option {
	return func(o *priorityQueueOptions) {
		o.queueingHintMap = m
	}
}

// WithPreEnqueuePluginMap sets preEnqueuePluginMap for PriorityQueue.
func WithPreEnqueuePluginMap(m map[string][]framework.PreEnqueuePlugin) Option {
	return func(o *priorityQueueOptions) {
		o.preEnqueuePluginMap = m
	}
}

// WithMetricsRecorder sets metrics recorder.
func WithMetricsRecorder(recorder metrics.MetricAsyncRecorder) Option {
	return func(o *priorityQueueOptions) {
		o.metricsRecorder = recorder
	}
}

// WithPluginMetricsSamplePercent sets the percentage of plugin metrics to be sampled.
func WithPluginMetricsSamplePercent(percent int) Option {
	return func(o *priorityQueueOptions) {
		o.pluginMetricsSamplePercent = percent
	}
}

var defaultPriorityQueueOptions = priorityQueueOptions{
	clock:                                   clock.RealClock{},
	regionInitialBackoffDuration:            DefaultRegionInitialBackoffDuration,
	regionMaxBackoffDuration:                DefaultRegionMaxBackoffDuration,
	regionMaxInUnschedulableRegionsDuration: DefaultRegionMaxInUnschedulableRegionsDuration,
}

// Making sure that PriorityQueue implements SchedulingQueue.
var _ SchedulingQueue = &PriorityQueue{}

// newQueuedRegionInfoForLookup builds a QueuedRegionInfo object for a lookup in the queue.
func newQueuedRegionInfoForLookup(region *corev1.Region, plugins ...string) *framework.QueuedRegionInfo {
	// Since this is only used for a lookup in the queue, we only need to set the Region,
	// and so we avoid creating a full RegionInfo, which is expensive to instantiate frequently.
	return &framework.QueuedRegionInfo{
		RegionInfo:           &framework.RegionInfo{Region: region},
		UnschedulablePlugins: sets.New(plugins...),
	}
}

// NewPriorityQueue creates a PriorityQueue object.
func NewPriorityQueue(
	lessFn framework.LessFunc,
	informerFactory informers.SharedInformerFactory,
	opts ...Option,
) *PriorityQueue {
	options := defaultPriorityQueueOptions
	if options.regionLister == nil {
		options.regionLister = informerFactory.Core().V1().Regions().Lister()
	}
	for _, opt := range opts {
		opt(&options)
	}

	comp := func(regionInfo1, regionInfo2 interface{}) bool {
		pInfo1 := regionInfo1.(*framework.QueuedRegionInfo)
		pInfo2 := regionInfo2.(*framework.QueuedRegionInfo)
		return lessFn(pInfo1, pInfo2)
	}

	pq := &PriorityQueue{
		nominator:                               newRegionNominator(options.regionLister),
		clock:                                   options.clock,
		stop:                                    make(chan struct{}),
		regionInitialBackoffDuration:            options.regionInitialBackoffDuration,
		regionMaxBackoffDuration:                options.regionMaxBackoffDuration,
		regionMaxInUnschedulableRegionsDuration: options.regionMaxInUnschedulableRegionsDuration,
		activeQ:                                 heap.NewWithRecorder(regionInfoKeyFunc, comp, metrics.NewActiveRegionsRecorder()),
		unschedulableRegions:                    newUnschedulableRegions(metrics.NewUnschedulableRegionsRecorder(), metrics.NewGatedRegionsRecorder()),
		inFlightRegions:                         make(map[types.UID]*list.Element),
		inFlightEvents:                          list.New(),
		preEnqueuePluginMap:                     options.preEnqueuePluginMap,
		queueingHintMap:                         options.queueingHintMap,
		metricsRecorder:                         options.metricsRecorder,
		pluginMetricsSamplePercent:              options.pluginMetricsSamplePercent,
		moveRequestCycle:                        -1,
		isSchedulingQueueHintEnabled:            true,
	}
	pq.cond.L = &pq.lock
	pq.regionBackoffQ = heap.NewWithRecorder(regionInfoKeyFunc, pq.regionsCompareBackoffCompleted, metrics.NewBackoffRegionsRecorder())

	return pq
}

// Run starts the goroutine to pump from regionBackoffQ to activeQ
func (p *PriorityQueue) Run(logger klog.Logger) {
	go wait.Until(func() {
		p.flushBackoffQCompleted(logger)
	}, 1.0*time.Second, p.stop)
	go wait.Until(func() {
		p.flushUnschedulableRegionsLeftover(logger)
	}, 30*time.Second, p.stop)
}

// queueingStrategy indicates how the scheduling queue should enqueue the Region from unschedulable region pool.
type queueingStrategy int

const (
	// queueSkip indicates that the scheduling queue should skip requeuing the Region to activeQ/backoffQ.
	queueSkip queueingStrategy = iota
	// queueAfterBackoff indicates that the scheduling queue should requeue the Region after backoff is completed.
	queueAfterBackoff
	// queueImmediately indicates that the scheduling queue should skip backoff and requeue the Region immediately to activeQ.
	queueImmediately
)

// isEventOfInterest returns true if the event is of interest by some plugins.
func (p *PriorityQueue) isEventOfInterest(logger klog.Logger, event framework.ClusterEvent) bool {
	if event.IsWildCard() {
		return true
	}

	for _, hintMap := range p.queueingHintMap {
		for eventToMatch := range hintMap {
			if eventToMatch.Match(event) {
				// This event is interested by some plugins.
				return true
			}
		}
	}

	logger.V(6).Info("receive an event that isn't interested by any enabled plugins", "event", event)

	return false
}

// isRegionWorthRequeuing calls QueueingHintFn of only plugins registered in pInfo.unschedulablePlugins and pInfo.PendingPlugins.
//
// If any of pInfo.PendingPlugins return Queue,
// the scheduling queue is supposed to enqueue this Region to activeQ, skipping backoffQ.
// If any of pInfo.unschedulablePlugins return Queue,
// the scheduling queue is supposed to enqueue this Region to activeQ/backoffQ depending on the remaining backoff time of the Region.
// If all QueueingHintFns returns Skip, the scheduling queue enqueues the Region back to unschedulable Region pool
// because no plugin changes the scheduling result via the event.
func (p *PriorityQueue) isRegionWorthRequeuing(logger klog.Logger, pInfo *framework.QueuedRegionInfo, event framework.ClusterEvent, oldObj, newObj interface{}) queueingStrategy {
	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)
	if rejectorPlugins.Len() == 0 {
		logger.V(6).Info("Worth requeuing because no failed plugins", "region", klog.KObj(pInfo.Region))
		return queueAfterBackoff
	}

	if event.IsWildCard() {
		// If the wildcard event is special one as someone wants to force all Regions to move to activeQ/backoffQ.
		// We return queueAfterBackoff in this case, while resetting all blocked plugins.
		logger.V(6).Info("Worth requeuing because the event is wildcard", "region", klog.KObj(pInfo.Region))
		return queueAfterBackoff
	}

	hintMap, ok := p.queueingHintMap[pInfo.Region.Spec.SchedulerName]
	if !ok {
		// shouldn't reach here unless bug.
		logger.Error(nil, "No QueueingHintMap is registered for this profile", "profile", "region", klog.KObj(pInfo.Region))
		return queueAfterBackoff
	}

	region := pInfo.Region
	queueStrategy := queueSkip
	for eventToMatch, hintfns := range hintMap {
		if !eventToMatch.Match(event) {
			continue
		}

		for _, hintfn := range hintfns {
			if !rejectorPlugins.Has(hintfn.PluginName) {
				// skip if it's not hintfn from rejectorPlugins.
				continue
			}

			hint, err := hintfn.QueueingHintFn(logger, region, oldObj, newObj)
			if err != nil {
				// If the QueueingHintFn returned an error, we should treat the event as Queue so that we can prevent
				// the Region from being stuck in the unschedulable region pool.
				oldObjMeta, newObjMeta, asErr := util.As[klog.KMetadata](oldObj, newObj)
				if asErr != nil {
					logger.Error(err, "QueueingHintFn returns error", "event", event, "plugin", hintfn.PluginName, "region", klog.KObj(region))
				} else {
					logger.Error(err, "QueueingHintFn returns error", "event", event, "plugin", hintfn.PluginName, "region", klog.KObj(region), "oldObj", klog.KObj(oldObjMeta), "newObj", klog.KObj(newObjMeta))
				}
				hint = framework.Queue
			}
			if hint == framework.QueueSkip {
				continue
			}

			if pInfo.PendingPlugins.Has(hintfn.PluginName) {
				// interprets Queue from the Pending plugin as queueImmediately.
				// We can return immediately because queueImmediately is the highest priority.
				return queueImmediately
			}

			// interprets Queue from the unschedulable plugin as queueAfterBackoff.

			if pInfo.PendingPlugins.Len() == 0 {
				// We can return immediately because no Pending plugins, which only can make queueImmediately, registered in this Region,
				// and queueAfterBackoff is the second highest priority.
				return queueAfterBackoff
			}

			// We can't return immediately because there are some Pending plugins registered in this Region.
			// We need to check if those plugins return Queue or not and if they do, we return queueImmediately.
			queueStrategy = queueAfterBackoff
		}
	}

	return queueStrategy
}

// runPreEnqueuePlugins iterates PreEnqueue function in each registered PreEnqueuePlugin.
// It returns true if all PreEnqueue function run successfully; otherwise returns false
// upon the first failure.
// Note: we need to associate the failed plugin to `pInfo`, so that the region can be moved back
// to activeQ by related cluster event.
func (p *PriorityQueue) runPreEnqueuePlugins(ctx context.Context, pInfo *framework.QueuedRegionInfo) bool {
	logger := klog.FromContext(ctx)
	var s *framework.Status
	region := pInfo.Region
	startTime := p.clock.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(preEnqueue, s.Code().String(), region.Spec.SchedulerName).Observe(metrics.SinceInSeconds(startTime))
	}()

	shouldRecordMetric := rand.Intn(100) < p.pluginMetricsSamplePercent
	for _, pl := range p.preEnqueuePluginMap[region.Spec.SchedulerName] {
		s = p.runPreEnqueuePlugin(ctx, pl, region, shouldRecordMetric)
		if s.IsSuccess() {
			continue
		}
		pInfo.UnschedulablePlugins.Insert(pl.Name())
		metrics.UnschedulableReason(pl.Name(), region.Spec.SchedulerName).Inc()
		if s.Code() == framework.Error {
			logger.Error(s.AsError(), "Unexpected error running PreEnqueue plugin", "region", klog.KObj(region), "plugin", pl.Name())
		} else {
			logger.V(4).Info("Status after running PreEnqueue plugin", "region", klog.KObj(region), "plugin", pl.Name(), "status", s)
		}
		return false
	}
	return true
}

func (p *PriorityQueue) runPreEnqueuePlugin(ctx context.Context, pl framework.PreEnqueuePlugin, region *corev1.Region, shouldRecordMetric bool) *framework.Status {
	if !shouldRecordMetric {
		return pl.PreEnqueue(ctx, region)
	}
	startTime := p.clock.Now()
	s := pl.PreEnqueue(ctx, region)
	p.metricsRecorder.ObservePluginDurationAsync(preEnqueue, pl.Name(), s.Code().String(), p.clock.Since(startTime).Seconds())
	return s
}

// addToActiveQ tries to add region to active queue. It returns 2 parameters:
// 1. a boolean flag to indicate whether the region is added successfully.
// 2. an error for the caller to act on.
func (p *PriorityQueue) addToActiveQ(logger klog.Logger, pInfo *framework.QueuedRegionInfo) (bool, error) {
	pInfo.Gated = !p.runPreEnqueuePlugins(context.Background(), pInfo)
	if pInfo.Gated {
		// Add the Region to unschedulableRegions if it's not passing PreEnqueuePlugins.
		p.unschedulableRegions.addOrUpdate(pInfo)
		return false, nil
	}
	if pInfo.InitialAttemptTimestamp == nil {
		now := p.clock.Now()
		pInfo.InitialAttemptTimestamp = &now
	}
	if err := p.activeQ.Add(pInfo); err != nil {
		logger.Error(err, "Error adding region to the active queue", "region", klog.KObj(pInfo.Region))
		return false, err
	}
	return true, nil
}

// Add adds a region to the active queue. It should be called only when a new region
// is added so there is no chance the region is already in active/unschedulable/backoff queues
func (p *PriorityQueue) Add(logger klog.Logger, region *corev1.Region) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	pInfo := p.newQueuedRegionInfo(region)
	gated := pInfo.Gated
	if added, err := p.addToActiveQ(logger, pInfo); !added {
		return err
	}
	if p.unschedulableRegions.get(region) != nil {
		logger.Error(nil, "Error: region is already in the unschedulable queue", "region", klog.KObj(region))
		p.unschedulableRegions.delete(region, gated)
	}
	// Delete region from backoffQ if it is backing off
	if err := p.regionBackoffQ.Delete(pInfo); err == nil {
		logger.Error(nil, "Error: region is already in the regionBackoff queue", "region", klog.KObj(region))
	}
	logger.V(5).Info("Region moved to an internal scheduling queue", "region", klog.KObj(region), "event", RegionAdd, "queue", activeQ)
	metrics.SchedulerQueueIncomingRegions.WithLabelValues("active", RegionAdd).Inc()
	p.addNominatedRegionUnlocked(logger, pInfo.RegionInfo, nil)
	p.cond.Broadcast()

	return nil
}

// Activate moves the given regions to activeQ iff they're in unschedulableRegions or backoffQ.
func (p *PriorityQueue) Activate(logger klog.Logger, regions map[string]*corev1.Region) {
	p.lock.Lock()
	defer p.lock.Unlock()

	activated := false
	for _, region := range regions {
		if p.activate(logger, region) {
			activated = true
		}
	}

	if activated {
		p.cond.Broadcast()
	}
}

func (p *PriorityQueue) activate(logger klog.Logger, region *corev1.Region) bool {
	// Verify if the region is present in activeQ.
	if _, exists, _ := p.activeQ.Get(newQueuedRegionInfoForLookup(region)); exists {
		// No need to activate if it's already present in activeQ.
		return false
	}
	var pInfo *framework.QueuedRegionInfo
	// Verify if the region is present in unschedulableRegions or backoffQ.
	if pInfo = p.unschedulableRegions.get(region); pInfo == nil {
		// If the region doesn't belong to unschedulableRegions or backoffQ, don't activate it.
		if obj, exists, _ := p.regionBackoffQ.Get(newQueuedRegionInfoForLookup(region)); !exists {
			logger.Error(nil, "To-activate region does not exist in unschedulableRegions or backoffQ", "region", klog.KObj(region))
			return false
		} else {
			pInfo = obj.(*framework.QueuedRegionInfo)
		}
	}

	if pInfo == nil {
		// Redundant safe check. We shouldn't reach here.
		logger.Error(nil, "Internal error: cannot obtain pInfo")
		return false
	}

	gated := pInfo.Gated
	if added, _ := p.addToActiveQ(logger, pInfo); !added {
		return false
	}
	p.unschedulableRegions.delete(pInfo.Region, gated)
	p.regionBackoffQ.Delete(pInfo)
	metrics.SchedulerQueueIncomingRegions.WithLabelValues("active", ForceActivate).Inc()
	p.addNominatedRegionUnlocked(logger, pInfo.RegionInfo, nil)
	return true
}

// isRegionBackingoff returns true if a region is still waiting for its backoff timer.
// If this returns true, the region should not be re-tried.
func (p *PriorityQueue) isRegionBackingoff(regionInfo *framework.QueuedRegionInfo) bool {
	if regionInfo.Gated {
		return false
	}
	boTime := p.getBackoffTime(regionInfo)
	return boTime.After(p.clock.Now())
}

// SchedulingCycle returns current scheduling cycle.
func (p *PriorityQueue) SchedulingCycle() int64 {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.schedulingCycle
}

// determineSchedulingHintForInFlightRegion looks at the unschedulable plugins of the given Region
// and determines the scheduling hint for this Region while checking the events that happened during in-flight.
func (p *PriorityQueue) determineSchedulingHintForInFlightRegion(logger klog.Logger, pInfo *framework.QueuedRegionInfo) queueingStrategy {
	logger.V(5).Info("Checking events for in-flight region", "region", klog.KObj(pInfo.Region), "unschedulablePlugins", pInfo.UnschedulablePlugins, "inFlightEventsSize", p.inFlightEvents.Len(), "inFlightRegionsSize", len(p.inFlightRegions))

	// AddUnschedulableIfNotPresent is called with the Region at the end of scheduling or binding.
	// So, given pInfo should have been Pop()ed before,
	// we can assume pInfo must be recorded in inFlightRegions and thus inFlightEvents.
	inFlightRegion, ok := p.inFlightRegions[pInfo.Region.UID]
	if !ok {
		// This can happen while updating a region. In that case pInfo.UnschedulablePlugins should
		// be empty. If it is not, we may have a problem.
		if len(pInfo.UnschedulablePlugins) != 0 {
			logger.Error(nil, "In flight Region isn't found in the scheduling queue. If you see this error log, it's likely a bug in the scheduler.", "region", klog.KObj(pInfo.Region))
			return queueAfterBackoff
		}
		if p.inFlightEvents.Len() > len(p.inFlightRegions) {
			return queueAfterBackoff
		}
		return queueSkip
	}

	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)
	if len(rejectorPlugins) == 0 {
		// No failed plugins are associated with this Region.
		// Meaning something unusual (a temporal failure on kube-apiserver, etc) happened and this Region gets moved back to the queue.
		// In this case, we should retry scheduling it because this Region may not be retried until the next flush.
		return queueAfterBackoff
	}

	// check if there is an event that makes this Region schedulable based on pInfo.UnschedulablePlugins.
	queueingStrategy := queueSkip
	for event := inFlightRegion.Next(); event != nil; event = event.Next() {
		e, ok := event.Value.(*clusterEvent)
		if !ok {
			// Must be another in-flight Region (*corev1.Region). Can be ignored.
			continue
		}
		logger.V(5).Info("Checking event for in-flight region", "region", klog.KObj(pInfo.Region), "event", e.event.Label)

		switch p.isRegionWorthRequeuing(logger, pInfo, e.event, e.oldObj, e.newObj) {
		case queueSkip:
			continue
		case queueImmediately:
			// queueImmediately is the highest priority.
			// No need to go through the rest of the events.
			return queueImmediately
		case queueAfterBackoff:
			// replace schedulingHint with queueAfterBackoff
			queueingStrategy = queueAfterBackoff
			if pInfo.PendingPlugins.Len() == 0 {
				// We can return immediately because no Pending plugins, which only can make queueImmediately, registered in this Region,
				// and queueAfterBackoff is the second highest priority.
				return queueAfterBackoff
			}
		}
	}
	return queueingStrategy
}

// addUnschedulableIfNotPresentWithoutQueueingHint inserts a region that cannot be scheduled into
// the queue, unless it is already in the queue. Normally, PriorityQueue puts
// unschedulable regions in `unschedulableRegions`. But if there has been a recent move
// request, then the region is put in `regionBackoffQ`.
// TODO: This function is called only when p.isSchedulingQueueHintEnabled is false,
// and this will be removed after SchedulingQueueHint goes to stable and the feature gate is removed.
func (p *PriorityQueue) addUnschedulableWithoutQueueingHint(logger klog.Logger, pInfo *framework.QueuedRegionInfo, regionSchedulingCycle int64) error {
	region := pInfo.Region
	// Refresh the timestamp since the region is re-added.
	pInfo.Timestamp = p.clock.Now()

	// When the queueing hint is enabled, they are used differently.
	// But, we use all of them as UnschedulablePlugins when the queueing hint isn't enabled so that we don't break the old behaviour.
	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)

	// If a move request has been received, move it to the BackoffQ, otherwise move
	// it to unschedulableRegions.
	for plugin := range rejectorPlugins {
		metrics.UnschedulableReason(plugin, pInfo.Region.Spec.SchedulerName).Inc()
	}
	if p.moveRequestCycle >= regionSchedulingCycle || len(rejectorPlugins) == 0 {
		// Two cases to move a Region to the active/backoff queue:
		// - The Region is rejected by some plugins, but a move request is received after this Region's scheduling cycle is started.
		//   In this case, the received event may be make Region schedulable and we should retry scheduling it.
		// - No unschedulable plugins are associated with this Region,
		//   meaning something unusual (a temporal failure on kube-apiserver, etc) happened and this Region gets moved back to the queue.
		//   In this case, we should retry scheduling it because this Region may not be retried until the next flush.
		if err := p.regionBackoffQ.Add(pInfo); err != nil {
			return fmt.Errorf("error adding region %v to the backoff queue: %v", klog.KObj(region), err)
		}
		logger.V(5).Info("Region moved to an internal scheduling queue", "region", klog.KObj(region), "event", ScheduleAttemptFailure, "queue", backoffQ)
		metrics.SchedulerQueueIncomingRegions.WithLabelValues("backoff", ScheduleAttemptFailure).Inc()
	} else {
		p.unschedulableRegions.addOrUpdate(pInfo)
		logger.V(5).Info("Region moved to an internal scheduling queue", "region", klog.KObj(region), "event", ScheduleAttemptFailure, "queue", unschedulableRegions)
		metrics.SchedulerQueueIncomingRegions.WithLabelValues("unschedulable", ScheduleAttemptFailure).Inc()
	}

	p.addNominatedRegionUnlocked(logger, pInfo.RegionInfo, nil)
	return nil
}

// AddUnschedulableIfNotPresent inserts a region that cannot be scheduled into
// the queue, unless it is already in the queue. Normally, PriorityQueue puts
// unschedulable regions in `unschedulableRegions`. But if there has been a recent move
// request, then the region is put in `regionBackoffQ`.
func (p *PriorityQueue) AddUnschedulableIfNotPresent(logger klog.Logger, pInfo *framework.QueuedRegionInfo, regionSchedulingCycle int64) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	// In any case, this Region will be moved back to the queue and we should call Done.
	defer p.done(pInfo.Region.UID)

	region := pInfo.Region
	if p.unschedulableRegions.get(region) != nil {
		return fmt.Errorf("Region %v is already present in unschedulable queue", klog.KObj(region))
	}

	if _, exists, _ := p.activeQ.Get(pInfo); exists {
		return fmt.Errorf("Region %v is already present in the active queue", klog.KObj(region))
	}
	if _, exists, _ := p.regionBackoffQ.Get(pInfo); exists {
		return fmt.Errorf("Region %v is already present in the backoff queue", klog.KObj(region))
	}

	if !p.isSchedulingQueueHintEnabled {
		// fall back to the old behavior which doesn't depend on the queueing hint.
		return p.addUnschedulableWithoutQueueingHint(logger, pInfo, regionSchedulingCycle)
	}

	// Refresh the timestamp since the region is re-added.
	pInfo.Timestamp = p.clock.Now()

	// If a move request has been received, move it to the BackoffQ, otherwise move
	// it to unschedulableRegions.
	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)
	for plugin := range rejectorPlugins {
		metrics.UnschedulableReason(plugin, pInfo.Region.Spec.SchedulerName).Inc()
	}

	// We check whether this Region may change its scheduling result by any of events that happened during scheduling.
	schedulingHint := p.determineSchedulingHintForInFlightRegion(logger, pInfo)

	// In this case, we try to requeue this Region to activeQ/backoffQ.
	queue := p.requeueRegionViaQueueingHint(logger, pInfo, schedulingHint, ScheduleAttemptFailure)
	logger.V(3).Info("Region moved to an internal scheduling queue", "region", klog.KObj(region), "event", ScheduleAttemptFailure, "queue", queue, "schedulingCycle", regionSchedulingCycle, "hint", schedulingHint, "unschedulable plugins", rejectorPlugins)
	if queue == activeQ {
		// When the Region is moved to activeQ, need to let p.cond know so that the Region will be pop()ed out.
		p.cond.Broadcast()
	}

	p.addNominatedRegionUnlocked(logger, pInfo.RegionInfo, nil)
	return nil
}

// flushBackoffQCompleted Moves all regions from backoffQ which have completed backoff in to activeQ
func (p *PriorityQueue) flushBackoffQCompleted(logger klog.Logger) {
	p.lock.Lock()
	defer p.lock.Unlock()
	activated := false
	for {
		rawRegionInfo := p.regionBackoffQ.Peek()
		if rawRegionInfo == nil {
			break
		}
		pInfo := rawRegionInfo.(*framework.QueuedRegionInfo)
		region := pInfo.Region
		if p.isRegionBackingoff(pInfo) {
			break
		}
		_, err := p.regionBackoffQ.Pop()
		if err != nil {
			logger.Error(err, "Unable to pop region from backoff queue despite backoff completion", "region", klog.KObj(region))
			break
		}
		if added, _ := p.addToActiveQ(logger, pInfo); added {
			logger.V(5).Info("Region moved to an internal scheduling queue", "region", klog.KObj(region), "event", BackoffComplete, "queue", activeQ)
			metrics.SchedulerQueueIncomingRegions.WithLabelValues("active", BackoffComplete).Inc()
			activated = true
		}
	}

	if activated {
		p.cond.Broadcast()
	}
}

// flushUnschedulableRegionsLeftover moves regions which stay in unschedulableRegions
// longer than regionMaxInUnschedulableRegionsDuration to backoffQ or activeQ.
func (p *PriorityQueue) flushUnschedulableRegionsLeftover(logger klog.Logger) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var regionsToMove []*framework.QueuedRegionInfo
	currentTime := p.clock.Now()
	for _, pInfo := range p.unschedulableRegions.regionInfoMap {
		lastScheduleTime := pInfo.Timestamp
		if currentTime.Sub(lastScheduleTime) > p.regionMaxInUnschedulableRegionsDuration {
			regionsToMove = append(regionsToMove, pInfo)
		}
	}

	if len(regionsToMove) > 0 {
		p.moveRegionsToActiveOrBackoffQueue(logger, regionsToMove, UnschedulableTimeout, nil, nil)
	}
}

// Pop removes the head of the active queue and returns it. It blocks if the
// activeQ is empty and waits until a new item is added to the queue. It
// increments scheduling cycle when a region is popped.
func (p *PriorityQueue) Pop(logger klog.Logger) (*framework.QueuedRegionInfo, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	for p.activeQ.Len() == 0 {
		// When the queue is empty, invocation of Pop() is blocked until new item is enqueued.
		// When Close() is called, the p.closed is set and the condition is broadcast,
		// which causes this loop to continue and return from the Pop().
		if p.closed {
			logger.V(2).Info("Scheduling queue is closed")
			return nil, nil
		}
		p.cond.Wait()
	}
	obj, err := p.activeQ.Pop()
	if err != nil {
		return nil, err
	}
	pInfo := obj.(*framework.QueuedRegionInfo)
	pInfo.Attempts++
	p.schedulingCycle++
	// In flight, no concurrent events yet.
	if p.isSchedulingQueueHintEnabled {
		p.inFlightRegions[pInfo.Region.UID] = p.inFlightEvents.PushBack(pInfo.Region)
	}

	// Update metrics and reset the set of unschedulable plugins for the next attempt.
	for plugin := range pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins) {
		metrics.UnschedulableReason(plugin, pInfo.Region.Spec.SchedulerName).Dec()
	}
	pInfo.UnschedulablePlugins.Clear()
	pInfo.PendingPlugins.Clear()

	return pInfo, nil
}

// Done must be called for region returned by Pop. This allows the queue to
// keep track of which regions are currently being processed.
func (p *PriorityQueue) Done(region types.UID) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.done(region)
}

func (p *PriorityQueue) done(region types.UID) {
	if !p.isSchedulingQueueHintEnabled {
		// do nothing if schedulingQueueHint is disabled.
		// In that case, we don't have inFlightRegions and inFlightEvents.
		return
	}
	inFlightRegion, ok := p.inFlightRegions[region]
	if !ok {
		// This Region is already done()ed.
		return
	}
	delete(p.inFlightRegions, region)

	// Remove the region from the list.
	p.inFlightEvents.Remove(inFlightRegion)

	// Remove events which are only referred to by this Region
	// so that the inFlightEvents list doesn't grow infinitely.
	// If the region was at the head of the list, then all
	// events between it and the next region are no longer needed
	// and can be removed.
	for {
		e := p.inFlightEvents.Front()
		if e == nil {
			// Empty list.
			break
		}
		if _, ok := e.Value.(*clusterEvent); !ok {
			// A region, must stop pruning.
			break
		}
		p.inFlightEvents.Remove(e)
	}
}

// isRegionUpdated checks if the region is updated in a way that it may have become
// schedulable. It drops status of the region and compares it with old version,
// except for region.status.resourceClaimStatuses: changing that may have an
// effect on scheduling.
func isRegionUpdated(oldRegion, newRegion *corev1.Region) bool {
	strip := func(region *corev1.Region) *corev1.Region {
		p := region.DeepCopy()
		p.ResourceVersion = ""
		p.Generation = 0
		p.Status = corev1.RegionStatus{
			//ResourceClaimStatuses: region.Status.ResourceClaimStatuses,
		}
		p.ManagedFields = nil
		p.Finalizers = nil
		return p
	}
	return !reflect.DeepEqual(strip(oldRegion), strip(newRegion))
}

// Update updates a region in the active or backoff queue if present. Otherwise, it removes
// the item from the unschedulable queue if region is updated in a way that it may
// become schedulable and adds the updated one to the active queue.
// If region is not present in any of the queues, it is added to the active queue.
func (p *PriorityQueue) Update(logger klog.Logger, oldRegion, newRegion *corev1.Region) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if oldRegion != nil {
		oldRegionInfo := newQueuedRegionInfoForLookup(oldRegion)
		// If the region is already in the active queue, just update it there.
		if oldRegionInfo, exists, _ := p.activeQ.Get(oldRegionInfo); exists {
			pInfo := updateRegion(oldRegionInfo, newRegion)
			p.updateNominatedRegionUnlocked(logger, oldRegion, pInfo.RegionInfo)
			return p.activeQ.Update(pInfo)
		}

		// If the region is in the backoff queue, update it there.
		if oldRegionInfo, exists, _ := p.regionBackoffQ.Get(oldRegionInfo); exists {
			pInfo := updateRegion(oldRegionInfo, newRegion)
			p.updateNominatedRegionUnlocked(logger, oldRegion, pInfo.RegionInfo)
			return p.regionBackoffQ.Update(pInfo)
		}
	}

	// If the region is in the unschedulable queue, updating it may make it schedulable.
	if usRegionInfo := p.unschedulableRegions.get(newRegion); usRegionInfo != nil {
		pInfo := updateRegion(usRegionInfo, newRegion)
		p.updateNominatedRegionUnlocked(logger, oldRegion, pInfo.RegionInfo)
		if isRegionUpdated(oldRegion, newRegion) {
			gated := usRegionInfo.Gated
			if p.isRegionBackingoff(usRegionInfo) {
				if err := p.regionBackoffQ.Add(pInfo); err != nil {
					return err
				}
				p.unschedulableRegions.delete(usRegionInfo.Region, gated)
				logger.V(5).Info("Region moved to an internal scheduling queue", "region", klog.KObj(pInfo.Region), "event", RegionUpdate, "queue", backoffQ)
			} else {
				if added, err := p.addToActiveQ(logger, pInfo); !added {
					return err
				}
				p.unschedulableRegions.delete(usRegionInfo.Region, gated)
				logger.V(5).Info("Region moved to an internal scheduling queue", "region", klog.KObj(pInfo.Region), "event", BackoffComplete, "queue", activeQ)
				p.cond.Broadcast()
			}
		} else {
			// Region update didn't make it schedulable, keep it in the unschedulable queue.
			p.unschedulableRegions.addOrUpdate(pInfo)
		}

		return nil
	}
	// If region is not in any of the queues, we put it in the active queue.
	pInfo := p.newQueuedRegionInfo(newRegion)
	if added, err := p.addToActiveQ(logger, pInfo); !added {
		return err
	}
	p.addNominatedRegionUnlocked(logger, pInfo.RegionInfo, nil)
	logger.V(5).Info("Region moved to an internal scheduling queue", "region", klog.KObj(pInfo.Region), "event", RegionUpdate, "queue", activeQ)
	p.cond.Broadcast()
	return nil
}

// Delete deletes the item from either of the two queues. It assumes the region is
// only in one queue.
func (p *PriorityQueue) Delete(region *corev1.Region) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.deleteNominatedRegionIfExistsUnlocked(region)
	pInfo := newQueuedRegionInfoForLookup(region)
	if err := p.activeQ.Delete(pInfo); err != nil {
		// The item was probably not found in the activeQ.
		p.regionBackoffQ.Delete(pInfo)
		if pInfo = p.unschedulableRegions.get(region); pInfo != nil {
			p.unschedulableRegions.delete(region, pInfo.Gated)
		}
	}
	return nil
}

// AssignedRegionAdded is called when a bound region is added. Creation of this region
// may make pending regions with matching affinity terms schedulable.
func (p *PriorityQueue) AssignedRegionAdded(logger klog.Logger, region *corev1.Region) {
	p.lock.Lock()
	//p.moveRegionsToActiveOrBackoffQueue(logger, p.getUnschedulableRegionsWithMatchingAffinityTerm(logger, region), AssignedRegionAdd, nil, region)
	p.lock.Unlock()
}

// isRegionResourcesResizedDown returns true if a region CPU and/or memory resize request has been
// admitted by kubelet, is 'InProgress', and results in a net sizing down of updated resources.
// It returns false if either CPU or memory resource is net resized up, or if no resize is in progress.
func isRegionResourcesResizedDown(region *corev1.Region) bool {
	//if utilfeature.DefaultFeatureGate.Enabled(features.InPlaceRegionVerticalScaling) {
	//	// TODO(vinaykul,wangchen615,InPlaceRegionVerticalScaling): Fix this to determine when a
	//	// region is truly resized down (might need oldRegion if we cannot determine from Status alone)
	//	if region.Status.Resize == v1.RegionResizeStatusInProgress {
	//		return true
	//	}
	//}
	return false
}

// AssignedRegionUpdated is called when a bound region is updated. Change of labels
// may make pending regions with matching affinity terms schedulable.
func (p *PriorityQueue) AssignedRegionUpdated(logger klog.Logger, oldRegion, newRegion *corev1.Region) {
	p.lock.Lock()
	if isRegionResourcesResizedDown(newRegion) {
		p.moveAllToActiveOrBackoffQueue(logger, AssignedRegionUpdate, oldRegion, newRegion, nil)
	} else {
		//p.moveRegionsToActiveOrBackoffQueue(logger, p.getUnschedulableRegionsWithMatchingAffinityTerm(logger, newRegion), AssignedRegionUpdate, oldRegion, newRegion)
	}
	p.lock.Unlock()
}

// NOTE: this function assumes a lock has been acquired in the caller.
// moveAllToActiveOrBackoffQueue moves all regions from unschedulableRegions to activeQ or backoffQ.
// This function adds all regions and then signals the condition variable to ensure that
// if Pop() is waiting for an item, it receives the signal after all the regions are in the
// queue and the head is the highest priority region.
func (p *PriorityQueue) moveAllToActiveOrBackoffQueue(logger klog.Logger, event framework.ClusterEvent, oldObj, newObj interface{}, preCheck PreEnqueueCheck) {
	if !p.isEventOfInterest(logger, event) {
		// No plugin is interested in this event.
		// Return early before iterating all regions in unschedulableRegions for preCheck.
		return
	}

	unschedulableRegions := make([]*framework.QueuedRegionInfo, 0, len(p.unschedulableRegions.regionInfoMap))
	for _, pInfo := range p.unschedulableRegions.regionInfoMap {
		if preCheck == nil || preCheck(pInfo.Region) {
			unschedulableRegions = append(unschedulableRegions, pInfo)
		}
	}
	p.moveRegionsToActiveOrBackoffQueue(logger, unschedulableRegions, event, oldObj, newObj)
}

// MoveAllToActiveOrBackoffQueue moves all regions from unschedulableRegions to activeQ or backoffQ.
// This function adds all regions and then signals the condition variable to ensure that
// if Pop() is waiting for an item, it receives the signal after all the regions are in the
// queue and the head is the highest priority region.
func (p *PriorityQueue) MoveAllToActiveOrBackoffQueue(logger klog.Logger, event framework.ClusterEvent, oldObj, newObj interface{}, preCheck PreEnqueueCheck) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.moveAllToActiveOrBackoffQueue(logger, event, oldObj, newObj, preCheck)
}

// requeueRegionViaQueueingHint tries to requeue Region to activeQ, backoffQ or unschedulable region pool based on schedulingHint.
// It returns the queue name Region goes.
//
// NOTE: this function assumes lock has been acquired in caller
func (p *PriorityQueue) requeueRegionViaQueueingHint(logger klog.Logger, pInfo *framework.QueuedRegionInfo, strategy queueingStrategy, event string) string {
	if strategy == queueSkip {
		p.unschedulableRegions.addOrUpdate(pInfo)
		metrics.SchedulerQueueIncomingRegions.WithLabelValues("unschedulable", event).Inc()
		return unschedulableRegions
	}

	region := pInfo.Region
	if strategy == queueAfterBackoff && p.isRegionBackingoff(pInfo) {
		if err := p.regionBackoffQ.Add(pInfo); err != nil {
			logger.Error(err, "Error adding region to the backoff queue, queue this Region to unschedulable region pool", "region", klog.KObj(region))
			p.unschedulableRegions.addOrUpdate(pInfo)
			return unschedulableRegions
		}

		metrics.SchedulerQueueIncomingRegions.WithLabelValues("backoff", event).Inc()
		return backoffQ
	}

	// Reach here if schedulingHint is QueueImmediately, or schedulingHint is Queue but the region is not backing off.

	added, err := p.addToActiveQ(logger, pInfo)
	if err != nil {
		logger.Error(err, "Error adding region to the active queue, queue this Region to unschedulable region pool", "region", klog.KObj(region))
	}
	if added {
		metrics.SchedulerQueueIncomingRegions.WithLabelValues("active", event).Inc()
		return activeQ
	}
	if pInfo.Gated {
		// In case the region is gated, the Region is pushed back to unschedulable Regions pool in addToActiveQ.
		return unschedulableRegions
	}

	p.unschedulableRegions.addOrUpdate(pInfo)
	metrics.SchedulerQueueIncomingRegions.WithLabelValues("unschedulable", ScheduleAttemptFailure).Inc()
	return unschedulableRegions
}

// NOTE: this function assumes lock has been acquired in caller
func (p *PriorityQueue) moveRegionsToActiveOrBackoffQueue(logger klog.Logger, regionInfoList []*framework.QueuedRegionInfo, event framework.ClusterEvent, oldObj, newObj interface{}) {
	if !p.isEventOfInterest(logger, event) {
		// No plugin is interested in this event.
		return
	}

	activated := false
	for _, pInfo := range regionInfoList {
		schedulingHint := p.isRegionWorthRequeuing(logger, pInfo, event, oldObj, newObj)
		if schedulingHint == queueSkip {
			// QueueingHintFn determined that this Region isn't worth putting to activeQ or backoffQ by this event.
			logger.V(5).Info("Event is not making region schedulable", "region", klog.KObj(pInfo.Region), "event", event.Label)
			continue
		}

		p.unschedulableRegions.delete(pInfo.Region, pInfo.Gated)
		queue := p.requeueRegionViaQueueingHint(logger, pInfo, schedulingHint, event.Label)
		logger.V(4).Info("Region moved to an internal scheduling queue", "region", klog.KObj(pInfo.Region), "event", event.Label, "queue", queue, "hint", schedulingHint)
		if queue == activeQ {
			activated = true
		}
	}

	p.moveRequestCycle = p.schedulingCycle

	if p.isSchedulingQueueHintEnabled && len(p.inFlightRegions) != 0 {
		logger.V(5).Info("Event received while regions are in flight", "event", event.Label, "numRegions", len(p.inFlightRegions))
		// AddUnschedulableIfNotPresent might get called for in-flight Regions later, and in
		// AddUnschedulableIfNotPresent we need to know whether events were
		// observed while scheduling them.
		p.inFlightEvents.PushBack(&clusterEvent{
			event:  event,
			oldObj: oldObj,
			newObj: newObj,
		})
	}

	if activated {
		p.cond.Broadcast()
	}
}

// getUnschedulableRegionsWithMatchingAffinityTerm returns unschedulable regions which have
// any affinity term that matches "region".
// NOTE: this function assumes lock has been acquired in caller.
//func (p *PriorityQueue) getUnschedulableRegionsWithMatchingAffinityTerm(logger klog.Logger, region *corev1.Region) []*framework.QueuedRegionInfo {
//	nsLabels := interregionaffinity.GetNamespaceLabelsSnapshot(logger, region.Namespace, p.nsLister)
//
//	var regionsToMove []*framework.QueuedRegionInfo
//	for _, pInfo := range p.unschedulableRegions.regionInfoMap {
//		for _, term := range pInfo.RequiredAffinityTerms {
//			if term.Matches(region, nsLabels) {
//				regionsToMove = append(regionsToMove, pInfo)
//				break
//			}
//		}
//
//	}
//	return regionsToMove
//}

// RegionsInActiveQ returns all the Regions in the activeQ.
// This function is only used in tests.
func (p *PriorityQueue) RegionsInActiveQ() []*corev1.Region {
	p.lock.RLock()
	defer p.lock.RUnlock()
	var result []*corev1.Region
	for _, pInfo := range p.activeQ.List() {
		result = append(result, pInfo.(*framework.QueuedRegionInfo).Region)
	}
	return result
}

var pendingRegionsSummary = "activeQ:%v; backoffQ:%v; unschedulableRegions:%v"

// PendingRegions returns all the pending regions in the queue; accompanied by a debugging string
// recording showing the number of regions in each queue respectively.
// This function is used for debugging purposes in the scheduler cache dumper and comparer.
func (p *PriorityQueue) PendingRegions() ([]*corev1.Region, string) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	var result []*corev1.Region
	for _, pInfo := range p.activeQ.List() {
		result = append(result, pInfo.(*framework.QueuedRegionInfo).Region)
	}
	for _, pInfo := range p.regionBackoffQ.List() {
		result = append(result, pInfo.(*framework.QueuedRegionInfo).Region)
	}
	for _, pInfo := range p.unschedulableRegions.regionInfoMap {
		result = append(result, pInfo.Region)
	}
	return result, fmt.Sprintf(pendingRegionsSummary, p.activeQ.Len(), p.regionBackoffQ.Len(), len(p.unschedulableRegions.regionInfoMap))
}

// Close closes the priority queue.
func (p *PriorityQueue) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()
	close(p.stop)
	p.closed = true
	p.cond.Broadcast()
}

// DeleteNominatedRegionIfExists deletes <region> from nominatedRegions.
func (npm *nominator) DeleteNominatedRegionIfExists(region *corev1.Region) {
	npm.lock.Lock()
	npm.deleteNominatedRegionIfExistsUnlocked(region)
	npm.lock.Unlock()
}

func (npm *nominator) deleteNominatedRegionIfExistsUnlocked(region *corev1.Region) {
	npm.delete(region)
}

// AddNominatedRegion adds a region to the nominated regions of the given node.
// This is called during the preemption process after a node is nominated to run
// the region. We update the structure before sending a request to update the region
// object to avoid races with the following scheduling cycles.
func (npm *nominator) AddNominatedRegion(logger klog.Logger, pi *framework.RegionInfo, nominatingInfo *framework.NominatingInfo) {
	npm.lock.Lock()
	npm.addNominatedRegionUnlocked(logger, pi, nominatingInfo)
	npm.lock.Unlock()
}

// NominatedRegionsForRunner returns a copy of regions that are nominated to run on the given node,
// but they are waiting for other regions to be removed from the node.
func (npm *nominator) NominatedRegionsForRunner(nodeName string) []*framework.RegionInfo {
	npm.lock.RLock()
	defer npm.lock.RUnlock()
	// Make a copy of the nominated Regions so the caller can mutate safely.
	regions := make([]*framework.RegionInfo, len(npm.nominatedRegions[nodeName]))
	for i := 0; i < len(regions); i++ {
		regions[i] = npm.nominatedRegions[nodeName][i].DeepCopy()
	}
	return regions
}

func (p *PriorityQueue) regionsCompareBackoffCompleted(regionInfo1, regionInfo2 interface{}) bool {
	pInfo1 := regionInfo1.(*framework.QueuedRegionInfo)
	pInfo2 := regionInfo2.(*framework.QueuedRegionInfo)
	bo1 := p.getBackoffTime(pInfo1)
	bo2 := p.getBackoffTime(pInfo2)
	return bo1.Before(bo2)
}

// newQueuedRegionInfo builds a QueuedRegionInfo object.
func (p *PriorityQueue) newQueuedRegionInfo(region *corev1.Region, plugins ...string) *framework.QueuedRegionInfo {
	now := p.clock.Now()
	// ignore this err since apiserver doesn't properly validate affinity terms
	// and we can't fix the validation for backwards compatibility.
	regionInfo, _ := framework.NewRegionInfo(region)
	return &framework.QueuedRegionInfo{
		RegionInfo:              regionInfo,
		Timestamp:               now,
		InitialAttemptTimestamp: nil,
		UnschedulablePlugins:    sets.New(plugins...),
	}
}

// getBackoffTime returns the time that regionInfo completes backoff
func (p *PriorityQueue) getBackoffTime(regionInfo *framework.QueuedRegionInfo) time.Time {
	duration := p.calculateBackoffDuration(regionInfo)
	backoffTime := regionInfo.Timestamp.Add(duration)
	return backoffTime
}

// calculateBackoffDuration is a helper function for calculating the backoffDuration
// based on the number of attempts the region has made.
func (p *PriorityQueue) calculateBackoffDuration(regionInfo *framework.QueuedRegionInfo) time.Duration {
	duration := p.regionInitialBackoffDuration
	for i := 1; i < regionInfo.Attempts; i++ {
		// Use subtraction instead of addition or multiplication to avoid overflow.
		if duration > p.regionMaxBackoffDuration-duration {
			return p.regionMaxBackoffDuration
		}
		duration += duration
	}
	return duration
}

func updateRegion(oldRegionInfo interface{}, newRegion *corev1.Region) *framework.QueuedRegionInfo {
	pInfo := oldRegionInfo.(*framework.QueuedRegionInfo)
	pInfo.Update(newRegion)
	return pInfo
}

// UnschedulableRegions holds regions that cannot be scheduled. This data structure
// is used to implement unschedulableRegions.
type UnschedulableRegions struct {
	// regionInfoMap is a map key by a region's full-name and the value is a pointer to the QueuedRegionInfo.
	regionInfoMap map[string]*framework.QueuedRegionInfo
	keyFunc       func(*corev1.Region) string
	// unschedulableRecorder/gatedRecorder updates the counter when elements of an unschedulableRegionsMap
	// get added or removed, and it does nothing if it's nil.
	unschedulableRecorder, gatedRecorder metrics.MetricRecorder
}

// addOrUpdate adds a region to the unschedulable regionInfoMap.
func (u *UnschedulableRegions) addOrUpdate(pInfo *framework.QueuedRegionInfo) {
	regionID := u.keyFunc(pInfo.Region)
	if _, exists := u.regionInfoMap[regionID]; !exists {
		if pInfo.Gated && u.gatedRecorder != nil {
			u.gatedRecorder.Inc()
		} else if !pInfo.Gated && u.unschedulableRecorder != nil {
			u.unschedulableRecorder.Inc()
		}
	}
	u.regionInfoMap[regionID] = pInfo
}

// delete deletes a region from the unschedulable regionInfoMap.
// The `gated` parameter is used to figure out which metric should be decreased.
func (u *UnschedulableRegions) delete(region *corev1.Region, gated bool) {
	regionID := u.keyFunc(region)
	if _, exists := u.regionInfoMap[regionID]; exists {
		if gated && u.gatedRecorder != nil {
			u.gatedRecorder.Dec()
		} else if !gated && u.unschedulableRecorder != nil {
			u.unschedulableRecorder.Dec()
		}
	}
	delete(u.regionInfoMap, regionID)
}

// get returns the QueuedRegionInfo if a region with the same key as the key of the given "region"
// is found in the map. It returns nil otherwise.
func (u *UnschedulableRegions) get(region *corev1.Region) *framework.QueuedRegionInfo {
	regionKey := u.keyFunc(region)
	if pInfo, exists := u.regionInfoMap[regionKey]; exists {
		return pInfo
	}
	return nil
}

// clear removes all the entries from the unschedulable regionInfoMap.
func (u *UnschedulableRegions) clear() {
	u.regionInfoMap = make(map[string]*framework.QueuedRegionInfo)
	if u.unschedulableRecorder != nil {
		u.unschedulableRecorder.Clear()
	}
	if u.gatedRecorder != nil {
		u.gatedRecorder.Clear()
	}
}

// newUnschedulableRegions initializes a new object of UnschedulableRegions.
func newUnschedulableRegions(unschedulableRecorder, gatedRecorder metrics.MetricRecorder) *UnschedulableRegions {
	return &UnschedulableRegions{
		regionInfoMap:         make(map[string]*framework.QueuedRegionInfo),
		keyFunc:               util.GetRegionFullName,
		unschedulableRecorder: unschedulableRecorder,
		gatedRecorder:         gatedRecorder,
	}
}

// nominator is a structure that stores regions nominated to run on nodes.
// It exists because nominatedRunnerName of region objects stored in the structure
// may be different than what scheduler has here. We should be able to find regions
// by their UID and update/delete them.
type nominator struct {
	// regionLister is used to verify if the given region is alive.
	regionLister listersv1.RegionLister
	// nominatedRegions is a map keyed by a node name and the value is a list of
	// regions which are nominated to run on the node. These are regions which can be in
	// the activeQ or unschedulableRegions.
	nominatedRegions map[string][]*framework.RegionInfo
	// nominatedRegionToRunner is map keyed by a Region UID to the node name where it is
	// nominated.
	nominatedRegionToRunner map[types.UID]string

	lock sync.RWMutex
}

func (npm *nominator) addNominatedRegionUnlocked(logger klog.Logger, pi *framework.RegionInfo, nominatingInfo *framework.NominatingInfo) {
	// Always delete the region if it already exists, to ensure we never store more than
	// one instance of the region.
	npm.delete(pi.Region)

	var nodeName string
	if nominatingInfo.Mode() == framework.ModeOverride {
		nodeName = nominatingInfo.NominatedRunnerName
	} else if nominatingInfo.Mode() == framework.ModeNoop {
		//if pi.Region.Status.NominatedRunnerName == "" {
		//	return
		//}
		//nodeName = pi.Region.Status.NominatedRunnerName
	}

	if npm.regionLister != nil {
		// If the region was removed or if it was already scheduled, don't nominate it.
		updatedRegion, err := npm.regionLister.Get(pi.Region.Name)
		if err != nil {
			logger.V(4).Info("Region doesn't exist in regionLister, aborted adding it to the nominator", "region", klog.KObj(pi.Region))
			return
		}
		_ = updatedRegion
		//if updatedRegion.Spec.RunnerName != "" {
		//	logger.V(4).Info("Region is already scheduled to a node, aborted adding it to the nominator", "region", klog.KObj(pi.Region), "node", updatedRegion.Spec.RunnerName)
		//	return
		//}
	}

	npm.nominatedRegionToRunner[pi.Region.UID] = nodeName
	for _, npi := range npm.nominatedRegions[nodeName] {
		if npi.Region.UID == pi.Region.UID {
			logger.V(4).Info("Region already exists in the nominator", "region", klog.KObj(npi.Region))
			return
		}
	}
	npm.nominatedRegions[nodeName] = append(npm.nominatedRegions[nodeName], pi)
}

func (npm *nominator) delete(p *corev1.Region) {
	nnn, ok := npm.nominatedRegionToRunner[p.UID]
	if !ok {
		return
	}
	for i, np := range npm.nominatedRegions[nnn] {
		if np.Region.UID == p.UID {
			npm.nominatedRegions[nnn] = append(npm.nominatedRegions[nnn][:i], npm.nominatedRegions[nnn][i+1:]...)
			if len(npm.nominatedRegions[nnn]) == 0 {
				delete(npm.nominatedRegions, nnn)
			}
			break
		}
	}
	delete(npm.nominatedRegionToRunner, p.UID)
}

// UpdateNominatedRegion updates the <oldRegion> with <newRegion>.
func (npm *nominator) UpdateNominatedRegion(logger klog.Logger, oldRegion *corev1.Region, newRegionInfo *framework.RegionInfo) {
	npm.lock.Lock()
	defer npm.lock.Unlock()
	npm.updateNominatedRegionUnlocked(logger, oldRegion, newRegionInfo)
}

func (npm *nominator) updateNominatedRegionUnlocked(logger klog.Logger, oldRegion *corev1.Region, newRegionInfo *framework.RegionInfo) {
	// In some cases, an Update event with no "NominatedRunner" present is received right
	// after a node("NominatedRunner") is reserved for this region in memory.
	// In this case, we need to keep reserving the NominatedRunner when updating the region pointer.
	var nominatingInfo *framework.NominatingInfo
	// We won't fall into below `if` block if the Update event represents:
	// (1) NominatedRunner info is added
	// (2) NominatedRunner info is updated
	// (3) NominatedRunner info is removed
	if NominatedRunnerName(oldRegion) == "" && NominatedRunnerName(newRegionInfo.Region) == "" {
		if nnn, ok := npm.nominatedRegionToRunner[oldRegion.UID]; ok {
			// This is the only case we should continue reserving the NominatedRunner
			nominatingInfo = &framework.NominatingInfo{
				NominatingMode:      framework.ModeOverride,
				NominatedRunnerName: nnn,
			}
		}
	}
	// We update irrespective of the nominatedRunnerName changed or not, to ensure
	// that region pointer is updated.
	npm.delete(oldRegion)
	npm.addNominatedRegionUnlocked(logger, newRegionInfo, nominatingInfo)
}

// NewRegionNominator creates a nominator as a backing of framework.RegionNominator.
// A regionLister is passed in so as to check if the region exists
// before adding its nominatedRunner info.
func NewRegionNominator(regionLister listersv1.RegionLister) framework.RegionNominator {
	return newRegionNominator(regionLister)
}

func newRegionNominator(regionLister listersv1.RegionLister) *nominator {
	return &nominator{
		regionLister:            regionLister,
		nominatedRegions:        make(map[string][]*framework.RegionInfo),
		nominatedRegionToRunner: make(map[types.UID]string),
	}
}

func regionInfoKeyFunc(obj interface{}) (string, error) {
	return cache.MetaNamespaceKeyFunc(obj.(*framework.QueuedRegionInfo).Region)
}
