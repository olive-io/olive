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
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/dynamicinformer"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/olive-io/olive/apis"
	configv1 "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	coreinformers "github.com/olive-io/olive/client-go/generated/informers/externalversions/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/parallelize"
	frameworkplugins "github.com/olive-io/olive/mon/scheduler/framework/plugins"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runnerresources"
	frameworkruntime "github.com/olive-io/olive/mon/scheduler/framework/runtime"
	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
	cachedebugger "github.com/olive-io/olive/mon/scheduler/internal/cache/debugger"
	internalqueue "github.com/olive-io/olive/mon/scheduler/internal/queue"
	"github.com/olive-io/olive/mon/scheduler/metrics"
	"github.com/olive-io/olive/mon/scheduler/profile"
)

const (
	// Duration the scheduler will wait before expiring an assumed region.
	// See issue #106361 for more details about this parameter and its value.
	durationToExpireAssumedRegion time.Duration = 0
)

// ErrNoRunnersAvailable is used to describe the error that no runners available to schedule regions.
var ErrNoRunnersAvailable = fmt.Errorf("no runners available to schedule regions")

// Scheduler watches for new unscheduled regions. It attempts to find
// runners that they fit on and writes bindings back to the api server.
type Scheduler struct {
	// It is expected that changes made via Cache will be observed
	// by RunnerLister and Algorithm.
	Cache internalcache.Cache

	Extenders []framework.Extender

	// NextRegion should be a function that blocks until the next region
	// is available. We don't use a channel for this, because scheduling
	// a region may take some amount of time and we don't want regions to get
	// stale while they sit in a channel.
	NextRegion func(logger klog.Logger) (*framework.QueuedRegionInfo, error)

	// FailureHandler is called upon a scheduling failure.
	FailureHandler FailureHandlerFn

	// ScheduleRegion tries to schedule the given region to one of the runners in the runner list.
	// Return a struct of ScheduleResult with the name of suggested host on success,
	// otherwise will return a FitError with reasons.
	ScheduleRegion func(ctx context.Context, fwk framework.Framework, state *framework.CycleState, region *corev1.Region) (ScheduleResult, error)

	// Close this to shut down the scheduler.
	StopEverything <-chan struct{}

	// SchedulingQueue holds regions to be scheduled
	SchedulingQueue internalqueue.SchedulingQueue

	// Profiles are the scheduling profiles.
	Profiles profile.Map

	client clientset.Interface

	runnerInfoSnapshot *internalcache.Snapshot

	percentageOfRunnersToScore int32

	nextStartRunnerIndex int

	// logger *must* be initialized when creating a Scheduler,
	// otherwise logging functions will access a nil sink and
	// panic.
	logger klog.Logger

	// registeredHandlers contains the registrations of all handlers. It's used to check if all handlers have finished syncing before the scheduling cycles start.
	registeredHandlers []cache.ResourceEventHandlerRegistration
}

func (sched *Scheduler) applyDefaultHandlers() {
	sched.ScheduleRegion = sched.scheduleRegion
	sched.FailureHandler = sched.handleSchedulingFailure
}

type schedulerOptions struct {
	componentConfigVersion string
	kubeConfig             *restclient.Config
	// Overridden by profile level percentageOfRunnersToScore if set in v1.
	percentageOfRunnersToScore              int32
	regionInitialBackoffSeconds             int64
	regionMaxBackoffSeconds                 int64
	regionMaxInUnschedulableRegionsDuration time.Duration
	// Contains out-of-tree plugins to be merged with the in-tree registry.
	frameworkOutOfTreeRegistry frameworkruntime.Registry
	profiles                   []configv1.SchedulerProfile
	extenders                  []configv1.Extender
	frameworkCapturer          FrameworkCapturer
	parallelism                int32
	applyDefaultProfile        bool
}

// Option configures a Scheduler
type Option func(*schedulerOptions)

// ScheduleResult represents the result of scheduling a region.
type ScheduleResult struct {
	// Name of the selected runner.
	SuggestedHost string
	// The number of runners the scheduler evaluated the region against in the filtering
	// phase and beyond.
	// Note that it contains the number of runners that filtered out by PreFilterResult.
	EvaluatedRunners int
	// The number of runners out of the evaluated ones that fit the region.
	FeasibleRunners int
	// The nominating info for scheduling cycle.
	nominatingInfo *framework.NominatingInfo
}

// WithComponentConfigVersion sets the component config version to the
// OliveSchedulerConfiguration version used. The string should be the full
// scheme group/version of the external type we converted from (for example
// "kubescheduler.config.k8s.io/v1")
func WithComponentConfigVersion(apiVersion string) Option {
	return func(o *schedulerOptions) {
		o.componentConfigVersion = apiVersion
	}
}

// WithOliveConfig sets the kube config for Scheduler.
func WithOliveConfig(cfg *restclient.Config) Option {
	return func(o *schedulerOptions) {
		o.kubeConfig = cfg
	}
}

// WithProfiles sets profiles for Scheduler. By default, there is one profile
// with the name "default-scheduler".
func WithProfiles(p ...configv1.SchedulerProfile) Option {
	return func(o *schedulerOptions) {
		o.profiles = p
		o.applyDefaultProfile = false
	}
}

// WithParallelism sets the parallelism for all scheduler algorithms. Default is 16.
func WithParallelism(threads int32) Option {
	return func(o *schedulerOptions) {
		o.parallelism = threads
	}
}

// WithPercentageOfRunnersToScore sets percentageOfRunnersToScore for Scheduler.
// The default value of 0 will use an adaptive percentage: 50 - (num of runners)/125.
func WithPercentageOfRunnersToScore(percentageOfRunnersToScore *int32) Option {
	return func(o *schedulerOptions) {
		if percentageOfRunnersToScore != nil {
			o.percentageOfRunnersToScore = *percentageOfRunnersToScore
		}
	}
}

// WithFrameworkOutOfTreeRegistry sets the registry for out-of-tree plugins. Those plugins
// will be appended to the default registry.
func WithFrameworkOutOfTreeRegistry(registry frameworkruntime.Registry) Option {
	return func(o *schedulerOptions) {
		o.frameworkOutOfTreeRegistry = registry
	}
}

// WithRegionInitialBackoffSeconds sets regionInitialBackoffSeconds for Scheduler, the default value is 1
func WithRegionInitialBackoffSeconds(regionInitialBackoffSeconds int64) Option {
	return func(o *schedulerOptions) {
		o.regionInitialBackoffSeconds = regionInitialBackoffSeconds
	}
}

// WithRegionMaxBackoffSeconds sets regionMaxBackoffSeconds for Scheduler, the default value is 10
func WithRegionMaxBackoffSeconds(regionMaxBackoffSeconds int64) Option {
	return func(o *schedulerOptions) {
		o.regionMaxBackoffSeconds = regionMaxBackoffSeconds
	}
}

// WithRegionMaxInUnschedulableRegionsDuration sets regionMaxInUnschedulableRegionsDuration for PriorityQueue.
func WithRegionMaxInUnschedulableRegionsDuration(duration time.Duration) Option {
	return func(o *schedulerOptions) {
		o.regionMaxInUnschedulableRegionsDuration = duration
	}
}

// WithExtenders sets extenders for the Scheduler
func WithExtenders(e ...configv1.Extender) Option {
	return func(o *schedulerOptions) {
		o.extenders = e
	}
}

// FrameworkCapturer is used for registering a notify function in building framework.
type FrameworkCapturer func(configv1.SchedulerProfile)

// WithBuildFrameworkCapturer sets a notify function for getting buildFramework details.
func WithBuildFrameworkCapturer(fc FrameworkCapturer) Option {
	return func(o *schedulerOptions) {
		o.frameworkCapturer = fc
	}
}

var defaultSchedulerOptions = schedulerOptions{
	percentageOfRunnersToScore:              configv1.DefaultPercentageOfRunnersToScore,
	regionInitialBackoffSeconds:             int64(internalqueue.DefaultRegionInitialBackoffDuration.Seconds()),
	regionMaxBackoffSeconds:                 int64(internalqueue.DefaultRegionMaxBackoffDuration.Seconds()),
	regionMaxInUnschedulableRegionsDuration: internalqueue.DefaultRegionMaxInUnschedulableRegionsDuration,
	parallelism:                             int32(parallelize.DefaultParallelism),
	// Ideally we would statically set the default profile here, but we can't because
	// creating the default profile may require testing feature gates, which may get
	// set dynamically in tests. Therefore, we delay creating it until New is actually
	// invoked.
	applyDefaultProfile: true,
}

// New returns a Scheduler
func New(ctx context.Context,
	client clientset.Interface,
	informerFactory informers.SharedInformerFactory,
	dynInformerFactory dynamicinformer.DynamicSharedInformerFactory,
	recorderFactory profile.RecorderFactory,
	opts ...Option) (*Scheduler, error) {

	logger := klog.FromContext(ctx)
	stopEverything := ctx.Done()

	options := defaultSchedulerOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.applyDefaultProfile {
		var versionedCfg configv1.SchedulerConfiguration
		apis.Scheme.Default(&versionedCfg)
		cfg := configv1.SchedulerConfiguration{}
		if err := apis.Scheme.Convert(&versionedCfg, &cfg, nil); err != nil {
			return nil, err
		}
		options.profiles = cfg.Profiles
	}

	registry := frameworkplugins.NewInTreeRegistry()
	if err := registry.Merge(options.frameworkOutOfTreeRegistry); err != nil {
		return nil, err
	}

	metrics.Register()

	extenders, err := buildExtenders(logger, options.extenders, options.profiles)
	if err != nil {
		return nil, fmt.Errorf("couldn't build extenders: %w", err)
	}

	regionLister := informerFactory.Core().V1().Regions().Lister()
	runnerLister := informerFactory.Core().V1().Runners().Lister()

	snapshot := internalcache.NewEmptySnapshot()
	metricsRecorder := metrics.NewMetricsAsyncRecorder(1000, time.Second, stopEverything)

	profiles, err := profile.NewMap(ctx, options.profiles, registry, recorderFactory,
		frameworkruntime.WithComponentConfigVersion(options.componentConfigVersion),
		frameworkruntime.WithClientSet(client),
		frameworkruntime.WithOliveConfig(options.kubeConfig),
		frameworkruntime.WithInformerFactory(informerFactory),
		frameworkruntime.WithSnapshotSharedLister(snapshot),
		frameworkruntime.WithCaptureProfile(frameworkruntime.CaptureProfile(options.frameworkCapturer)),
		frameworkruntime.WithParallelism(int(options.parallelism)),
		frameworkruntime.WithExtenders(extenders),
		frameworkruntime.WithMetricsRecorder(metricsRecorder),
	)
	if err != nil {
		return nil, fmt.Errorf("initializing profiles: %v", err)
	}

	if len(profiles) == 0 {
		return nil, errors.New("at least one profile is required")
	}

	preEnqueuePluginMap := make(map[string][]framework.PreEnqueuePlugin)
	queueingHintsPerProfile := make(internalqueue.QueueingHintMapPerProfile)
	for profileName, profile := range profiles {
		preEnqueuePluginMap[profileName] = profile.PreEnqueuePlugins()
		queueingHintsPerProfile[profileName] = buildQueueingHintMap(profile.EnqueueExtensions())
	}

	regionQueue := internalqueue.NewSchedulingQueue(
		profiles[options.profiles[0].SchedulerName].QueueSortFunc(),
		informerFactory,
		internalqueue.WithRegionInitialBackoffDuration(time.Duration(options.regionInitialBackoffSeconds)*time.Second),
		internalqueue.WithRegionMaxBackoffDuration(time.Duration(options.regionMaxBackoffSeconds)*time.Second),
		internalqueue.WithRegionLister(regionLister),
		internalqueue.WithRegionMaxInUnschedulableRegionsDuration(options.regionMaxInUnschedulableRegionsDuration),
		internalqueue.WithPreEnqueuePluginMap(preEnqueuePluginMap),
		internalqueue.WithQueueingHintMapPerProfile(queueingHintsPerProfile),
		internalqueue.WithPluginMetricsSamplePercent(pluginMetricsSamplePercent),
		internalqueue.WithMetricsRecorder(*metricsRecorder),
	)

	for _, fwk := range profiles {
		fwk.SetRegionNominator(regionQueue)
	}

	schedulerCache := internalcache.New(ctx, durationToExpireAssumedRegion)

	// Setup cache debugger.
	debugger := cachedebugger.New(runnerLister, regionLister, schedulerCache, regionQueue)
	debugger.ListenForSignal(ctx)

	sched := &Scheduler{
		Cache:                      schedulerCache,
		client:                     client,
		runnerInfoSnapshot:         snapshot,
		percentageOfRunnersToScore: options.percentageOfRunnersToScore,
		Extenders:                  extenders,
		StopEverything:             stopEverything,
		SchedulingQueue:            regionQueue,
		Profiles:                   profiles,
		logger:                     logger,
	}
	sched.NextRegion = regionQueue.Pop
	sched.applyDefaultHandlers()

	if err = addAllEventHandlers(sched, informerFactory, dynInformerFactory, unionedGVKs(queueingHintsPerProfile)); err != nil {
		return nil, fmt.Errorf("adding event handlers: %w", err)
	}

	return sched, nil
}

// defaultQueueingHintFn is the default queueing hint function.
// It always returns Queue as the queueing hint.
var defaultQueueingHintFn = func(_ klog.Logger, _ *corev1.Region, _, _ interface{}) (framework.QueueingHint, error) {
	return framework.Queue, nil
}

func buildQueueingHintMap(es []framework.EnqueueExtensions) internalqueue.QueueingHintMap {
	queueingHintMap := make(internalqueue.QueueingHintMap)
	for _, e := range es {
		events := e.EventsToRegister()

		// This will happen when plugin registers with empty events, it's usually the case a region
		// will become reschedulable only for self-update, e.g. schedulingGates plugin, the region
		// will enter into the activeQ via priorityQueue.Update().
		if len(events) == 0 {
			continue
		}

		// Note: Rarely, a plugin implements EnqueueExtensions but returns nil.
		// We treat it as: the plugin is not interested in any event, and hence region failed by that plugin
		// cannot be moved by any regular cluster event.
		// So, we can just ignore such EventsToRegister here.

		registerRunnerAdded := false
		registerRunnerTaintUpdated := false
		for _, event := range events {
			fn := event.QueueingHintFn

			if event.Event.Resource == framework.Runner {
				if event.Event.ActionType&framework.Add != 0 {
					registerRunnerAdded = true
				}
				if event.Event.ActionType&framework.UpdateRunnerTaint != 0 {
					registerRunnerTaintUpdated = true
				}
			}

			queueingHintMap[event.Event] = append(queueingHintMap[event.Event], &internalqueue.QueueingHintFunction{
				PluginName:     e.Name(),
				QueueingHintFn: fn,
			})
		}
		if registerRunnerAdded && !registerRunnerTaintUpdated {
			// RunnerAdded QueueingHint isn't always called because of preCheck.
			// It's definitely not something expected for plugin developers,
			// and registering UpdateRunnerTaint event is the only mitigation for now.
			//
			// So, here registers UpdateRunnerTaint event for plugins that has RunnerAdded event, but don't have UpdateRunnerTaint event.
			// It has a bad impact for the requeuing efficiency though, a lot better than some Regions being stuch in the
			// unschedulable region pool.
			// This behavior will be removed when we remove the preCheck feature.
			queueingHintMap[framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerTaint}] =
				append(queueingHintMap[framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerTaint}],
					&internalqueue.QueueingHintFunction{
						PluginName:     e.Name(),
						QueueingHintFn: defaultQueueingHintFn,
					},
				)
		}
	}
	return queueingHintMap
}

// Run begins watching and scheduling. It starts scheduling and blocked until the context is done.
func (sched *Scheduler) Run(ctx context.Context) {
	logger := klog.FromContext(ctx)
	sched.SchedulingQueue.Run(logger)

	// We need to start scheduleOne loop in a dedicated goroutine,
	// because scheduleOne function hangs on getting the next item
	// from the SchedulingQueue.
	// If there are no new regions to schedule, it will be hanging there
	// and if done in this goroutine it will be blocking closing
	// SchedulingQueue, in effect causing a deadlock on shutdown.
	go wait.UntilWithContext(ctx, sched.ScheduleOne, 0)

	<-ctx.Done()
	sched.SchedulingQueue.Close()

	// If the plugins satisfy the io.Closer interface, they are closed.
	err := sched.Profiles.Close()
	if err != nil {
		logger.Error(err, "Failed to close plugins")
	}
}

// NewInformerFactory creates a SharedInformerFactory and initializes a scheduler specific
// in-place regionInformer.
func NewInformerFactory(cs clientset.Interface, resyncPeriod time.Duration) informers.SharedInformerFactory {
	informerFactory := informers.NewSharedInformerFactory(cs, resyncPeriod)
	informerFactory.InformerFor(&corev1.Region{}, newRegionInformer)
	return informerFactory
}

func buildExtenders(logger klog.Logger, extenders []configv1.Extender, profiles []configv1.SchedulerProfile) ([]framework.Extender, error) {
	var fExtenders []framework.Extender
	if len(extenders) == 0 {
		return nil, nil
	}

	var ignoredExtendedResources []string
	var ignorableExtenders []framework.Extender
	for i := range extenders {
		logger.V(2).Info("Creating extender", "extender", extenders[i])
		extender, err := NewHTTPExtender(&extenders[i])
		if err != nil {
			return nil, err
		}
		if !extender.IsIgnorable() {
			fExtenders = append(fExtenders, extender)
		} else {
			ignorableExtenders = append(ignorableExtenders, extender)
		}
		for _, r := range extenders[i].ManagedResources {
			if r.IgnoredByScheduler {
				ignoredExtendedResources = append(ignoredExtendedResources, r.Name)
			}
		}
	}
	// place ignorable extenders to the tail of extenders
	fExtenders = append(fExtenders, ignorableExtenders...)

	// If there are any extended resources found from the Extenders, append them to the pluginConfig for each profile.
	// This should only have an effect on ComponentConfig, where it is possible to configure Extenders and
	// plugin args (and in which case the extender ignored resources take precedence).
	if len(ignoredExtendedResources) == 0 {
		return fExtenders, nil
	}

	for i := range profiles {
		prof := &profiles[i]
		var found = false
		for k := range prof.PluginConfig {
			if prof.PluginConfig[k].Name == runnerresources.Name {
				// Update the existing args
				pc := &prof.PluginConfig[k]
				args, ok := pc.Args.Object.(*configv1.RunnerResourcesFitArgs)
				if !ok {
					return nil, fmt.Errorf("want args to be of type RunnerResourcesFitArgs, got %T", pc.Args)
				}
				args.IgnoredResources = ignoredExtendedResources
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("can't find RunnerResourcesFitArgs in plugin config")
		}
	}
	return fExtenders, nil
}

type FailureHandlerFn func(ctx context.Context, fwk framework.Framework, regionInfo *framework.QueuedRegionInfo, status *framework.Status, nominatingInfo *framework.NominatingInfo, start time.Time)

func unionedGVKs(queueingHintsPerProfile internalqueue.QueueingHintMapPerProfile) map[framework.GVK]framework.ActionType {
	gvkMap := make(map[framework.GVK]framework.ActionType)
	for _, queueingHints := range queueingHintsPerProfile {
		for evt := range queueingHints {
			if _, ok := gvkMap[evt.Resource]; ok {
				gvkMap[evt.Resource] |= evt.ActionType
			} else {
				gvkMap[evt.Resource] = evt.ActionType
			}
		}
	}
	return gvkMap
}

// newRegionInformer creates a shared index informer that returns only non-terminal regions.
// The RegionInformer allows indexers to be added, but note that only non-conflict indexers are allowed.
func newRegionInformer(cs clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	selector := fmt.Sprintf("status.phase!=%v,status.phase!=%v", corev1.RegionSucceeded, corev1.RegionFailed)
	tweakListOptions := func(options *metav1.ListOptions) {
		options.FieldSelector = selector
	}

	informer := coreinformers.NewFilteredRegionInformer(cs, resyncPeriod, cache.Indexers{}, tweakListOptions)

	// Dropping `.metadata.managedFields` to improve memory usage.
	// The Extract workflow (i.e. `ExtractRegion`) should be unused.
	trim := func(obj interface{}) (interface{}, error) {
		if accessor, err := meta.Accessor(obj); err == nil {
			if accessor.GetManagedFields() != nil {
				accessor.SetManagedFields(nil)
			}
		}
		return obj, nil
	}
	informer.SetTransform(trim)
	return informer
}
