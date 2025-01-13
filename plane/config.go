/*
Copyright 2023 The olive Authors

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

package plane

import (
	"context"
	"time"

	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	"go.uber.org/zap"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/client-go/dynamic"

	"github.com/olive-io/olive/apis"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
	"github.com/olive-io/olive/plane/embed"
	planeleader "github.com/olive-io/olive/plane/leader"
	apidiscoveryrest "github.com/olive-io/olive/plane/registry/apidiscovery/rest"
	corerest "github.com/olive-io/olive/plane/registry/core/rest"
	planescheduler "github.com/olive-io/olive/plane/scheduler"
)

const (
	DefaultRegionLimit            = 100
	DefaultRegionDefinitionsLimit = 500
	OliveComponentName            = "plane"
)

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   *ExtraConfig
}

func NewConfig() *Config {
	genericConfig := genericapiserver.NewRecommendedConfig(apis.Codecs)

	cfg := Config{
		GenericConfig: genericConfig,
		ExtraConfig:   NewExtraConfig(),
	}

	return &cfg
}

func (cfg *Config) Validate() (err error) {
	return
}

type ExtraConfig struct {
	Logger *zap.Logger
	Etcd   *embed.Etcd

	APIResourceConfigSource serverstorage.APIResourceConfigSource

	ClientSet             clientset.Interface
	SharedInformerFactory informers.SharedInformerFactory
	DynamicClient         dynamic.Interface
}

func NewExtraConfig() *ExtraConfig {
	return &ExtraConfig{
		APIResourceConfigSource: DefaultAPIResourceConfigSource(),
	}
}

type completedConfig struct {
	EtcdConfig    *embed.Config
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		GenericConfig: cfg.GenericConfig.Complete(),
		ExtraConfig:   cfg.ExtraConfig,
	}

	return CompletedConfig{&c}
}

// New returns a new instance of Server from the given config.
func (c completedConfig) New() (*PlaneServer, error) {
	lg := c.ExtraConfig.Logger
	etcd := c.ExtraConfig.Etcd
	embedDaemon := genericdaemon.NewEmbedDaemon(lg)

	genericServer, err := c.GenericConfig.New(OliveComponentName, genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	informersFactory := c.ExtraConfig.SharedInformerFactory
	genericServer.AddPostStartHookOrDie("start-olive-plane-informers", func(ctx genericapiserver.PostStartHookContext) error {
		informersFactory.Start(ctx.Done())
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	etcdClient := v3client.New(etcd.Server)

	leaderNotifier := planeleader.NewNotify(etcd.Server)
	clientSet := c.ExtraConfig.ClientSet

	planeServer := &PlaneServer{
		IDaemon: embedDaemon,

		ctx:    ctx,
		cancel: cancel,

		lg:       lg,
		etcd:     etcd,
		v3cli:    etcdClient,
		idGen:    idutil.NewGenerator(uint16(etcd.Server.ID()), time.Now()),
		notifier: leaderNotifier,

		genericAPIServer: genericServer,
	}

	apiDiscoveryRestStorageProvider, err := apidiscoveryrest.NewRESTStorageProvider()
	if err != nil {
		return nil, err
	}
	coreRestStorageProvider, err := corerest.NewRESTStorageProvider(etcdClient, planeServer.StoppingNotify())
	if err != nil {
		return nil, err
	}

	restStorageProviders := []RESTStorageProvider{
		coreRestStorageProvider,
		apiDiscoveryRestStorageProvider,
	}

	if err = planeServer.InstallAPIs(
		c.ExtraConfig.APIResourceConfigSource,
		c.GenericConfig.RESTOptionsGetter,
		restStorageProviders...); err != nil {
		return nil, err
	}

	schedulerOptions := []planescheduler.Option{}
	planeServer.scheduler, err = planescheduler.NewScheduler(ctx, leaderNotifier, clientSet, informersFactory, schedulerOptions...)
	if err != nil {
		return nil, err
	}

	return planeServer, nil
}
