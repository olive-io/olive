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
// Scheduling queues hold definitions waiting to be scheduled. This file implements a
// priority queue which has two sub queues and a additional data structure,
// namely: activeQ, backoffQ and unschedulableDefinitions.
// - activeQ holds definitions that are being considered for scheduling.
// - backoffQ holds definitions that moved from unschedulableDefinitions and will move to
//   activeQ when their backoff periods complete.
// - unschedulableDefinitions holds definitions that were already attempted for scheduling and
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
	informers "github.com/olive-io/olive/client/generated/informers/externalversions"
	listersv1 "github.com/olive-io/olive/client/generated/listers/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/interdefinitionaffinity"
	"github.com/olive-io/olive/mon/scheduler/internal/heap"
	"github.com/olive-io/olive/mon/scheduler/metrics"
	"github.com/olive-io/olive/mon/scheduler/util"
)

const (
	// DefaultDefinitionMaxInUnschedulableDefinitionsDuration is the default value for the maximum
	// time a definition can stay in unschedulableDefinitions. If a definition stays in unschedulableDefinitions
	// for longer than this value, the definition will be moved from unschedulableDefinitions to
	// backoffQ or activeQ. If this value is empty, the default value (5min)
	// will be used.
	DefaultDefinitionMaxInUnschedulableDefinitionsDuration time.Duration = 5 * time.Minute
	// Scheduling queue names
	activeQ                  = "Active"
	backoffQ                 = "Backoff"
	unschedulableDefinitions = "Unschedulable"

	preEnqueue = "PreEnqueue"
)

const (
	// DefaultDefinitionInitialBackoffDuration is the default value for the initial backoff duration
	// for unschedulable definitions. To change the default definitionInitialBackoffDurationSeconds used by the
	// scheduler, update the ComponentConfig value in defaults.go
	DefaultDefinitionInitialBackoffDuration time.Duration = 1 * time.Second
	// DefaultDefinitionMaxBackoffDuration is the default value for the max backoff duration
	// for unschedulable definitions. To change the default definitionMaxBackoffDurationSeconds used by the
	// scheduler, update the ComponentConfig value in defaults.go
	DefaultDefinitionMaxBackoffDuration time.Duration = 10 * time.Second
)

// PreEnqueueCheck is a function type. It's used to build functions that
// run against a Definition and the caller can choose to enqueue or skip the Definition
// by the checking result.
type PreEnqueueCheck func(definition *corev1.Definition) bool

// SchedulingQueue is an interface for a queue to store definitions waiting to be scheduled.
// The interface follows a pattern similar to cache.FIFO and cache.Heap and
// makes it easy to use those data structures as a SchedulingQueue.
type SchedulingQueue interface {
	framework.DefinitionNominator
	Add(logger klog.Logger, definition *corev1.Definition) error
	// Activate moves the given definitions to activeQ iff they're in unschedulableDefinitions or backoffQ.
	// The passed-in definitions are originally compiled from plugins that want to activate Definitions,
	// by injecting the definitions through a reserved CycleState struct (DefinitionsToActivate).
	Activate(logger klog.Logger, definitions map[string]*corev1.Definition)
	// AddUnschedulableIfNotPresent adds an unschedulable definition back to scheduling queue.
	// The definitionSchedulingCycle represents the current scheduling cycle number which can be
	// returned by calling SchedulingCycle().
	AddUnschedulableIfNotPresent(logger klog.Logger, definition *framework.QueuedDefinitionInfo, definitionSchedulingCycle int64) error
	// SchedulingCycle returns the current number of scheduling cycle which is
	// cached by scheduling queue. Normally, incrementing this number whenever
	// a definition is popped (e.g. called Pop()) is enough.
	SchedulingCycle() int64
	// Pop removes the head of the queue and returns it. It blocks if the
	// queue is empty and waits until a new item is added to the queue.
	Pop(logger klog.Logger) (*framework.QueuedDefinitionInfo, error)
	// Done must be called for definition returned by Pop. This allows the queue to
	// keep track of which definitions are currently being processed.
	Done(types.UID)
	Update(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition) error
	Delete(definition *corev1.Definition) error
	// TODO(sanposhiho): move all PreEnqueueCkeck to Requeue and delete it from this parameter eventually.
	// Some PreEnqueueCheck include event filtering logic based on some in-tree plugins
	// and it affect badly to other plugins.
	// See https://github.com/kubernetes/kubernetes/issues/110175
	MoveAllToActiveOrBackoffQueue(logger klog.Logger, event framework.ClusterEvent, oldObj, newObj interface{}, preCheck PreEnqueueCheck)
	AssignedDefinitionAdded(logger klog.Logger, definition *corev1.Definition)
	AssignedDefinitionUpdated(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition)
	PendingDefinitions() ([]*corev1.Definition, string)
	DefinitionsInActiveQ() []*corev1.Definition
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

// NominatedRunnerName returns nominated node name of a Definition.
func NominatedRunnerName(definition *corev1.Definition) string {
	//return definition.Status.NominatedRunnerName
	return ""
}

// PriorityQueue implements a scheduling queue.
// The head of PriorityQueue is the highest priority pending definition. This structure
// has two sub queues and a additional data structure, namely: activeQ,
// backoffQ and unschedulableDefinitions.
//   - activeQ holds definitions that are being considered for scheduling.
//   - backoffQ holds definitions that moved from unschedulableDefinitions and will move to
//     activeQ when their backoff periods complete.
//   - unschedulableDefinitions holds definitions that were already attempted for scheduling and
//     are currently determined to be unschedulable.
type PriorityQueue struct {
	*nominator

	stop  chan struct{}
	clock clock.Clock

	// definition initial backoff duration.
	definitionInitialBackoffDuration time.Duration
	// definition maximum backoff duration.
	definitionMaxBackoffDuration time.Duration
	// the maximum time a definition can stay in the unschedulableDefinitions.
	definitionMaxInUnschedulableDefinitionsDuration time.Duration

	cond sync.Cond

	// inFlightDefinitions holds the UID of all definitions which have been popped out for which Done
	// hasn't been called yet - in other words, all definitions that are currently being
	// processed (being scheduled, in permit, or in the binding cycle).
	//
	// The values in the map are the entry of each definition in the inFlightEvents list.
	// The value of that entry is the *corev1.Definition at the time that scheduling of that
	// definition started, which can be useful for logging or debugging.
	inFlightDefinitions map[types.UID]*list.Element

	// inFlightEvents holds the events received by the scheduling queue
	// (entry value is clusterEvent) together with in-flight definitions (entry
	// value is *corev1.Definition). Entries get added at the end while the mutex is
	// locked, so they get serialized.
	//
	// The definition entries are added in Pop and used to track which events
	// occurred after the definition scheduling attempt for that definition started.
	// They get removed when the scheduling attempt is done, at which
	// point all events that occurred in the meantime are processed.
	//
	// After removal of a definition, events at the start of the list are no
	// longer needed because all of the other in-flight definitions started
	// later. Those events can be removed.
	inFlightEvents *list.List

	// activeQ is heap structure that scheduler actively looks at to find definitions to
	// schedule. Head of heap is the highest priority definition.
	activeQ *heap.Heap
	// definitionBackoffQ is a heap ordered by backoff expiry. Definitions which have completed backoff
	// are popped from this heap before the scheduler looks at activeQ
	definitionBackoffQ *heap.Heap
	// unschedulableDefinitions holds definitions that have been tried and determined unschedulable.
	unschedulableDefinitions *UnschedulableDefinitions
	// schedulingCycle represents sequence number of scheduling cycle and is incremented
	// when a definition is popped.
	schedulingCycle int64
	// moveRequestCycle caches the sequence number of scheduling cycle when we
	// received a move request. Unschedulable definitions in and before this scheduling
	// cycle will be put back to activeQueue if we were trying to schedule them
	// when we received move request.
	// TODO: this will be removed after SchedulingQueueHint goes to stable and the feature gate is removed.
	moveRequestCycle int64

	// preEnqueuePluginMap is keyed with profile name, valued with registered preEnqueue plugins.
	preEnqueuePluginMap map[string][]framework.PreEnqueuePlugin
	// queueingHintMap is keyed with profile name, valued with registered queueing hint functions.
	queueingHintMap QueueingHintMapPerProfile

	// closed indicates that the queue is closed.
	// It is mainly used to let Pop() exit its control loop while waiting for an item.
	closed bool

	nsLister listersv1.NamespaceLister

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
	clock                                           clock.Clock
	definitionInitialBackoffDuration                time.Duration
	definitionMaxBackoffDuration                    time.Duration
	definitionMaxInUnschedulableDefinitionsDuration time.Duration
	definitionLister                                listersv1.DefinitionLister
	metricsRecorder                                 metrics.MetricAsyncRecorder
	pluginMetricsSamplePercent                      int
	preEnqueuePluginMap                             map[string][]framework.PreEnqueuePlugin
	queueingHintMap                                 QueueingHintMapPerProfile
}

// Option configures a PriorityQueue
type Option func(*priorityQueueOptions)

// WithClock sets clock for PriorityQueue, the default clock is clock.RealClock.
func WithClock(clock clock.Clock) Option {
	return func(o *priorityQueueOptions) {
		o.clock = clock
	}
}

// WithDefinitionInitialBackoffDuration sets definition initial backoff duration for PriorityQueue.
func WithDefinitionInitialBackoffDuration(duration time.Duration) Option {
	return func(o *priorityQueueOptions) {
		o.definitionInitialBackoffDuration = duration
	}
}

// WithDefinitionMaxBackoffDuration sets definition max backoff duration for PriorityQueue.
func WithDefinitionMaxBackoffDuration(duration time.Duration) Option {
	return func(o *priorityQueueOptions) {
		o.definitionMaxBackoffDuration = duration
	}
}

// WithDefinitionLister sets definition lister for PriorityQueue.
func WithDefinitionLister(pl listersv1.DefinitionLister) Option {
	return func(o *priorityQueueOptions) {
		o.definitionLister = pl
	}
}

// WithDefinitionMaxInUnschedulableDefinitionsDuration sets definitionMaxInUnschedulableDefinitionsDuration for PriorityQueue.
func WithDefinitionMaxInUnschedulableDefinitionsDuration(duration time.Duration) Option {
	return func(o *priorityQueueOptions) {
		o.definitionMaxInUnschedulableDefinitionsDuration = duration
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
	clock:                            clock.RealClock{},
	definitionInitialBackoffDuration: DefaultDefinitionInitialBackoffDuration,
	definitionMaxBackoffDuration:     DefaultDefinitionMaxBackoffDuration,
	definitionMaxInUnschedulableDefinitionsDuration: DefaultDefinitionMaxInUnschedulableDefinitionsDuration,
}

// Making sure that PriorityQueue implements SchedulingQueue.
var _ SchedulingQueue = &PriorityQueue{}

// newQueuedDefinitionInfoForLookup builds a QueuedDefinitionInfo object for a lookup in the queue.
func newQueuedDefinitionInfoForLookup(definition *corev1.Definition, plugins ...string) *framework.QueuedDefinitionInfo {
	// Since this is only used for a lookup in the queue, we only need to set the Definition,
	// and so we avoid creating a full DefinitionInfo, which is expensive to instantiate frequently.
	return &framework.QueuedDefinitionInfo{
		DefinitionInfo:       &framework.DefinitionInfo{Definition: definition},
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
	if options.definitionLister == nil {
		options.definitionLister = informerFactory.Core().V1().Definitions().Lister()
	}
	for _, opt := range opts {
		opt(&options)
	}

	comp := func(definitionInfo1, definitionInfo2 interface{}) bool {
		pInfo1 := definitionInfo1.(*framework.QueuedDefinitionInfo)
		pInfo2 := definitionInfo2.(*framework.QueuedDefinitionInfo)
		return lessFn(pInfo1, pInfo2)
	}

	pq := &PriorityQueue{
		nominator:                        newDefinitionNominator(options.definitionLister),
		clock:                            options.clock,
		stop:                             make(chan struct{}),
		definitionInitialBackoffDuration: options.definitionInitialBackoffDuration,
		definitionMaxBackoffDuration:     options.definitionMaxBackoffDuration,
		definitionMaxInUnschedulableDefinitionsDuration: options.definitionMaxInUnschedulableDefinitionsDuration,
		activeQ:                    heap.NewWithRecorder(definitionInfoKeyFunc, comp, metrics.NewActiveDefinitionsRecorder()),
		unschedulableDefinitions:   newUnschedulableDefinitions(metrics.NewUnschedulableDefinitionsRecorder(), metrics.NewGatedDefinitionsRecorder()),
		inFlightDefinitions:        make(map[types.UID]*list.Element),
		inFlightEvents:             list.New(),
		preEnqueuePluginMap:        options.preEnqueuePluginMap,
		queueingHintMap:            options.queueingHintMap,
		metricsRecorder:            options.metricsRecorder,
		pluginMetricsSamplePercent: options.pluginMetricsSamplePercent,
		moveRequestCycle:           -1,
		//isSchedulingQueueHintEnabled: utilfeature.DefaultFeatureGate.Enabled(features.SchedulerQueueingHints),
	}
	pq.cond.L = &pq.lock
	pq.definitionBackoffQ = heap.NewWithRecorder(definitionInfoKeyFunc, pq.definitionsCompareBackoffCompleted, metrics.NewBackoffDefinitionsRecorder())
	pq.nsLister = informerFactory.Core().V1().Namespaces().Lister()

	return pq
}

// Run starts the goroutine to pump from definitionBackoffQ to activeQ
func (p *PriorityQueue) Run(logger klog.Logger) {
	go wait.Until(func() {
		p.flushBackoffQCompleted(logger)
	}, 1.0*time.Second, p.stop)
	go wait.Until(func() {
		p.flushUnschedulableDefinitionsLeftover(logger)
	}, 30*time.Second, p.stop)
}

// queueingStrategy indicates how the scheduling queue should enqueue the Definition from unschedulable definition pool.
type queueingStrategy int

const (
	// queueSkip indicates that the scheduling queue should skip requeuing the Definition to activeQ/backoffQ.
	queueSkip queueingStrategy = iota
	// queueAfterBackoff indicates that the scheduling queue should requeue the Definition after backoff is completed.
	queueAfterBackoff
	// queueImmediately indicates that the scheduling queue should skip backoff and requeue the Definition immediately to activeQ.
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

// isDefinitionWorthRequeuing calls QueueingHintFn of only plugins registered in pInfo.unschedulablePlugins and pInfo.PendingPlugins.
//
// If any of pInfo.PendingPlugins return Queue,
// the scheduling queue is supposed to enqueue this Definition to activeQ, skipping backoffQ.
// If any of pInfo.unschedulablePlugins return Queue,
// the scheduling queue is supposed to enqueue this Definition to activeQ/backoffQ depending on the remaining backoff time of the Definition.
// If all QueueingHintFns returns Skip, the scheduling queue enqueues the Definition back to unschedulable Definition pool
// because no plugin changes the scheduling result via the event.
func (p *PriorityQueue) isDefinitionWorthRequeuing(logger klog.Logger, pInfo *framework.QueuedDefinitionInfo, event framework.ClusterEvent, oldObj, newObj interface{}) queueingStrategy {
	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)
	if rejectorPlugins.Len() == 0 {
		logger.V(6).Info("Worth requeuing because no failed plugins", "definition", klog.KObj(pInfo.Definition))
		return queueAfterBackoff
	}

	if event.IsWildCard() {
		// If the wildcard event is special one as someone wants to force all Definitions to move to activeQ/backoffQ.
		// We return queueAfterBackoff in this case, while resetting all blocked plugins.
		logger.V(6).Info("Worth requeuing because the event is wildcard", "definition", klog.KObj(pInfo.Definition))
		return queueAfterBackoff
	}

	hintMap, ok := p.queueingHintMap[pInfo.Definition.Spec.SchedulerName]
	if !ok {
		// shouldn't reach here unless bug.
		logger.Error(nil, "No QueueingHintMap is registered for this profile", "profile", "definition", klog.KObj(pInfo.Definition))
		return queueAfterBackoff
	}

	definition := pInfo.Definition
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

			hint, err := hintfn.QueueingHintFn(logger, definition, oldObj, newObj)
			if err != nil {
				// If the QueueingHintFn returned an error, we should treat the event as Queue so that we can prevent
				// the Definition from being stuck in the unschedulable definition pool.
				oldObjMeta, newObjMeta, asErr := util.As[klog.KMetadata](oldObj, newObj)
				if asErr != nil {
					logger.Error(err, "QueueingHintFn returns error", "event", event, "plugin", hintfn.PluginName, "definition", klog.KObj(definition))
				} else {
					logger.Error(err, "QueueingHintFn returns error", "event", event, "plugin", hintfn.PluginName, "definition", klog.KObj(definition), "oldObj", klog.KObj(oldObjMeta), "newObj", klog.KObj(newObjMeta))
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
				// We can return immediately because no Pending plugins, which only can make queueImmediately, registered in this Definition,
				// and queueAfterBackoff is the second highest priority.
				return queueAfterBackoff
			}

			// We can't return immediately because there are some Pending plugins registered in this Definition.
			// We need to check if those plugins return Queue or not and if they do, we return queueImmediately.
			queueStrategy = queueAfterBackoff
		}
	}

	return queueStrategy
}

// runPreEnqueuePlugins iterates PreEnqueue function in each registered PreEnqueuePlugin.
// It returns true if all PreEnqueue function run successfully; otherwise returns false
// upon the first failure.
// Note: we need to associate the failed plugin to `pInfo`, so that the definition can be moved back
// to activeQ by related cluster event.
func (p *PriorityQueue) runPreEnqueuePlugins(ctx context.Context, pInfo *framework.QueuedDefinitionInfo) bool {
	logger := klog.FromContext(ctx)
	var s *framework.Status
	definition := pInfo.Definition
	startTime := p.clock.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(preEnqueue, s.Code().String(), definition.Spec.SchedulerName).Observe(metrics.SinceInSeconds(startTime))
	}()

	shouldRecordMetric := rand.Intn(100) < p.pluginMetricsSamplePercent
	for _, pl := range p.preEnqueuePluginMap[definition.Spec.SchedulerName] {
		s = p.runPreEnqueuePlugin(ctx, pl, definition, shouldRecordMetric)
		if s.IsSuccess() {
			continue
		}
		pInfo.UnschedulablePlugins.Insert(pl.Name())
		metrics.UnschedulableReason(pl.Name(), definition.Spec.SchedulerName).Inc()
		if s.Code() == framework.Error {
			logger.Error(s.AsError(), "Unexpected error running PreEnqueue plugin", "definition", klog.KObj(definition), "plugin", pl.Name())
		} else {
			logger.V(4).Info("Status after running PreEnqueue plugin", "definition", klog.KObj(definition), "plugin", pl.Name(), "status", s)
		}
		return false
	}
	return true
}

func (p *PriorityQueue) runPreEnqueuePlugin(ctx context.Context, pl framework.PreEnqueuePlugin, definition *corev1.Definition, shouldRecordMetric bool) *framework.Status {
	if !shouldRecordMetric {
		return pl.PreEnqueue(ctx, definition)
	}
	startTime := p.clock.Now()
	s := pl.PreEnqueue(ctx, definition)
	p.metricsRecorder.ObservePluginDurationAsync(preEnqueue, pl.Name(), s.Code().String(), p.clock.Since(startTime).Seconds())
	return s
}

// addToActiveQ tries to add definition to active queue. It returns 2 parameters:
// 1. a boolean flag to indicate whether the definition is added successfully.
// 2. an error for the caller to act on.
func (p *PriorityQueue) addToActiveQ(logger klog.Logger, pInfo *framework.QueuedDefinitionInfo) (bool, error) {
	pInfo.Gated = !p.runPreEnqueuePlugins(context.Background(), pInfo)
	if pInfo.Gated {
		// Add the Definition to unschedulableDefinitions if it's not passing PreEnqueuePlugins.
		p.unschedulableDefinitions.addOrUpdate(pInfo)
		return false, nil
	}
	if pInfo.InitialAttemptTimestamp == nil {
		now := p.clock.Now()
		pInfo.InitialAttemptTimestamp = &now
	}
	if err := p.activeQ.Add(pInfo); err != nil {
		logger.Error(err, "Error adding definition to the active queue", "definition", klog.KObj(pInfo.Definition))
		return false, err
	}
	return true, nil
}

// Add adds a definition to the active queue. It should be called only when a new definition
// is added so there is no chance the definition is already in active/unschedulable/backoff queues
func (p *PriorityQueue) Add(logger klog.Logger, definition *corev1.Definition) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	pInfo := p.newQueuedDefinitionInfo(definition)
	gated := pInfo.Gated
	if added, err := p.addToActiveQ(logger, pInfo); !added {
		return err
	}
	if p.unschedulableDefinitions.get(definition) != nil {
		logger.Error(nil, "Error: definition is already in the unschedulable queue", "definition", klog.KObj(definition))
		p.unschedulableDefinitions.delete(definition, gated)
	}
	// Delete definition from backoffQ if it is backing off
	if err := p.definitionBackoffQ.Delete(pInfo); err == nil {
		logger.Error(nil, "Error: definition is already in the definitionBackoff queue", "definition", klog.KObj(definition))
	}
	logger.V(5).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(definition), "event", DefinitionAdd, "queue", activeQ)
	metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("active", DefinitionAdd).Inc()
	p.addNominatedDefinitionUnlocked(logger, pInfo.DefinitionInfo, nil)
	p.cond.Broadcast()

	return nil
}

// Activate moves the given definitions to activeQ iff they're in unschedulableDefinitions or backoffQ.
func (p *PriorityQueue) Activate(logger klog.Logger, definitions map[string]*corev1.Definition) {
	p.lock.Lock()
	defer p.lock.Unlock()

	activated := false
	for _, definition := range definitions {
		if p.activate(logger, definition) {
			activated = true
		}
	}

	if activated {
		p.cond.Broadcast()
	}
}

func (p *PriorityQueue) activate(logger klog.Logger, definition *corev1.Definition) bool {
	// Verify if the definition is present in activeQ.
	if _, exists, _ := p.activeQ.Get(newQueuedDefinitionInfoForLookup(definition)); exists {
		// No need to activate if it's already present in activeQ.
		return false
	}
	var pInfo *framework.QueuedDefinitionInfo
	// Verify if the definition is present in unschedulableDefinitions or backoffQ.
	if pInfo = p.unschedulableDefinitions.get(definition); pInfo == nil {
		// If the definition doesn't belong to unschedulableDefinitions or backoffQ, don't activate it.
		if obj, exists, _ := p.definitionBackoffQ.Get(newQueuedDefinitionInfoForLookup(definition)); !exists {
			logger.Error(nil, "To-activate definition does not exist in unschedulableDefinitions or backoffQ", "definition", klog.KObj(definition))
			return false
		} else {
			pInfo = obj.(*framework.QueuedDefinitionInfo)
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
	p.unschedulableDefinitions.delete(pInfo.Definition, gated)
	p.definitionBackoffQ.Delete(pInfo)
	metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("active", ForceActivate).Inc()
	p.addNominatedDefinitionUnlocked(logger, pInfo.DefinitionInfo, nil)
	return true
}

// isDefinitionBackingoff returns true if a definition is still waiting for its backoff timer.
// If this returns true, the definition should not be re-tried.
func (p *PriorityQueue) isDefinitionBackingoff(definitionInfo *framework.QueuedDefinitionInfo) bool {
	if definitionInfo.Gated {
		return false
	}
	boTime := p.getBackoffTime(definitionInfo)
	return boTime.After(p.clock.Now())
}

// SchedulingCycle returns current scheduling cycle.
func (p *PriorityQueue) SchedulingCycle() int64 {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.schedulingCycle
}

// determineSchedulingHintForInFlightDefinition looks at the unschedulable plugins of the given Definition
// and determines the scheduling hint for this Definition while checking the events that happened during in-flight.
func (p *PriorityQueue) determineSchedulingHintForInFlightDefinition(logger klog.Logger, pInfo *framework.QueuedDefinitionInfo) queueingStrategy {
	logger.V(5).Info("Checking events for in-flight definition", "definition", klog.KObj(pInfo.Definition), "unschedulablePlugins", pInfo.UnschedulablePlugins, "inFlightEventsSize", p.inFlightEvents.Len(), "inFlightDefinitionsSize", len(p.inFlightDefinitions))

	// AddUnschedulableIfNotPresent is called with the Definition at the end of scheduling or binding.
	// So, given pInfo should have been Pop()ed before,
	// we can assume pInfo must be recorded in inFlightDefinitions and thus inFlightEvents.
	inFlightDefinition, ok := p.inFlightDefinitions[pInfo.Definition.UID]
	if !ok {
		// This can happen while updating a definition. In that case pInfo.UnschedulablePlugins should
		// be empty. If it is not, we may have a problem.
		if len(pInfo.UnschedulablePlugins) != 0 {
			logger.Error(nil, "In flight Definition isn't found in the scheduling queue. If you see this error log, it's likely a bug in the scheduler.", "definition", klog.KObj(pInfo.Definition))
			return queueAfterBackoff
		}
		if p.inFlightEvents.Len() > len(p.inFlightDefinitions) {
			return queueAfterBackoff
		}
		return queueSkip
	}

	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)
	if len(rejectorPlugins) == 0 {
		// No failed plugins are associated with this Definition.
		// Meaning something unusual (a temporal failure on kube-apiserver, etc) happened and this Definition gets moved back to the queue.
		// In this case, we should retry scheduling it because this Definition may not be retried until the next flush.
		return queueAfterBackoff
	}

	// check if there is an event that makes this Definition schedulable based on pInfo.UnschedulablePlugins.
	queueingStrategy := queueSkip
	for event := inFlightDefinition.Next(); event != nil; event = event.Next() {
		e, ok := event.Value.(*clusterEvent)
		if !ok {
			// Must be another in-flight Definition (*corev1.Definition). Can be ignored.
			continue
		}
		logger.V(5).Info("Checking event for in-flight definition", "definition", klog.KObj(pInfo.Definition), "event", e.event.Label)

		switch p.isDefinitionWorthRequeuing(logger, pInfo, e.event, e.oldObj, e.newObj) {
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
				// We can return immediately because no Pending plugins, which only can make queueImmediately, registered in this Definition,
				// and queueAfterBackoff is the second highest priority.
				return queueAfterBackoff
			}
		}
	}
	return queueingStrategy
}

// addUnschedulableIfNotPresentWithoutQueueingHint inserts a definition that cannot be scheduled into
// the queue, unless it is already in the queue. Normally, PriorityQueue puts
// unschedulable definitions in `unschedulableDefinitions`. But if there has been a recent move
// request, then the definition is put in `definitionBackoffQ`.
// TODO: This function is called only when p.isSchedulingQueueHintEnabled is false,
// and this will be removed after SchedulingQueueHint goes to stable and the feature gate is removed.
func (p *PriorityQueue) addUnschedulableWithoutQueueingHint(logger klog.Logger, pInfo *framework.QueuedDefinitionInfo, definitionSchedulingCycle int64) error {
	definition := pInfo.Definition
	// Refresh the timestamp since the definition is re-added.
	pInfo.Timestamp = p.clock.Now()

	// When the queueing hint is enabled, they are used differently.
	// But, we use all of them as UnschedulablePlugins when the queueing hint isn't enabled so that we don't break the old behaviour.
	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)

	// If a move request has been received, move it to the BackoffQ, otherwise move
	// it to unschedulableDefinitions.
	for plugin := range rejectorPlugins {
		metrics.UnschedulableReason(plugin, pInfo.Definition.Spec.SchedulerName).Inc()
	}
	if p.moveRequestCycle >= definitionSchedulingCycle || len(rejectorPlugins) == 0 {
		// Two cases to move a Definition to the active/backoff queue:
		// - The Definition is rejected by some plugins, but a move request is received after this Definition's scheduling cycle is started.
		//   In this case, the received event may be make Definition schedulable and we should retry scheduling it.
		// - No unschedulable plugins are associated with this Definition,
		//   meaning something unusual (a temporal failure on kube-apiserver, etc) happened and this Definition gets moved back to the queue.
		//   In this case, we should retry scheduling it because this Definition may not be retried until the next flush.
		if err := p.definitionBackoffQ.Add(pInfo); err != nil {
			return fmt.Errorf("error adding definition %v to the backoff queue: %v", klog.KObj(definition), err)
		}
		logger.V(5).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(definition), "event", ScheduleAttemptFailure, "queue", backoffQ)
		metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("backoff", ScheduleAttemptFailure).Inc()
	} else {
		p.unschedulableDefinitions.addOrUpdate(pInfo)
		logger.V(5).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(definition), "event", ScheduleAttemptFailure, "queue", unschedulableDefinitions)
		metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("unschedulable", ScheduleAttemptFailure).Inc()
	}

	p.addNominatedDefinitionUnlocked(logger, pInfo.DefinitionInfo, nil)
	return nil
}

// AddUnschedulableIfNotPresent inserts a definition that cannot be scheduled into
// the queue, unless it is already in the queue. Normally, PriorityQueue puts
// unschedulable definitions in `unschedulableDefinitions`. But if there has been a recent move
// request, then the definition is put in `definitionBackoffQ`.
func (p *PriorityQueue) AddUnschedulableIfNotPresent(logger klog.Logger, pInfo *framework.QueuedDefinitionInfo, definitionSchedulingCycle int64) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	// In any case, this Definition will be moved back to the queue and we should call Done.
	defer p.done(pInfo.Definition.UID)

	definition := pInfo.Definition
	if p.unschedulableDefinitions.get(definition) != nil {
		return fmt.Errorf("Definition %v is already present in unschedulable queue", klog.KObj(definition))
	}

	if _, exists, _ := p.activeQ.Get(pInfo); exists {
		return fmt.Errorf("Definition %v is already present in the active queue", klog.KObj(definition))
	}
	if _, exists, _ := p.definitionBackoffQ.Get(pInfo); exists {
		return fmt.Errorf("Definition %v is already present in the backoff queue", klog.KObj(definition))
	}

	if !p.isSchedulingQueueHintEnabled {
		// fall back to the old behavior which doesn't depend on the queueing hint.
		return p.addUnschedulableWithoutQueueingHint(logger, pInfo, definitionSchedulingCycle)
	}

	// Refresh the timestamp since the definition is re-added.
	pInfo.Timestamp = p.clock.Now()

	// If a move request has been received, move it to the BackoffQ, otherwise move
	// it to unschedulableDefinitions.
	rejectorPlugins := pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins)
	for plugin := range rejectorPlugins {
		metrics.UnschedulableReason(plugin, pInfo.Definition.Spec.SchedulerName).Inc()
	}

	// We check whether this Definition may change its scheduling result by any of events that happened during scheduling.
	schedulingHint := p.determineSchedulingHintForInFlightDefinition(logger, pInfo)

	// In this case, we try to requeue this Definition to activeQ/backoffQ.
	queue := p.requeueDefinitionViaQueueingHint(logger, pInfo, schedulingHint, ScheduleAttemptFailure)
	logger.V(3).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(definition), "event", ScheduleAttemptFailure, "queue", queue, "schedulingCycle", definitionSchedulingCycle, "hint", schedulingHint, "unschedulable plugins", rejectorPlugins)
	if queue == activeQ {
		// When the Definition is moved to activeQ, need to let p.cond know so that the Definition will be pop()ed out.
		p.cond.Broadcast()
	}

	p.addNominatedDefinitionUnlocked(logger, pInfo.DefinitionInfo, nil)
	return nil
}

// flushBackoffQCompleted Moves all definitions from backoffQ which have completed backoff in to activeQ
func (p *PriorityQueue) flushBackoffQCompleted(logger klog.Logger) {
	p.lock.Lock()
	defer p.lock.Unlock()
	activated := false
	for {
		rawDefinitionInfo := p.definitionBackoffQ.Peek()
		if rawDefinitionInfo == nil {
			break
		}
		pInfo := rawDefinitionInfo.(*framework.QueuedDefinitionInfo)
		definition := pInfo.Definition
		if p.isDefinitionBackingoff(pInfo) {
			break
		}
		_, err := p.definitionBackoffQ.Pop()
		if err != nil {
			logger.Error(err, "Unable to pop definition from backoff queue despite backoff completion", "definition", klog.KObj(definition))
			break
		}
		if added, _ := p.addToActiveQ(logger, pInfo); added {
			logger.V(5).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(definition), "event", BackoffComplete, "queue", activeQ)
			metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("active", BackoffComplete).Inc()
			activated = true
		}
	}

	if activated {
		p.cond.Broadcast()
	}
}

// flushUnschedulableDefinitionsLeftover moves definitions which stay in unschedulableDefinitions
// longer than definitionMaxInUnschedulableDefinitionsDuration to backoffQ or activeQ.
func (p *PriorityQueue) flushUnschedulableDefinitionsLeftover(logger klog.Logger) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var definitionsToMove []*framework.QueuedDefinitionInfo
	currentTime := p.clock.Now()
	for _, pInfo := range p.unschedulableDefinitions.definitionInfoMap {
		lastScheduleTime := pInfo.Timestamp
		if currentTime.Sub(lastScheduleTime) > p.definitionMaxInUnschedulableDefinitionsDuration {
			definitionsToMove = append(definitionsToMove, pInfo)
		}
	}

	if len(definitionsToMove) > 0 {
		p.moveDefinitionsToActiveOrBackoffQueue(logger, definitionsToMove, UnschedulableTimeout, nil, nil)
	}
}

// Pop removes the head of the active queue and returns it. It blocks if the
// activeQ is empty and waits until a new item is added to the queue. It
// increments scheduling cycle when a definition is popped.
func (p *PriorityQueue) Pop(logger klog.Logger) (*framework.QueuedDefinitionInfo, error) {
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
	pInfo := obj.(*framework.QueuedDefinitionInfo)
	pInfo.Attempts++
	p.schedulingCycle++
	// In flight, no concurrent events yet.
	if p.isSchedulingQueueHintEnabled {
		p.inFlightDefinitions[pInfo.Definition.UID] = p.inFlightEvents.PushBack(pInfo.Definition)
	}

	// Update metrics and reset the set of unschedulable plugins for the next attempt.
	for plugin := range pInfo.UnschedulablePlugins.Union(pInfo.PendingPlugins) {
		metrics.UnschedulableReason(plugin, pInfo.Definition.Spec.SchedulerName).Dec()
	}
	pInfo.UnschedulablePlugins.Clear()
	pInfo.PendingPlugins.Clear()

	return pInfo, nil
}

// Done must be called for definition returned by Pop. This allows the queue to
// keep track of which definitions are currently being processed.
func (p *PriorityQueue) Done(definition types.UID) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.done(definition)
}

func (p *PriorityQueue) done(definition types.UID) {
	if !p.isSchedulingQueueHintEnabled {
		// do nothing if schedulingQueueHint is disabled.
		// In that case, we don't have inFlightDefinitions and inFlightEvents.
		return
	}
	inFlightDefinition, ok := p.inFlightDefinitions[definition]
	if !ok {
		// This Definition is already done()ed.
		return
	}
	delete(p.inFlightDefinitions, definition)

	// Remove the definition from the list.
	p.inFlightEvents.Remove(inFlightDefinition)

	// Remove events which are only referred to by this Definition
	// so that the inFlightEvents list doesn't grow infinitely.
	// If the definition was at the head of the list, then all
	// events between it and the next definition are no longer needed
	// and can be removed.
	for {
		e := p.inFlightEvents.Front()
		if e == nil {
			// Empty list.
			break
		}
		if _, ok := e.Value.(*clusterEvent); !ok {
			// A definition, must stop pruning.
			break
		}
		p.inFlightEvents.Remove(e)
	}
}

// isDefinitionUpdated checks if the definition is updated in a way that it may have become
// schedulable. It drops status of the definition and compares it with old version,
// except for definition.status.resourceClaimStatuses: changing that may have an
// effect on scheduling.
func isDefinitionUpdated(oldDefinition, newDefinition *corev1.Definition) bool {
	strip := func(definition *corev1.Definition) *corev1.Definition {
		p := definition.DeepCopy()
		p.ResourceVersion = ""
		p.Generation = 0
		p.Status = corev1.DefinitionStatus{
			//ResourceClaimStatuses: definition.Status.ResourceClaimStatuses,
		}
		p.ManagedFields = nil
		p.Finalizers = nil
		return p
	}
	return !reflect.DeepEqual(strip(oldDefinition), strip(newDefinition))
}

// Update updates a definition in the active or backoff queue if present. Otherwise, it removes
// the item from the unschedulable queue if definition is updated in a way that it may
// become schedulable and adds the updated one to the active queue.
// If definition is not present in any of the queues, it is added to the active queue.
func (p *PriorityQueue) Update(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if oldDefinition != nil {
		oldDefinitionInfo := newQueuedDefinitionInfoForLookup(oldDefinition)
		// If the definition is already in the active queue, just update it there.
		if oldDefinitionInfo, exists, _ := p.activeQ.Get(oldDefinitionInfo); exists {
			pInfo := updateDefinition(oldDefinitionInfo, newDefinition)
			p.updateNominatedDefinitionUnlocked(logger, oldDefinition, pInfo.DefinitionInfo)
			return p.activeQ.Update(pInfo)
		}

		// If the definition is in the backoff queue, update it there.
		if oldDefinitionInfo, exists, _ := p.definitionBackoffQ.Get(oldDefinitionInfo); exists {
			pInfo := updateDefinition(oldDefinitionInfo, newDefinition)
			p.updateNominatedDefinitionUnlocked(logger, oldDefinition, pInfo.DefinitionInfo)
			return p.definitionBackoffQ.Update(pInfo)
		}
	}

	// If the definition is in the unschedulable queue, updating it may make it schedulable.
	if usDefinitionInfo := p.unschedulableDefinitions.get(newDefinition); usDefinitionInfo != nil {
		pInfo := updateDefinition(usDefinitionInfo, newDefinition)
		p.updateNominatedDefinitionUnlocked(logger, oldDefinition, pInfo.DefinitionInfo)
		if isDefinitionUpdated(oldDefinition, newDefinition) {
			gated := usDefinitionInfo.Gated
			if p.isDefinitionBackingoff(usDefinitionInfo) {
				if err := p.definitionBackoffQ.Add(pInfo); err != nil {
					return err
				}
				p.unschedulableDefinitions.delete(usDefinitionInfo.Definition, gated)
				logger.V(5).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(pInfo.Definition), "event", DefinitionUpdate, "queue", backoffQ)
			} else {
				if added, err := p.addToActiveQ(logger, pInfo); !added {
					return err
				}
				p.unschedulableDefinitions.delete(usDefinitionInfo.Definition, gated)
				logger.V(5).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(pInfo.Definition), "event", BackoffComplete, "queue", activeQ)
				p.cond.Broadcast()
			}
		} else {
			// Definition update didn't make it schedulable, keep it in the unschedulable queue.
			p.unschedulableDefinitions.addOrUpdate(pInfo)
		}

		return nil
	}
	// If definition is not in any of the queues, we put it in the active queue.
	pInfo := p.newQueuedDefinitionInfo(newDefinition)
	if added, err := p.addToActiveQ(logger, pInfo); !added {
		return err
	}
	p.addNominatedDefinitionUnlocked(logger, pInfo.DefinitionInfo, nil)
	logger.V(5).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(pInfo.Definition), "event", DefinitionUpdate, "queue", activeQ)
	p.cond.Broadcast()
	return nil
}

// Delete deletes the item from either of the two queues. It assumes the definition is
// only in one queue.
func (p *PriorityQueue) Delete(definition *corev1.Definition) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.deleteNominatedDefinitionIfExistsUnlocked(definition)
	pInfo := newQueuedDefinitionInfoForLookup(definition)
	if err := p.activeQ.Delete(pInfo); err != nil {
		// The item was probably not found in the activeQ.
		p.definitionBackoffQ.Delete(pInfo)
		if pInfo = p.unschedulableDefinitions.get(definition); pInfo != nil {
			p.unschedulableDefinitions.delete(definition, pInfo.Gated)
		}
	}
	return nil
}

// AssignedDefinitionAdded is called when a bound definition is added. Creation of this definition
// may make pending definitions with matching affinity terms schedulable.
func (p *PriorityQueue) AssignedDefinitionAdded(logger klog.Logger, definition *corev1.Definition) {
	p.lock.Lock()
	p.moveDefinitionsToActiveOrBackoffQueue(logger, p.getUnschedulableDefinitionsWithMatchingAffinityTerm(logger, definition), AssignedDefinitionAdd, nil, definition)
	p.lock.Unlock()
}

// isDefinitionResourcesResizedDown returns true if a definition CPU and/or memory resize request has been
// admitted by kubelet, is 'InProgress', and results in a net sizing down of updated resources.
// It returns false if either CPU or memory resource is net resized up, or if no resize is in progress.
func isDefinitionResourcesResizedDown(definition *corev1.Definition) bool {
	//if utilfeature.DefaultFeatureGate.Enabled(features.InPlaceDefinitionVerticalScaling) {
	//	// TODO(vinaykul,wangchen615,InPlaceDefinitionVerticalScaling): Fix this to determine when a
	//	// definition is truly resized down (might need oldDefinition if we cannot determine from Status alone)
	//	if definition.Status.Resize == v1.DefinitionResizeStatusInProgress {
	//		return true
	//	}
	//}
	return false
}

// AssignedDefinitionUpdated is called when a bound definition is updated. Change of labels
// may make pending definitions with matching affinity terms schedulable.
func (p *PriorityQueue) AssignedDefinitionUpdated(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition) {
	p.lock.Lock()
	if isDefinitionResourcesResizedDown(newDefinition) {
		p.moveAllToActiveOrBackoffQueue(logger, AssignedDefinitionUpdate, oldDefinition, newDefinition, nil)
	} else {
		p.moveDefinitionsToActiveOrBackoffQueue(logger, p.getUnschedulableDefinitionsWithMatchingAffinityTerm(logger, newDefinition), AssignedDefinitionUpdate, oldDefinition, newDefinition)
	}
	p.lock.Unlock()
}

// NOTE: this function assumes a lock has been acquired in the caller.
// moveAllToActiveOrBackoffQueue moves all definitions from unschedulableDefinitions to activeQ or backoffQ.
// This function adds all definitions and then signals the condition variable to ensure that
// if Pop() is waiting for an item, it receives the signal after all the definitions are in the
// queue and the head is the highest priority definition.
func (p *PriorityQueue) moveAllToActiveOrBackoffQueue(logger klog.Logger, event framework.ClusterEvent, oldObj, newObj interface{}, preCheck PreEnqueueCheck) {
	if !p.isEventOfInterest(logger, event) {
		// No plugin is interested in this event.
		// Return early before iterating all definitions in unschedulableDefinitions for preCheck.
		return
	}

	unschedulableDefinitions := make([]*framework.QueuedDefinitionInfo, 0, len(p.unschedulableDefinitions.definitionInfoMap))
	for _, pInfo := range p.unschedulableDefinitions.definitionInfoMap {
		if preCheck == nil || preCheck(pInfo.Definition) {
			unschedulableDefinitions = append(unschedulableDefinitions, pInfo)
		}
	}
	p.moveDefinitionsToActiveOrBackoffQueue(logger, unschedulableDefinitions, event, oldObj, newObj)
}

// MoveAllToActiveOrBackoffQueue moves all definitions from unschedulableDefinitions to activeQ or backoffQ.
// This function adds all definitions and then signals the condition variable to ensure that
// if Pop() is waiting for an item, it receives the signal after all the definitions are in the
// queue and the head is the highest priority definition.
func (p *PriorityQueue) MoveAllToActiveOrBackoffQueue(logger klog.Logger, event framework.ClusterEvent, oldObj, newObj interface{}, preCheck PreEnqueueCheck) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.moveAllToActiveOrBackoffQueue(logger, event, oldObj, newObj, preCheck)
}

// requeueDefinitionViaQueueingHint tries to requeue Definition to activeQ, backoffQ or unschedulable definition pool based on schedulingHint.
// It returns the queue name Definition goes.
//
// NOTE: this function assumes lock has been acquired in caller
func (p *PriorityQueue) requeueDefinitionViaQueueingHint(logger klog.Logger, pInfo *framework.QueuedDefinitionInfo, strategy queueingStrategy, event string) string {
	if strategy == queueSkip {
		p.unschedulableDefinitions.addOrUpdate(pInfo)
		metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("unschedulable", event).Inc()
		return unschedulableDefinitions
	}

	definition := pInfo.Definition
	if strategy == queueAfterBackoff && p.isDefinitionBackingoff(pInfo) {
		if err := p.definitionBackoffQ.Add(pInfo); err != nil {
			logger.Error(err, "Error adding definition to the backoff queue, queue this Definition to unschedulable definition pool", "definition", klog.KObj(definition))
			p.unschedulableDefinitions.addOrUpdate(pInfo)
			return unschedulableDefinitions
		}

		metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("backoff", event).Inc()
		return backoffQ
	}

	// Reach here if schedulingHint is QueueImmediately, or schedulingHint is Queue but the definition is not backing off.

	added, err := p.addToActiveQ(logger, pInfo)
	if err != nil {
		logger.Error(err, "Error adding definition to the active queue, queue this Definition to unschedulable definition pool", "definition", klog.KObj(definition))
	}
	if added {
		metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("active", event).Inc()
		return activeQ
	}
	if pInfo.Gated {
		// In case the definition is gated, the Definition is pushed back to unschedulable Definitions pool in addToActiveQ.
		return unschedulableDefinitions
	}

	p.unschedulableDefinitions.addOrUpdate(pInfo)
	metrics.SchedulerQueueIncomingDefinitions.WithLabelValues("unschedulable", ScheduleAttemptFailure).Inc()
	return unschedulableDefinitions
}

// NOTE: this function assumes lock has been acquired in caller
func (p *PriorityQueue) moveDefinitionsToActiveOrBackoffQueue(logger klog.Logger, definitionInfoList []*framework.QueuedDefinitionInfo, event framework.ClusterEvent, oldObj, newObj interface{}) {
	if !p.isEventOfInterest(logger, event) {
		// No plugin is interested in this event.
		return
	}

	activated := false
	for _, pInfo := range definitionInfoList {
		schedulingHint := p.isDefinitionWorthRequeuing(logger, pInfo, event, oldObj, newObj)
		if schedulingHint == queueSkip {
			// QueueingHintFn determined that this Definition isn't worth putting to activeQ or backoffQ by this event.
			logger.V(5).Info("Event is not making definition schedulable", "definition", klog.KObj(pInfo.Definition), "event", event.Label)
			continue
		}

		p.unschedulableDefinitions.delete(pInfo.Definition, pInfo.Gated)
		queue := p.requeueDefinitionViaQueueingHint(logger, pInfo, schedulingHint, event.Label)
		logger.V(4).Info("Definition moved to an internal scheduling queue", "definition", klog.KObj(pInfo.Definition), "event", event.Label, "queue", queue, "hint", schedulingHint)
		if queue == activeQ {
			activated = true
		}
	}

	p.moveRequestCycle = p.schedulingCycle

	if p.isSchedulingQueueHintEnabled && len(p.inFlightDefinitions) != 0 {
		logger.V(5).Info("Event received while definitions are in flight", "event", event.Label, "numDefinitions", len(p.inFlightDefinitions))
		// AddUnschedulableIfNotPresent might get called for in-flight Definitions later, and in
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

// getUnschedulableDefinitionsWithMatchingAffinityTerm returns unschedulable definitions which have
// any affinity term that matches "definition".
// NOTE: this function assumes lock has been acquired in caller.
func (p *PriorityQueue) getUnschedulableDefinitionsWithMatchingAffinityTerm(logger klog.Logger, definition *corev1.Definition) []*framework.QueuedDefinitionInfo {
	nsLabels := interdefinitionaffinity.GetNamespaceLabelsSnapshot(logger, definition.Namespace, p.nsLister)

	var definitionsToMove []*framework.QueuedDefinitionInfo
	for _, pInfo := range p.unschedulableDefinitions.definitionInfoMap {
		for _, term := range pInfo.RequiredAffinityTerms {
			if term.Matches(definition, nsLabels) {
				definitionsToMove = append(definitionsToMove, pInfo)
				break
			}
		}

	}
	return definitionsToMove
}

// DefinitionsInActiveQ returns all the Definitions in the activeQ.
// This function is only used in tests.
func (p *PriorityQueue) DefinitionsInActiveQ() []*corev1.Definition {
	p.lock.RLock()
	defer p.lock.RUnlock()
	var result []*corev1.Definition
	for _, pInfo := range p.activeQ.List() {
		result = append(result, pInfo.(*framework.QueuedDefinitionInfo).Definition)
	}
	return result
}

var pendingDefinitionsSummary = "activeQ:%v; backoffQ:%v; unschedulableDefinitions:%v"

// PendingDefinitions returns all the pending definitions in the queue; accompanied by a debugging string
// recording showing the number of definitions in each queue respectively.
// This function is used for debugging purposes in the scheduler cache dumper and comparer.
func (p *PriorityQueue) PendingDefinitions() ([]*corev1.Definition, string) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	var result []*corev1.Definition
	for _, pInfo := range p.activeQ.List() {
		result = append(result, pInfo.(*framework.QueuedDefinitionInfo).Definition)
	}
	for _, pInfo := range p.definitionBackoffQ.List() {
		result = append(result, pInfo.(*framework.QueuedDefinitionInfo).Definition)
	}
	for _, pInfo := range p.unschedulableDefinitions.definitionInfoMap {
		result = append(result, pInfo.Definition)
	}
	return result, fmt.Sprintf(pendingDefinitionsSummary, p.activeQ.Len(), p.definitionBackoffQ.Len(), len(p.unschedulableDefinitions.definitionInfoMap))
}

// Close closes the priority queue.
func (p *PriorityQueue) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()
	close(p.stop)
	p.closed = true
	p.cond.Broadcast()
}

// DeleteNominatedDefinitionIfExists deletes <definition> from nominatedDefinitions.
func (npm *nominator) DeleteNominatedDefinitionIfExists(definition *corev1.Definition) {
	npm.lock.Lock()
	npm.deleteNominatedDefinitionIfExistsUnlocked(definition)
	npm.lock.Unlock()
}

func (npm *nominator) deleteNominatedDefinitionIfExistsUnlocked(definition *corev1.Definition) {
	npm.delete(definition)
}

// AddNominatedDefinition adds a definition to the nominated definitions of the given node.
// This is called during the preemption process after a node is nominated to run
// the definition. We update the structure before sending a request to update the definition
// object to avoid races with the following scheduling cycles.
func (npm *nominator) AddNominatedDefinition(logger klog.Logger, pi *framework.DefinitionInfo, nominatingInfo *framework.NominatingInfo) {
	npm.lock.Lock()
	npm.addNominatedDefinitionUnlocked(logger, pi, nominatingInfo)
	npm.lock.Unlock()
}

// NominatedDefinitionsForRunner returns a copy of definitions that are nominated to run on the given node,
// but they are waiting for other definitions to be removed from the node.
func (npm *nominator) NominatedDefinitionsForRunner(nodeName string) []*framework.DefinitionInfo {
	npm.lock.RLock()
	defer npm.lock.RUnlock()
	// Make a copy of the nominated Definitions so the caller can mutate safely.
	definitions := make([]*framework.DefinitionInfo, len(npm.nominatedDefinitions[nodeName]))
	for i := 0; i < len(definitions); i++ {
		definitions[i] = npm.nominatedDefinitions[nodeName][i].DeepCopy()
	}
	return definitions
}

func (p *PriorityQueue) definitionsCompareBackoffCompleted(definitionInfo1, definitionInfo2 interface{}) bool {
	pInfo1 := definitionInfo1.(*framework.QueuedDefinitionInfo)
	pInfo2 := definitionInfo2.(*framework.QueuedDefinitionInfo)
	bo1 := p.getBackoffTime(pInfo1)
	bo2 := p.getBackoffTime(pInfo2)
	return bo1.Before(bo2)
}

// newQueuedDefinitionInfo builds a QueuedDefinitionInfo object.
func (p *PriorityQueue) newQueuedDefinitionInfo(definition *corev1.Definition, plugins ...string) *framework.QueuedDefinitionInfo {
	now := p.clock.Now()
	// ignore this err since apiserver doesn't properly validate affinity terms
	// and we can't fix the validation for backwards compatibility.
	definitionInfo, _ := framework.NewDefinitionInfo(definition)
	return &framework.QueuedDefinitionInfo{
		DefinitionInfo:          definitionInfo,
		Timestamp:               now,
		InitialAttemptTimestamp: nil,
		UnschedulablePlugins:    sets.New(plugins...),
	}
}

// getBackoffTime returns the time that definitionInfo completes backoff
func (p *PriorityQueue) getBackoffTime(definitionInfo *framework.QueuedDefinitionInfo) time.Time {
	duration := p.calculateBackoffDuration(definitionInfo)
	backoffTime := definitionInfo.Timestamp.Add(duration)
	return backoffTime
}

// calculateBackoffDuration is a helper function for calculating the backoffDuration
// based on the number of attempts the definition has made.
func (p *PriorityQueue) calculateBackoffDuration(definitionInfo *framework.QueuedDefinitionInfo) time.Duration {
	duration := p.definitionInitialBackoffDuration
	for i := 1; i < definitionInfo.Attempts; i++ {
		// Use subtraction instead of addition or multiplication to avoid overflow.
		if duration > p.definitionMaxBackoffDuration-duration {
			return p.definitionMaxBackoffDuration
		}
		duration += duration
	}
	return duration
}

func updateDefinition(oldDefinitionInfo interface{}, newDefinition *corev1.Definition) *framework.QueuedDefinitionInfo {
	pInfo := oldDefinitionInfo.(*framework.QueuedDefinitionInfo)
	pInfo.Update(newDefinition)
	return pInfo
}

// UnschedulableDefinitions holds definitions that cannot be scheduled. This data structure
// is used to implement unschedulableDefinitions.
type UnschedulableDefinitions struct {
	// definitionInfoMap is a map key by a definition's full-name and the value is a pointer to the QueuedDefinitionInfo.
	definitionInfoMap map[string]*framework.QueuedDefinitionInfo
	keyFunc           func(*corev1.Definition) string
	// unschedulableRecorder/gatedRecorder updates the counter when elements of an unschedulableDefinitionsMap
	// get added or removed, and it does nothing if it's nil.
	unschedulableRecorder, gatedRecorder metrics.MetricRecorder
}

// addOrUpdate adds a definition to the unschedulable definitionInfoMap.
func (u *UnschedulableDefinitions) addOrUpdate(pInfo *framework.QueuedDefinitionInfo) {
	definitionID := u.keyFunc(pInfo.Definition)
	if _, exists := u.definitionInfoMap[definitionID]; !exists {
		if pInfo.Gated && u.gatedRecorder != nil {
			u.gatedRecorder.Inc()
		} else if !pInfo.Gated && u.unschedulableRecorder != nil {
			u.unschedulableRecorder.Inc()
		}
	}
	u.definitionInfoMap[definitionID] = pInfo
}

// delete deletes a definition from the unschedulable definitionInfoMap.
// The `gated` parameter is used to figure out which metric should be decreased.
func (u *UnschedulableDefinitions) delete(definition *corev1.Definition, gated bool) {
	definitionID := u.keyFunc(definition)
	if _, exists := u.definitionInfoMap[definitionID]; exists {
		if gated && u.gatedRecorder != nil {
			u.gatedRecorder.Dec()
		} else if !gated && u.unschedulableRecorder != nil {
			u.unschedulableRecorder.Dec()
		}
	}
	delete(u.definitionInfoMap, definitionID)
}

// get returns the QueuedDefinitionInfo if a definition with the same key as the key of the given "definition"
// is found in the map. It returns nil otherwise.
func (u *UnschedulableDefinitions) get(definition *corev1.Definition) *framework.QueuedDefinitionInfo {
	definitionKey := u.keyFunc(definition)
	if pInfo, exists := u.definitionInfoMap[definitionKey]; exists {
		return pInfo
	}
	return nil
}

// clear removes all the entries from the unschedulable definitionInfoMap.
func (u *UnschedulableDefinitions) clear() {
	u.definitionInfoMap = make(map[string]*framework.QueuedDefinitionInfo)
	if u.unschedulableRecorder != nil {
		u.unschedulableRecorder.Clear()
	}
	if u.gatedRecorder != nil {
		u.gatedRecorder.Clear()
	}
}

// newUnschedulableDefinitions initializes a new object of UnschedulableDefinitions.
func newUnschedulableDefinitions(unschedulableRecorder, gatedRecorder metrics.MetricRecorder) *UnschedulableDefinitions {
	return &UnschedulableDefinitions{
		definitionInfoMap:     make(map[string]*framework.QueuedDefinitionInfo),
		keyFunc:               util.GetDefinitionFullName,
		unschedulableRecorder: unschedulableRecorder,
		gatedRecorder:         gatedRecorder,
	}
}

// nominator is a structure that stores definitions nominated to run on nodes.
// It exists because nominatedRunnerName of definition objects stored in the structure
// may be different than what scheduler has here. We should be able to find definitions
// by their UID and update/delete them.
type nominator struct {
	// definitionLister is used to verify if the given definition is alive.
	definitionLister listersv1.DefinitionLister
	// nominatedDefinitions is a map keyed by a node name and the value is a list of
	// definitions which are nominated to run on the node. These are definitions which can be in
	// the activeQ or unschedulableDefinitions.
	nominatedDefinitions map[string][]*framework.DefinitionInfo
	// nominatedDefinitionToRunner is map keyed by a Definition UID to the node name where it is
	// nominated.
	nominatedDefinitionToRunner map[types.UID]string

	lock sync.RWMutex
}

func (npm *nominator) addNominatedDefinitionUnlocked(logger klog.Logger, pi *framework.DefinitionInfo, nominatingInfo *framework.NominatingInfo) {
	// Always delete the definition if it already exists, to ensure we never store more than
	// one instance of the definition.
	npm.delete(pi.Definition)

	var nodeName string
	if nominatingInfo.Mode() == framework.ModeOverride {
		nodeName = nominatingInfo.NominatedRunnerName
	} else if nominatingInfo.Mode() == framework.ModeNoop {
		//if pi.Definition.Status.NominatedRunnerName == "" {
		//	return
		//}
		//nodeName = pi.Definition.Status.NominatedRunnerName
	}

	if npm.definitionLister != nil {
		// If the definition was removed or if it was already scheduled, don't nominate it.
		updatedDefinition, err := npm.definitionLister.Definitions(pi.Definition.Namespace).Get(pi.Definition.Name)
		if err != nil {
			logger.V(4).Info("Definition doesn't exist in definitionLister, aborted adding it to the nominator", "definition", klog.KObj(pi.Definition))
			return
		}
		_ = updatedDefinition
		//if updatedDefinition.Spec.RunnerName != "" {
		//	logger.V(4).Info("Definition is already scheduled to a node, aborted adding it to the nominator", "definition", klog.KObj(pi.Definition), "node", updatedDefinition.Spec.RunnerName)
		//	return
		//}
	}

	npm.nominatedDefinitionToRunner[pi.Definition.UID] = nodeName
	for _, npi := range npm.nominatedDefinitions[nodeName] {
		if npi.Definition.UID == pi.Definition.UID {
			logger.V(4).Info("Definition already exists in the nominator", "definition", klog.KObj(npi.Definition))
			return
		}
	}
	npm.nominatedDefinitions[nodeName] = append(npm.nominatedDefinitions[nodeName], pi)
}

func (npm *nominator) delete(p *corev1.Definition) {
	nnn, ok := npm.nominatedDefinitionToRunner[p.UID]
	if !ok {
		return
	}
	for i, np := range npm.nominatedDefinitions[nnn] {
		if np.Definition.UID == p.UID {
			npm.nominatedDefinitions[nnn] = append(npm.nominatedDefinitions[nnn][:i], npm.nominatedDefinitions[nnn][i+1:]...)
			if len(npm.nominatedDefinitions[nnn]) == 0 {
				delete(npm.nominatedDefinitions, nnn)
			}
			break
		}
	}
	delete(npm.nominatedDefinitionToRunner, p.UID)
}

// UpdateNominatedDefinition updates the <oldDefinition> with <newDefinition>.
func (npm *nominator) UpdateNominatedDefinition(logger klog.Logger, oldDefinition *corev1.Definition, newDefinitionInfo *framework.DefinitionInfo) {
	npm.lock.Lock()
	defer npm.lock.Unlock()
	npm.updateNominatedDefinitionUnlocked(logger, oldDefinition, newDefinitionInfo)
}

func (npm *nominator) updateNominatedDefinitionUnlocked(logger klog.Logger, oldDefinition *corev1.Definition, newDefinitionInfo *framework.DefinitionInfo) {
	// In some cases, an Update event with no "NominatedRunner" present is received right
	// after a node("NominatedRunner") is reserved for this definition in memory.
	// In this case, we need to keep reserving the NominatedRunner when updating the definition pointer.
	var nominatingInfo *framework.NominatingInfo
	// We won't fall into below `if` block if the Update event represents:
	// (1) NominatedRunner info is added
	// (2) NominatedRunner info is updated
	// (3) NominatedRunner info is removed
	if NominatedRunnerName(oldDefinition) == "" && NominatedRunnerName(newDefinitionInfo.Definition) == "" {
		if nnn, ok := npm.nominatedDefinitionToRunner[oldDefinition.UID]; ok {
			// This is the only case we should continue reserving the NominatedRunner
			nominatingInfo = &framework.NominatingInfo{
				NominatingMode:      framework.ModeOverride,
				NominatedRunnerName: nnn,
			}
		}
	}
	// We update irrespective of the nominatedRunnerName changed or not, to ensure
	// that definition pointer is updated.
	npm.delete(oldDefinition)
	npm.addNominatedDefinitionUnlocked(logger, newDefinitionInfo, nominatingInfo)
}

// NewDefinitionNominator creates a nominator as a backing of framework.DefinitionNominator.
// A definitionLister is passed in so as to check if the definition exists
// before adding its nominatedRunner info.
func NewDefinitionNominator(definitionLister listersv1.DefinitionLister) framework.DefinitionNominator {
	return newDefinitionNominator(definitionLister)
}

func newDefinitionNominator(definitionLister listersv1.DefinitionLister) *nominator {
	return &nominator{
		definitionLister:            definitionLister,
		nominatedDefinitions:        make(map[string][]*framework.DefinitionInfo),
		nominatedDefinitionToRunner: make(map[types.UID]string),
	}
}

func definitionInfoKeyFunc(obj interface{}) (string, error) {
	return cache.MetaNamespaceKeyFunc(obj.(*framework.QueuedDefinitionInfo).Definition)
}
