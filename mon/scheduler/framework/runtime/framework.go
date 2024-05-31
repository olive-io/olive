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

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"

	config "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/parallelize"
	"github.com/olive-io/olive/mon/scheduler/metrics"
	"github.com/olive-io/olive/mon/scheduler/util/slice"
	scheudulingcorev1 "github.com/olive-io/olive/pkg/scheduling/corev1"
)

const (
	// Specifies the maximum timeout a permit plugin can return.
	maxTimeout = 15 * time.Minute
)

// frameworkImpl is the component responsible for initializing and running scheduler
// plugins.
type frameworkImpl struct {
	registry             Registry
	snapshotSharedLister framework.SharedLister
	waitingRegions       *waitingRegionsMap
	scorePluginWeight    map[string]int
	preEnqueuePlugins    []framework.PreEnqueuePlugin
	enqueueExtensions    []framework.EnqueueExtensions
	queueSortPlugins     []framework.QueueSortPlugin
	preFilterPlugins     []framework.PreFilterPlugin
	filterPlugins        []framework.FilterPlugin
	postFilterPlugins    []framework.PostFilterPlugin
	preScorePlugins      []framework.PreScorePlugin
	scorePlugins         []framework.ScorePlugin
	reservePlugins       []framework.ReservePlugin
	preBindPlugins       []framework.PreBindPlugin
	bindPlugins          []framework.BindPlugin
	postBindPlugins      []framework.PostBindPlugin
	permitPlugins        []framework.PermitPlugin

	// pluginsMap contains all plugins, by name.
	pluginsMap map[string]framework.Plugin

	clientSet       clientset.Interface
	kubeConfig      *restclient.Config
	eventRecorder   events.EventRecorder
	informerFactory informers.SharedInformerFactory
	logger          klog.Logger

	metricsRecorder            *metrics.MetricAsyncRecorder
	profileName                string
	percentageOfRunnersToScore *int32

	extenders []framework.Extender
	framework.RegionNominator

	parallelizer parallelize.Parallelizer
}

// extensionPoint encapsulates desired and applied set of plugins at a specific extension
// point. This is used to simplify iterating over all extension points supported by the
// frameworkImpl.
type extensionPoint struct {
	// the set of plugins to be configured at this extension point.
	plugins *config.PluginSet
	// a pointer to the slice storing plugins implementations that will run at this
	// extension point.
	slicePtr interface{}
}

func (f *frameworkImpl) getExtensionPoints(plugins *config.Plugins) []extensionPoint {
	return []extensionPoint{
		{&plugins.PreFilter, &f.preFilterPlugins},
		{&plugins.Filter, &f.filterPlugins},
		{&plugins.PostFilter, &f.postFilterPlugins},
		{&plugins.Reserve, &f.reservePlugins},
		{&plugins.PreScore, &f.preScorePlugins},
		{&plugins.Score, &f.scorePlugins},
		{&plugins.PreBind, &f.preBindPlugins},
		{&plugins.Bind, &f.bindPlugins},
		{&plugins.PostBind, &f.postBindPlugins},
		{&plugins.Permit, &f.permitPlugins},
		{&plugins.PreEnqueue, &f.preEnqueuePlugins},
		{&plugins.QueueSort, &f.queueSortPlugins},
	}
}

// Extenders returns the registered extenders.
func (f *frameworkImpl) Extenders() []framework.Extender {
	return f.extenders
}

type frameworkOptions struct {
	componentConfigVersion string
	clientSet              clientset.Interface
	kubeConfig             *restclient.Config
	eventRecorder          events.EventRecorder
	informerFactory        informers.SharedInformerFactory
	snapshotSharedLister   framework.SharedLister
	metricsRecorder        *metrics.MetricAsyncRecorder
	regionNominator        framework.RegionNominator
	extenders              []framework.Extender
	captureProfile         CaptureProfile
	parallelizer           parallelize.Parallelizer
	logger                 *klog.Logger
}

// Option for the frameworkImpl.
type Option func(*frameworkOptions)

// WithComponentConfigVersion sets the component config version to the
// OliveSchedulerConfiguration version used. The string should be the full
// scheme group/version of the external type we converted from (for example
// "scheduler.config.olive.io/v1")
func WithComponentConfigVersion(componentConfigVersion string) Option {
	return func(o *frameworkOptions) {
		o.componentConfigVersion = componentConfigVersion
	}
}

// WithClientSet sets clientSet for the scheduling frameworkImpl.
func WithClientSet(clientSet clientset.Interface) Option {
	return func(o *frameworkOptions) {
		o.clientSet = clientSet
	}
}

// WithOliveConfig sets kubeConfig for the scheduling frameworkImpl.
func WithOliveConfig(kubeConfig *restclient.Config) Option {
	return func(o *frameworkOptions) {
		o.kubeConfig = kubeConfig
	}
}

// WithEventRecorder sets clientSet for the scheduling frameworkImpl.
func WithEventRecorder(recorder events.EventRecorder) Option {
	return func(o *frameworkOptions) {
		o.eventRecorder = recorder
	}
}

// WithInformerFactory sets informer factory for the scheduling frameworkImpl.
func WithInformerFactory(informerFactory informers.SharedInformerFactory) Option {
	return func(o *frameworkOptions) {
		o.informerFactory = informerFactory
	}
}

// WithSnapshotSharedLister sets the SharedLister of the snapshot.
func WithSnapshotSharedLister(snapshotSharedLister framework.SharedLister) Option {
	return func(o *frameworkOptions) {
		o.snapshotSharedLister = snapshotSharedLister
	}
}

// WithRegionNominator sets regionNominator for the scheduling frameworkImpl.
func WithRegionNominator(nominator framework.RegionNominator) Option {
	return func(o *frameworkOptions) {
		o.regionNominator = nominator
	}
}

// WithExtenders sets extenders for the scheduling frameworkImpl.
func WithExtenders(extenders []framework.Extender) Option {
	return func(o *frameworkOptions) {
		o.extenders = extenders
	}
}

// WithParallelism sets parallelism for the scheduling frameworkImpl.
func WithParallelism(parallelism int) Option {
	return func(o *frameworkOptions) {
		o.parallelizer = parallelize.NewParallelizer(parallelism)
	}
}

// CaptureProfile is a callback to capture a finalized profile.
type CaptureProfile func(config.SchedulerProfile)

// WithCaptureProfile sets a callback to capture the finalized profile.
func WithCaptureProfile(c CaptureProfile) Option {
	return func(o *frameworkOptions) {
		o.captureProfile = c
	}
}

// WithMetricsRecorder sets metrics recorder for the scheduling frameworkImpl.
func WithMetricsRecorder(r *metrics.MetricAsyncRecorder) Option {
	return func(o *frameworkOptions) {
		o.metricsRecorder = r
	}
}

// WithLogger overrides the default logger from k8s.io/klog.
func WithLogger(logger klog.Logger) Option {
	return func(o *frameworkOptions) {
		o.logger = &logger
	}
}

// defaultFrameworkOptions are applied when no option corresponding to those fields exist.
func defaultFrameworkOptions(stopCh <-chan struct{}) frameworkOptions {
	return frameworkOptions{
		metricsRecorder: metrics.NewMetricsAsyncRecorder(1000, time.Second, stopCh),
		parallelizer:    parallelize.NewParallelizer(parallelize.DefaultParallelism),
	}
}

var _ framework.Framework = &frameworkImpl{}

// NewFramework initializes plugins given the configuration and the registry.
func NewFramework(ctx context.Context, r Registry, profile *config.SchedulerProfile, opts ...Option) (framework.Framework, error) {
	options := defaultFrameworkOptions(ctx.Done())
	for _, opt := range opts {
		opt(&options)
	}

	logger := klog.FromContext(ctx)
	if options.logger != nil {
		logger = *options.logger
	}

	f := &frameworkImpl{
		registry:             r,
		snapshotSharedLister: options.snapshotSharedLister,
		scorePluginWeight:    make(map[string]int),
		waitingRegions:       newWaitingRegionsMap(),
		clientSet:            options.clientSet,
		kubeConfig:           options.kubeConfig,
		eventRecorder:        options.eventRecorder,
		informerFactory:      options.informerFactory,
		metricsRecorder:      options.metricsRecorder,
		extenders:            options.extenders,
		RegionNominator:      options.regionNominator,
		parallelizer:         options.parallelizer,
		logger:               logger,
	}

	if len(f.extenders) > 0 {
		// Extender doesn't support any kind of requeueing feature like EnqueueExtensions in the scheduling framework.
		// We register a defaultEnqueueExtension to framework.ExtenderName here.
		// And, in the scheduling cycle, when Extenders reject some Runners and the region ends up being unschedulable,
		// we put framework.ExtenderName to pInfo.UnschedulablePlugins.
		f.enqueueExtensions = []framework.EnqueueExtensions{&defaultEnqueueExtension{pluginName: framework.ExtenderName}}
	}

	if profile == nil {
		return f, nil
	}

	f.profileName = profile.SchedulerName
	f.percentageOfRunnersToScore = &profile.PercentageOfRunnersToScore
	if profile.Plugins == nil {
		return f, nil
	}

	// get needed plugins from config
	pg := f.pluginsNeeded(profile.Plugins)

	pluginConfig := make(map[string]runtime.Object, len(profile.PluginConfig))
	for i := range profile.PluginConfig {
		name := profile.PluginConfig[i].Name
		if _, ok := pluginConfig[name]; ok {
			return nil, fmt.Errorf("repeated config for plugin %s", name)
		}
		pluginConfig[name] = profile.PluginConfig[i].Args.Object
	}
	outputProfile := config.SchedulerProfile{
		SchedulerName:              f.profileName,
		PercentageOfRunnersToScore: *f.percentageOfRunnersToScore,
		Plugins:                    profile.Plugins,
		PluginConfig:               make([]config.PluginConfig, 0, len(pg)),
	}

	f.pluginsMap = make(map[string]framework.Plugin)
	for name, factory := range r {
		// initialize only needed plugins.
		if !pg.Has(name) {
			continue
		}

		args := pluginConfig[name]
		if args != nil {
			outputProfile.PluginConfig = append(outputProfile.PluginConfig, config.PluginConfig{
				Name: name,
				Args: runtime.RawExtension{
					Object: args,
				},
			})
		}
		p, err := factory(ctx, args, f)
		if err != nil {
			return nil, fmt.Errorf("initializing plugin %q: %w", name, err)
		}
		f.pluginsMap[name] = p

		f.fillEnqueueExtensions(p)
	}

	// initialize plugins per individual extension points
	for _, e := range f.getExtensionPoints(profile.Plugins) {
		if err := updatePluginList(e.slicePtr, *e.plugins, f.pluginsMap); err != nil {
			return nil, err
		}
	}

	// initialize multiPoint plugins to their expanded extension points
	if len(profile.Plugins.MultiPoint.Enabled) > 0 {
		if err := f.expandMultiPointPlugins(logger, profile); err != nil {
			return nil, err
		}
	}

	if len(f.queueSortPlugins) != 1 {
		return nil, fmt.Errorf("only one queue sort plugin required for profile with scheduler name %q, but got %d", profile.SchedulerName, len(f.queueSortPlugins))
	}
	if len(f.bindPlugins) == 0 {
		return nil, fmt.Errorf("at least one bind plugin is needed for profile with scheduler name %q", profile.SchedulerName)
	}

	if err := getScoreWeights(f, append(profile.Plugins.Score.Enabled, profile.Plugins.MultiPoint.Enabled...)); err != nil {
		return nil, err
	}

	// Verifying the score weights again since Plugin.Name() could return a different
	// value from the one used in the configuration.
	for _, scorePlugin := range f.scorePlugins {
		if f.scorePluginWeight[scorePlugin.Name()] == 0 {
			return nil, fmt.Errorf("score plugin %q is not configured with weight", scorePlugin.Name())
		}
	}

	if options.captureProfile != nil {
		if len(outputProfile.PluginConfig) != 0 {
			sort.Slice(outputProfile.PluginConfig, func(i, j int) bool {
				return outputProfile.PluginConfig[i].Name < outputProfile.PluginConfig[j].Name
			})
		} else {
			outputProfile.PluginConfig = nil
		}
		options.captureProfile(outputProfile)
	}

	// Logs Enabled Plugins at each extension point, taking default plugins, given config, and multipoint into consideration
	logger.V(2).Info("the scheduler starts to work with those plugins", "Plugins", *f.ListPlugins())
	f.setInstrumentedPlugins()
	return f, nil
}

// setInstrumentedPlugins initializes instrumented plugins from current plugins that frameworkImpl has.
func (f *frameworkImpl) setInstrumentedPlugins() {
	// Cache metric streams for prefilter and filter plugins.
	for i, pl := range f.preFilterPlugins {
		f.preFilterPlugins[i] = &instrumentedPreFilterPlugin{
			PreFilterPlugin: f.preFilterPlugins[i],
			metric:          metrics.PluginEvaluationTotal.WithLabelValues(pl.Name(), metrics.PreFilter, f.profileName),
		}
	}
	for i, pl := range f.filterPlugins {
		f.filterPlugins[i] = &instrumentedFilterPlugin{
			FilterPlugin: f.filterPlugins[i],
			metric:       metrics.PluginEvaluationTotal.WithLabelValues(pl.Name(), metrics.Filter, f.profileName),
		}
	}

	// Cache metric streams for prescore and score plugins.
	for i, pl := range f.preScorePlugins {
		f.preScorePlugins[i] = &instrumentedPreScorePlugin{
			PreScorePlugin: f.preScorePlugins[i],
			metric:         metrics.PluginEvaluationTotal.WithLabelValues(pl.Name(), metrics.PreScore, f.profileName),
		}
	}
	for i, pl := range f.scorePlugins {
		f.scorePlugins[i] = &instrumentedScorePlugin{
			ScorePlugin: f.scorePlugins[i],
			metric:      metrics.PluginEvaluationTotal.WithLabelValues(pl.Name(), metrics.Score, f.profileName),
		}
	}
}

func (f *frameworkImpl) SetRegionNominator(n framework.RegionNominator) {
	f.RegionNominator = n
}

// Close closes each plugin, when they implement io.Closer interface.
func (f *frameworkImpl) Close() error {
	var errs []error
	for name, plugin := range f.pluginsMap {
		if closer, ok := plugin.(io.Closer); ok {
			err := closer.Close()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s failed to close: %w", name, err))
				// We try to close all plugins even if we got errors from some.
			}
		}
	}
	return errors.Join(errs...)
}

// getScoreWeights makes sure that, between MultiPoint-Score plugin weights and individual Score
// plugin weights there is not an overflow of MaxTotalScore.
func getScoreWeights(f *frameworkImpl, plugins []config.Plugin) error {
	var totalPriority int64
	scorePlugins := reflect.ValueOf(&f.scorePlugins).Elem()
	pluginType := scorePlugins.Type().Elem()
	for _, e := range plugins {
		pg := f.pluginsMap[e.Name]
		if !reflect.TypeOf(pg).Implements(pluginType) {
			continue
		}

		// We append MultiPoint plugins to the list of Score plugins. So if this plugin has already been
		// encountered, let the individual Score weight take precedence.
		if _, ok := f.scorePluginWeight[e.Name]; ok {
			continue
		}
		// a weight of zero is not permitted, plugins can be disabled explicitly
		// when configured.
		f.scorePluginWeight[e.Name] = int(e.Weight)
		if f.scorePluginWeight[e.Name] == 0 {
			f.scorePluginWeight[e.Name] = 1
		}

		// Checks totalPriority against MaxTotalScore to avoid overflow
		if int64(f.scorePluginWeight[e.Name])*framework.MaxRunnerScore > framework.MaxTotalScore-totalPriority {
			return fmt.Errorf("total score of Score plugins could overflow")
		}
		totalPriority += int64(f.scorePluginWeight[e.Name]) * framework.MaxRunnerScore
	}
	return nil
}

type orderedSet struct {
	set         map[string]int
	list        []string
	deletionCnt int
}

func newOrderedSet() *orderedSet {
	return &orderedSet{set: make(map[string]int)}
}

func (os *orderedSet) insert(s string) {
	if os.has(s) {
		return
	}
	os.set[s] = len(os.list)
	os.list = append(os.list, s)
}

func (os *orderedSet) has(s string) bool {
	_, found := os.set[s]
	return found
}

func (os *orderedSet) delete(s string) {
	if i, found := os.set[s]; found {
		delete(os.set, s)
		os.list = append(os.list[:i-os.deletionCnt], os.list[i+1-os.deletionCnt:]...)
		os.deletionCnt++
	}
}

func (f *frameworkImpl) expandMultiPointPlugins(logger klog.Logger, profile *config.SchedulerProfile) error {
	// initialize MultiPoint plugins
	for _, e := range f.getExtensionPoints(profile.Plugins) {
		plugins := reflect.ValueOf(e.slicePtr).Elem()
		pluginType := plugins.Type().Elem()
		// build enabledSet of plugins already registered via normal extension points
		// to check double registration
		enabledSet := newOrderedSet()
		for _, plugin := range e.plugins.Enabled {
			enabledSet.insert(plugin.Name)
		}

		disabledSet := sets.New[string]()
		for _, disabledPlugin := range e.plugins.Disabled {
			disabledSet.Insert(disabledPlugin.Name)
		}
		if disabledSet.Has("*") {
			logger.V(4).Info("Skipped MultiPoint expansion because all plugins are disabled for extension point", "extension", pluginType)
			continue
		}

		// track plugins enabled via multipoint separately from those enabled by specific extensions,
		// so that we can distinguish between double-registration and explicit overrides
		multiPointEnabled := newOrderedSet()
		overridePlugins := newOrderedSet()
		for _, ep := range profile.Plugins.MultiPoint.Enabled {
			pg, ok := f.pluginsMap[ep.Name]
			if !ok {
				return fmt.Errorf("%s %q does not exist", pluginType.Name(), ep.Name)
			}

			// if this plugin doesn't implement the type for the current extension we're trying to expand, skip
			if !reflect.TypeOf(pg).Implements(pluginType) {
				continue
			}

			// a plugin that's enabled via MultiPoint can still be disabled for specific extension points
			if disabledSet.Has(ep.Name) {
				logger.V(4).Info("Skipped disabled plugin for extension point", "plugin", ep.Name, "extension", pluginType)
				continue
			}

			// if this plugin has already been enabled by the specific extension point,
			// the user intent is to override the default plugin or make some other explicit setting.
			// Either way, discard the MultiPoint value for this plugin.
			// This maintains expected behavior for overriding default plugins (see https://github.com/kubernetes/kubernetes/pull/99582)
			if enabledSet.has(ep.Name) {
				overridePlugins.insert(ep.Name)
				logger.Info("MultiPoint plugin is explicitly re-configured; overriding", "plugin", ep.Name)
				continue
			}

			// if this plugin is already registered via MultiPoint, then this is
			// a double registration and an error in the config.
			if multiPointEnabled.has(ep.Name) {
				return fmt.Errorf("plugin %q already registered as %q", ep.Name, pluginType.Name())
			}

			// we only need to update the multipoint set, since we already have the specific extension set from above
			multiPointEnabled.insert(ep.Name)
		}

		// Reorder plugins. Here is the expected order:
		// - part 1: overridePlugins. Their order stay intact as how they're specified in regular extension point.
		// - part 2: multiPointEnabled - i.e., plugin defined in multipoint but not in regular extension point.
		// - part 3: other plugins (excluded by part 1 & 2) in regular extension point.
		newPlugins := reflect.New(reflect.TypeOf(e.slicePtr).Elem()).Elem()
		// part 1
		for _, name := range slice.CopyStrings(enabledSet.list) {
			if overridePlugins.has(name) {
				newPlugins = reflect.Append(newPlugins, reflect.ValueOf(f.pluginsMap[name]))
				enabledSet.delete(name)
			}
		}
		// part 2
		for _, name := range multiPointEnabled.list {
			newPlugins = reflect.Append(newPlugins, reflect.ValueOf(f.pluginsMap[name]))
		}
		// part 3
		for _, name := range enabledSet.list {
			newPlugins = reflect.Append(newPlugins, reflect.ValueOf(f.pluginsMap[name]))
		}
		plugins.Set(newPlugins)
	}
	return nil
}

func shouldHaveEnqueueExtensions(p framework.Plugin) bool {
	switch p.(type) {
	// Only PreEnqueue, PreFilter, Filter, Reserve, and Permit plugins can (should) have EnqueueExtensions.
	// See the comment of EnqueueExtensions for more detailed reason here.
	case framework.PreEnqueuePlugin, framework.PreFilterPlugin, framework.FilterPlugin, framework.ReservePlugin, framework.PermitPlugin:
		return true
	}
	return false
}

func (f *frameworkImpl) fillEnqueueExtensions(p framework.Plugin) {
	if !shouldHaveEnqueueExtensions(p) {
		// Ignore EnqueueExtensions from plugin which isn't PreEnqueue, PreFilter, Filter, Reserve, and Permit.
		return
	}

	ext, ok := p.(framework.EnqueueExtensions)
	if !ok {
		// If interface EnqueueExtensions is not implemented, register the default enqueue extensions
		// to the plugin because we don't know which events the plugin is interested in.
		// This is to ensure backward compatibility.
		f.enqueueExtensions = append(f.enqueueExtensions, &defaultEnqueueExtension{pluginName: p.Name()})
		return
	}

	f.enqueueExtensions = append(f.enqueueExtensions, ext)
}

// defaultEnqueueExtension is used when a plugin does not implement EnqueueExtensions interface.
type defaultEnqueueExtension struct {
	pluginName string
}

func (p *defaultEnqueueExtension) Name() string { return p.pluginName }
func (p *defaultEnqueueExtension) EventsToRegister() []framework.ClusterEventWithHint {
	// need to return all specific cluster events with framework.All action instead of wildcard event
	// because the returning values are used to register event handlers.
	// If we return the wildcard here, it won't affect the event handlers registered by the plugin
	// and some events may not be registered in the event handlers.
	return framework.UnrollWildCardResource()
}

func updatePluginList(pluginList interface{}, pluginSet config.PluginSet, pluginsMap map[string]framework.Plugin) error {
	plugins := reflect.ValueOf(pluginList).Elem()
	pluginType := plugins.Type().Elem()
	set := sets.New[string]()
	for _, ep := range pluginSet.Enabled {
		pg, ok := pluginsMap[ep.Name]
		if !ok {
			return fmt.Errorf("%s %q does not exist", pluginType.Name(), ep.Name)
		}

		if !reflect.TypeOf(pg).Implements(pluginType) {
			return fmt.Errorf("plugin %q does not extend %s plugin", ep.Name, pluginType.Name())
		}

		if set.Has(ep.Name) {
			return fmt.Errorf("plugin %q already registered as %q", ep.Name, pluginType.Name())
		}

		set.Insert(ep.Name)

		newPlugins := reflect.Append(plugins, reflect.ValueOf(pg))
		plugins.Set(newPlugins)
	}
	return nil
}

// PreEnqueuePlugins returns the registered preEnqueue plugins.
func (f *frameworkImpl) PreEnqueuePlugins() []framework.PreEnqueuePlugin {
	return f.preEnqueuePlugins
}

// EnqueueExtensions returns the registered reenqueue plugins.
func (f *frameworkImpl) EnqueueExtensions() []framework.EnqueueExtensions {
	return f.enqueueExtensions
}

// QueueSortFunc returns the function to sort regions in scheduling queue
func (f *frameworkImpl) QueueSortFunc() framework.LessFunc {
	if f == nil {
		// If frameworkImpl is nil, simply keep their order unchanged.
		// NOTE: this is primarily for tests.
		return func(_, _ *framework.QueuedRegionInfo) bool { return false }
	}

	if len(f.queueSortPlugins) == 0 {
		panic("No QueueSort plugin is registered in the frameworkImpl.")
	}

	// Only one QueueSort plugin can be enabled.
	return f.queueSortPlugins[0].Less
}

// RunPreFilterPlugins runs the set of configured PreFilter plugins. It returns
// *Status and its code is set to non-success if any of the plugins returns
// anything but Success/Skip.
// When it returns Skip status, returned PreFilterResult and other fields in status are just ignored,
// and coupled Filter plugin/PreFilterExtensions() will be skipped in this scheduling cycle.
// If a non-success status is returned, then the scheduling cycle is aborted.
func (f *frameworkImpl) RunPreFilterPlugins(ctx context.Context, state *framework.CycleState, region *corev1.Region) (_ *framework.PreFilterResult, status *framework.Status) {
	startTime := time.Now()
	skipPlugins := sets.New[string]()
	defer func() {
		state.SkipFilterPlugins = skipPlugins
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.PreFilter, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	var result *framework.PreFilterResult
	var pluginsWithRunners []string
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "PreFilter")
	}
	var returnStatus *framework.Status
	for _, pl := range f.preFilterPlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		r, s := f.runPreFilterPlugin(ctx, pl, state, region)
		if s.IsSkip() {
			skipPlugins.Insert(pl.Name())
			continue
		}
		if !s.IsSuccess() {
			s.SetPlugin(pl.Name())
			if s.Code() == framework.UnschedulableAndUnresolvable {
				// In this case, the preemption shouldn't happen in this scheduling cycle.
				// So, no need to execute all PreFilter.
				return nil, s
			}
			if s.Code() == framework.Unschedulable {
				// In this case, the preemption should happen later in this scheduling cycle.
				// So we need to execute all PreFilter.
				// https://github.com/kubernetes/kubernetes/issues/119770
				returnStatus = s
				continue
			}
			return nil, framework.AsStatus(fmt.Errorf("running PreFilter plugin %q: %w", pl.Name(), s.AsError())).WithPlugin(pl.Name())
		}
		if !r.AllRunners() {
			pluginsWithRunners = append(pluginsWithRunners, pl.Name())
		}
		result = result.Merge(r)
		if !result.AllRunners() && len(result.RunnerNames) == 0 {
			msg := fmt.Sprintf("runner(s) didn't satisfy plugin(s) %v simultaneously", pluginsWithRunners)
			if len(pluginsWithRunners) == 1 {
				msg = fmt.Sprintf("runner(s) didn't satisfy plugin %v", pluginsWithRunners[0])
			}

			// When PreFilterResult filters out Runners, the framework considers Runners that are filtered out as getting "UnschedulableAndUnresolvable".
			return result, framework.NewStatus(framework.UnschedulableAndUnresolvable, msg)
		}
	}
	return result, returnStatus
}

func (f *frameworkImpl) runPreFilterPlugin(ctx context.Context, pl framework.PreFilterPlugin, state *framework.CycleState, region *corev1.Region) (*framework.PreFilterResult, *framework.Status) {
	if !state.ShouldRecordPluginMetrics() {
		return pl.PreFilter(ctx, state, region)
	}
	startTime := time.Now()
	result, status := pl.PreFilter(ctx, state, region)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.PreFilter, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return result, status
}

// RunPreFilterExtensionAddRegion calls the AddRegion interface for the set of configured
// PreFilter plugins. It returns directly if any of the plugins return any
// status other than Success.
func (f *frameworkImpl) RunPreFilterExtensionAddRegion(
	ctx context.Context,
	state *framework.CycleState,
	regionToSchedule *corev1.Region,
	regionInfoToAdd *framework.RegionInfo,
	runnerInfo *framework.RunnerInfo,
) (status *framework.Status) {
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "PreFilterExtension")
	}
	for _, pl := range f.preFilterPlugins {
		if pl.PreFilterExtensions() == nil || state.SkipFilterPlugins.Has(pl.Name()) {
			continue
		}
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		status = f.runPreFilterExtensionAddRegion(ctx, pl, state, regionToSchedule, regionInfoToAdd, runnerInfo)
		if !status.IsSuccess() {
			err := status.AsError()
			logger.Error(err, "Plugin failed", "region", klog.KObj(regionToSchedule), "runner", klog.KObj(runnerInfo.Runner()), "operation", "addRegion", "plugin", pl.Name())
			return framework.AsStatus(fmt.Errorf("running AddRegion on PreFilter plugin %q: %w", pl.Name(), err))
		}
	}

	return nil
}

func (f *frameworkImpl) runPreFilterExtensionAddRegion(ctx context.Context, pl framework.PreFilterPlugin, state *framework.CycleState, regionToSchedule *corev1.Region, regionInfoToAdd *framework.RegionInfo, runnerInfo *framework.RunnerInfo) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return pl.PreFilterExtensions().AddRegion(ctx, state, regionToSchedule, regionInfoToAdd, runnerInfo)
	}
	startTime := time.Now()
	status := pl.PreFilterExtensions().AddRegion(ctx, state, regionToSchedule, regionInfoToAdd, runnerInfo)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.PreFilterExtensionAddRegion, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunPreFilterExtensionRemoveRegion calls the RemoveRegion interface for the set of configured
// PreFilter plugins. It returns directly if any of the plugins return any
// status other than Success.
func (f *frameworkImpl) RunPreFilterExtensionRemoveRegion(
	ctx context.Context,
	state *framework.CycleState,
	regionToSchedule *corev1.Region,
	regionInfoToRemove *framework.RegionInfo,
	runnerInfo *framework.RunnerInfo,
) (status *framework.Status) {
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "PreFilterExtension")
	}
	for _, pl := range f.preFilterPlugins {
		if pl.PreFilterExtensions() == nil || state.SkipFilterPlugins.Has(pl.Name()) {
			continue
		}
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		status = f.runPreFilterExtensionRemoveRegion(ctx, pl, state, regionToSchedule, regionInfoToRemove, runnerInfo)
		if !status.IsSuccess() {
			err := status.AsError()
			logger.Error(err, "Plugin failed", "runner", klog.KObj(runnerInfo.Runner()), "operation", "removeRegion", "plugin", pl.Name(), "region", klog.KObj(regionToSchedule))
			return framework.AsStatus(fmt.Errorf("running RemoveRegion on PreFilter plugin %q: %w", pl.Name(), err))
		}
	}

	return nil
}

func (f *frameworkImpl) runPreFilterExtensionRemoveRegion(ctx context.Context, pl framework.PreFilterPlugin, state *framework.CycleState, regionToSchedule *corev1.Region, regionInfoToRemove *framework.RegionInfo, runnerInfo *framework.RunnerInfo) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return pl.PreFilterExtensions().RemoveRegion(ctx, state, regionToSchedule, regionInfoToRemove, runnerInfo)
	}
	startTime := time.Now()
	status := pl.PreFilterExtensions().RemoveRegion(ctx, state, regionToSchedule, regionInfoToRemove, runnerInfo)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.PreFilterExtensionRemoveRegion, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunFilterPlugins runs the set of configured Filter plugins for region on
// the given runner. If any of these plugins doesn't return "Success", the
// given runner is not suitable for running region.
// Meanwhile, the failure message and status are set for the given runner.
func (f *frameworkImpl) RunFilterPlugins(
	ctx context.Context,
	state *framework.CycleState,
	region *corev1.Region,
	runnerInfo *framework.RunnerInfo,
) *framework.Status {
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "Filter")
	}

	for _, pl := range f.filterPlugins {
		if state.SkipFilterPlugins.Has(pl.Name()) {
			continue
		}
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		if status := f.runFilterPlugin(ctx, pl, state, region, runnerInfo); !status.IsSuccess() {
			if !status.IsRejected() {
				// Filter plugins are not supposed to return any status other than
				// Success or Unschedulable.
				status = framework.AsStatus(fmt.Errorf("running %q filter plugin: %w", pl.Name(), status.AsError()))
			}
			status.SetPlugin(pl.Name())
			return status
		}
	}

	return nil
}

func (f *frameworkImpl) runFilterPlugin(ctx context.Context, pl framework.FilterPlugin, state *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return pl.Filter(ctx, state, region, runnerInfo)
	}
	startTime := time.Now()
	status := pl.Filter(ctx, state, region, runnerInfo)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.Filter, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunPostFilterPlugins runs the set of configured PostFilter plugins until the first
// Success, Error or UnschedulableAndUnresolvable is met; otherwise continues to execute all plugins.
func (f *frameworkImpl) RunPostFilterPlugins(ctx context.Context, state *framework.CycleState, region *corev1.Region, filteredRunnerStatusMap framework.RunnerToStatusMap) (_ *framework.PostFilterResult, status *framework.Status) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.PostFilter, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()

	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "PostFilter")
	}

	// `result` records the last meaningful(non-noop) PostFilterResult.
	var result *framework.PostFilterResult
	var reasons []string
	var rejectorPlugin string
	for _, pl := range f.postFilterPlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		r, s := f.runPostFilterPlugin(ctx, pl, state, region, filteredRunnerStatusMap)
		if s.IsSuccess() {
			return r, s
		} else if s.Code() == framework.UnschedulableAndUnresolvable {
			return r, s.WithPlugin(pl.Name())
		} else if !s.IsRejected() {
			// Any status other than Success, Unschedulable or UnschedulableAndUnresolvable is Error.
			return nil, framework.AsStatus(s.AsError()).WithPlugin(pl.Name())
		} else if r != nil && r.Mode() != framework.ModeNoop {
			result = r
		}

		reasons = append(reasons, s.Reasons()...)
		// Record the first failed plugin unless we proved that
		// the latter is more relevant.
		if len(rejectorPlugin) == 0 {
			rejectorPlugin = pl.Name()
		}
	}

	return result, framework.NewStatus(framework.Unschedulable, reasons...).WithPlugin(rejectorPlugin)
}

func (f *frameworkImpl) runPostFilterPlugin(ctx context.Context, pl framework.PostFilterPlugin, state *framework.CycleState, region *corev1.Region, filteredRunnerStatusMap framework.RunnerToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	if !state.ShouldRecordPluginMetrics() {
		return pl.PostFilter(ctx, state, region, filteredRunnerStatusMap)
	}
	startTime := time.Now()
	r, s := pl.PostFilter(ctx, state, region, filteredRunnerStatusMap)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.PostFilter, pl.Name(), s.Code().String(), metrics.SinceInSeconds(startTime))
	return r, s
}

// RunFilterPluginsWithNominatedRegions runs the set of configured filter plugins
// for nominated region on the given runner.
// This function is called from two different places: Schedule and Preempt.
// When it is called from Schedule, we want to test whether the region is
// schedulable on the runner with all the existing regions on the runner plus higher
// and equal priority regions nominated to run on the runner.
// When it is called from Preempt, we should remove the victims of preemption
// and add the nominated regions. Removal of the victims is done by
// SelectVictimsOnRunner(). Preempt removes victims from PreFilter state and
// RunnerInfo before calling this function.
func (f *frameworkImpl) RunFilterPluginsWithNominatedRegions(ctx context.Context, state *framework.CycleState, region *corev1.Region, info *framework.RunnerInfo) *framework.Status {
	var status *framework.Status

	regionsAdded := false
	// We run filters twice in some cases. If the runner has greater or equal priority
	// nominated regions, we run them when those regions are added to PreFilter state and runnerInfo.
	// If all filters succeed in this pass, we run them again when these
	// nominated regions are not added. This second pass is necessary because some
	// filters such as inter-region affinity may not pass without the nominated regions.
	// If there are no nominated regions for the runner or if the first run of the
	// filters fail, we don't run the second pass.
	// We consider only equal or higher priority regions in the first pass, because
	// those are the current "region" must yield to them and not take a space opened
	// for running them. It is ok if the current "region" take resources freed for
	// lower priority regions.
	// Requiring that the new region is schedulable in both circumstances ensures that
	// we are making a conservative decision: filters like resources and inter-region
	// anti-affinity are more likely to fail when the nominated regions are treated
	// as running, while filters like region affinity are more likely to fail when
	// the nominated regions are treated as not running. We can't just assume the
	// nominated regions are running because they are not running right now and in fact,
	// they may end up getting scheduled to a different runner.
	logger := klog.FromContext(ctx)
	logger = klog.LoggerWithName(logger, "FilterWithNominatedRegions")
	ctx = klog.NewContext(ctx, logger)
	for i := 0; i < 2; i++ {
		stateToUse := state
		runnerInfoToUse := info
		if i == 0 {
			var err error
			regionsAdded, stateToUse, runnerInfoToUse, err = addNominatedRegions(ctx, f, region, state, info)
			if err != nil {
				return framework.AsStatus(err)
			}
		} else if !regionsAdded || !status.IsSuccess() {
			break
		}

		status = f.RunFilterPlugins(ctx, stateToUse, region, runnerInfoToUse)
		if !status.IsSuccess() && !status.IsRejected() {
			return status
		}
	}

	return status
}

// addNominatedRegions adds regions with equal or greater priority which are nominated
// to run on the runner. It returns 1) whether any region was added, 2) augmented cycleState,
// 3) augmented runnerInfo.
func addNominatedRegions(ctx context.Context, fh framework.Handle, region *corev1.Region, state *framework.CycleState, runnerInfo *framework.RunnerInfo) (bool, *framework.CycleState, *framework.RunnerInfo, error) {
	if fh == nil {
		// This may happen only in tests.
		return false, state, runnerInfo, nil
	}
	nominatedRegionInfos := fh.NominatedRegionsForRunner(runnerInfo.Runner().Name)
	if len(nominatedRegionInfos) == 0 {
		return false, state, runnerInfo, nil
	}
	runnerInfoOut := runnerInfo.Snapshot()
	stateOut := state.Clone()
	regionsAdded := false
	for _, pi := range nominatedRegionInfos {
		if scheudulingcorev1.RegionPriority(pi.Region) >= scheudulingcorev1.RegionPriority(region) && pi.Region.UID != region.UID {
			runnerInfoOut.AddRegionInfo(pi)
			status := fh.RunPreFilterExtensionAddRegion(ctx, stateOut, region, pi, runnerInfoOut)
			if !status.IsSuccess() {
				return false, state, runnerInfo, status.AsError()
			}
			regionsAdded = true
		}
	}
	return regionsAdded, stateOut, runnerInfoOut, nil
}

// RunPreScorePlugins runs the set of configured pre-score plugins. If any
// of these plugins returns any status other than Success/Skip, the given region is rejected.
// When it returns Skip status, other fields in status are just ignored,
// and coupled Score plugin will be skipped in this scheduling cycle.
func (f *frameworkImpl) RunPreScorePlugins(
	ctx context.Context,
	state *framework.CycleState,
	region *corev1.Region,
	runners []*framework.RunnerInfo,
) (status *framework.Status) {
	startTime := time.Now()
	skipPlugins := sets.New[string]()
	defer func() {
		state.SkipScorePlugins = skipPlugins
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.PreScore, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "PreScore")
	}
	for _, pl := range f.preScorePlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		status = f.runPreScorePlugin(ctx, pl, state, region, runners)
		if status.IsSkip() {
			skipPlugins.Insert(pl.Name())
			continue
		}
		if !status.IsSuccess() {
			return framework.AsStatus(fmt.Errorf("running PreScore plugin %q: %w", pl.Name(), status.AsError()))
		}
	}
	return nil
}

func (f *frameworkImpl) runPreScorePlugin(ctx context.Context, pl framework.PreScorePlugin, state *framework.CycleState, region *corev1.Region, runners []*framework.RunnerInfo) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return pl.PreScore(ctx, state, region, runners)
	}
	startTime := time.Now()
	status := pl.PreScore(ctx, state, region, runners)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.PreScore, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunScorePlugins runs the set of configured scoring plugins.
// It returns a list that stores scores from each plugin and total score for each Runner.
// It also returns *Status, which is set to non-success if any of the plugins returns
// a non-success status.
func (f *frameworkImpl) RunScorePlugins(ctx context.Context, state *framework.CycleState, region *corev1.Region, runners []*framework.RunnerInfo) (ns []framework.RunnerPluginScores, status *framework.Status) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.Score, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	allRunnerPluginScores := make([]framework.RunnerPluginScores, len(runners))
	numPlugins := len(f.scorePlugins)
	plugins := make([]framework.ScorePlugin, 0, numPlugins)
	pluginToRunnerScores := make(map[string]framework.RunnerScoreList, numPlugins)
	for _, pl := range f.scorePlugins {
		if state.SkipScorePlugins.Has(pl.Name()) {
			continue
		}
		plugins = append(plugins, pl)
		pluginToRunnerScores[pl.Name()] = make(framework.RunnerScoreList, len(runners))
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := parallelize.NewErrorChannel()

	if len(plugins) > 0 {
		logger := klog.FromContext(ctx)
		verboseLogs := logger.V(4).Enabled()
		if verboseLogs {
			logger = klog.LoggerWithName(logger, "Score")
		}
		// Run Score method for each runner in parallel.
		f.Parallelizer().Until(ctx, len(runners), func(index int) {
			runnerName := runners[index].Runner().Name
			logger := logger
			if verboseLogs {
				logger = klog.LoggerWithValues(logger, "runner", klog.ObjectRef{Name: runnerName})
			}
			for _, pl := range plugins {
				ctx := ctx
				if verboseLogs {
					logger := klog.LoggerWithName(logger, pl.Name())
					ctx = klog.NewContext(ctx, logger)
				}
				s, status := f.runScorePlugin(ctx, pl, state, region, runnerName)
				if !status.IsSuccess() {
					err := fmt.Errorf("plugin %q failed with: %w", pl.Name(), status.AsError())
					errCh.SendErrorWithCancel(err, cancel)
					return
				}
				pluginToRunnerScores[pl.Name()][index] = framework.RunnerScore{
					Name:  runnerName,
					Score: s,
				}
			}
		}, metrics.Score)
		if err := errCh.ReceiveError(); err != nil {
			return nil, framework.AsStatus(fmt.Errorf("running Score plugins: %w", err))
		}
	}

	// Run NormalizeScore method for each ScorePlugin in parallel.
	f.Parallelizer().Until(ctx, len(plugins), func(index int) {
		pl := plugins[index]
		if pl.ScoreExtensions() == nil {
			return
		}
		runnerScoreList := pluginToRunnerScores[pl.Name()]
		status := f.runScoreExtension(ctx, pl, state, region, runnerScoreList)
		if !status.IsSuccess() {
			err := fmt.Errorf("plugin %q failed with: %w", pl.Name(), status.AsError())
			errCh.SendErrorWithCancel(err, cancel)
			return
		}
	}, metrics.Score)
	if err := errCh.ReceiveError(); err != nil {
		return nil, framework.AsStatus(fmt.Errorf("running Normalize on Score plugins: %w", err))
	}

	// Apply score weight for each ScorePlugin in parallel,
	// and then, build allRunnerPluginScores.
	f.Parallelizer().Until(ctx, len(runners), func(index int) {
		runnerPluginScores := framework.RunnerPluginScores{
			Name:   runners[index].Runner().Name,
			Scores: make([]framework.PluginScore, len(plugins)),
		}

		for i, pl := range plugins {
			weight := f.scorePluginWeight[pl.Name()]
			runnerScoreList := pluginToRunnerScores[pl.Name()]
			score := runnerScoreList[index].Score

			if score > framework.MaxRunnerScore || score < framework.MinRunnerScore {
				err := fmt.Errorf("plugin %q returns an invalid score %v, it should in the range of [%v, %v] after normalizing", pl.Name(), score, framework.MinRunnerScore, framework.MaxRunnerScore)
				errCh.SendErrorWithCancel(err, cancel)
				return
			}
			weightedScore := score * int64(weight)
			runnerPluginScores.Scores[i] = framework.PluginScore{
				Name:  pl.Name(),
				Score: weightedScore,
			}
			runnerPluginScores.TotalScore += weightedScore
		}
		allRunnerPluginScores[index] = runnerPluginScores
	}, metrics.Score)
	if err := errCh.ReceiveError(); err != nil {
		return nil, framework.AsStatus(fmt.Errorf("applying score defaultWeights on Score plugins: %w", err))
	}

	return allRunnerPluginScores, nil
}

func (f *frameworkImpl) runScorePlugin(ctx context.Context, pl framework.ScorePlugin, state *framework.CycleState, region *corev1.Region, runnerName string) (int64, *framework.Status) {
	if !state.ShouldRecordPluginMetrics() {
		return pl.Score(ctx, state, region, runnerName)
	}
	startTime := time.Now()
	s, status := pl.Score(ctx, state, region, runnerName)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.Score, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return s, status
}

func (f *frameworkImpl) runScoreExtension(ctx context.Context, pl framework.ScorePlugin, state *framework.CycleState, region *corev1.Region, runnerScoreList framework.RunnerScoreList) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return pl.ScoreExtensions().NormalizeScore(ctx, state, region, runnerScoreList)
	}
	startTime := time.Now()
	status := pl.ScoreExtensions().NormalizeScore(ctx, state, region, runnerScoreList)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.ScoreExtensionNormalize, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunPreBindPlugins runs the set of configured prebind plugins. It returns a
// failure (bool) if any of the plugins returns an error. It also returns an
// error containing the rejection message or the error occurred in the plugin.
func (f *frameworkImpl) RunPreBindPlugins(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (status *framework.Status) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.PreBind, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "PreBind")
		logger = klog.LoggerWithValues(logger, "runner", klog.ObjectRef{Name: runnerName})
	}
	for _, pl := range f.preBindPlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		status = f.runPreBindPlugin(ctx, pl, state, region, runnerName)
		if !status.IsSuccess() {
			if status.IsRejected() {
				logger.V(4).Info("Region rejected by PreBind plugin", "region", klog.KObj(region), "runner", runnerName, "plugin", pl.Name(), "status", status.Message())
				status.SetPlugin(pl.Name())
				return status
			}
			err := status.AsError()
			logger.Error(err, "Plugin failed", "plugin", pl.Name(), "region", klog.KObj(region), "runner", runnerName)
			return framework.AsStatus(fmt.Errorf("running PreBind plugin %q: %w", pl.Name(), err))
		}
	}
	return nil
}

func (f *frameworkImpl) runPreBindPlugin(ctx context.Context, pl framework.PreBindPlugin, state *framework.CycleState, region *corev1.Region, runnerName string) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return pl.PreBind(ctx, state, region, runnerName)
	}
	startTime := time.Now()
	status := pl.PreBind(ctx, state, region, runnerName)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.PreBind, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunBindPlugins runs the set of configured bind plugins until one returns a non `Skip` status.
func (f *frameworkImpl) RunBindPlugins(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (status *framework.Status) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.Bind, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	if len(f.bindPlugins) == 0 {
		return framework.NewStatus(framework.Skip, "")
	}
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "Bind")
	}
	for _, pl := range f.bindPlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		status = f.runBindPlugin(ctx, pl, state, region, runnerName)
		if status.IsSkip() {
			continue
		}
		if !status.IsSuccess() {
			if status.IsRejected() {
				logger.V(4).Info("Region rejected by Bind plugin", "region", klog.KObj(region), "runner", runnerName, "plugin", pl.Name(), "status", status.Message())
				status.SetPlugin(pl.Name())
				return status
			}
			err := status.AsError()
			logger.Error(err, "Plugin Failed", "plugin", pl.Name(), "region", klog.KObj(region), "runner", runnerName)
			return framework.AsStatus(fmt.Errorf("running Bind plugin %q: %w", pl.Name(), err))
		}
		return status
	}
	return status
}

func (f *frameworkImpl) runBindPlugin(ctx context.Context, bp framework.BindPlugin, state *framework.CycleState, region *corev1.Region, runnerName string) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return bp.Bind(ctx, state, region, runnerName)
	}
	startTime := time.Now()
	status := bp.Bind(ctx, state, region, runnerName)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.Bind, bp.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunPostBindPlugins runs the set of configured postbind plugins.
func (f *frameworkImpl) RunPostBindPlugins(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.PostBind, framework.Success.String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "PostBind")
	}
	for _, pl := range f.postBindPlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		f.runPostBindPlugin(ctx, pl, state, region, runnerName)
	}
}

func (f *frameworkImpl) runPostBindPlugin(ctx context.Context, pl framework.PostBindPlugin, state *framework.CycleState, region *corev1.Region, runnerName string) {
	if !state.ShouldRecordPluginMetrics() {
		pl.PostBind(ctx, state, region, runnerName)
		return
	}
	startTime := time.Now()
	pl.PostBind(ctx, state, region, runnerName)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.PostBind, pl.Name(), framework.Success.String(), metrics.SinceInSeconds(startTime))
}

// RunReservePluginsReserve runs the Reserve method in the set of configured
// reserve plugins. If any of these plugins returns an error, it does not
// continue running the remaining ones and returns the error. In such a case,
// the region will not be scheduled and the caller will be expected to call
// RunReservePluginsUnreserve.
func (f *frameworkImpl) RunReservePluginsReserve(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (status *framework.Status) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.Reserve, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "Reserve")
		logger = klog.LoggerWithValues(logger, "runner", klog.ObjectRef{Name: runnerName})
	}
	for _, pl := range f.reservePlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		status = f.runReservePluginReserve(ctx, pl, state, region, runnerName)
		if !status.IsSuccess() {
			if status.IsRejected() {
				logger.V(4).Info("Region rejected by plugin", "region", klog.KObj(region), "plugin", pl.Name(), "status", status.Message())
				status.SetPlugin(pl.Name())
				return status
			}
			err := status.AsError()
			logger.Error(err, "Plugin failed", "plugin", pl.Name(), "region", klog.KObj(region))
			return framework.AsStatus(fmt.Errorf("running Reserve plugin %q: %w", pl.Name(), err))
		}
	}
	return nil
}

func (f *frameworkImpl) runReservePluginReserve(ctx context.Context, pl framework.ReservePlugin, state *framework.CycleState, region *corev1.Region, runnerName string) *framework.Status {
	if !state.ShouldRecordPluginMetrics() {
		return pl.Reserve(ctx, state, region, runnerName)
	}
	startTime := time.Now()
	status := pl.Reserve(ctx, state, region, runnerName)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.Reserve, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status
}

// RunReservePluginsUnreserve runs the Unreserve method in the set of
// configured reserve plugins.
func (f *frameworkImpl) RunReservePluginsUnreserve(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.Unreserve, framework.Success.String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	// Execute the Unreserve operation of each reserve plugin in the
	// *reverse* order in which the Reserve operation was executed.
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "Unreserve")
		logger = klog.LoggerWithValues(logger, "runner", klog.ObjectRef{Name: runnerName})
	}
	for i := len(f.reservePlugins) - 1; i >= 0; i-- {
		pl := f.reservePlugins[i]
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		f.runReservePluginUnreserve(ctx, pl, state, region, runnerName)
	}
}

func (f *frameworkImpl) runReservePluginUnreserve(ctx context.Context, pl framework.ReservePlugin, state *framework.CycleState, region *corev1.Region, runnerName string) {
	if !state.ShouldRecordPluginMetrics() {
		pl.Unreserve(ctx, state, region, runnerName)
		return
	}
	startTime := time.Now()
	pl.Unreserve(ctx, state, region, runnerName)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.Unreserve, pl.Name(), framework.Success.String(), metrics.SinceInSeconds(startTime))
}

// RunPermitPlugins runs the set of configured permit plugins. If any of these
// plugins returns a status other than "Success" or "Wait", it does not continue
// running the remaining plugins and returns an error. Otherwise, if any of the
// plugins returns "Wait", then this function will create and add waiting region
// to a map of currently waiting regions and return status with "Wait" code.
// Region will remain waiting region for the minimum duration returned by the permit plugins.
func (f *frameworkImpl) RunPermitPlugins(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (status *framework.Status) {
	startTime := time.Now()
	defer func() {
		metrics.FrameworkExtensionPointDuration.WithLabelValues(metrics.Permit, status.Code().String(), f.profileName).Observe(metrics.SinceInSeconds(startTime))
	}()
	pluginsWaitTime := make(map[string]time.Duration)
	statusCode := framework.Success
	logger := klog.FromContext(ctx)
	verboseLogs := logger.V(4).Enabled()
	if verboseLogs {
		logger = klog.LoggerWithName(logger, "Permit")
		logger = klog.LoggerWithValues(logger, "runner", klog.ObjectRef{Name: runnerName})
	}
	for _, pl := range f.permitPlugins {
		ctx := ctx
		if verboseLogs {
			logger := klog.LoggerWithName(logger, pl.Name())
			ctx = klog.NewContext(ctx, logger)
		}
		status, timeout := f.runPermitPlugin(ctx, pl, state, region, runnerName)
		if !status.IsSuccess() {
			if status.IsRejected() {
				logger.V(4).Info("Region rejected by plugin", "region", klog.KObj(region), "plugin", pl.Name(), "status", status.Message())
				return status.WithPlugin(pl.Name())
			}
			if status.IsWait() {
				// Not allowed to be greater than maxTimeout.
				if timeout > maxTimeout {
					timeout = maxTimeout
				}
				pluginsWaitTime[pl.Name()] = timeout
				statusCode = framework.Wait
			} else {
				err := status.AsError()
				logger.Error(err, "Plugin failed", "plugin", pl.Name(), "region", klog.KObj(region))
				return framework.AsStatus(fmt.Errorf("running Permit plugin %q: %w", pl.Name(), err)).WithPlugin(pl.Name())
			}
		}
	}
	if statusCode == framework.Wait {
		waitingRegion := newWaitingRegion(region, pluginsWaitTime)
		f.waitingRegions.add(waitingRegion)
		msg := fmt.Sprintf("one or more plugins asked to wait and no plugin rejected region %q", region.Name)
		logger.V(4).Info("One or more plugins asked to wait and no plugin rejected region", "region", klog.KObj(region))
		return framework.NewStatus(framework.Wait, msg)
	}
	return nil
}

func (f *frameworkImpl) runPermitPlugin(ctx context.Context, pl framework.PermitPlugin, state *framework.CycleState, region *corev1.Region, runnerName string) (*framework.Status, time.Duration) {
	if !state.ShouldRecordPluginMetrics() {
		return pl.Permit(ctx, state, region, runnerName)
	}
	startTime := time.Now()
	status, timeout := pl.Permit(ctx, state, region, runnerName)
	f.metricsRecorder.ObservePluginDurationAsync(metrics.Permit, pl.Name(), status.Code().String(), metrics.SinceInSeconds(startTime))
	return status, timeout
}

// WaitOnPermit will block, if the region is a waiting region, until the waiting region is rejected or allowed.
func (f *frameworkImpl) WaitOnPermit(ctx context.Context, region *corev1.Region) *framework.Status {
	waitingRegion := f.waitingRegions.get(region.UID)
	if waitingRegion == nil {
		return nil
	}
	defer f.waitingRegions.remove(region.UID)

	logger := klog.FromContext(ctx)
	logger.V(4).Info("Region waiting on permit", "region", klog.KObj(region))

	startTime := time.Now()
	s := <-waitingRegion.s
	metrics.PermitWaitDuration.WithLabelValues(s.Code().String()).Observe(metrics.SinceInSeconds(startTime))

	if !s.IsSuccess() {
		if s.IsRejected() {
			logger.V(4).Info("Region rejected while waiting on permit", "region", klog.KObj(region), "status", s.Message())
			return s
		}
		err := s.AsError()
		logger.Error(err, "Failed waiting on permit for region", "region", klog.KObj(region))
		return framework.AsStatus(fmt.Errorf("waiting on permit for region: %w", err)).WithPlugin(s.Plugin())
	}
	return nil
}

// SnapshotSharedLister returns the scheduler's SharedLister of the latest RunnerInfo
// snapshot. The snapshot is taken at the beginning of a scheduling cycle and remains
// unchanged until a region finishes "Reserve". There is no guarantee that the information
// remains unchanged after "Reserve".
func (f *frameworkImpl) SnapshotSharedLister() framework.SharedLister {
	return f.snapshotSharedLister
}

// IterateOverWaitingRegions acquires a read lock and iterates over the WaitingRegions map.
func (f *frameworkImpl) IterateOverWaitingRegions(callback func(framework.WaitingRegion)) {
	f.waitingRegions.iterate(callback)
}

// GetWaitingRegion returns a reference to a WaitingRegion given its UID.
func (f *frameworkImpl) GetWaitingRegion(uid types.UID) framework.WaitingRegion {
	if wp := f.waitingRegions.get(uid); wp != nil {
		return wp
	}
	return nil // Returning nil instead of *waitingRegion(nil).
}

// RejectWaitingRegion rejects a WaitingRegion given its UID.
// The returned value indicates if the given region is waiting or not.
func (f *frameworkImpl) RejectWaitingRegion(uid types.UID) bool {
	if waitingRegion := f.waitingRegions.get(uid); waitingRegion != nil {
		waitingRegion.Reject("", "removed")
		return true
	}
	return false
}

// HasFilterPlugins returns true if at least one filter plugin is defined.
func (f *frameworkImpl) HasFilterPlugins() bool {
	return len(f.filterPlugins) > 0
}

// HasPostFilterPlugins returns true if at least one postFilter plugin is defined.
func (f *frameworkImpl) HasPostFilterPlugins() bool {
	return len(f.postFilterPlugins) > 0
}

// HasScorePlugins returns true if at least one score plugin is defined.
func (f *frameworkImpl) HasScorePlugins() bool {
	return len(f.scorePlugins) > 0
}

// ListPlugins returns a map of extension point name to plugin names configured at each extension
// point. Returns nil if no plugins where configured.
func (f *frameworkImpl) ListPlugins() *config.Plugins {
	m := config.Plugins{}

	for _, e := range f.getExtensionPoints(&m) {
		plugins := reflect.ValueOf(e.slicePtr).Elem()
		extName := plugins.Type().Elem().Name()
		var cfgs []config.Plugin
		for i := 0; i < plugins.Len(); i++ {
			name := plugins.Index(i).Interface().(framework.Plugin).Name()
			p := config.Plugin{Name: name}
			if extName == "ScorePlugin" {
				// Weights apply only to score plugins.
				p.Weight = int32(f.scorePluginWeight[name])
			}
			cfgs = append(cfgs, p)
		}
		if len(cfgs) > 0 {
			e.plugins.Enabled = cfgs
		}
	}
	return &m
}

// ClientSet returns a kubernetes clientset.
func (f *frameworkImpl) ClientSet() clientset.Interface {
	return f.clientSet
}

// OliveConfig returns a olive config.
func (f *frameworkImpl) OliveConfig() *restclient.Config {
	return f.kubeConfig
}

// EventRecorder returns an event recorder.
func (f *frameworkImpl) EventRecorder() events.EventRecorder {
	return f.eventRecorder
}

// SharedInformerFactory returns a shared informer factory.
func (f *frameworkImpl) SharedInformerFactory() informers.SharedInformerFactory {
	return f.informerFactory
}

func (f *frameworkImpl) pluginsNeeded(plugins *config.Plugins) sets.Set[string] {
	pgSet := sets.Set[string]{}

	if plugins == nil {
		return pgSet
	}

	find := func(pgs *config.PluginSet) {
		for _, pg := range pgs.Enabled {
			pgSet.Insert(pg.Name)
		}
	}

	for _, e := range f.getExtensionPoints(plugins) {
		find(e.plugins)
	}
	// Parse MultiPoint separately since they are not returned by f.getExtensionPoints()
	find(&plugins.MultiPoint)

	return pgSet
}

// ProfileName returns the profile name associated to this framework.
func (f *frameworkImpl) ProfileName() string {
	return f.profileName
}

// PercentageOfRunnersToScore returns percentageOfRunnersToScore associated to a profile.
func (f *frameworkImpl) PercentageOfRunnersToScore() *int32 {
	return f.percentageOfRunnersToScore
}

// Parallelizer returns a parallelizer holding parallelism for scheduler.
func (f *frameworkImpl) Parallelizer() parallelize.Parallelizer {
	return f.parallelizer
}
