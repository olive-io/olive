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

package v1

import (
	"bytes"
	"fmt"
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	// SchedulerDefaultLockObjectNamespace defines default scheduler lock object namespace ("olive-system")
	SchedulerDefaultLockObjectNamespace string = "olive-system"

	// SchedulerDefaultLockObjectName defines default scheduler lock object name ("kube-scheduler")
	SchedulerDefaultLockObjectName = "olive-mon-scheduler"

	// SchedulerDefaultProviderName defines the default provider names
	SchedulerDefaultProviderName = "DefaultProvider"
)

// DebuggingConfiguration holds configuration for Debugging related features.
type DebuggingConfiguration struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SchedulerConfiguration configures a scheduler
type SchedulerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Parallelism defines the amount of parallelism in algorithms for scheduling a Regions. Must be greater than 0. Defaults to 16
	Parallelism int32 `json:"parallelism,omitempty" protobuf:"varint,1,opt,name=parallelism"`

	// enableProfiling enables profiling via web interface host:port/debug/pprof/
	EnableProfiling bool `json:"enableProfiling,omitempty" protobuf:"varint,2,opt,name=enableProfiling"`
	// enableContentionProfiling enables block profiling, if
	// enableProfiling is true.
	EnableContentionProfiling bool `json:"enableContentionProfiling,omitempty" protobuf:"varint,3,opt,name=enableContentionProfiling"`

	// PercentageOfRunnersToScore is the percentage of all runners that once found feasible
	// for running a region, the scheduler stops its search for more feasible runners in
	// the cluster. This helps improve scheduler's performance. Scheduler always tries to find
	// at least "minFeasibleRunnersToFind" feasible runners no matter what the value of this flag is.
	// Example: if the cluster size is 500 runners and the value of this flag is 30,
	// then scheduler stops finding further feasible runners once it finds 150 feasible ones.
	// When the value is 0, default percentage (5%--50% based on the size of the cluster) of the
	// runners will be scored. It is overridden by profile level PercentageofRunnersToScore.
	PercentageOfRunnersToScore int32 `json:"percentageOfRunnersToScore,omitempty" protobuf:"varint,4,opt,name=percentageOfRunnersToScore"`

	// RegionInitialBackoffSeconds is the initial backoff for unschedulable regions.
	// If specified, it must be greater than 0. If this value is null, the default value (1s)
	// will be used.
	RegionInitialBackoffSeconds int64 `json:"regionInitialBackoffSeconds,omitempty" protobuf:"varint,5,opt,name=regionInitialBackoffSeconds"`

	// RegionMaxBackoffSeconds is the max backoff for unschedulable regions.
	// If specified, it must be greater than regionInitialBackoffSeconds. If this value is null,
	// the default value (10s) will be used.
	RegionMaxBackoffSeconds int64 `json:"regionMaxBackoffSeconds,omitempty" protobuf:"varint,6,opt,name=regionMaxBackoffSeconds"`

	// Profiles are scheduling profiles that kube-scheduler supports. Regions can
	// choose to be scheduled under a particular profile by setting its associated
	// scheduler name. Regions that don't specify any scheduler name are scheduled
	// with the "default-scheduler" profile, if present here.
	// +listType=map
	// +listMapKey=schedulerName
	Profiles []SchedulerProfile `json:"profiles,omitempty" protobuf:"bytes,7,rep,name=profiles"`

	// Extenders are the list of scheduler extenders, each holding the values of how to communicate
	// with the extender. These extenders are shared by all scheduler profiles.
	// +listType=set
	Extenders []Extender `json:"extenders,omitempty" protobuf:"bytes,8,rep,name=extenders"`

	// DelayCacheUntilActive specifies when to start caching. If this is true and leader election is enabled,
	// the scheduler will wait to fill informer caches until it is the leader. Doing so will have slower
	// failover with the benefit of lower memory overhead while waiting to become leader.
	// Defaults to false.
	DelayCacheUntilActive bool `json:"delayCacheUntilActive,omitempty" protobuf:"varint,9,opt,name=delayCacheUntilActive"`
}

// DecodeNestedObjects decodes plugin args for known types.
func (c *SchedulerConfiguration) DecodeNestedObjects(d runtime.Decoder) error {
	var strictDecodingErrs []error
	for i := range c.Profiles {
		prof := &c.Profiles[i]
		for j := range prof.PluginConfig {
			err := prof.PluginConfig[j].decodeNestedObjects(d)
			if err != nil {
				decodingErr := fmt.Errorf("decoding .profiles[%d].pluginConfig[%d]: %w", i, j, err)
				if runtime.IsStrictDecodingError(err) {
					strictDecodingErrs = append(strictDecodingErrs, decodingErr)
				} else {
					return decodingErr
				}
			}
		}
	}
	if len(strictDecodingErrs) > 0 {
		return runtime.NewStrictDecodingError(strictDecodingErrs)
	}
	return nil
}

// EncodeNestedObjects encodes plugin args.
func (c *SchedulerConfiguration) EncodeNestedObjects(e runtime.Encoder) error {
	for i := range c.Profiles {
		prof := &c.Profiles[i]
		for j := range prof.PluginConfig {
			err := prof.PluginConfig[j].encodeNestedObjects(e)
			if err != nil {
				return fmt.Errorf("encoding .profiles[%d].pluginConfig[%d]: %w", i, j, err)
			}
		}
	}
	return nil
}

// SchedulerProfile is a scheduling profile.
type SchedulerProfile struct {
	// SchedulerName is the name of the scheduler associated to this profile.
	// If SchedulerName matches with the region's "spec.schedulerName", then the region
	// is scheduled with this profile.
	SchedulerName string `json:"schedulerName,omitempty" protobuf:"bytes,1,opt,name=schedulerName"`

	// PercentageOfRunnersToScore is the percentage of all runners that once found feasible
	// for running a region, the scheduler stops its search for more feasible runners in
	// the cluster. This helps improve scheduler's performance. Scheduler always tries to find
	// at least "minFeasibleRunnersToFind" feasible runners no matter what the value of this flag is.
	// Example: if the cluster size is 500 runners and the value of this flag is 30,
	// then scheduler stops finding further feasible runners once it finds 150 feasible ones.
	// When the value is 0, default percentage (5%--50% based on the size of the cluster) of the
	// runners will be scored. It will override global PercentageOfRunnersToScore. If it is empty,
	// global PercentageOfRunnersToScore will be used.
	PercentageOfRunnersToScore int32 `json:"percentageOfRunnersToScore,omitempty" protobuf:"varint,2,opt,name=percentageOfRunnersToScore"`

	// Plugins specify the set of plugins that should be enabled or disabled.
	// Enabled plugins are the ones that should be enabled in addition to the
	// default plugins. Disabled plugins are any of the default plugins that
	// should be disabled.
	// When no enabled or disabled plugin is specified for an extension point,
	// default plugins for that extension point will be used if there is any.
	// If a QueueSort plugin is specified, the same QueueSort Plugin and
	// PluginConfig must be specified for all profiles.
	Plugins *Plugins `json:"plugins,omitempty" protobuf:"bytes,3,opt,name=plugins"`

	// PluginConfig is an optional set of custom plugin arguments for each plugin.
	// Omitting config args for a plugin is equivalent to using the default config
	// for that plugin.
	// +listType=map
	// +listMapKey=name
	PluginConfig []PluginConfig `json:"pluginConfig,omitempty" protobuf:"bytes,4,rep,name=pluginConfig"`
}

// Plugins include multiple extension points. When specified, the list of plugins for
// a particular extension point are the only ones enabled. If an extension point is
// omitted from the config, then the default set of plugins is used for that extension point.
// Enabled plugins are called in the order specified here, after default plugins. If they need to
// be invoked before default plugins, default plugins must be disabled and re-enabled here in desired order.
type Plugins struct {
	// PreEnqueue is a list of plugins that should be invoked before adding regions to the scheduling queue.
	PreEnqueue PluginSet `json:"preEnqueue,omitempty" protobuf:"bytes,1,opt,name=preEnqueue"`

	// QueueSort is a list of plugins that should be invoked when sorting regions in the scheduling queue.
	QueueSort PluginSet `json:"queueSort,omitempty" protobuf:"bytes,2,opt,name=queueSort"`

	// PreFilter is a list of plugins that should be invoked at "PreFilter" extension point of the scheduling framework.
	PreFilter PluginSet `json:"preFilter,omitempty" protobuf:"bytes,3,opt,name=preFilter"`

	// Filter is a list of plugins that should be invoked when filtering out runners that cannot run the Region.
	Filter PluginSet `json:"filter,omitempty" protobuf:"bytes,4,opt,name=filter"`

	// PostFilter is a list of plugins that are invoked after filtering phase, but only when no feasible runners were found for the region.
	PostFilter PluginSet `json:"postFilter,omitempty" protobuf:"bytes,5,opt,name=postFilter"`

	// PreScore is a list of plugins that are invoked before scoring.
	PreScore PluginSet `json:"preScore,omitempty" protobuf:"bytes,6,opt,name=preScore"`

	// Score is a list of plugins that should be invoked when ranking runners that have passed the filtering phase.
	Score PluginSet `json:"score,omitempty" protobuf:"bytes,7,opt,name=score"`

	// Reserve is a list of plugins invoked when reserving/unreserving resources
	// after a runner is assigned to run the region.
	Reserve PluginSet `json:"reserve,omitempty" protobuf:"bytes,8,opt,name=reserve"`

	// Permit is a list of plugins that control binding of a Region. These plugins can prevent or delay binding of a Region.
	Permit PluginSet `json:"permit,omitempty" protobuf:"bytes,9,opt,name=permit"`

	// PreBind is a list of plugins that should be invoked before a region is bound.
	PreBind PluginSet `json:"preBind,omitempty" protobuf:"bytes,10,opt,name=preBind"`

	// Bind is a list of plugins that should be invoked at "Bind" extension point of the scheduling framework.
	// The scheduler call these plugins in order. Scheduler skips the rest of these plugins as soon as one returns success.
	Bind PluginSet `json:"bind,omitempty" protobuf:"bytes,11,opt,name=bind"`

	// PostBind is a list of plugins that should be invoked after a region is successfully bound.
	PostBind PluginSet `json:"postBind,omitempty" protobuf:"bytes,12,opt,name=postBind"`

	// MultiPoint is a simplified config section to enable plugins for all valid extension points.
	// Plugins enabled through MultiPoint will automatically register for every individual extension
	// point the plugin has implemented. Disabling a plugin through MultiPoint disables that behavior.
	// The same is true for disabling "*" through MultiPoint (no default plugins will be automatically registered).
	// Plugins can still be disabled through their individual extension points.
	//
	// In terms of precedence, plugin config follows this basic hierarchy
	//   1. Specific extension points
	//   2. Explicitly configured MultiPoint plugins
	//   3. The set of default plugins, as MultiPoint plugins
	// This implies that a higher precedence plugin will run first and overwrite any settings within MultiPoint.
	// Explicitly user-configured plugins also take a higher precedence over default plugins.
	// Within this hierarchy, an Enabled setting takes precedence over Disabled. For example, if a plugin is
	// set in both `multiPoint.Enabled` and `multiPoint.Disabled`, the plugin will be enabled. Similarly,
	// including `multiPoint.Disabled = '*'` and `multiPoint.Enabled = pluginA` will still register that specific
	// plugin through MultiPoint. This follows the same behavior as all other extension point configurations.
	MultiPoint PluginSet `json:"multiPoint,omitempty" protobuf:"bytes,13,opt,name=multiPoint"`
}

// PluginSet specifies enabled and disabled plugins for an extension point.
// If an array is empty, missing, or nil, default plugins at that extension point will be used.
type PluginSet struct {
	// Enabled specifies plugins that should be enabled in addition to default plugins.
	// If the default plugin is also configured in the scheduler config file, the weight of plugin will
	// be overridden accordingly.
	// These are called after default plugins and in the same order specified here.
	// +listType=atomic
	Enabled []Plugin `json:"enabled,omitempty" protobuf:"bytes,1,rep,name=enabled"`
	// Disabled specifies default plugins that should be disabled.
	// When all default plugins need to be disabled, an array containing only one "*" should be provided.
	// +listType=map
	// +listMapKey=name
	Disabled []Plugin `json:"disabled,omitempty" protobuf:"bytes,2,rep,name=disabled"`
}

/*
 * NOTE: The following variables and methods are intentionally left out of the staging mirror.
 */
const (
	// DefaultPercentageOfRunnersToScore defines the percentage of runners of all runners
	// that once found feasible, the scheduler stops looking for more runners.
	// A value of 0 means adaptive, meaning the scheduler figures out a proper default.
	DefaultPercentageOfRunnersToScore = 0

	// MaxCustomPriorityScore is the max score UtilizationShapePoint expects.
	MaxCustomPriorityScore int64 = 10

	// MaxTotalScore is the maximum total score.
	MaxTotalScore int64 = math.MaxInt64

	// MaxWeight defines the max weight value allowed for custom PriorityPolicy
	MaxWeight = MaxTotalScore / MaxCustomPriorityScore
)

// Plugin specifies a plugin name and its weight when applicable. Weight is used only for Score plugins.
type Plugin struct {
	// Name defines the name of plugin
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Weight defines the weight of plugin, only used for Score plugins.
	Weight int32 `json:"weight,omitempty" protobuf:"varint,2,opt,name=weight"`
}

// PluginConfig specifies arguments that should be passed to a plugin at the time of initialization.
// A plugin that is invoked at multiple extension points is initialized once. Args can have arbitrary structure.
// It is up to the plugin to process these Args.
type PluginConfig struct {
	// Name defines the name of plugin being configured
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Args defines the arguments passed to the plugins at the time of initialization. Args can have arbitrary structure.
	Args runtime.RawExtension `json:"args,omitempty" protobuf:"bytes,2,opt,name=args"`
}

func (c *PluginConfig) decodeNestedObjects(d runtime.Decoder) error {
	gvk := SchemeGroupVersion.WithKind(c.Name + "Args")
	// dry-run to detect and skip out-of-tree plugin args.
	if _, _, err := d.Decode(nil, &gvk, nil); runtime.IsNotRegisteredError(err) {
		return nil
	}

	var strictDecodingErr error
	obj, parsedGvk, err := d.Decode(c.Args.Raw, &gvk, nil)
	if err != nil {
		decodingArgsErr := fmt.Errorf("decoding args for plugin %s: %w", c.Name, err)
		if obj != nil && runtime.IsStrictDecodingError(err) {
			strictDecodingErr = runtime.NewStrictDecodingError([]error{decodingArgsErr})
		} else {
			return decodingArgsErr
		}
	}
	if parsedGvk.GroupKind() != gvk.GroupKind() {
		return fmt.Errorf("args for plugin %s were not of type %s, got %s", c.Name, gvk.GroupKind(), parsedGvk.GroupKind())
	}
	c.Args.Object = obj
	return strictDecodingErr
}

func (c *PluginConfig) encodeNestedObjects(e runtime.Encoder) error {
	if c.Args.Object == nil {
		return nil
	}
	var buf bytes.Buffer
	err := e.Encode(c.Args.Object, &buf)
	if err != nil {
		return err
	}
	// The <e> encoder might be a YAML encoder, but the parent encoder expects
	// JSON output, so we convert YAML back to JSON.
	// This is a no-op if <e> produces JSON.
	json, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return err
	}
	c.Args.Raw = json
	return nil
}

// Extender holds the parameters used to communicate with the extender. If a verb is unspecified/empty,
// it is assumed that the extender chose not to provide that extension.
type Extender struct {
	// URLPrefix at which the extender is available
	URLPrefix string `json:"urlPrefix" protobuf:"bytes,1,opt,name=urlPrefix"`
	// Verb for the filter call, empty if not supported. This verb is appended to the URLPrefix when issuing the filter call to extender.
	FilterVerb string `json:"filterVerb,omitempty" protobuf:"bytes,2,opt,name=filterVerb"`
	// Verb for the preempt call, empty if not supported. This verb is appended to the URLPrefix when issuing the preempt call to extender.
	PreemptVerb string `json:"preemptVerb,omitempty" protobuf:"bytes,3,opt,name=preemptVerb"`
	// Verb for the prioritize call, empty if not supported. This verb is appended to the URLPrefix when issuing the prioritize call to extender.
	PrioritizeVerb string `json:"prioritizeVerb,omitempty" protobuf:"bytes,4,opt,name=prioritizeVerb"`
	// The numeric multiplier for the runner scores that the prioritize call generates.
	// The weight should be a positive integer
	Weight int64 `json:"weight,omitempty" protobuf:"varint,5,opt,name=weight"`
	// Verb for the bind call, empty if not supported. This verb is appended to the URLPrefix when issuing the bind call to extender.
	// If this method is implemented by the extender, it is the extender's responsibility to bind the region to apiserver. Only one extender
	// can implement this function.
	BindVerb string `json:"bindVerb,omitempty" protobuf:"bytes,6,opt,name=bindVerb"`
	// EnableHTTPS specifies whether https should be used to communicate with the extender
	EnableHTTPS bool `json:"enableHTTPS,omitempty" protobuf:"varint,7,opt,name=enableHTTPS"`
	// TLSConfig specifies the transport layer security config
	TLSConfig *ExtenderTLSConfig `json:"tlsConfig,omitempty" protobuf:"bytes,8,opt,name=tlsConfig"`
	// HTTPTimeout specifies the timeout duration for a call to the extender. Filter timeout fails the scheduling of the region. Prioritize
	// timeout is ignored, k8s/other extenders priorities are used to select the runner.
	HTTPTimeout metav1.Duration `json:"httpTimeout,omitempty" protobuf:"bytes,9,opt,name=httpTimeout"`
	// RunnerCacheCapable specifies that the extender is capable of caching runner information,
	// so the scheduler should only send minimal information about the eligible runners
	// assuming that the extender already cached full details of all runners in the cluster
	RunnerCacheCapable bool `json:"runnerCacheCapable,omitempty" protobuf:"varint,10,opt,name=runnerCacheCapable"`
	// ManagedResources is a list of extended resources that are managed by
	// this extender.
	// - A region will be sent to the extender on the Filter, Prioritize and Bind
	//   (if the extender is the binder) phases iff the region requests at least
	//   one of the extended resources in this list. If empty or unspecified,
	//   all regions will be sent to this extender.
	// - If IgnoredByScheduler is set to true for a resource, kube-scheduler
	//   will skip checking the resource in predicates.
	// +optional
	// +listType=atomic
	ManagedResources []ExtenderManagedResource `json:"managedResources,omitempty" protobuf:"bytes,11,rep,name=managedResources"`
	// Ignorable specifies if the extender is ignorable, i.e. scheduling should not
	// fail when the extender returns an error or is not reachable.
	Ignorable bool `json:"ignorable,omitempty" protobuf:"varint,12,opt,name=ignorable"`
}

// ExtenderManagedResource describes the arguments of extended resources
// managed by an extender.
type ExtenderManagedResource struct {
	// Name is the extended resource name.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// IgnoredByScheduler indicates whether kube-scheduler should ignore this
	// resource when applying predicates.
	IgnoredByScheduler bool `json:"ignoredByScheduler,omitempty" protobuf:"varint,2,opt,name=ignoredByScheduler"`
}

// ExtenderTLSConfig contains settings to enable TLS with extender
type ExtenderTLSConfig struct {
	// Server should be accessed without verifying the TLS certificate. For testing only.
	Insecure bool `json:"insecure,omitempty" protobuf:"varint,1,opt,name=insecure"`
	// ServerName is passed to the server for SNI and is used in the client to check server
	// certificates against. If ServerName is empty, the hostname used to contact the
	// server is used.
	ServerName string `json:"serverName,omitempty" protobuf:"bytes,2,opt,name=serverName"`

	// Server requires TLS client certificate authentication
	CertFile string `json:"certFile,omitempty" protobuf:"bytes,3,opt,name=certFile"`
	// Server requires TLS client certificate authentication
	KeyFile string `json:"keyFile,omitempty" protobuf:"bytes,4,opt,name=keyFile"`
	// Trusted root certificates for server
	CAFile string `json:"caFile,omitempty" protobuf:"bytes,5,opt,name=caFile"`

	// CertData holds PEM-encoded bytes (typically read from a client certificate file).
	// CertData takes precedence over CertFile
	// +listType=atomic
	CertData []byte `json:"certData,omitempty" protobuf:"bytes,6,opt,name=certData"`
	// KeyData holds PEM-encoded bytes (typically read from a client certificate key file).
	// KeyData takes precedence over KeyFile
	// +listType=atomic
	KeyData []byte `json:"keyData,omitempty" protobuf:"bytes,7,opt,name=keyData"`
	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CAData takes precedence over CAFile
	// +listType=atomic
	CAData []byte `json:"caData,omitempty" protobuf:"bytes,8,opt,name=caData"`
}
