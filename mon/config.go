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

package mon

import (
	"context"
	"time"

	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/client-go/dynamic"

	"github.com/olive-io/olive/apis"
	clientset "github.com/olive-io/olive/client/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client/generated/informers/externalversions"
	"github.com/olive-io/olive/mon/embed"
	"github.com/olive-io/olive/mon/leader"
	apidiscoveryrest "github.com/olive-io/olive/mon/registry/apidiscovery/rest"
	corerest "github.com/olive-io/olive/mon/registry/core/rest"
	monrest "github.com/olive-io/olive/mon/registry/mon/rest"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
)

const (
	DefaultRegionLimit            = 100
	DefaultRegionDefinitionsLimit = 500
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

	KubeClient            clientset.Interface
	SharedInformerFactory informers.SharedInformerFactory
	DynamicClient         dynamic.Interface

	// The maximum number of regions in a runner
	RegionLimit int
	// The maximum number of bpmn definitions in a region
	RegionDefinitionsLimit int
}

func NewExtraConfig() *ExtraConfig {
	return &ExtraConfig{
		APIResourceConfigSource: DefaultAPIResourceConfigSource(),
		RegionLimit:             DefaultRegionLimit,
		RegionDefinitionsLimit:  DefaultRegionDefinitionsLimit,
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

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of Server from the given config.
func (c completedConfig) New() (*MonitorServer, error) {
	lg := c.ExtraConfig.Logger
	etcd := c.ExtraConfig.Etcd
	embedDaemon := genericdaemon.NewEmbedDaemon(lg)

	genericServer, err := c.GenericConfig.New("olive-mon", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	informersFactory := c.ExtraConfig.SharedInformerFactory
	genericServer.AddPostStartHookOrDie("start-olive-mon-informers", func(ctx genericapiserver.PostStartHookContext) error {
		informersFactory.Start(ctx.StopCh)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	etcdClient := v3client.New(etcd.Server)

	monServer := &MonitorServer{
		IDaemon: embedDaemon,

		ctx:    ctx,
		cancel: cancel,

		lg:       lg,
		etcd:     etcd,
		v3cli:    etcdClient,
		idGen:    idutil.NewGenerator(uint16(etcd.Server.ID()), time.Now()),
		notifier: leader.NewNotify(etcd.Server),

		genericAPIServer: genericServer,
	}

	monRestStorageProvider, err := monrest.NewRESTStorageProvider(etcdClient, monServer.StoppingNotify())
	if err != nil {
		return nil, err
	}
	apiDiscoveryRestStorageProvider, err := apidiscoveryrest.NewRESTStorageProvider()
	if err != nil {
		return nil, err
	}
	coreRestStorageProvider, err := corerest.NewRESTStorageProvider()
	if err != nil {
		return nil, err
	}

	restStorageProviders := []RESTStorageProvider{
		monRestStorageProvider,
		apiDiscoveryRestStorageProvider,
		coreRestStorageProvider,
	}

	if err = monServer.InstallAPIs(
		c.ExtraConfig.APIResourceConfigSource,
		c.GenericConfig.RESTOptionsGetter,
		restStorageProviders...); err != nil {
		return nil, err
	}

	return monServer, nil
}
