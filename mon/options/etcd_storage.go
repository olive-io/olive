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

package options

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	genericserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/server/options/encryptionconfig"
	encryptionconfigcontroller "k8s.io/apiserver/pkg/server/options/encryptionconfig/controller"
	"k8s.io/apiserver/pkg/storage/etcd3/metrics"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	storagefactory "k8s.io/apiserver/pkg/storage/storagebackend/factory"
	storagevalue "k8s.io/apiserver/pkg/storage/value"
	"k8s.io/klog/v2"

	"github.com/olive-io/olive/apis"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	serverstorage "github.com/olive-io/olive/mon/storage"
)

type EtcdStorageOptions struct {
	StorageConfig                           storagebackend.Config
	EncryptionProviderConfigFilepath        string
	EncryptionProviderConfigAutomaticReload bool

	EtcdServersOverrides []string

	// To enable protobuf as storage format, it is enough
	// to set it to "application/vnd.kubernetes.protobuf".
	DefaultStorageMediaType string
	DeleteCollectionWorkers int
	EnableGarbageCollection bool

	// Set EnableWatchCache to false to disable all watch caches
	EnableWatchCache bool
	// Set DefaultWatchCacheSize to zero to disable watch caches for those resources that have no explicit cache size set
	DefaultWatchCacheSize int
	// WatchCacheSizes represents override to a given resource
	WatchCacheSizes []string

	// SkipHealthEndpoints, when true, causes the Apply methods to not set up health endpoints.
	// This allows multiple invocations of the Apply methods without duplication of said endpoints.
	SkipHealthEndpoints bool
}

func NewStorageEtcdOptions(backendConfig *storagebackend.Config) *EtcdStorageOptions {
	options := &EtcdStorageOptions{
		StorageConfig:           *backendConfig,
		DefaultStorageMediaType: krt.ContentTypeJSON,
		DeleteCollectionWorkers: 1,
		EnableGarbageCollection: true,
		EnableWatchCache:        true,
		DefaultWatchCacheSize:   100,
	}
	options.StorageConfig.Type = storagebackend.StorageTypeETCD3
	options.StorageConfig.CountMetricPollPeriod = time.Minute
	return options
}

var storageMediaTypes = sets.New(
	krt.ContentTypeJSON,
	krt.ContentTypeYAML,
	krt.ContentTypeProtobuf,
)

func (s *EtcdStorageOptions) Validate() []error {
	if s == nil {
		return nil
	}

	allErrors := []error{}
	for _, override := range s.EtcdServersOverrides {
		tokens := strings.Split(override, "#")
		if len(tokens) != 2 {
			allErrors = append(allErrors, fmt.Errorf("--etcd-servers-overrides invalid, must be of format: group/resource#servers, where servers are URLs, semicolon separated"))
			continue
		}

		apiresource := strings.Split(tokens[0], "/")
		if len(apiresource) != 2 {
			allErrors = append(allErrors, fmt.Errorf("--etcd-servers-overrides invalid, must be of format: group/resource#servers, where servers are URLs, semicolon separated"))
			continue
		}

	}

	if len(s.EncryptionProviderConfigFilepath) == 0 && s.EncryptionProviderConfigAutomaticReload {
		allErrors = append(allErrors, fmt.Errorf("--encryption-provider-config-automatic-reload must be set with --encryption-provider-config"))
	}

	if s.DefaultStorageMediaType != "" && !storageMediaTypes.Has(s.DefaultStorageMediaType) {
		allErrors = append(allErrors, fmt.Errorf("--storage-media-type %q invalid, allowed values: %s", s.DefaultStorageMediaType, strings.Join(sets.List(storageMediaTypes), ", ")))
	}

	return allErrors
}

// AddFlags adds flags related to etcd storage for a specific APIServer to the specified FlagSet
func (s *EtcdStorageOptions) AddFlags(fs *pflag.FlagSet) {
	if s == nil {
		return
	}

	fs.StringSliceVar(&s.EtcdServersOverrides, "etcd-servers-overrides", s.EtcdServersOverrides, ""+
		"Per-resource etcd servers overrides, comma separated. The individual override "+
		"format: group/resource#servers, where servers are URLs, semicolon separated. "+
		"Note that this applies only to resources compiled into this server binary. ")

	fs.StringVar(&s.DefaultStorageMediaType, "storage-media-type", s.DefaultStorageMediaType, ""+
		"The media type to use to store objects in storage. "+
		"Some resources or storage backends may only support a specific media type and will ignore this setting. "+
		"Supported media types: [application/json, application/yaml, application/vnd.kubernetes.protobuf]")
	fs.IntVar(&s.DeleteCollectionWorkers, "delete-collection-workers", s.DeleteCollectionWorkers,
		"Number of workers spawned for DeleteCollection call. These are used to speed up namespace cleanup.")

	fs.BoolVar(&s.EnableGarbageCollection, "enable-garbage-collector", s.EnableGarbageCollection, ""+
		"Enables the generic garbage collector. MUST be synced with the corresponding flag "+
		"of the kube-controller-manager.")

	fs.BoolVar(&s.EnableWatchCache, "watch-cache", s.EnableWatchCache,
		"Enable watch caching in the apiserver")

	fs.IntVar(&s.DefaultWatchCacheSize, "default-watch-cache-size", s.DefaultWatchCacheSize,
		"Default watch cache size. If zero, watch cache will be disabled for resources that do not have a default watch size set.")

	_ = fs.MarkDeprecated("default-watch-cache-size",
		"watch caches are sized automatically and this flag will be removed in a future version")

	fs.StringSliceVar(&s.WatchCacheSizes, "watch-cache-sizes", s.WatchCacheSizes, ""+
		"Watch cache size settings for some resources (pods, nodes, etc.), comma separated. "+
		"The individual setting format: resource[.group]#size, where resource is lowercase plural (no version), "+
		"group is omitted for resources of apiVersion v1 (the legacy core API) and included for others, "+
		"and size is a number. This option is only meaningful for resources built into the apiserver, "+
		"not ones defined by CRDs or aggregated from external servers, and is only consulted if the "+
		"watch-cache is enabled. The only meaningful size setting to supply here is zero, which means to "+
		"disable watch caching for the associated resource; all non-zero values are equivalent and mean "+
		"to not disable watch caching for that resource")

	//fs.StringSliceVar(&s.StorageConfig.Transport.ServerList, "etcd-servers", s.StorageConfig.Transport.ServerList,
	//	"List of etcd servers to connect with (scheme://ip:port), comma separated.")

	fs.StringVar(&s.StorageConfig.Prefix, "etcd-prefix", s.StorageConfig.Prefix,
		"The prefix to prepend to all resource paths in etcd.")

	fs.StringVar(&s.StorageConfig.Transport.KeyFile, "etcd-keyfile", s.StorageConfig.Transport.KeyFile,
		"SSL key file used to secure etcd communication.")

	fs.StringVar(&s.StorageConfig.Transport.CertFile, "etcd-certfile", s.StorageConfig.Transport.CertFile,
		"SSL certification file used to secure etcd communication.")

	fs.StringVar(&s.StorageConfig.Transport.TrustedCAFile, "etcd-cafile", s.StorageConfig.Transport.TrustedCAFile,
		"SSL Certificate Authority file used to secure etcd communication.")

	fs.StringVar(&s.EncryptionProviderConfigFilepath, "encryption-provider-config", s.EncryptionProviderConfigFilepath,
		"The file containing configuration for encryption providers to be used for storing secrets in etcd")

	fs.BoolVar(&s.EncryptionProviderConfigAutomaticReload, "encryption-provider-config-automatic-reload", s.EncryptionProviderConfigAutomaticReload,
		"Determines if the file set by --encryption-provider-config should be automatically reloaded if the disk contents change. "+
			"Setting this to true disables the ability to uniquely identify distinct KMS plugins via the API server healthz endpoints.")

	fs.DurationVar(&s.StorageConfig.CompactionInterval, "etcd-compaction-interval", s.StorageConfig.CompactionInterval,
		"The interval of compaction requests. If 0, the compaction request from olive is disabled.")

	fs.DurationVar(&s.StorageConfig.CountMetricPollPeriod, "etcd-count-metric-poll-period", s.StorageConfig.CountMetricPollPeriod, ""+
		"Frequency of polling etcd for number of resources per type. 0 disables the metric collection.")

	fs.DurationVar(&s.StorageConfig.DBMetricPollInterval, "etcd-db-metric-poll-interval", s.StorageConfig.DBMetricPollInterval,
		"The interval of requests to poll etcd and update metric. 0 disables the metric collection")

	fs.DurationVar(&s.StorageConfig.HealthcheckTimeout, "etcd-healthcheck-timeout", s.StorageConfig.HealthcheckTimeout,
		"The timeout to use when checking etcd health.")

	fs.DurationVar(&s.StorageConfig.ReadycheckTimeout, "etcd-readycheck-timeout", s.StorageConfig.ReadycheckTimeout,
		"The timeout to use when checking etcd readiness")

	fs.Int64Var(&s.StorageConfig.LeaseManagerConfig.ReuseDurationSeconds, "lease-reuse-duration-seconds", s.StorageConfig.LeaseManagerConfig.ReuseDurationSeconds,
		"The time in seconds that each lease is reused. A lower value could avoid large number of objects reusing the same lease. Notice that a too small value may cause performance problems at storage layer.")
}

// ApplyTo mutates the provided server.Config.  It must never mutate the receiver (EtcdOptions).
func (s *EtcdStorageOptions) ApplyTo(c *genericserver.Config) error {
	if s == nil {
		return nil
	}

	storageFactoryConfig := NewStorageFactoryConfig()
	if c.MergedResourceConfig != nil {
		storageFactoryConfig.APIResourceConfig = &serverstorage.ResourceConfig{
			GroupVersionConfigs: c.MergedResourceConfig.GroupVersionConfigs,
			ResourceConfigs:     c.MergedResourceConfig.ResourceConfigs,
		}
	}
	storageFactory, err := storageFactoryConfig.Complete(s).New()
	if err != nil {
		return err
	}

	return s.ApplyWithStorageFactoryTo(storageFactory, c)
}

// ApplyWithStorageFactoryTo mutates the provided server.Config.  It must never mutate the receiver (EtcdOptions).
func (s *EtcdStorageOptions) ApplyWithStorageFactoryTo(factory serverstorage.StorageFactory, c *genericserver.Config) error {
	if s == nil {
		return nil
	}

	if !s.SkipHealthEndpoints {
		if err := s.addEtcdHealthEndpoint(c); err != nil {
			return err
		}
	}

	// setup encryption
	if err := s.maybeApplyResourceTransformers(c); err != nil {
		return err
	}

	metrics.SetStorageMonitorGetter(monitorGetter(factory))

	c.RESTOptionsGetter = s.CreateRESTOptionsGetter(factory, c.ResourceTransformers)
	return nil
}

func monitorGetter(factory serverstorage.StorageFactory) func() (monitors []metrics.Monitor, err error) {
	return func() (monitors []metrics.Monitor, err error) {
		defer func() {
			if err != nil {
				for _, m := range monitors {
					m.Close()
				}
			}
		}()

		var m metrics.Monitor
		for _, cfg := range factory.Configs() {
			m, err = storagefactory.CreateMonitor(cfg)
			if err != nil {
				return nil, err
			}
			monitors = append(monitors, m)
		}
		return monitors, nil
	}
}

func (s *EtcdStorageOptions) CreateRESTOptionsGetter(factory serverstorage.StorageFactory, resourceTransformers storagevalue.ResourceTransformers) generic.RESTOptionsGetter {
	if resourceTransformers != nil {
		factory = &transformerStorageFactory{
			delegate:             factory,
			resourceTransformers: resourceTransformers,
		}
	}
	return &StorageFactoryRestOptionsFactory{Options: *s, StorageFactory: factory}
}

func (s *EtcdStorageOptions) maybeApplyResourceTransformers(c *genericserver.Config) (err error) {
	if c.ResourceTransformers != nil {
		return nil
	}
	if len(s.EncryptionProviderConfigFilepath) == 0 {
		return nil
	}

	ctxServer := wait.ContextForChannel(c.DrainedNotify())
	ctxTransformers, closeTransformers := context.WithCancel(ctxServer)
	defer func() {
		// in case of error, we want to close partially initialized (if any) transformers
		if err != nil {
			closeTransformers()
		}
	}()

	encryptionConfiguration, err := encryptionconfig.LoadEncryptionConfig(ctxTransformers, s.EncryptionProviderConfigFilepath, s.EncryptionProviderConfigAutomaticReload, c.APIServerID)
	if err != nil {
		return err
	}

	if s.EncryptionProviderConfigAutomaticReload {
		// with reload=true we will always have 1 health check
		if len(encryptionConfiguration.HealthChecks) != 1 {
			return fmt.Errorf("failed to start kms encryption config hot reload controller. only 1 health check should be available when reload is enabled")
		}

		// Here the dynamic transformers take ownership of the transformers and their cancellation.
		dynamicTransformers := encryptionconfig.NewDynamicTransformers(encryptionConfiguration.Transformers, encryptionConfiguration.HealthChecks[0], closeTransformers, encryptionConfiguration.KMSCloseGracePeriod)

		// add post start hook to start hot reload controller
		// adding this hook here will ensure that it gets configured exactly once
		err = c.AddPostStartHook(
			"start-encryption-provider-config-automatic-reload",
			func(_ genericserver.PostStartHookContext) error {
				dynamicEncryptionConfigController := encryptionconfigcontroller.NewDynamicEncryptionConfiguration(
					"encryption-provider-config-automatic-reload-controller",
					s.EncryptionProviderConfigFilepath,
					dynamicTransformers,
					encryptionConfiguration.EncryptionFileContentHash,
					c.APIServerID,
				)

				go dynamicEncryptionConfigController.Run(ctxServer)

				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("failed to add post start hook for kms encryption config hot reload controller: %w", err)
		}

		c.ResourceTransformers = dynamicTransformers
		if !s.SkipHealthEndpoints {
			addHealthChecksWithoutLivez(c, dynamicTransformers)
		}
	} else {
		c.ResourceTransformers = encryptionconfig.StaticTransformers(encryptionConfiguration.Transformers)
		if !s.SkipHealthEndpoints {
			addHealthChecksWithoutLivez(c, encryptionConfiguration.HealthChecks...)
		}
	}

	return nil
}

func addHealthChecksWithoutLivez(c *genericserver.Config, healthChecks ...healthz.HealthChecker) {
	c.HealthzChecks = append(c.HealthzChecks, healthChecks...)
	c.ReadyzChecks = append(c.ReadyzChecks, healthChecks...)
}

func (s *EtcdStorageOptions) addEtcdHealthEndpoint(c *genericserver.Config) error {
	healthCheck, err := storagefactory.CreateHealthCheck(s.StorageConfig, c.DrainedNotify())
	if err != nil {
		return err
	}
	c.AddHealthChecks(healthz.NamedCheck("etcd", func(r *http.Request) error {
		return healthCheck()
	}))

	readyCheck, err := storagefactory.CreateReadyCheck(s.StorageConfig, c.DrainedNotify())
	if err != nil {
		return err
	}
	c.AddReadyzChecks(healthz.NamedCheck("etcd-readiness", func(r *http.Request) error {
		return readyCheck()
	}))

	return nil
}

type StorageFactoryRestOptionsFactory struct {
	Options        EtcdStorageOptions
	StorageFactory serverstorage.StorageFactory
}

func (f *StorageFactoryRestOptionsFactory) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	storageConfig, err := f.StorageFactory.NewConfig(resource)
	if err != nil {
		return generic.RESTOptions{}, fmt.Errorf("unable to find storage destination for %v, due to %v", resource, err.Error())
	}

	ret := generic.RESTOptions{
		StorageConfig:             storageConfig,
		Decorator:                 generic.UndecoratedStorage,
		DeleteCollectionWorkers:   f.Options.DeleteCollectionWorkers,
		EnableGarbageCollection:   f.Options.EnableGarbageCollection,
		ResourcePrefix:            f.StorageFactory.ResourcePrefix(resource),
		CountMetricPollPeriod:     f.Options.StorageConfig.CountMetricPollPeriod,
		StorageObjectCountTracker: f.Options.StorageConfig.StorageObjectCountTracker,
	}

	if f.Options.EnableWatchCache {
		sizes, err := ParseWatchCacheSizes(f.Options.WatchCacheSizes)
		if err != nil {
			return generic.RESTOptions{}, err
		}
		size, ok := sizes[resource]
		if ok && size > 0 {
			klog.Warningf("Dropping watch-cache-size for %v - watchCache size is now dynamic", resource)
		}
		if ok && size <= 0 {
			klog.V(3).InfoS("Not using watch cache", "resource", resource)
			ret.Decorator = generic.UndecoratedStorage
		} else {
			klog.V(3).InfoS("Using watch cache", "resource", resource)
			ret.Decorator = genericregistry.StorageWithCacher()
		}
	}

	return ret, nil
}

// ParseWatchCacheSizes turns a list of cache size values into a map of group resources
// to requested sizes.
func ParseWatchCacheSizes(cacheSizes []string) (map[schema.GroupResource]int, error) {
	watchCacheSizes := make(map[schema.GroupResource]int)
	for _, c := range cacheSizes {
		tokens := strings.Split(c, "#")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid value of watch cache size: %s", c)
		}

		size, err := strconv.Atoi(tokens[1])
		if err != nil {
			return nil, fmt.Errorf("invalid size of watch cache size: %s", c)
		}
		if size < 0 {
			return nil, fmt.Errorf("watch cache size cannot be negative: %s", c)
		}
		watchCacheSizes[schema.ParseGroupResource(tokens[0])] = size
	}
	return watchCacheSizes, nil
}

// WriteWatchCacheSizes turns a map of cache size values into a list of string specifications.
func WriteWatchCacheSizes(watchCacheSizes map[schema.GroupResource]int) ([]string, error) {
	var cacheSizes []string

	for resource, size := range watchCacheSizes {
		if size < 0 {
			return nil, fmt.Errorf("watch cache size cannot be negative for resource %s", resource)
		}
		cacheSizes = append(cacheSizes, fmt.Sprintf("%s#%d", resource.String(), size))
	}
	return cacheSizes, nil
}

type transformerStorageFactory struct {
	delegate             serverstorage.StorageFactory
	resourceTransformers storagevalue.ResourceTransformers
}

func (t *transformerStorageFactory) NewConfig(resource schema.GroupResource) (*storagebackend.ConfigForResource, error) {
	config, err := t.delegate.NewConfig(resource)
	if err != nil {
		return nil, err
	}

	configCopy := *config
	resourceConfig := configCopy.Config
	resourceConfig.Transformer = t.resourceTransformers.TransformerForResource(resource)
	configCopy.Config = resourceConfig

	return &configCopy, nil
}

func (t *transformerStorageFactory) ResourcePrefix(resource schema.GroupResource) string {
	return t.delegate.ResourcePrefix(resource)
}

func (t *transformerStorageFactory) Configs() []storagebackend.Config {
	return t.delegate.Configs()
}

func (t *transformerStorageFactory) Backends() []serverstorage.Backend {
	return t.delegate.Backends()
}

// SpecialDefaultResourcePrefixes are prefixes compiled into Kubernetes.
var SpecialDefaultResourcePrefixes = map[schema.GroupResource]string{
	//{Group: "", Resource: "replicationcontrollers"}:     "controllers",
	//{Group: "", Resource: "endpoints"}:                  "services/endpoints",
	//{Group: "", Resource: "nodes"}:                      "minions",
	//{Group: "", Resource: "services"}:                   "services/specs",
	//{Group: "extensions", Resource: "ingresses"}:        "ingress",
	//{Group: "networking.k8s.io", Resource: "ingresses"}: "ingress",
}

// DefaultWatchCacheSizes defines default resources for which watchcache
// should be disabled.
func DefaultWatchCacheSizes() map[schema.GroupResource]int {
	return map[schema.GroupResource]int{
		//{Resource: "events"}:                         0,
		//{Group: "events.k8s.io", Resource: "events"}: 0,
	}
}

// NewStorageFactoryConfig returns a new StorageFactoryConfig set up with necessary resource overrides.
func NewStorageFactoryConfig() *StorageFactoryConfig {
	resources := []schema.GroupVersionResource{
		// If a resource has to be stored in a version that is not the
		// latest, then it can be listed here. Usually this is the case
		// when a new version for a resource gets introduced and a
		// downgrade to an older apiserver that doesn't know the new
		// version still needs to be supported for one release.
		//
		corev1.Resource("runners").WithVersion("v1"),
		corev1.Resource("runnerStat").WithVersion("v1"),
		corev1.Resource("regions").WithVersion("v1"),
	}

	return &StorageFactoryConfig{
		Serializer:                apis.Codecs,
		DefaultResourceEncoding:   serverstorage.NewDefaultResourceEncodingConfig(apis.Scheme),
		ResourceEncodingOverrides: resources,
	}
}

// StorageFactoryConfig is a configuration for creating storage factory.
type StorageFactoryConfig struct {
	StorageConfig             storagebackend.Config
	APIResourceConfig         *serverstorage.ResourceConfig
	DefaultResourceEncoding   *serverstorage.DefaultResourceEncodingConfig
	DefaultStorageMediaType   string
	Serializer                krt.StorageSerializer
	ResourceEncodingOverrides []schema.GroupVersionResource
	EtcdServersOverrides      []string
}

// Complete completes the StorageFactoryConfig with provided etcdOptions returning completedStorageFactoryConfig.
// This method mutates the receiver (StorageFactoryConfig).  It must never mutate the inputs.
func (c *StorageFactoryConfig) Complete(etcdOptions *EtcdStorageOptions) *completedStorageFactoryConfig {
	c.StorageConfig = etcdOptions.StorageConfig
	c.DefaultStorageMediaType = etcdOptions.DefaultStorageMediaType
	c.EtcdServersOverrides = etcdOptions.EtcdServersOverrides
	return &completedStorageFactoryConfig{c}
}

// completedStorageFactoryConfig is a wrapper around StorageFactoryConfig completed with etcd options.
//
// Note: this struct is intentionally unexported so that it can only be constructed via a StorageFactoryConfig.Complete
// call. The implied consequence is that this does not comply with golint.
type completedStorageFactoryConfig struct {
	*StorageFactoryConfig
}

// New returns a new storage factory created from the completed storage factory configuration.
func (c *completedStorageFactoryConfig) New() (*serverstorage.DefaultStorageFactory, error) {
	resourceEncoding := c.DefaultResourceEncoding
	resourceEncodingConfig := MergeResourceEncodingConfigs(resourceEncoding, c.ResourceEncodingOverrides)

	storageFactory := serverstorage.NewDefaultStorageFactory(
		c.StorageConfig,
		c.DefaultStorageMediaType,
		c.Serializer,
		resourceEncodingConfig,
		c.APIResourceConfig,
		SpecialDefaultResourcePrefixes)

	for _, override := range c.EtcdServersOverrides {
		tokens := strings.Split(override, "#")
		apiresource := strings.Split(tokens[0], "/")

		group := apiresource[0]
		resource := apiresource[1]
		groupResource := schema.GroupResource{Group: group, Resource: resource}

		servers := strings.Split(tokens[1], ";")
		storageFactory.SetEtcdLocation(groupResource, servers)
	}
	return storageFactory, nil
}

// MergeResourceEncodingConfigs merges the given defaultResourceConfig with specific GroupVersionResource overrides.
func MergeResourceEncodingConfigs(
	defaultResourceEncoding *serverstorage.DefaultResourceEncodingConfig,
	resourceEncodingOverrides []schema.GroupVersionResource,
) *serverstorage.DefaultResourceEncodingConfig {
	resourceEncodingConfig := defaultResourceEncoding
	for _, gvr := range resourceEncodingOverrides {
		resourceEncodingConfig.SetResourceEncoding(gvr.GroupResource(), gvr.GroupVersion(),
			schema.GroupVersion{Group: gvr.Group, Version: krt.APIVersionInternal})
	}
	return resourceEncodingConfig
}
