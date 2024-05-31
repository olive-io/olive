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

package runtime

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/yaml"

	"github.com/olive-io/olive/mon/scheduler/framework"
	plfeature "github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
)

// PluginFactory is a function that builds a plugin.
type PluginFactory = func(ctx context.Context, configuration runtime.Object, f framework.Handle) (framework.Plugin, error)

// PluginFactoryWithFts is a function that builds a plugin with certain feature gates.
type PluginFactoryWithFts func(context.Context, runtime.Object, framework.Handle, plfeature.Features) (framework.Plugin, error)

// FactoryAdapter can be used to inject feature gates for a plugin that needs
// them when the caller expects the older PluginFactory method.
func FactoryAdapter(fts plfeature.Features, withFts PluginFactoryWithFts) PluginFactory {
	return func(ctx context.Context, plArgs runtime.Object, fh framework.Handle) (framework.Plugin, error) {
		return withFts(ctx, plArgs, fh, fts)
	}
}

// DecodeInto decodes configuration whose type is *runtime.Unknown to the interface into.
func DecodeInto(obj runtime.Object, into interface{}) error {
	if obj == nil {
		return nil
	}
	configuration, ok := obj.(*runtime.Unknown)
	if !ok {
		return fmt.Errorf("want args of type runtime.Unknown, got %T", obj)
	}
	if configuration.Raw == nil {
		return nil
	}

	switch configuration.ContentType {
	// If ContentType is empty, it means ContentTypeJSON by default.
	case runtime.ContentTypeJSON, "":
		return json.Unmarshal(configuration.Raw, into)
	case runtime.ContentTypeYAML:
		return yaml.Unmarshal(configuration.Raw, into)
	default:
		return fmt.Errorf("not supported content type %s", configuration.ContentType)
	}
}

// Registry is a collection of all available plugins. The framework uses a
// registry to enable and initialize configured plugins.
// All plugins must be in the registry before initializing the framework.
type Registry map[string]PluginFactory

// Register adds a new plugin to the registry. If a plugin with the same name
// exists, it returns an error.
func (r Registry) Register(name string, factory PluginFactory) error {
	if _, ok := r[name]; ok {
		return fmt.Errorf("a plugin named %v already exists", name)
	}
	r[name] = factory
	return nil
}

// Unregister removes an existing plugin from the registry. If no plugin with
// the provided name exists, it returns an error.
func (r Registry) Unregister(name string) error {
	if _, ok := r[name]; !ok {
		return fmt.Errorf("no plugin named %v exists", name)
	}
	delete(r, name)
	return nil
}

// Merge merges the provided registry to the current one.
func (r Registry) Merge(in Registry) error {
	for name, factory := range in {
		if err := r.Register(name, factory); err != nil {
			return err
		}
	}
	return nil
}
