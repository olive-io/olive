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

package app

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
	utilversion "k8s.io/component-base/version"

	"github.com/olive-io/olive/plane/options"
)

// NewPlaneServer provides a CLI handler for 'start master' command
// with a default ServerOptions.
func NewPlaneServer(ctx context.Context, defaults *options.ServerOptions, skipDefaultComponentGlobalsRegistrySet bool) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Use:   "olive-plane",
		Short: "a component of olive",
		//Version: version.,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if skipDefaultComponentGlobalsRegistrySet {
				return nil
			}
			return featuregate.DefaultComponentGlobalsRegistry.Set()
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.StartPlaneServer(c.Context()); err != nil {
				return err
			}
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.SetContext(ctx)

	cmd.SetOut(o.StdOut)
	cmd.SetErr(o.StdErr)

	//cmd.SetUsageFunc(func(cmd *cobra.Command) error {
	//	if err := PrintUsage(o.StdOut); err != nil {
	//		return err
	//	}
	//
	//	return PrintFlags(o.StdOut)
	//})

	flags := cmd.Flags()
	o.EmbedEtcdOptions.AddFlags(flags)
	o.RecommendedOptions.AddFlags(flags)

	defaultOliveVersion := "1.0"
	// Register the "Wardle" component with the global component registry,
	// associating it with its effective version and feature gate configuration.
	// Will skip if the component has been registered, like in the integration test.
	_, wardleFeatureGate := featuregate.DefaultComponentGlobalsRegistry.ComponentGlobalsOrRegister(
		options.PlaneComponentName, utilversion.NewEffectiveVersion(defaultOliveVersion),
		featuregate.NewVersionedFeatureGate(version.MustParse(defaultOliveVersion)))

	// Add versioned feature specifications for the "BanFlunder" feature.
	// These specifications, together with the effective version, determine if the feature is enabled.
	utilruntime.Must(wardleFeatureGate.AddVersioned(map[featuregate.Feature]featuregate.VersionedSpecs{
		"BanFlunder": {
			{Version: version.MustParse("1.2"), Default: true, PreRelease: featuregate.GA, LockToDefault: true},
			{Version: version.MustParse("1.1"), Default: true, PreRelease: featuregate.Beta},
			{Version: version.MustParse("1.0"), Default: false, PreRelease: featuregate.Alpha},
		},
	}))

	// Register the default kube component if not already present in the global registry.
	_, _ = featuregate.DefaultComponentGlobalsRegistry.ComponentGlobalsOrRegister(featuregate.DefaultKubeComponent,
		utilversion.NewEffectiveVersion(utilversion.DefaultKubeBinaryVersion), utilfeature.DefaultMutableFeatureGate)

	// Set the emulation version mapping from the "olive" component to the kube component.
	// This ensures that the emulation version of the latter is determined by the emulation version of the former.
	utilruntime.Must(featuregate.DefaultComponentGlobalsRegistry.SetEmulationVersionMapping(options.PlaneComponentName, featuregate.DefaultKubeComponent, OliveVersionToKubeVersion))

	featuregate.DefaultComponentGlobalsRegistry.AddFlags(flags)

	return cmd
}

func OliveVersionToKubeVersion(ver *version.Version) *version.Version {
	if ver.Major() != 1 {
		return nil
	}
	kubeVer := utilversion.DefaultKubeEffectiveVersion().BinaryVersion()
	// "1.2" maps to kubeVer
	offset := int(ver.Minor()) - 2
	mappedVer := kubeVer.OffsetMinor(offset)
	if mappedVer.GreaterThan(kubeVer) {
		return kubeVer
	}
	return mappedVer
}

func PrintUsage(out io.Writer) error {
	_, err := fmt.Fprintln(out, usageline)
	return err
}

func PrintFlags(out io.Writer) error {
	_, err := fmt.Fprintln(out, flagsline)
	return err
}
