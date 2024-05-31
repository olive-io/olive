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

package framework

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	schedulerconfigv1 "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	schedulerapi "github.com/olive-io/olive/mon/scheduler/apis/config"
	"github.com/olive-io/olive/mon/scheduler/apis/config/scheme"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
)

var configDecoder = scheme.Codecs.UniversalDecoder()

// NewFramework creates a Framework from the register functions and options.
func NewFramework(ctx context.Context, fns []RegisterPluginFunc, profileName string, opts ...runtime.Option) (framework.Framework, error) {
	registry := runtime.Registry{}
	profile := &schedulerapi.SchedulerProfile{
		SchedulerName: profileName,
		Plugins:       &schedulerapi.Plugins{},
	}
	for _, f := range fns {
		f(&registry, profile)
	}
	return runtime.NewFramework(ctx, registry, profile, opts...)
}

// RegisterPluginFunc is a function signature used in method RegisterFilterPlugin()
// to register a Filter Plugin to a given registry.
type RegisterPluginFunc func(reg *runtime.Registry, profile *schedulerapi.SchedulerProfile)

// RegisterQueueSortPlugin returns a function to register a QueueSort Plugin to a given registry.
func RegisterQueueSortPlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "QueueSort")
}

// RegisterPreFilterPlugin returns a function to register a PreFilter Plugin to a given registry.
func RegisterPreFilterPlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "PreFilter")
}

// RegisterFilterPlugin returns a function to register a Filter Plugin to a given registry.
func RegisterFilterPlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "Filter")
}

// RegisterReservePlugin returns a function to register a Reserve Plugin to a given registry.
func RegisterReservePlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "Reserve")
}

// RegisterPermitPlugin returns a function to register a Permit Plugin to a given registry.
func RegisterPermitPlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "Permit")
}

// RegisterPreBindPlugin returns a function to register a PreBind Plugin to a given registry.
func RegisterPreBindPlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "PreBind")
}

// RegisterScorePlugin returns a function to register a Score Plugin to a given registry.
func RegisterScorePlugin(pluginName string, pluginNewFunc runtime.PluginFactory, weight int32) RegisterPluginFunc {
	return RegisterPluginAsExtensionsWithWeight(pluginName, weight, pluginNewFunc, "Score")
}

// RegisterPreScorePlugin returns a function to register a Score Plugin to a given registry.
func RegisterPreScorePlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "PreScore")
}

// RegisterBindPlugin returns a function to register a Bind Plugin to a given registry.
func RegisterBindPlugin(pluginName string, pluginNewFunc runtime.PluginFactory) RegisterPluginFunc {
	return RegisterPluginAsExtensions(pluginName, pluginNewFunc, "Bind")
}

// RegisterPluginAsExtensions returns a function to register a Plugin as given extensionPoints to a given registry.
func RegisterPluginAsExtensions(pluginName string, pluginNewFunc runtime.PluginFactory, extensions ...string) RegisterPluginFunc {
	return RegisterPluginAsExtensionsWithWeight(pluginName, 1, pluginNewFunc, extensions...)
}

// RegisterPluginAsExtensionsWithWeight returns a function to register a Plugin as given extensionPoints with weight to a given registry.
func RegisterPluginAsExtensionsWithWeight(pluginName string, weight int32, pluginNewFunc runtime.PluginFactory, extensions ...string) RegisterPluginFunc {
	return func(reg *runtime.Registry, profile *schedulerapi.SchedulerProfile) {
		reg.Register(pluginName, pluginNewFunc)
		for _, extension := range extensions {
			ps := getPluginSetByExtension(profile.Plugins, extension)
			if ps == nil {
				continue
			}
			ps.Enabled = append(ps.Enabled, schedulerapi.Plugin{Name: pluginName, Weight: weight})
		}
		// Use defaults from latest config API version.
		var gvk schema.GroupVersionKind
		gvk = schedulerconfigv1.SchemeGroupVersion.WithKind(pluginName + "Args")
		if args, _, err := configDecoder.Decode(nil, &gvk, nil); err == nil {
			profile.PluginConfig = append(profile.PluginConfig, schedulerapi.PluginConfig{
				Name: pluginName,
				Args: args,
			})
		}
	}
}

func getPluginSetByExtension(plugins *schedulerapi.Plugins, extension string) *schedulerapi.PluginSet {
	switch extension {
	case "QueueSort":
		return &plugins.QueueSort
	case "Filter":
		return &plugins.Filter
	case "PreFilter":
		return &plugins.PreFilter
	case "PreScore":
		return &plugins.PreScore
	case "Score":
		return &plugins.Score
	case "Bind":
		return &plugins.Bind
	case "Reserve":
		return &plugins.Reserve
	case "Permit":
		return &plugins.Permit
	case "PreBind":
		return &plugins.PreBind
	case "PostBind":
		return &plugins.PostBind
	default:
		return nil
	}
}

// BuildRunnerInfos build RunnerInfo slice from a v1.Runner slice
func BuildRunnerInfos(runners []*corev1.Runner) []*framework.RunnerInfo {
	res := make([]*framework.RunnerInfo, len(runners))
	for i := 0; i < len(runners); i++ {
		res[i] = framework.NewRunnerInfo()
		res[i].SetRunner(runners[i])
	}
	return res
}
