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
	"time"

	"github.com/spf13/pflag"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	genericserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/util/feature"
	dynamicclient "k8s.io/client-go/dynamic"
	"k8s.io/component-base/featuregate"

	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"

	monserver "github.com/olive-io/olive/mon"
)

// RecommendedOptions contains the recommended options for running an API server.
// If you add something to this list, it should be in a logical grouping.
// Each of them can be nil to leave the feature unconfigured on ApplyTo.
type RecommendedOptions struct {
	SecureServing  *genericoptions.SecureServingOptionsWithLoopback
	EtcdStorage    *EtcdStorageOptions
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions
	Audit          *genericoptions.AuditOptions
	Features       *FeatureOptions

	// FeatureGate is a way to plumb feature gate through if you have them.
	FeatureGate featuregate.FeatureGate
	// ExtraAdmissionInitializers is called once after all ApplyTo from the options above, to pass the returned
	// admission plugin initializers to Admission.ApplyTo.
	ExtraAdmissionInitializers func(c *genericserver.RecommendedConfig, extraConfig *monserver.ExtraConfig) ([]admission.PluginInitializer, error)
	Admission                  *AdmissionOptions
	// API Server Egress Selector is used to control outbound traffic from the API Server
	EgressSelector *genericoptions.EgressSelectorOptions
	// Traces contains options to control distributed request tracing.
	Traces *genericoptions.TracingOptions
}

func NewRecommendedOptions(prefix string, codec krt.Codec) *RecommendedOptions {
	secureServingOptions := genericoptions.NewSecureServingOptions()
	// We are composing recommended options for an aggregated api-server,
	// whose client is typically a proxy multiplexing many operations ---
	// notably including long-running ones --- into one HTTP/2 connection
	// into this server.  So allow many concurrent operations.
	secureServingOptions.HTTP2MaxStreamsPerConnection = 1000

	return &RecommendedOptions{
		SecureServing:  secureServingOptions.WithLoopback(),
		EtcdStorage:    NewStorageEtcdOptions(storagebackend.NewDefaultConfig(prefix, codec)),
		Authentication: genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericoptions.NewDelegatingAuthorizationOptions(),
		Audit:          genericoptions.NewAuditOptions(),
		Features:       NewFeatureOptions(),
		FeatureGate:    feature.DefaultFeatureGate,
		ExtraAdmissionInitializers: func(c *genericserver.RecommendedConfig, extraConfig *monserver.ExtraConfig) ([]admission.PluginInitializer, error) {
			return nil, nil
		},
		Admission:      NewAdmissionOptions(),
		EgressSelector: genericoptions.NewEgressSelectorOptions(),
		Traces:         genericoptions.NewTracingOptions(),
	}
}

func (o *RecommendedOptions) AddFlags(fs *pflag.FlagSet) {
	o.SecureServing.AddFlags(fs)
	o.EtcdStorage.AddFlags(fs)
	o.Authentication.AddFlags(fs)
	o.Authorization.AddFlags(fs)
	o.Audit.AddFlags(fs)
	o.Features.AddFlags(fs)
	o.Admission.AddFlags(fs)
	o.EgressSelector.AddFlags(fs)
	o.Traces.AddFlags(fs)
}

// ApplyTo adds RecommendedOptions to the server configuration.
// pluginInitializers can be empty, it is only need for additional initializers.
func (o *RecommendedOptions) ApplyTo(config *genericserver.RecommendedConfig, extraConfig *monserver.ExtraConfig) error {
	if err := o.SecureServing.ApplyTo(&config.Config.SecureServing, &config.Config.LoopbackClientConfig); err != nil {
		return err
	}
	if err := o.EtcdStorage.ApplyTo(&config.Config); err != nil {
		return err
	}
	if err := o.EgressSelector.ApplyTo(&config.Config); err != nil {
		return err
	}
	if err := o.Traces.ApplyTo(config.Config.EgressSelector, &config.Config); err != nil {
		return err
	}
	if err := o.Authentication.ApplyTo(&config.Config.Authentication, config.SecureServing, config.OpenAPIConfig); err != nil {
		return err
	}
	if err := o.Authorization.ApplyTo(&config.Config.Authorization); err != nil {
		return err
	}
	if err := o.Audit.ApplyTo(&config.Config); err != nil {
		return err
	}

	config.LoopbackClientConfig.ContentConfig.ContentType = krt.ContentTypeProtobuf
	config.LoopbackClientConfig.DisableCompression = false
	config.LoopbackClientConfig.Timeout = time.Minute * 10

	clientSet, err := clientset.NewForConfig(config.LoopbackClientConfig)
	if err != nil {
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Second*15)
	dynamicClient, err := dynamicclient.NewForConfig(config.LoopbackClientConfig)
	if err != nil {
		return err
	}

	extraConfig.ClientSet = clientSet
	extraConfig.SharedInformerFactory = informerFactory
	extraConfig.DynamicClient = dynamicClient

	if err := o.Features.ApplyTo(&config.Config, clientSet, informerFactory); err != nil {
		return err
	}
	initializers, err := o.ExtraAdmissionInitializers(config, extraConfig)
	if err != nil {
		return err
	}
	if err := o.Admission.ApplyTo(&config.Config, informerFactory, clientSet, dynamicClient, o.FeatureGate,
		initializers...); err != nil {
		return err
	}
	return nil
}

func (o *RecommendedOptions) Validate() []error {
	errors := []error{}
	errors = append(errors, o.SecureServing.Validate()...)
	errors = append(errors, o.EtcdStorage.Validate()...)
	errors = append(errors, o.Authentication.Validate()...)
	errors = append(errors, o.Authorization.Validate()...)
	errors = append(errors, o.Audit.Validate()...)
	errors = append(errors, o.Features.Validate()...)
	errors = append(errors, o.Admission.Validate()...)
	errors = append(errors, o.EgressSelector.Validate()...)
	errors = append(errors, o.Traces.Validate()...)

	return errors
}
