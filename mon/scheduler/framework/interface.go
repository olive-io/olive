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

// This file defines the scheduling framework plugin interfaces.

package framework

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"

	config "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	"github.com/olive-io/olive/mon/scheduler/framework/parallelize"
)

// RunnerScoreList declares a list of runners and their scores.
type RunnerScoreList []RunnerScore

// RunnerScore is a struct with runner name and score.
type RunnerScore struct {
	Name  string
	Score int64
}

// RunnerToStatusMap declares map from runner name to its status.
type RunnerToStatusMap map[string]*Status

// RunnerPluginScores is a struct with runner name and scores for that runner.
type RunnerPluginScores struct {
	// Name is runner name.
	Name string
	// Scores is scores from plugins and extenders.
	Scores []PluginScore
	// TotalScore is the total score in Scores.
	TotalScore int64
}

// PluginScore is a struct with plugin/extender name and score.
type PluginScore struct {
	// Name is the name of plugin or extender.
	Name  string
	Score int64
}

// Code is the Status code/type which is returned from plugins.
type Code int

// These are predefined codes used in a Status.
// Note: when you add a new status, you have to add it in `codes` slice below.
const (
	// Success means that plugin ran correctly and found region schedulable.
	// NOTE: A nil status is also considered as "Success".
	Success Code = iota
	// Error is one of the failures, used for internal plugin errors, unexpected input, etc.
	// Plugin shouldn't return this code for expected failures, like Unschedulable.
	// Since it's the unexpected failure, the scheduling queue registers the region without unschedulable plugins.
	// Meaning, the Region will be requeued to activeQ/backoffQ soon.
	Error
	// Unschedulable is one of the failures, used when a plugin finds a region unschedulable.
	// If it's returned from PreFilter or Filter, the scheduler might attempt to
	// run other postFilter plugins like preemption to get this region scheduled.
	// Use UnschedulableAndUnresolvable to make the scheduler skipping other postFilter plugins.
	// The accompanying status message should explain why the region is unschedulable.
	//
	// We regard the backoff as a penalty of wasting the scheduling cycle.
	// When the scheduling queue requeues Regions, which was rejected with Unschedulable in the last scheduling,
	// the Region goes through backoff.
	Unschedulable
	// UnschedulableAndUnresolvable is used when a plugin finds a region unschedulable and
	// other postFilter plugins like preemption would not change anything.
	// See the comment on PostFilter interface for more details about how PostFilter should handle this status.
	// Plugins should return Unschedulable if it is possible that the region can get scheduled
	// after running other postFilter plugins.
	// The accompanying status message should explain why the region is unschedulable.
	//
	// We regard the backoff as a penalty of wasting the scheduling cycle.
	// When the scheduling queue requeues Regions, which was rejected with UnschedulableAndUnresolvable in the last scheduling,
	// the Region goes through backoff.
	UnschedulableAndUnresolvable
	// Wait is used when a Permit plugin finds a region scheduling should wait.
	Wait
	// Skip is used in the following scenarios:
	// - when a Bind plugin chooses to skip binding.
	// - when a PreFilter plugin returns Skip so that coupled Filter plugin/PreFilterExtensions() will be skipped.
	// - when a PreScore plugin returns Skip so that coupled Score plugin will be skipped.
	Skip
	// Pending means that the scheduling process is finished successfully,
	// but the plugin wants to stop the scheduling cycle/binding cycle here.
	//
	// For example, the DRA plugin sometimes needs to wait for the external device driver
	// to provision the resource for the Region.
	// It's different from when to return Unschedulable/UnschedulableAndUnresolvable,
	// because in this case, the scheduler decides where the Region can go successfully,
	// but we need to wait for the external component to do something based on that scheduling result.
	//
	// We regard the backoff as a penalty of wasting the scheduling cycle.
	// In the case of returning Pending, we cannot say the scheduling cycle is wasted
	// because the scheduling result is used to proceed the Region's scheduling forward,
	// that particular scheduling cycle is failed though.
	// So, Regions rejected by such reasons don't need to suffer a penalty (backoff).
	// When the scheduling queue requeues Regions, which was rejected with Pending in the last scheduling,
	// the Region goes to activeQ directly ignoring backoff.
	Pending
)

// This list should be exactly the same as the codes iota defined above in the same order.
var codes = []string{"Success", "Error", "Unschedulable", "UnschedulableAndUnresolvable", "Wait", "Skip", "Pending"}

func (c Code) String() string {
	return codes[c]
}

const (
	// MaxRunnerScore is the maximum score a Score plugin is expected to return.
	MaxRunnerScore int64 = 100

	// MinRunnerScore is the minimum score a Score plugin is expected to return.
	MinRunnerScore int64 = 0

	// MaxTotalScore is the maximum total score.
	MaxTotalScore int64 = math.MaxInt64
)

// RegionsToActivateKey is a reserved state key for stashing regions.
// If the stashed regions are present in unschedulableRegions or backoffQ，they will be
// activated (i.e., moved to activeQ) in two phases:
// - end of a scheduling cycle if it succeeds (will be cleared from `RegionsToActivate` if activated)
// - end of a binding cycle if it succeeds
var RegionsToActivateKey StateKey = "olive.io/regions-to-activate"

// RegionsToActivate stores regions to be activated.
type RegionsToActivate struct {
	sync.Mutex
	// Map is keyed with namespaced region name, and valued with the region.
	Map map[string]*corev1.Region
}

// Clone just returns the same state.
func (s *RegionsToActivate) Clone() StateData {
	return s
}

// NewRegionsToActivate instantiates a RegionsToActivate object.
func NewRegionsToActivate() *RegionsToActivate {
	return &RegionsToActivate{Map: make(map[string]*corev1.Region)}
}

// Status indicates the result of running a plugin. It consists of a code, a
// message, (optionally) an error, and a plugin name it fails by.
// When the status code is not Success, the reasons should explain why.
// And, when code is Success, all the other fields should be empty.
// NOTE: A nil Status is also considered as Success.
type Status struct {
	code    Code
	reasons []string
	err     error
	// plugin is an optional field that records the plugin name causes this status.
	// It's set by the framework when code is Unschedulable, UnschedulableAndUnresolvable or Pending.
	plugin string
}

func (s *Status) WithError(err error) *Status {
	s.err = err
	return s
}

// Code returns code of the Status.
func (s *Status) Code() Code {
	if s == nil {
		return Success
	}
	return s.code
}

// Message returns a concatenated message on reasons of the Status.
func (s *Status) Message() string {
	if s == nil {
		return ""
	}
	return strings.Join(s.Reasons(), ", ")
}

// SetPlugin sets the given plugin name to s.plugin.
func (s *Status) SetPlugin(plugin string) {
	s.plugin = plugin
}

// WithPlugin sets the given plugin name to s.plugin,
// and returns the given status object.
func (s *Status) WithPlugin(plugin string) *Status {
	s.SetPlugin(plugin)
	return s
}

// Plugin returns the plugin name which caused this status.
func (s *Status) Plugin() string {
	return s.plugin
}

// Reasons returns reasons of the Status.
func (s *Status) Reasons() []string {
	if s.err != nil {
		return append([]string{s.err.Error()}, s.reasons...)
	}
	return s.reasons
}

// AppendReason appends given reason to the Status.
func (s *Status) AppendReason(reason string) {
	s.reasons = append(s.reasons, reason)
}

// IsSuccess returns true if and only if "Status" is nil or Code is "Success".
func (s *Status) IsSuccess() bool {
	return s.Code() == Success
}

// IsWait returns true if and only if "Status" is non-nil and its Code is "Wait".
func (s *Status) IsWait() bool {
	return s.Code() == Wait
}

// IsSkip returns true if and only if "Status" is non-nil and its Code is "Skip".
func (s *Status) IsSkip() bool {
	return s.Code() == Skip
}

// IsRejected returns true if "Status" is Unschedulable (Unschedulable, UnschedulableAndUnresolvable, or Pending).
func (s *Status) IsRejected() bool {
	code := s.Code()
	return code == Unschedulable || code == UnschedulableAndUnresolvable || code == Pending
}

// AsError returns nil if the status is a success, a wait or a skip; otherwise returns an "error" object
// with a concatenated message on reasons of the Status.
func (s *Status) AsError() error {
	if s.IsSuccess() || s.IsWait() || s.IsSkip() {
		return nil
	}
	if s.err != nil {
		return s.err
	}
	return errors.New(s.Message())
}

// Equal checks equality of two statuses. This is useful for testing with
// cmp.Equal.
func (s *Status) Equal(x *Status) bool {
	if s == nil || x == nil {
		return s.IsSuccess() && x.IsSuccess()
	}
	if s.code != x.code {
		return false
	}
	if !cmp.Equal(s.err, x.err, cmpopts.EquateErrors()) {
		return false
	}
	if !cmp.Equal(s.reasons, x.reasons) {
		return false
	}
	return cmp.Equal(s.plugin, x.plugin)
}

func (s *Status) String() string {
	return s.Message()
}

// NewStatus makes a Status out of the given arguments and returns its pointer.
func NewStatus(code Code, reasons ...string) *Status {
	s := &Status{
		code:    code,
		reasons: reasons,
	}
	return s
}

// AsStatus wraps an error in a Status.
func AsStatus(err error) *Status {
	if err == nil {
		return nil
	}
	return &Status{
		code: Error,
		err:  err,
	}
}

// WaitingRegion represents a region currently waiting in the permit phase.
type WaitingRegion interface {
	// GetRegion returns a reference to the waiting region.
	GetRegion() *corev1.Region
	// GetPendingPlugins returns a list of pending Permit plugin's name.
	GetPendingPlugins() []string
	// Allow declares the waiting region is allowed to be scheduled by the plugin named as "pluginName".
	// If this is the last remaining plugin to allow, then a success signal is delivered
	// to unblock the region.
	Allow(pluginName string)
	// Reject declares the waiting region unschedulable.
	Reject(pluginName, msg string)
}

// Plugin is the parent type for all the scheduling framework plugins.
type Plugin interface {
	Name() string
}

// PreEnqueuePlugin is an interface that must be implemented by "PreEnqueue" plugins.
// These plugins are called prior to adding Regions to activeQ.
// Note: an preEnqueue plugin is expected to be lightweight and efficient, so it's not expected to
// involve expensive calls like accessing external endpoints; otherwise it'd block other
// Regions' enqueuing in event handlers.
type PreEnqueuePlugin interface {
	Plugin
	// PreEnqueue is called prior to adding Regions to activeQ.
	PreEnqueue(ctx context.Context, p *corev1.Region) *Status
}

// LessFunc is the function to sort region info
type LessFunc func(regionInfo1, regionInfo2 *QueuedRegionInfo) bool

// QueueSortPlugin is an interface that must be implemented by "QueueSort" plugins.
// These plugins are used to sort regions in the scheduling queue. Only one queue sort
// plugin may be enabled at a time.
type QueueSortPlugin interface {
	Plugin
	// Less are used to sort regions in the scheduling queue.
	Less(*QueuedRegionInfo, *QueuedRegionInfo) bool
}

// EnqueueExtensions is an optional interface that plugins can implement to efficiently
// move unschedulable Regions in internal scheduling queues.
// In the scheduler, Regions can be unschedulable by PreEnqueue, PreFilter, Filter, Reserve, and Permit plugins,
// and Regions rejected by these plugins are requeued based on this extension point.
// Failures from other extension points are regarded as temporal errors (e.g., network failure),
// and the scheduler requeue Regions without this extension point - always requeue Regions to activeQ after backoff.
// This is because such temporal errors cannot be resolved by specific cluster events,
// and we have no choise but keep retrying scheduling until the failure is resolved.
//
// Plugins that make region unschedulable (PreEnqueue, PreFilter, Filter, Reserve, and Permit plugins) should implement this interface,
// otherwise the default implementation will be used, which is less efficient in requeueing Regions rejected by the plugin.
// And, if plugins other than above extension points support this interface, they are just ignored.
type EnqueueExtensions interface {
	Plugin
	// EventsToRegister returns a series of possible events that may cause a Region
	// failed by this plugin schedulable. Each event has a callback function that
	// filters out events to reduce useless retry of Region's scheduling.
	// The events will be registered when instantiating the internal scheduling queue,
	// and leveraged to build event handlers dynamically.
	// Note: the returned list needs to be static (not depend on configuration parameters);
	// otherwise it would lead to undefined behavior.
	//
	// Appropriate implementation of this function will make Region's re-scheduling accurate and performant.
	EventsToRegister() []ClusterEventWithHint
}

// PreFilterExtensions is an interface that is included in plugins that allow specifying
// callbacks to make incremental updates to its supposedly pre-calculated
// state.
type PreFilterExtensions interface {
	// AddRegion is called by the framework while trying to evaluate the impact
	// of adding regionToAdd to the runner while scheduling regionToSchedule.
	AddRegion(ctx context.Context, state *CycleState, regionToSchedule *corev1.Region, regionInfoToAdd *RegionInfo, runnerInfo *RunnerInfo) *Status
	// RemoveRegion is called by the framework while trying to evaluate the impact
	// of removing regionToRemove from the runner while scheduling regionToSchedule.
	RemoveRegion(ctx context.Context, state *CycleState, regionToSchedule *corev1.Region, regionInfoToRemove *RegionInfo, runnerInfo *RunnerInfo) *Status
}

// PreFilterPlugin is an interface that must be implemented by "PreFilter" plugins.
// These plugins are called at the beginning of the scheduling cycle.
type PreFilterPlugin interface {
	Plugin
	// PreFilter is called at the beginning of the scheduling cycle. All PreFilter
	// plugins must return success or the region will be rejected. PreFilter could optionally
	// return a PreFilterResult to influence which runners to evaluate downstream. This is useful
	// for cases where it is possible to determine the subset of runners to process in O(1) time.
	// When PreFilterResult filters out some Runners, the framework considers Runners that are filtered out as getting "UnschedulableAndUnresolvable".
	// i.e., those Runners will be out of the candidates of the preemption.
	//
	// When it returns Skip status, returned PreFilterResult and other fields in status are just ignored,
	// and coupled Filter plugin/PreFilterExtensions() will be skipped in this scheduling cycle.
	PreFilter(ctx context.Context, state *CycleState, p *corev1.Region) (*PreFilterResult, *Status)
	// PreFilterExtensions returns a PreFilterExtensions interface if the plugin implements one,
	// or nil if it does not. A Pre-filter plugin can provide extensions to incrementally
	// modify its pre-processed info. The framework guarantees that the extensions
	// AddRegion/RemoveRegion will only be called after PreFilter, possibly on a cloned
	// CycleState, and may call those functions more than once before calling
	// Filter again on a specific runner.
	PreFilterExtensions() PreFilterExtensions
}

// FilterPlugin is an interface for Filter plugins. These plugins are called at the
// filter extension point for filtering out hosts that cannot run a region.
// This concept used to be called 'predicate' in the original scheduler.
// These plugins should return "Success", "Unschedulable" or "Error" in Status.code.
// However, the scheduler accepts other valid codes as well.
// Anything other than "Success" will lead to exclusion of the given host from
// running the region.
type FilterPlugin interface {
	Plugin
	// Filter is called by the scheduling framework.
	// All FilterPlugins should return "Success" to declare that
	// the given runner fits the region. If Filter doesn't return "Success",
	// it will return "Unschedulable", "UnschedulableAndUnresolvable" or "Error".
	// For the runner being evaluated, Filter plugins should look at the passed
	// runnerInfo reference for this particular runner's information (e.g., regions
	// considered to be running on the runner) instead of looking it up in the
	// RunnerInfoSnapshot because we don't guarantee that they will be the same.
	// For example, during preemption, we may pass a copy of the original
	// runnerInfo object that has some regions removed from it to evaluate the
	// possibility of preempting them to schedule the target region.
	Filter(ctx context.Context, state *CycleState, region *corev1.Region, runnerInfo *RunnerInfo) *Status
}

// PostFilterPlugin is an interface for "PostFilter" plugins. These plugins are called
// after a region cannot be scheduled.
type PostFilterPlugin interface {
	Plugin
	// PostFilter is called by the scheduling framework
	// when the scheduling cycle failed at PreFilter or Filter by Unschedulable or UnschedulableAndUnresolvable.
	// RunnerToStatusMap has statuses that each Runner got in the Filter phase.
	// If this scheduling cycle failed at PreFilter, all Runners have the status from the rejector PreFilter plugin in RunnerToStatusMap.
	// Note that the scheduling framework runs PostFilter plugins even when PreFilter returned UnschedulableAndUnresolvable.
	// In that case, RunnerToStatusMap contains all Runners with UnschedulableAndUnresolvable.
	//
	// Also, ignoring Runners with UnschedulableAndUnresolvable is the responsibility of each PostFilter plugin,
	// meaning RunnerToStatusMap obviously could have Runners with UnschedulableAndUnresolvable
	// and the scheduling framework does call PostFilter even when all Runners in RunnerToStatusMap are UnschedulableAndUnresolvable.
	//
	// A PostFilter plugin should return one of the following statuses:
	// - Unschedulable: the plugin gets executed successfully but the region cannot be made schedulable.
	// - Success: the plugin gets executed successfully and the region can be made schedulable.
	// - Error: the plugin aborts due to some internal error.
	//
	// Informational plugins should be configured ahead of other ones, and always return Unschedulable status.
	// Optionally, a non-nil PostFilterResult may be returned along with a Success status. For example,
	// a preemption plugin may choose to return nominatedRunnerName, so that framework can reuse that to update the
	// preemptor region's .spec.status.nominatedRunnerName field.
	PostFilter(ctx context.Context, state *CycleState, region *corev1.Region, filteredRunnerStatusMap RunnerToStatusMap) (*PostFilterResult, *Status)
}

// PreScorePlugin is an interface for "PreScore" plugin. PreScore is an
// informational extension point. Plugins will be called with a list of runners
// that passed the filtering phase. A plugin may use this data to update internal
// state or to generate logs/metrics.
type PreScorePlugin interface {
	Plugin
	// PreScore is called by the scheduling framework after a list of runners
	// passed the filtering phase. All prescore plugins must return success or
	// the region will be rejected
	// When it returns Skip status, other fields in status are just ignored,
	// and coupled Score plugin will be skipped in this scheduling cycle.
	PreScore(ctx context.Context, state *CycleState, region *corev1.Region, runners []*RunnerInfo) *Status
}

// ScoreExtensions is an interface for Score extended functionality.
type ScoreExtensions interface {
	// NormalizeScore is called for all runner scores produced by the same plugin's "Score"
	// method. A successful run of NormalizeScore will update the scores list and return
	// a success status.
	NormalizeScore(ctx context.Context, state *CycleState, p *corev1.Region, scores RunnerScoreList) *Status
}

// ScorePlugin is an interface that must be implemented by "Score" plugins to rank
// runners that passed the filtering phase.
type ScorePlugin interface {
	Plugin
	// Score is called on each filtered runner. It must return success and an integer
	// indicating the rank of the runner. All scoring plugins must return success or
	// the region will be rejected.
	Score(ctx context.Context, state *CycleState, p *corev1.Region, runnerName string) (int64, *Status)

	// ScoreExtensions returns a ScoreExtensions interface if it implements one, or nil if does not.
	ScoreExtensions() ScoreExtensions
}

// ReservePlugin is an interface for plugins with Reserve and Unreserve
// methods. These are meant to update the state of the plugin. This concept
// used to be called 'assume' in the original scheduler. These plugins should
// return only Success or Error in Status.code. However, the scheduler accepts
// other valid codes as well. Anything other than Success will lead to
// rejection of the region.
type ReservePlugin interface {
	Plugin
	// Reserve is called by the scheduling framework when the scheduler cache is
	// updated. If this method returns a failed Status, the scheduler will call
	// the Unreserve method for all enabled ReservePlugins.
	Reserve(ctx context.Context, state *CycleState, p *corev1.Region, runnerName string) *Status
	// Unreserve is called by the scheduling framework when a reserved region was
	// rejected, an error occurred during reservation of subsequent plugins, or
	// in a later phase. The Unreserve method implementation must be idempotent
	// and may be called by the scheduler even if the corresponding Reserve
	// method for the same plugin was not called.
	Unreserve(ctx context.Context, state *CycleState, p *corev1.Region, runnerName string)
}

// PreBindPlugin is an interface that must be implemented by "PreBind" plugins.
// These plugins are called before a region being scheduled.
type PreBindPlugin interface {
	Plugin
	// PreBind is called before binding a region. All prebind plugins must return
	// success or the region will be rejected and won't be sent for binding.
	PreBind(ctx context.Context, state *CycleState, p *corev1.Region, runnerName string) *Status
}

// PostBindPlugin is an interface that must be implemented by "PostBind" plugins.
// These plugins are called after a region is successfully bound to a runner.
type PostBindPlugin interface {
	Plugin
	// PostBind is called after a region is successfully bound. These plugins are
	// informational. A common application of this extension point is for cleaning
	// up. If a plugin needs to clean-up its state after a region is scheduled and
	// bound, PostBind is the extension point that it should register.
	PostBind(ctx context.Context, state *CycleState, p *corev1.Region, runnerName string)
}

// PermitPlugin is an interface that must be implemented by "Permit" plugins.
// These plugins are called before a region is bound to a runner.
type PermitPlugin interface {
	Plugin
	// Permit is called before binding a region (and before prebind plugins). Permit
	// plugins are used to prevent or delay the binding of a Region. A permit plugin
	// must return success or wait with timeout duration, or the region will be rejected.
	// The region will also be rejected if the wait timeout or the region is rejected while
	// waiting. Note that if the plugin returns "wait", the framework will wait only
	// after running the remaining plugins given that no other plugin rejects the region.
	Permit(ctx context.Context, state *CycleState, p *corev1.Region, runnerName string) (*Status, time.Duration)
}

// BindPlugin is an interface that must be implemented by "Bind" plugins. Bind
// plugins are used to bind a region to a Runner.
type BindPlugin interface {
	Plugin
	// Bind plugins will not be called until all pre-bind plugins have completed. Each
	// bind plugin is called in the configured order. A bind plugin may choose whether
	// or not to handle the given Region. If a bind plugin chooses to handle a Region, the
	// remaining bind plugins are skipped. When a bind plugin does not handle a region,
	// it must return Skip in its Status code. If a bind plugin returns an Error, the
	// region is rejected and will not be bound.
	Bind(ctx context.Context, state *CycleState, p *corev1.Region, runnerNames ...string) *Status
}

// Framework manages the set of plugins in use by the scheduling framework.
// Configured plugins are called at specified points in a scheduling context.
type Framework interface {
	Handle

	// PreEnqueuePlugins returns the registered preEnqueue plugins.
	PreEnqueuePlugins() []PreEnqueuePlugin

	// EnqueueExtensions returns the registered Enqueue extensions.
	EnqueueExtensions() []EnqueueExtensions

	// QueueSortFunc returns the function to sort regions in scheduling queue
	QueueSortFunc() LessFunc

	// RunPreFilterPlugins runs the set of configured PreFilter plugins. It returns
	// *Status and its code is set to non-success if any of the plugins returns
	// anything but Success. If a non-success status is returned, then the scheduling
	// cycle is aborted.
	// It also returns a PreFilterResult, which may influence what or how many runners to
	// evaluate downstream.
	RunPreFilterPlugins(ctx context.Context, state *CycleState, region *corev1.Region) (*PreFilterResult, *Status)

	// RunPostFilterPlugins runs the set of configured PostFilter plugins.
	// PostFilter plugins can either be informational, in which case should be configured
	// to execute first and return Unschedulable status, or ones that try to change the
	// cluster state to make the region potentially schedulable in a future scheduling cycle.
	RunPostFilterPlugins(ctx context.Context, state *CycleState, region *corev1.Region, filteredRunnerStatusMap RunnerToStatusMap) (*PostFilterResult, *Status)

	// RunPreBindPlugins runs the set of configured PreBind plugins. It returns
	// *Status and its code is set to non-success if any of the plugins returns
	// anything but Success. If the Status code is "Unschedulable", it is
	// considered as a scheduling check failure, otherwise, it is considered as an
	// internal error. In either case the region is not going to be bound.
	RunPreBindPlugins(ctx context.Context, state *CycleState, region *corev1.Region, runnerName string) *Status

	// RunPostBindPlugins runs the set of configured PostBind plugins.
	RunPostBindPlugins(ctx context.Context, state *CycleState, region *corev1.Region, runnerName string)

	// RunReservePluginsReserve runs the Reserve method of the set of
	// configured Reserve plugins. If any of these calls returns an error, it
	// does not continue running the remaining ones and returns the error. In
	// such case, region will not be scheduled.
	RunReservePluginsReserve(ctx context.Context, state *CycleState, region *corev1.Region, runnerName string) *Status

	// RunReservePluginsUnreserve runs the Unreserve method of the set of
	// configured Reserve plugins.
	RunReservePluginsUnreserve(ctx context.Context, state *CycleState, region *corev1.Region, runnerName string)

	// RunPermitPlugins runs the set of configured Permit plugins. If any of these
	// plugins returns a status other than "Success" or "Wait", it does not continue
	// running the remaining plugins and returns an error. Otherwise, if any of the
	// plugins returns "Wait", then this function will create and add waiting region
	// to a map of currently waiting regions and return status with "Wait" code.
	// Region will remain waiting region for the minimum duration returned by the Permit plugins.
	RunPermitPlugins(ctx context.Context, state *CycleState, region *corev1.Region, runnerName string) *Status

	// WaitOnPermit will block, if the region is a waiting region, until the waiting region is rejected or allowed.
	WaitOnPermit(ctx context.Context, region *corev1.Region) *Status

	// RunBindPlugins runs the set of configured Bind plugins. A Bind plugin may choose
	// whether or not to handle the given Region. If a Bind plugin chooses to skip the
	// binding, it should return code=5("skip") status. Otherwise, it should return "Error"
	// or "Success". If none of the plugins handled binding, RunBindPlugins returns
	// code=5("skip") status.
	RunBindPlugins(ctx context.Context, state *CycleState, region *corev1.Region, runnerName string) *Status

	// HasFilterPlugins returns true if at least one Filter plugin is defined.
	HasFilterPlugins() bool

	// HasPostFilterPlugins returns true if at least one PostFilter plugin is defined.
	HasPostFilterPlugins() bool

	// HasScorePlugins returns true if at least one Score plugin is defined.
	HasScorePlugins() bool

	// ListPlugins returns a map of extension point name to list of configured Plugins.
	ListPlugins() *config.Plugins

	// ProfileName returns the profile name associated to a profile.
	ProfileName() string

	// PercentageOfRunnersToScore returns percentageOfRunnersToScore associated to a profile.
	PercentageOfRunnersToScore() *int32

	// SetRegionNominator sets the RegionNominator
	SetRegionNominator(nominator RegionNominator)

	// Close calls Close method of each plugin.
	Close() error
}

// Handle provides data and some tools that plugins can use. It is
// passed to the plugin factories at the time of plugin initialization. Plugins
// must store and use this handle to call framework functions.
type Handle interface {
	// RegionNominator abstracts operations to maintain nominated Regions.
	RegionNominator
	// PluginsRunner abstracts operations to run some plugins.
	PluginsRunner
	// SnapshotSharedLister returns listers from the latest RunnerInfo Snapshot. The snapshot
	// is taken at the beginning of a scheduling cycle and remains unchanged until
	// a region finishes "Permit" point.
	//
	// It should be used only during scheduling cycle:
	// - There is no guarantee that the information remains unchanged in the binding phase of scheduling.
	//   So, plugins shouldn't use it in the binding cycle (pre-bind/bind/post-bind/un-reserve plugin)
	//   otherwise, a concurrent read/write error might occur.
	// - There is no guarantee that the information is always up-to-date.
	//   So, plugins shouldn't use it in QueueingHint and PreEnqueue
	//   otherwise, they might make a decision based on stale information.
	//
	// Instead, they should use the resources getting from Informer created from SharedInformerFactory().
	SnapshotSharedLister() SharedLister

	// IterateOverWaitingRegions acquires a read lock and iterates over the WaitingRegions map.
	IterateOverWaitingRegions(callback func(WaitingRegion))

	// GetWaitingRegion returns a waiting region given its UID.
	GetWaitingRegion(uid types.UID) WaitingRegion

	// RejectWaitingRegion rejects a waiting region given its UID.
	// The return value indicates if the region is waiting or not.
	RejectWaitingRegion(uid types.UID) bool

	// ClientSet returns a kubernetes clientSet.
	ClientSet() clientset.Interface

	// OliveConfig returns the raw olive config.
	OliveConfig() *restclient.Config

	// EventRecorder returns an event recorder.
	EventRecorder() events.EventRecorder

	SharedInformerFactory() informers.SharedInformerFactory

	// RunFilterPluginsWithNominatedRegions runs the set of configured filter plugins for nominated region on the given runner.
	RunFilterPluginsWithNominatedRegions(ctx context.Context, state *CycleState, region *corev1.Region, info *RunnerInfo) *Status

	// Extenders returns registered scheduler extenders.
	Extenders() []Extender

	// Parallelizer returns a parallelizer holding parallelism for scheduler.
	Parallelizer() parallelize.Parallelizer
}

// PreFilterResult wraps needed info for scheduler framework to act upon PreFilter phase.
type PreFilterResult struct {
	// The set of runners that should be considered downstream; if nil then
	// all runners are eligible.
	RunnerNames sets.Set[string]
}

func (p *PreFilterResult) AllRunners() bool {
	return p == nil || p.RunnerNames == nil
}

func (p *PreFilterResult) Merge(in *PreFilterResult) *PreFilterResult {
	if p.AllRunners() && in.AllRunners() {
		return nil
	}

	r := PreFilterResult{}
	if p.AllRunners() {
		r.RunnerNames = in.RunnerNames.Clone()
		return &r
	}
	if in.AllRunners() {
		r.RunnerNames = p.RunnerNames.Clone()
		return &r
	}

	r.RunnerNames = p.RunnerNames.Intersection(in.RunnerNames)
	return &r
}

type NominatingMode int

const (
	ModeNoop NominatingMode = iota
	ModeOverride
)

type NominatingInfo struct {
	NominatedRunnerName string
	NominatingMode      NominatingMode
}

// PostFilterResult wraps needed info for scheduler framework to act upon PostFilter phase.
type PostFilterResult struct {
	*NominatingInfo
}

func NewPostFilterResultWithNominatedRunner(name string) *PostFilterResult {
	return &PostFilterResult{
		NominatingInfo: &NominatingInfo{
			NominatedRunnerName: name,
			NominatingMode:      ModeOverride,
		},
	}
}

func (ni *NominatingInfo) Mode() NominatingMode {
	if ni == nil {
		return ModeNoop
	}
	return ni.NominatingMode
}

// RegionNominator abstracts operations to maintain nominated Regions.
type RegionNominator interface {
	// AddNominatedRegion adds the given region to the nominator or
	// updates it if it already exists.
	AddNominatedRegion(logger klog.Logger, region *RegionInfo, nominatingInfo *NominatingInfo)
	// DeleteNominatedRegionIfExists deletes nominatedRegion from internal cache. It's a no-op if it doesn't exist.
	DeleteNominatedRegionIfExists(region *corev1.Region)
	// UpdateNominatedRegion updates the <oldRegion> with <newRegion>.
	UpdateNominatedRegion(logger klog.Logger, oldRegion *corev1.Region, newRegionInfo *RegionInfo)
	// NominatedRegionsForRunner returns nominatedRegions on the given runner.
	NominatedRegionsForRunner(runnerName string) []*RegionInfo
}

// PluginsRunner abstracts operations to run some plugins.
// This is used by preemption PostFilter plugins when evaluating the feasibility of
// scheduling the region on runners when certain running regions get evicted.
type PluginsRunner interface {
	// RunPreScorePlugins runs the set of configured PreScore plugins. If any
	// of these plugins returns any status other than "Success", the given region is rejected.
	RunPreScorePlugins(context.Context, *CycleState, *corev1.Region, []*RunnerInfo) *Status
	// RunScorePlugins runs the set of configured scoring plugins.
	// It returns a list that stores scores from each plugin and total score for each Runner.
	// It also returns *Status, which is set to non-success if any of the plugins returns
	// a non-success status.
	RunScorePlugins(context.Context, *CycleState, *corev1.Region, []*RunnerInfo) ([]RunnerPluginScores, *Status)
	// RunFilterPlugins runs the set of configured Filter plugins for region on
	// the given runner. Note that for the runner being evaluated, the passed runnerInfo
	// reference could be different from the one in RunnerInfoSnapshot map (e.g., regions
	// considered to be running on the runner could be different). For example, during
	// preemption, we may pass a copy of the original runnerInfo object that has some regions
	// removed from it to evaluate the possibility of preempting them to
	// schedule the target region.
	RunFilterPlugins(context.Context, *CycleState, *corev1.Region, *RunnerInfo) *Status
	// RunPreFilterExtensionAddRegion calls the AddRegion interface for the set of configured
	// PreFilter plugins. It returns directly if any of the plugins return any
	// status other than Success.
	RunPreFilterExtensionAddRegion(ctx context.Context, state *CycleState, regionToSchedule *corev1.Region, regionInfoToAdd *RegionInfo, runnerInfo *RunnerInfo) *Status
	// RunPreFilterExtensionRemoveRegion calls the RemoveRegion interface for the set of configured
	// PreFilter plugins. It returns directly if any of the plugins return any
	// status other than Success.
	RunPreFilterExtensionRemoveRegion(ctx context.Context, state *CycleState, regionToSchedule *corev1.Region, regionInfoToRemove *RegionInfo, runnerInfo *RunnerInfo) *Status
}
