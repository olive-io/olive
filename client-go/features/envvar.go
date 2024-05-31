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

package features

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/util/naming"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

// internalPackages are packages that ignored when creating a name for featureGates. These packages are in the common
// call chains, so they'd be unhelpful as names.
var internalPackages = []string{"github.com/olive-io/olive/client-go/features/envvar.go"}

var _ Gates = &envVarFeatureGates{}

// newEnvVarFeatureGates creates a feature gate that allows for registration
// of features and checking if the features are enabled.
//
// On the first call to Enabled, the effective state of all known features is loaded from
// environment variables. The environment variable read for a given feature is formed by
// concatenating the prefix "KUBE_FEATURE_" with the feature's name.
//
// For example, if you have a feature named "MyFeature"
// setting an environmental variable "KUBE_FEATURE_MyFeature"
// will allow you to configure the state of that feature.
//
// Please note that environmental variables can only be set to the boolean value.
// Incorrect values will be ignored and logged.
func newEnvVarFeatureGates(features map[Feature]FeatureSpec) *envVarFeatureGates {
	known := map[Feature]FeatureSpec{}
	for name, spec := range features {
		known[name] = spec
	}

	fg := &envVarFeatureGates{
		callSiteName: naming.GetNameFromCallsite(internalPackages...),
		known:        known,
	}
	fg.enabled.Store(map[Feature]bool{})

	return fg
}

// envVarFeatureGates implements Gates and allows for feature registration.
type envVarFeatureGates struct {
	// callSiteName holds the name of the file
	// that created this instance
	callSiteName string

	// readEnvVarsOnce guards reading environmental variables
	readEnvVarsOnce sync.Once

	// known holds known feature gates
	known map[Feature]FeatureSpec

	// enabled holds a map[Feature]bool
	// with values explicitly set via env var
	enabled atomic.Value

	// readEnvVars holds the boolean value which
	// indicates whether readEnvVarsOnce has been called.
	readEnvVars atomic.Bool
}

// Enabled returns true if the key is enabled.  If the key is not known, this call will panic.
func (f *envVarFeatureGates) Enabled(key Feature) bool {
	if v, ok := f.getEnabledMapFromEnvVar()[key]; ok {
		return v
	}
	if v, ok := f.known[key]; ok {
		return v.Default
	}
	panic(fmt.Errorf("feature %q is not registered in FeatureGates %q", key, f.callSiteName))
}

// getEnabledMapFromEnvVar will fill the enabled map on the first call.
// This is the only time a known feature can be set to a value
// read from the corresponding environmental variable.
func (f *envVarFeatureGates) getEnabledMapFromEnvVar() map[Feature]bool {
	f.readEnvVarsOnce.Do(func() {
		featureGatesState := map[Feature]bool{}
		for feature, featureSpec := range f.known {
			featureState, featureStateSet := os.LookupEnv(fmt.Sprintf("KUBE_FEATURE_%s", feature))
			if !featureStateSet {
				continue
			}
			boolVal, boolErr := strconv.ParseBool(featureState)
			switch {
			case boolErr != nil:
				utilruntime.HandleError(fmt.Errorf("cannot set feature gate %q to %q, due to %v", feature, featureState, boolErr))
			case featureSpec.LockToDefault:
				if boolVal != featureSpec.Default {
					utilruntime.HandleError(fmt.Errorf("cannot set feature gate %q to %q, feature is locked to %v", feature, featureState, featureSpec.Default))
					break
				}
				featureGatesState[feature] = featureSpec.Default
			default:
				featureGatesState[feature] = boolVal
			}
		}
		f.enabled.Store(featureGatesState)
		f.readEnvVars.Store(true)

		for feature, featureSpec := range f.known {
			if featureState, ok := featureGatesState[feature]; ok {
				klog.V(1).InfoS("Feature gate updated state", "feature", feature, "enabled", featureState)
				continue
			}
			klog.V(1).InfoS("Feature gate default state", "feature", feature, "enabled", featureSpec.Default)
		}
	})
	return f.enabled.Load().(map[Feature]bool)
}

func (f *envVarFeatureGates) hasAlreadyReadEnvVar() bool {
	return f.readEnvVars.Load()
}
