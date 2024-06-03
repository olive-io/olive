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

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/server"

	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
)

type FeatureOptions struct {
	EnableProfiling           bool
	DebugSocketPath           string
	EnableContentionProfiling bool
	EnablePriorityAndFairness bool
}

func NewFeatureOptions() *FeatureOptions {
	defaults := server.NewConfig(serializer.CodecFactory{})

	return &FeatureOptions{
		EnableProfiling:           defaults.EnableProfiling,
		DebugSocketPath:           defaults.DebugSocketPath,
		EnableContentionProfiling: defaults.EnableContentionProfiling,
		EnablePriorityAndFairness: true,
	}
}

func (o *FeatureOptions) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.BoolVar(&o.EnableProfiling, "profiling", o.EnableProfiling,
		"Enable profiling via web interface host:port/debug/pprof/")
	fs.BoolVar(&o.EnableContentionProfiling, "contention-profiling", o.EnableContentionProfiling,
		"Enable block profiling, if profiling is enabled")
	fs.StringVar(&o.DebugSocketPath, "debug-socket-path", o.DebugSocketPath,
		"Use an unprotected (no authn/authz) unix-domain socket for profiling with the given path")
	fs.BoolVar(&o.EnablePriorityAndFairness, "enable-priority-and-fairness", o.EnablePriorityAndFairness, ""+
		"If true, replace the max-in-flight handler with an enhanced one that queues and dispatches with priority and fairness")
}

func (o *FeatureOptions) ApplyTo(c *server.Config, clientset clientset.Interface, informers informers.SharedInformerFactory) error {
	if o == nil {
		return nil
	}

	c.EnableProfiling = o.EnableProfiling
	c.DebugSocketPath = o.DebugSocketPath
	c.EnableContentionProfiling = o.EnableContentionProfiling

	if o.EnablePriorityAndFairness {
		if c.MaxRequestsInFlight+c.MaxMutatingRequestsInFlight <= 0 {
			return fmt.Errorf("invalid configuration: MaxRequestsInFlight=%d and MaxMutatingRequestsInFlight=%d; they must add up to something positive", c.MaxRequestsInFlight, c.MaxMutatingRequestsInFlight)

		}
		//c.FlowControl = utilflowcontrol.New(
		//	informers,
		//	clientset.FlowcontrolV1(),
		//	c.MaxRequestsInFlight+c.MaxMutatingRequestsInFlight,
		//)
	}

	return nil
}

func (o *FeatureOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errs := []error{}
	return errs
}
