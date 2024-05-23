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
	"fmt"
	"io"
	"net"

	"github.com/spf13/cobra"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	genericopenapi "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	netutils "k8s.io/utils/net"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/apis/version"
	informers "github.com/olive-io/olive/client/generated/informers/externalversions"
	"github.com/olive-io/olive/client/generated/openapi"
	"github.com/olive-io/olive/mon"
)

const defaultEtcdPathPrefix = "/registry/olive"

// ServerOptions contains state for master/api server
type ServerOptions struct {
	RecommendedOptions *RecommendedOptions

	SharedInformerFactory informers.SharedInformerFactory
	StdOut                io.Writer
	StdErr                io.Writer

	AlternateDNS []string
}

// NewServerOptions returns a new ServerOptions
func NewServerOptions(out, errOut io.Writer) *ServerOptions {
	o := &ServerOptions{
		RecommendedOptions: NewRecommendedOptions(
			defaultEtcdPathPrefix,
			apis.Codec,
		),

		StdOut: out,
		StdErr: errOut,
	}
	//o.RecommendedOptions.Etcd.StorageConfig.EncodeVersioner = runtime.NewMultiGroupVersioner(v1alpha1.SchemeGroupVersion, schema.GroupKind{Group: v1alpha1.GroupName})
	//o.RecommendedOptions.Admission = nil
	//o.RecommendedOptions.CoreAPI = nil
	//o.RecommendedOptions.Authentication = nil
	////o.RecommendedOptions.Authorization.RemoteKubeConfigFileOptional = true
	//o.RecommendedOptions.Authorization = nil
	return o
}

// NewStartServer provides a CLI handler for 'start master' command
// with a default ServerOptions.
func NewStartServer(defaults *ServerOptions, stopCh <-chan struct{}) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: "Launch a wardle API server",
		Long:  "Launch a wardle API server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunServer(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.RecommendedOptions.AddFlags(flags)
	utilfeature.DefaultMutableFeatureGate.AddFlag(flags)

	return cmd
}

// Validate validates ServerOptions
func (o ServerOptions) Validate(args []string) error {
	errors := []error{}
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
func (o *ServerOptions) Config() (*mon.Config, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", o.AlternateDNS, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	//o.RecommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error) {
	//	client, err := clientset.NewForConfig(c.LoopbackClientConfig)
	//	if err != nil {
	//		return nil, err
	//	}
	//	informerFactory := informers.NewSharedInformerFactory(client, c.LoopbackClientConfig.Timeout)
	//	o.SharedInformerFactory = informerFactory
	//	return []admission.PluginInitializer{wardleinitializer.New(informerFactory)}, nil
	//}

	serverConfig := genericapiserver.NewRecommendedConfig(apis.Codecs)

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(openapi.GetOpenAPIDefinitions, genericopenapi.NewDefinitionNamer(apis.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "Olive"
	serverConfig.OpenAPIConfig.Info.Version = version.APIVersion

	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(openapi.GetOpenAPIDefinitions, genericopenapi.NewDefinitionNamer(apis.Scheme))
	serverConfig.OpenAPIV3Config.Info.Title = "Olive"
	serverConfig.OpenAPIV3Config.Info.Version = version.APIVersion

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	config := &mon.Config{
		GenericConfig: serverConfig,
		//ExtraConfig:   apiserver.ExtraConfig{},
	}
	return config, nil
}

// RunServer starts a new Server given ServerOptions
func (o ServerOptions) RunServer(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	config.GenericConfig.CorsAllowedOriginList = []string{".*"}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}

	//server.GenericAPIServer.AddPostStartHookOrDie("start-sample-server-informers", func(context genericapiserver.PostStartHookContext) error {
	//	config.GenericConfig.SharedInformerFactory.Start(context.StopCh)
	//	o.SharedInformerFactory.Start(context.StopCh)
	//	return nil
	//})

	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}
