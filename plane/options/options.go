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
	"io"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	genericopenapi "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/featuregate"
	netutils "k8s.io/utils/net"

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/client-go/generated/openapi"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/apis/version"
	planeserver "github.com/olive-io/olive/plane"
	"github.com/olive-io/olive/plane/embed"
)

const (
	DefaultEtcdPathPrefix = "/registry/olive"
	PlaneComponentName    = "plane"
)

// ServerOptions contains state for master/api server
type ServerOptions struct {
	EmbedEtcdOptions   *EmbedEtcdOptions
	RecommendedOptions *RecommendedOptions

	StdOut io.Writer
	StdErr io.Writer

	AlternateDNS []string
}

// NewServerOptions returns a new ServerOptions
func NewServerOptions(out, errOut io.Writer) *ServerOptions {

	embedEtcdOptions := NewEmbedEtcdOptions()
	recommendedOptions := NewRecommendedOptions(DefaultEtcdPathPrefix, Codec)

	o := &ServerOptions{
		EmbedEtcdOptions:   embedEtcdOptions,
		RecommendedOptions: recommendedOptions,

		StdOut: out,
		StdErr: errOut,
	}
	o.RecommendedOptions.EtcdStorage.StorageConfig.EncodeVersioner = krt.NewMultiGroupVersioner(
		metav1.SchemeGroupVersion,
		schema.GroupKind{Group: corev1.GroupName},
		schema.GroupKind{Group: apidiscoveryv1.GroupName},
	)
	o.RecommendedOptions.Admission = nil
	o.RecommendedOptions.Authentication = nil
	o.RecommendedOptions.Authorization.RemoteKubeConfigFileOptional = true
	o.RecommendedOptions.Authorization = nil
	return o
}

// Validate validates ServerOptions
func (o *ServerOptions) Validate(args []string) error {
	errors := []error{}
	errors = append(errors, o.EmbedEtcdOptions.Validate()...)
	errors = append(errors, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

// Complete fills in fields required to have valid data
func (o *ServerOptions) Complete() error {
	// register admission plugins
	//banflunder.Register(o.RecommendedOptions.Admission.Plugins)
	//
	//// add admission plugins to the RecommendedPluginOrder
	//o.RecommendedOptions.Admission.RecommendedPluginOrder = append(o.RecommendedOptions.Admission.RecommendedPluginOrder, "BanFlunder")

	return nil
}

// Config returns config for the api server given ServerOptions
func (o *ServerOptions) Config() (*planeserver.Config, error) {
	etcdConfig := &embed.Config{}
	if err := o.EmbedEtcdOptions.ApplyTo(etcdConfig); err != nil {
		return nil, err
	}

	lg := etcdConfig.GetLogger()
	etcd, err := embed.StartEtcd(etcdConfig)
	if err != nil {
		return nil, err
	}
	<-etcd.Server.ReadyNotify()

	extraConfig := planeserver.NewExtraConfig()
	extraConfig.Logger = lg
	extraConfig.Etcd = etcd

	if err = o.EmbedEtcdOptions.ApplyToEtcdStorage(o.RecommendedOptions.EtcdStorage); err != nil {
		return nil, err
	}

	if err = o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", o.AlternateDNS, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	o.RecommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig, extraConfig *planeserver.ExtraConfig) ([]admission.PluginInitializer, error) {
		//client, err := clientset.NewForConfig(c.LoopbackClientConfig)
		//if err != nil {
		//	return nil, err
		//}
		//dynamicClient, err := dynamic.NewForConfig(c.LoopbackClientConfig)
		//if err != nil {
		//	return nil, err
		//}
		//informerFactory := informers.NewSharedInformerFactory(client, c.LoopbackClientConfig.Timeout)
		//o.SharedInformerFactory = informerFactory

		//
		//initializer.New(client, dynamicClient, informerFactory,  utilfeature.DefaultFeatureGate, )
		return []admission.PluginInitializer{}, nil
	}

	genericConfig := genericapiserver.NewRecommendedConfig(apis.Codecs)

	namer := genericopenapi.NewDefinitionNamer(apis.Scheme)
	genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(openapi.GetOpenAPIDefinitions, namer)
	genericConfig.OpenAPIConfig.Info.Title = "Olive"
	genericConfig.OpenAPIConfig.Info.Version = version.APIVersion
	genericConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(openapi.GetOpenAPIDefinitions, namer)
	genericConfig.OpenAPIV3Config.Info.Title = "Olive"
	genericConfig.OpenAPIV3Config.Info.Version = version.APIVersion

	genericConfig.CorsAllowedOriginList = []string{".*"}
	genericConfig.FeatureGate = featuregate.DefaultComponentGlobalsRegistry.FeatureGateFor(featuregate.DefaultKubeComponent)
	genericConfig.EffectiveVersion = featuregate.DefaultComponentGlobalsRegistry.EffectiveVersionFor(PlaneComponentName)

	if err = o.RecommendedOptions.ApplyTo(genericConfig, extraConfig); err != nil {
		return nil, err
	}

	config := &planeserver.Config{
		GenericConfig: genericConfig,
		ExtraConfig:   extraConfig,
	}
	return config, nil
}

// StartPlaneServer start a Plane Server by given ServerOptions
func (o *ServerOptions) StartPlaneServer(ctx context.Context) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	return server.Start(ctx.Done())
}
