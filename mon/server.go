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
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/klog/v2"

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/mon/embed"
	"github.com/olive-io/olive/mon/leader"
	"github.com/olive-io/olive/mon/schedule"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
)

var (
	// stableAPIGroupVersionsEnabledByDefault is a list of our stable versions.
	stableAPIGroupVersionsEnabledByDefault = []schema.GroupVersion{
		apidiscoveryv1.SchemeGroupVersion,
		corev1.SchemeGroupVersion,
		monv1.SchemeGroupVersion,
	}
)

// RESTStorageProvider is a factory type for REST storage.
type RESTStorageProvider interface {
	GroupName() string
	NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, error)
}

type MonitorServer struct {
	genericdaemon.IDaemon

	etcdConfig    *embed.Config
	genericConfig *genericapiserver.RecommendedConfig

	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	etcd  *embed.Etcd
	v3cli *clientv3.Client
	idGen *idutil.Generator

	notifier leader.Notifier

	genericAPIServer *genericapiserver.GenericAPIServer

	scheduler *schedule.Scheduler
}

// DefaultAPIResourceConfigSource returns default configuration for an APIResource.
func DefaultAPIResourceConfigSource() *serverstorage.ResourceConfig {
	ret := serverstorage.NewResourceConfig()
	ret.EnableVersions(stableAPIGroupVersionsEnabledByDefault...)

	return ret
}

// InstallAPIs will install the APIs for the restStorageProviders if they are enabled.
func (s *MonitorServer) InstallAPIs(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter, restStorageProviders ...RESTStorageProvider) error {
	nonLegacy := []*genericapiserver.APIGroupInfo{}

	// used later in the loop to filter the served resource by those that have expired.
	resourceExpirationEvaluator, err := genericapiserver.NewResourceExpirationEvaluator(*s.genericAPIServer.Version)
	if err != nil {
		return err
	}

	for _, restStorageBuilder := range restStorageProviders {
		groupName := restStorageBuilder.GroupName()
		apiGroupInfo, err := restStorageBuilder.NewRESTStorage(apiResourceConfigSource, restOptionsGetter)
		if err != nil {
			return fmt.Errorf("problem initializing API group %q : %v", groupName, err)
		}
		if len(apiGroupInfo.VersionedResourcesStorageMap) == 0 {
			// If we have no storage for any resource configured, this API group is effectively disabled.
			// This can happen when an entire API group, version, or development-stage (alpha, beta, GA) is disabled.
			klog.Infof("API group %q is not enabled, skipping.", groupName)
			continue
		}

		// Remove resources that serving kinds that are removed.
		// We do this here so that we don't accidentally serve versions without resources or openapi information that for kinds we don't serve.
		// This is a spot above the construction of individual storage handlers so that no sig accidentally forgets to check.
		resourceExpirationEvaluator.RemoveDeletedKinds(groupName, apiGroupInfo.Scheme, apiGroupInfo.VersionedResourcesStorageMap)
		if len(apiGroupInfo.VersionedResourcesStorageMap) == 0 {
			klog.V(1).Infof("Removing API group %v because it is time to stop serving it because it has no versions per APILifecycle.", groupName)
			continue
		}

		klog.V(1).Infof("Enabling API group %q.", groupName)

		if postHookProvider, ok := restStorageBuilder.(genericapiserver.PostStartHookProvider); ok {
			name, hook, err := postHookProvider.PostStartHook()
			if err != nil {
				klog.Fatalf("Error building PostStartHook: %v", err)
			}
			s.genericAPIServer.AddPostStartHookOrDie(name, hook)
		}

		if len(groupName) == 0 {
			// the legacy group for core APIs is special that it is installed into /api via this special install method.
			if err := s.genericAPIServer.InstallLegacyAPIGroup(genericapiserver.DefaultLegacyAPIPrefix, &apiGroupInfo); err != nil {
				return fmt.Errorf("error in registering legacy API: %w", err)
			}
		} else {
			// everything else goes to /apis
			nonLegacy = append(nonLegacy, &apiGroupInfo)
		}
	}

	if err := s.genericAPIServer.InstallAPIGroups(nonLegacy...); err != nil {
		return fmt.Errorf("error in registering group versions: %v", err)
	}
	return nil
}

func (s *MonitorServer) Start(stopc <-chan struct{}) error {
	//ec := s.etcdConfig
	//ec.EnableGRPCGateway = true
	//
	//ec.UserHandlers = map[string]http.Handler{
	//	"/metrics": promhttp.Handler(),
	//}
	//
	//clusterRPC, err := newClusterServer(s)
	//if err != nil {
	//	return err
	//}
	//runnerRPC, err := newRunnerServer(s)
	//if err != nil {
	//	return err
	//}
	//authRPC, err := newAuthServer(s)
	//if err != nil {
	//	return err
	//}
	//bpmnRPC, err := newBpmnServer(s)
	//if err != nil {
	//	return err
	//}
	//ec.ServiceRegister = func(gs *grpc.Server) {
	//	pb.RegisterMetaClusterRPCServer(gs, clusterRPC)
	//	pb.RegisterMetaRunnerRPCServer(gs, runnerRPC)
	//	pb.RegisterMetaRegionRPCServer(gs, runnerRPC)
	//	pb.RegisterAuthRPCServer(gs, authRPC)
	//	pb.RegisterRbacRPCServer(gs, authRPC)
	//	pb.RegisterBpmnRPCServer(gs, bpmnRPC)
	//}
	//
	//ec.UnaryInterceptor = s.unaryInterceptor()
	//s.etcd, err = embed.StartEtcd(ec)
	//if err != nil {
	//	return errors.Wrap(err, "start embed etcd")
	//}
	//
	//select {
	//case <-s.etcd.Server.ReadyNotify():
	//case <-stopc:
	//	return errors.New("etcd server not stops")
	//}

	//s.genericAPIServer.AddPostStartHookOrDie("start-olive-mon-informers", func(ctx genericapiserver.PostStartHookContext) error {
	//	config.GenericConfig.SharedInformerFactory.Start(ctx.StopCh)
	//	o.SharedInformerFactory.Start(ctx.StopCh)
	//	return nil
	//})
	//
	//return server.GenericAPIServer.PrepareRun().Run(stopCh)
	shutdownTimeout := time.Second * 10
	preparedServer := s.genericAPIServer.PrepareRun()
	stoppedCh, listenerStoppedCh, err := preparedServer.NonBlockingRun(stopc, shutdownTimeout)
	if err != nil {
		return err
	}

	scfg := &schedule.Config{
		Logger:          s.lg,
		Client:          s.v3cli,
		Notifier:        s.notifier,
		RegionLimit:     DefaultRegionLimit,
		DefinitionLimit: DefaultRegionDefinitionsLimit,
	}
	s.scheduler = schedule.New(scfg, s.StoppingNotify())
	if err = s.scheduler.Start(); err != nil {
		return err
	}

	//if err = authRPC.Prepare(s.ctx); err != nil {
	//	return err
	//}

	s.IDaemon.OnDestroy(s.destroy)

	<-stoppedCh
	<-listenerStoppedCh

	return s.stop()
}

func (s *MonitorServer) stop() error {
	s.IDaemon.Shutdown()
	return nil
}

func (s *MonitorServer) destroy() {
	s.etcd.Server.HardStop()
	<-s.etcd.Server.StopNotify()
	s.cancel()
}
