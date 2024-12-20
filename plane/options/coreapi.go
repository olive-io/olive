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
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/util/feature"
	clientgoinformers "k8s.io/client-go/informers"
	clientgoclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/tracing"
)

// CoreAPIOptions contains options to configure the connection to a core API Kubernetes apiserver.
type CoreAPIOptions struct {
	// CoreAPIKubeconfigPath is a filename for a kubeconfig file to contact the core API server with.
	// If it is not set, the in cluster config is used.
	CoreAPIKubeconfigPath string
}

func NewCoreAPIOptions() *CoreAPIOptions {
	return &CoreAPIOptions{}
}

func (o *CoreAPIOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.CoreAPIKubeconfigPath, "kubeconfig", o.CoreAPIKubeconfigPath,
		"kubeconfig file pointing at the 'core' kubernetes server.")
}

func (o *CoreAPIOptions) ApplyTo(config *server.RecommendedConfig) error {
	if o == nil {
		return nil
	}

	// create shared informer for Kubernetes APIs
	var kubeconfig *rest.Config
	var err error
	if len(o.CoreAPIKubeconfigPath) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: o.CoreAPIKubeconfigPath}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		kubeconfig, err = loader.ClientConfig()
		if err != nil {
			return fmt.Errorf("failed to load kubeconfig at %q: %v", o.CoreAPIKubeconfigPath, err)
		}
	} else {
		kubeconfig, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
	}
	if feature.DefaultFeatureGate.Enabled(features.APIServerTracing) {
		kubeconfig.Wrap(tracing.WrapperFor(config.TracerProvider))
	}
	clientgoExternalClient, err := clientgoclientset.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}
	config.ClientConfig = kubeconfig
	config.SharedInformerFactory = clientgoinformers.NewSharedInformerFactory(clientgoExternalClient, 10*time.Minute)

	return nil
}

func (o *CoreAPIOptions) Validate() []error {
	return nil
}
