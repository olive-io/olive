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
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

var generation int64

// ActionType is an integer to represent one type of resource change.
// Different ActionTypes can be bit-wised to compose new semantics.
type ActionType int64

// Constants for ActionTypes.
const (
	Add    ActionType = 1 << iota // 1
	Delete                        // 10
	// UpdateRunnerXYZ is only applicable for Runner events.
	UpdateRunnerAllocatable // 100
	UpdateRunnerLabel       // 1000
	UpdateRunnerTaint       // 10000
	UpdateRunnerCondition   // 100000
	UpdateRunnerAnnotation  // 1000000

	All ActionType = 1<<iota - 1 // 1111111

	// Use the general Update type if you don't either know or care the specific sub-Update type to use.
	Update = UpdateRunnerAllocatable | UpdateRunnerLabel | UpdateRunnerTaint | UpdateRunnerCondition | UpdateRunnerAnnotation
)

// GVK is short for group/version/kind, which can uniquely represent a particular API resource.
type GVK string

// Constants for GVKs.
const (
	// There are a couple of notes about how the scheduler notifies the events of Regions:
	// - Add: add events could be triggered by either a newly created Region or an existing Region that is scheduled to a Runner.
	// - Delete: delete events could be triggered by:
	//           - a Region that is deleted
	//           - a Region that was assumed, but gets un-assumed due to some errors in the binding cycle.
	//           - an existing Region that was unscheduled but gets scheduled to a Runner.
	Region GVK = "Region"
	// A note about RunnerAdd event and UpdateRunnerTaint event:
	// RunnerAdd QueueingHint isn't always called because of the internal feature called preCheck.
	// It's definitely not something expected for plugin developers,
	// and registering UpdateRunnerTaint event is the only mitigation for now.
	// So, kube-scheduler registers UpdateRunnerTaint event for plugins that has RunnerAdded event, but don't have UpdateRunnerTaint event.
	// It has a bad impact for the requeuing efficiency though, a lot better than some Regions being stuck in the
	// unschedulable region pool.
	// This behavior will be removed when we remove the preCheck feature.
	// See: https://github.com/kubernetes/kubernetes/issues/110175
	Runner                  GVK = "Runner"
	RegionSchedulingContext GVK = "RegionSchedulingContext"
	ResourceClaim           GVK = "ResourceClaim"
	ResourceClass           GVK = "ResourceClass"
	ResourceClaimParameters GVK = "ResourceClaimParameters"
	ResourceClassParameters GVK = "ResourceClassParameters"

	// WildCard is a special GVK to match all resources.
	// e.g., If you register `{Resource: "*", ActionType: All}` in EventsToRegister,
	// all coming clusterEvents will be admitted. Be careful to register it, it will
	// increase the computing pressure in requeueing unless you really need it.
	//
	// Meanwhile, if the coming clusterEvent is a wildcard one, all regions
	// will be moved from unschedulableRegion pool to activeQ/backoffQ forcibly.
	WildCard GVK = "*"
)

type ClusterEventWithHint struct {
	Event ClusterEvent
	// QueueingHintFn is executed for the plugin rejected by this plugin when the above Event happens,
	// and filters out events to reduce useless retry of Region's scheduling.
	// It's an optional field. If not set,
	// the scheduling of Regions will be always retried with backoff when this Event happens.
	// (the same as Queue)
	QueueingHintFn QueueingHintFn
}

// QueueingHintFn returns a hint that signals whether the event can make a Region,
// which was rejected by this plugin in the past scheduling cycle, schedulable or not.
// It's called before a Region gets moved from unschedulableQ to backoffQ or activeQ.
// If it returns an error, we'll take the returned QueueingHint as `Queue` at the caller whatever we returned here so that
// we can prevent the Region from being stuck in the unschedulable region pool.
//
// - `region`: the Region to be enqueued, which is rejected by this plugin in the past.
// - `oldObj` `newObj`: the object involved in that event.
//   - For example, the given event is "Runner deleted", the `oldObj` will be that deleted Runner.
//   - `oldObj` is nil if the event is add event.
//   - `newObj` is nil if the event is delete event.
type QueueingHintFn func(logger klog.Logger, region *corev1.Region, oldObj, newObj interface{}) (QueueingHint, error)

type QueueingHint int

const (
	// QueueSkip implies that the cluster event has no impact on
	// scheduling of the region.
	QueueSkip QueueingHint = iota

	// Queue implies that the Region may be schedulable by the event.
	Queue
)

func (s QueueingHint) String() string {
	switch s {
	case QueueSkip:
		return "QueueSkip"
	case Queue:
		return "Queue"
	}
	return ""
}

// ClusterEvent abstracts how a system resource's state gets changed.
// Resource represents the standard API resources such as Region, Runner, etc.
// ActionType denotes the specific change such as Add, Update or Delete.
type ClusterEvent struct {
	Resource   GVK
	ActionType ActionType
	Label      string
}

// IsWildCard returns true if ClusterEvent follows WildCard semantics
func (ce ClusterEvent) IsWildCard() bool {
	return ce.Resource == WildCard && ce.ActionType == All
}

// Match returns true if ClusterEvent is matched with the coming event.
// If the ce.Resource is "*", there's no requirement for the coming event' Resource.
// Contrarily, if the coming event's Resource is "*", the ce.Resource should only be "*".
//
// Note: we have a special case here when the coming event is a wildcard event,
// it will force all Regions to move to activeQ/backoffQ,
// but we take it as an unmatched event unless the ce is also a wildcard one.
func (ce ClusterEvent) Match(event ClusterEvent) bool {
	return ce.IsWildCard() || (ce.Resource == WildCard || ce.Resource == event.Resource) && ce.ActionType&event.ActionType != 0
}

func UnrollWildCardResource() []ClusterEventWithHint {
	return []ClusterEventWithHint{
		{Event: ClusterEvent{Resource: Region, ActionType: All}},
		{Event: ClusterEvent{Resource: Runner, ActionType: All}},
		{Event: ClusterEvent{Resource: RegionSchedulingContext, ActionType: All}},
		{Event: ClusterEvent{Resource: ResourceClaim, ActionType: All}},
		{Event: ClusterEvent{Resource: ResourceClass, ActionType: All}},
		{Event: ClusterEvent{Resource: ResourceClaimParameters, ActionType: All}},
		{Event: ClusterEvent{Resource: ResourceClassParameters, ActionType: All}},
	}
}

// QueuedRegionInfo is a Region wrapper with additional information related to
// the region's status in the scheduling queue, such as the timestamp when
// it's added to the queue.
type QueuedRegionInfo struct {
	*RegionInfo
	// The time region added to the scheduling queue.
	Timestamp time.Time
	// Number of schedule attempts before successfully scheduled.
	// It's used to record the # attempts metric.
	Attempts int
	// The time when the region is added to the queue for the first time. The region may be added
	// back to the queue multiple times before it's successfully scheduled.
	// It shouldn't be updated once initialized. It's used to record the e2e scheduling
	// latency for a region.
	InitialAttemptTimestamp *time.Time
	// UnschedulablePlugins records the plugin names that the Region failed with Unschedulable or UnschedulableAndUnresolvable status.
	// It's registered only when the Region is rejected in PreFilter, Filter, Reserve, or Permit (WaitOnPermit).
	UnschedulablePlugins sets.Set[string]
	// PendingPlugins records the plugin names that the Region failed with Pending status.
	PendingPlugins sets.Set[string]
	// Whether the Region is scheduling gated (by PreEnqueuePlugins) or not.
	Gated bool
}

// DeepCopy returns a deep copy of the QueuedRegionInfo object.
func (pqi *QueuedRegionInfo) DeepCopy() *QueuedRegionInfo {
	return &QueuedRegionInfo{
		RegionInfo:              pqi.RegionInfo.DeepCopy(),
		Timestamp:               pqi.Timestamp,
		Attempts:                pqi.Attempts,
		InitialAttemptTimestamp: pqi.InitialAttemptTimestamp,
		UnschedulablePlugins:    pqi.UnschedulablePlugins.Clone(),
		Gated:                   pqi.Gated,
	}
}

// RegionInfo is a wrapper to a Region with additional pre-computed information to
// accelerate processing. This information is typically immutable (e.g., pre-processed
// inter-region affinity selectors).
type RegionInfo struct {
	Region                     *corev1.Region
	RequiredAffinityTerms      []AffinityTerm
	RequiredAntiAffinityTerms  []AffinityTerm
	PreferredAffinityTerms     []WeightedAffinityTerm
	PreferredAntiAffinityTerms []WeightedAffinityTerm
}

// DeepCopy returns a deep copy of the RegionInfo object.
func (pi *RegionInfo) DeepCopy() *RegionInfo {
	return &RegionInfo{
		Region:                     pi.Region.DeepCopy(),
		RequiredAffinityTerms:      pi.RequiredAffinityTerms,
		RequiredAntiAffinityTerms:  pi.RequiredAntiAffinityTerms,
		PreferredAffinityTerms:     pi.PreferredAffinityTerms,
		PreferredAntiAffinityTerms: pi.PreferredAntiAffinityTerms,
	}
}

// Update creates a full new RegionInfo by default. And only updates the region when the RegionInfo
// has been instantiated and the passed region is the exact same one as the original region.
func (pi *RegionInfo) Update(region *corev1.Region) error {
	if region != nil && pi.Region != nil && pi.Region.UID == region.UID {
		// RegionInfo includes immutable information, and so it is safe to update the region in place if it is
		// the exact same region
		pi.Region = region
		return nil
	}
	// Attempt to parse the affinity terms
	var parseErrs []error

	pi.Region = region
	return utilerrors.NewAggregate(parseErrs)
}

// AffinityTerm is a processed version of corev1.RegionAffinityTerm.
type AffinityTerm struct {
	Namespaces        sets.Set[string]
	Selector          labels.Selector
	TopologyKey       string
	NamespaceSelector labels.Selector
}

// Matches returns true if the region matches the label selector and namespaces or namespace selector.
func (at *AffinityTerm) Matches(region *corev1.Region, nsLabels labels.Set) bool {
	if at.Namespaces.Has(region.Namespace) || at.NamespaceSelector.Matches(nsLabels) {
		return at.Selector.Matches(labels.Set(region.Labels))
	}
	return false
}

// WeightedAffinityTerm is a "processed" representation of corev1.WeightedAffinityTerm.
type WeightedAffinityTerm struct {
	AffinityTerm
	Weight int32
}

// ExtenderName is a fake plugin name put in UnschedulablePlugins when Extender rejected some Runners.
const ExtenderName = "Extender"

// Diagnosis records the details to diagnose a scheduling failure.
type Diagnosis struct {
	// RunnerToStatusMap records the status of each node
	// if they're rejected in PreFilter (via PreFilterResult) or Filter plugins.
	// Runners that pass PreFilter/Filter plugins are not included in this map.
	RunnerToStatusMap RunnerToStatusMap
	// UnschedulablePlugins are plugins that returns Unschedulable or UnschedulableAndUnresolvable.
	UnschedulablePlugins sets.Set[string]
	// UnschedulablePlugins are plugins that returns Pending.
	PendingPlugins sets.Set[string]
	// PreFilterMsg records the messages returned from PreFilter plugins.
	PreFilterMsg string
	// PostFilterMsg records the messages returned from PostFilter plugins.
	PostFilterMsg string
}

// FitError describes a fit error of a region.
type FitError struct {
	Region        *corev1.Region
	NumAllRunners int
	Diagnosis     Diagnosis
}

const (
	// NoRunnerAvailableMsg is used to format message when no nodes available.
	NoRunnerAvailableMsg = "0/%v nodes are available"
)

func (d *Diagnosis) AddPluginStatus(sts *Status) {
	if sts.Plugin() == "" {
		return
	}
	if sts.IsRejected() {
		if d.UnschedulablePlugins == nil {
			d.UnschedulablePlugins = sets.New[string]()
		}
		d.UnschedulablePlugins.Insert(sts.Plugin())
	}
	if sts.Code() == Pending {
		if d.PendingPlugins == nil {
			d.PendingPlugins = sets.New[string]()
		}
		d.PendingPlugins.Insert(sts.Plugin())
	}
}

// Error returns detailed information of why the region failed to fit on each node.
// A message format is "0/X nodes are available: <PreFilterMsg>. <FilterMsg>. <PostFilterMsg>."
func (f *FitError) Error() string {
	reasonMsg := fmt.Sprintf(NoRunnerAvailableMsg+":", f.NumAllRunners)
	preFilterMsg := f.Diagnosis.PreFilterMsg
	if preFilterMsg != "" {
		// PreFilter plugin returns unschedulable.
		// Add the messages from PreFilter plugins to reasonMsg.
		reasonMsg += fmt.Sprintf(" %v.", preFilterMsg)
	}

	if preFilterMsg == "" {
		// the scheduling cycle went through PreFilter extension point successfully.
		//
		// When the prefilter plugin returns unschedulable,
		// the scheduling framework inserts the same unschedulable status to all nodes in RunnerToStatusMap.
		// So, we shouldn't add the message from RunnerToStatusMap when the PreFilter failed.
		// Otherwise, we will have duplicated reasons in the error message.
		reasons := make(map[string]int)
		for _, status := range f.Diagnosis.RunnerToStatusMap {
			for _, reason := range status.Reasons() {
				reasons[reason]++
			}
		}

		sortReasonsHistogram := func() []string {
			var reasonStrings []string
			for k, v := range reasons {
				reasonStrings = append(reasonStrings, fmt.Sprintf("%v %v", v, k))
			}
			sort.Strings(reasonStrings)
			return reasonStrings
		}
		sortedFilterMsg := sortReasonsHistogram()
		if len(sortedFilterMsg) != 0 {
			reasonMsg += fmt.Sprintf(" %v.", strings.Join(sortedFilterMsg, ", "))
		}
	}

	// Add the messages from PostFilter plugins to reasonMsg.
	// We can add this message regardless of whether the scheduling cycle fails at PreFilter or Filter
	// since we may run PostFilter (if enabled) in both cases.
	postFilterMsg := f.Diagnosis.PostFilterMsg
	if postFilterMsg != "" {
		reasonMsg += fmt.Sprintf(" %v", postFilterMsg)
	}
	return reasonMsg
}

// ImageStateSummary provides summarized information about the state of an image.
type ImageStateSummary struct {
	// Size of the image
	Size int64
	// Used to track how many nodes have this image, it is computed from the Runners field below
	// during the execution of Snapshot.
	NumRunners int
	// A set of node names for nodes having this image present. This field is used for
	// keeping track of the nodes during update/add/remove events.
	Runners sets.Set[string]
}

// Snapshot returns a copy without Runners field of ImageStateSummary
func (iss *ImageStateSummary) Snapshot() *ImageStateSummary {
	return &ImageStateSummary{
		Size:       iss.Size,
		NumRunners: iss.Runners.Len(),
	}
}

// RunnerInfo is node level aggregated information.
type RunnerInfo struct {
	// Overall node information.
	node *corev1.Runner

	// Regions running on the node.
	Regions []*RegionInfo

	// The subset of regions with affinity.
	RegionsWithAffinity []*RegionInfo

	// The subset of regions with required anti-affinity.
	RegionsWithRequiredAntiAffinity []*RegionInfo

	// Total requested resources of all regions on this node. This includes assumed
	// regions, which scheduler has sent for binding, but may not be scheduled yet.
	Requested *Resource
	// Total requested resources of all regions on this node with a minimum value
	// applied to each container's CPU and memory requests. This does not reflect
	// the actual resource requests for this node, but is used to avoid scheduling
	// many zero-request regions onto one node.
	NonZeroRequested *Resource
	// We store allocatedResources (which is Runner.Status.Allocatable.*) explicitly
	// as int64, to avoid conversions and accessing map.
	Allocatable *Resource

	// Whenever RunnerInfo changes, generation is bumped.
	// This is used to avoid cloning it if the object didn't change.
	Generation int64
}

// nextGeneration: Let's make sure history never forgets the name...
// Increments the generation number monotonically ensuring that generation numbers never collide.
// Collision of the generation numbers would be particularly problematic if a node was deleted and
// added back with the same name. See issue#63262.
func nextGeneration() int64 {
	return atomic.AddInt64(&generation, 1)
}

// Resource is a collection of compute resource.
type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	// We store allowedRegionNumber (which is Runner.Status.Allocatable.Regions().Value())
	// explicitly as int, to avoid conversions and improve performance.
	AllowedRegionNumber int
	// ScalarResources
	ScalarResources map[corev1.ResourceName]int64
}

// NewResource creates a Resource from ResourceList
func NewResource(rl corev1.ResourceList) *Resource {
	r := &Resource{}
	r.Add(rl)
	return r
}

// Add adds ResourceList into Resource.
func (r *Resource) Add(rl corev1.ResourceList) {
	if r == nil {
		return
	}

	for rName, rQuant := range rl {
		switch rName {
		case corev1.ResourceCPU:
			r.MilliCPU += rQuant.MilliValue()
		case corev1.ResourceMemory:
			r.Memory += rQuant.Value()
		default:
		}
	}
}

// Clone returns a copy of this resource.
func (r *Resource) Clone() *Resource {
	res := &Resource{
		MilliCPU:            r.MilliCPU,
		Memory:              r.Memory,
		AllowedRegionNumber: r.AllowedRegionNumber,
		EphemeralStorage:    r.EphemeralStorage,
	}
	if r.ScalarResources != nil {
		res.ScalarResources = make(map[corev1.ResourceName]int64, len(r.ScalarResources))
		for k, v := range r.ScalarResources {
			res.ScalarResources[k] = v
		}
	}
	return res
}

// AddScalar adds a resource by a scalar value of this resource.
func (r *Resource) AddScalar(name corev1.ResourceName, quantity int64) {
	r.SetScalar(name, r.ScalarResources[name]+quantity)
}

// SetScalar sets a resource by a scalar value of this resource.
func (r *Resource) SetScalar(name corev1.ResourceName, quantity int64) {
	// Lazily allocate scalar resource map.
	if r.ScalarResources == nil {
		r.ScalarResources = map[corev1.ResourceName]int64{}
	}
	r.ScalarResources[name] = quantity
}

// SetMaxResource compares with ResourceList and takes max value for each Resource.
func (r *Resource) SetMaxResource(rl corev1.ResourceList) {
	if r == nil {
		return
	}

	for rName, rQuantity := range rl {
		switch rName {
		case corev1.ResourceMemory:
			r.Memory = max(r.Memory, rQuantity.Value())
		case corev1.ResourceCPU:
			r.MilliCPU = max(r.MilliCPU, rQuantity.MilliValue())
		case corev1.ResourceEphemeralStorage:
			r.EphemeralStorage = max(r.EphemeralStorage, rQuantity.Value())
		default:
		}
	}
}

// NewRunnerInfo returns a ready to use empty RunnerInfo object.
// If any regions are given in arguments, their information will be aggregated in
// the returned object.
func NewRunnerInfo(regions ...*corev1.Region) *RunnerInfo {
	ni := &RunnerInfo{
		Requested:        &Resource{},
		NonZeroRequested: &Resource{},
		Allocatable:      &Resource{},
		Generation:       nextGeneration(),
	}
	for _, region := range regions {
		ni.AddRegion(region)
	}
	return ni
}

// Runner returns overall information about this node.
func (n *RunnerInfo) Runner() *corev1.Runner {
	if n == nil {
		return nil
	}
	return n.node
}

// Snapshot returns a copy of this node, Except that ImageStates is copied without the Runners field.
func (n *RunnerInfo) Snapshot() *RunnerInfo {
	clone := &RunnerInfo{
		node:             n.node,
		Requested:        n.Requested.Clone(),
		NonZeroRequested: n.NonZeroRequested.Clone(),
		Allocatable:      n.Allocatable.Clone(),
		Generation:       n.Generation,
	}
	if len(n.Regions) > 0 {
		clone.Regions = append([]*RegionInfo(nil), n.Regions...)
	}
	if len(n.RegionsWithAffinity) > 0 {
		clone.RegionsWithAffinity = append([]*RegionInfo(nil), n.RegionsWithAffinity...)
	}
	if len(n.RegionsWithRequiredAntiAffinity) > 0 {
		clone.RegionsWithRequiredAntiAffinity = append([]*RegionInfo(nil), n.RegionsWithRequiredAntiAffinity...)
	}
	return clone
}

// String returns representation of human readable format of this RunnerInfo.
func (n *RunnerInfo) String() string {
	regionKeys := make([]string, len(n.Regions))
	for i, p := range n.Regions {
		regionKeys[i] = p.Region.Name
	}
	return fmt.Sprintf("&RunnerInfo{Regions:%v, RequestedResource:%#v, NonZeroRequest: %#v, AllocatableResource:%#v}",
		regionKeys, n.Requested, n.NonZeroRequested, n.Allocatable)
}

// AddRegionInfo adds region information to this RunnerInfo.
// Consider using this instead of AddRegion if a RegionInfo is already computed.
func (n *RunnerInfo) AddRegionInfo(regionInfo *RegionInfo) {
	n.Regions = append(n.Regions, regionInfo)
	if regionWithAffinity(regionInfo.Region) {
		n.RegionsWithAffinity = append(n.RegionsWithAffinity, regionInfo)
	}
	if regionWithRequiredAntiAffinity(regionInfo.Region) {
		n.RegionsWithRequiredAntiAffinity = append(n.RegionsWithRequiredAntiAffinity, regionInfo)
	}
	n.update(regionInfo.Region, 1)
}

// NewRegionInfo returns a new RegionInfo.
func NewRegionInfo(region *corev1.Region) (*RegionInfo, error) {
	pInfo := &RegionInfo{}
	err := pInfo.Update(region)
	return pInfo, err
}

// AddRegion is a wrapper around AddRegionInfo.
func (n *RunnerInfo) AddRegion(region *corev1.Region) {
	// ignore this err since apiserver doesn't properly validate affinity terms
	// and we can't fix the validation for backwards compatibility.
	regionInfo, _ := NewRegionInfo(region)
	n.AddRegionInfo(regionInfo)
}

func regionWithAffinity(p *corev1.Region) bool {
	return true
}

func regionWithRequiredAntiAffinity(p *corev1.Region) bool {
	return true
}

func removeFromSlice(logger klog.Logger, s []*RegionInfo, k string) ([]*RegionInfo, bool) {
	var removed bool
	for i := range s {
		tmpKey, err := GetRegionKey(s[i].Region)
		if err != nil {
			logger.Error(err, "Cannot get region key", "region", klog.KObj(s[i].Region))
			continue
		}
		if k == tmpKey {
			// delete the element
			s[i] = s[len(s)-1]
			s = s[:len(s)-1]
			removed = true
			break
		}
	}
	// resets the slices to nil so that we can do DeepEqual in unit tests.
	if len(s) == 0 {
		return nil, removed
	}
	return s, removed
}

// RemoveRegion subtracts region information from this RunnerInfo.
func (n *RunnerInfo) RemoveRegion(logger klog.Logger, region *corev1.Region) error {
	k, err := GetRegionKey(region)
	if err != nil {
		return err
	}
	if regionWithAffinity(region) {
		n.RegionsWithAffinity, _ = removeFromSlice(logger, n.RegionsWithAffinity, k)
	}
	if regionWithRequiredAntiAffinity(region) {
		n.RegionsWithRequiredAntiAffinity, _ = removeFromSlice(logger, n.RegionsWithRequiredAntiAffinity, k)
	}

	var removed bool
	if n.Regions, removed = removeFromSlice(logger, n.Regions, k); removed {
		n.update(region, -1)
		return nil
	}
	return fmt.Errorf("no corresponding region %s in regions of node %s", region.Name, n.node.Name)
}

// update node info based on the region and sign.
// The sign will be set to `+1` when AddRegion and to `-1` when RemoveRegion.
func (n *RunnerInfo) update(region *corev1.Region, sign int64) {
	res, non0CPU, non0Mem := calculateResource(region)
	n.Requested.MilliCPU += sign * res.MilliCPU
	n.Requested.Memory += sign * res.Memory
	n.Requested.EphemeralStorage += sign * res.EphemeralStorage
	if n.Requested.ScalarResources == nil && len(res.ScalarResources) > 0 {
		n.Requested.ScalarResources = map[corev1.ResourceName]int64{}
	}
	for rName, rQuant := range res.ScalarResources {
		n.Requested.ScalarResources[rName] += sign * rQuant
	}
	n.NonZeroRequested.MilliCPU += sign * non0CPU
	n.NonZeroRequested.Memory += sign * non0Mem

	n.Generation = nextGeneration()
}

func calculateResource(region *corev1.Region) (Resource, int64, int64) {
	var non0CPU, non0Mem int64
	var res Resource
	return res, non0CPU, non0Mem
}

// SetRunner sets the overall node information.
func (n *RunnerInfo) SetRunner(node *corev1.Runner) {
	n.node = node
	n.Generation = nextGeneration()
}

// RemoveRunner removes the node object, leaving all other tracking information.
func (n *RunnerInfo) RemoveRunner() {
	n.node = nil
	n.Generation = nextGeneration()
}

// GetRegionKey returns the string key of a region.
func GetRegionKey(region *corev1.Region) (string, error) {
	uid := string(region.UID)
	if len(uid) == 0 {
		return "", errors.New("cannot get cache key for region with empty UID")
	}
	return uid, nil
}

// GetNamespacedName returns the string format of a namespaced resource name.
func GetNamespacedName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}
