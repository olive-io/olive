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

	"k8s.io/component-base/featuregate"

	clientfeatures "github.com/olive-io/olive/client-go/features"
)

// clientAdapter adapts a k8s.io/component-base/featuregate.MutableFeatureGate to client-go's
// feature Gate and Registry interfaces. The component-base types Feature, FeatureSpec, and
// prerelease, and the component-base prerelease constants, are duplicated by parallel types and
// constants in client-go. The parallel types exist to allow the feature gate mechanism to be used
// for client-go features without introducing a circular dependency between component-base and
// client-go.
type clientAdapter struct {
	mfg featuregate.MutableFeatureGate
}

var _ clientfeatures.Gates = &clientAdapter{}

func (a *clientAdapter) Enabled(name clientfeatures.Feature) bool {
	return a.mfg.Enabled(featuregate.Feature(name))
}

var _ clientfeatures.Registry = &clientAdapter{}

func (a *clientAdapter) Add(in map[clientfeatures.Feature]clientfeatures.FeatureSpec) error {
	out := map[featuregate.Feature]featuregate.FeatureSpec{}
	for name, spec := range in {
		converted := featuregate.FeatureSpec{
			Default:       spec.Default,
			LockToDefault: spec.LockToDefault,
		}
		switch spec.PreRelease {
		case clientfeatures.Alpha:
			converted.PreRelease = featuregate.Alpha
		case clientfeatures.Beta:
			converted.PreRelease = featuregate.Beta
		case clientfeatures.GA:
			converted.PreRelease = featuregate.GA
		case clientfeatures.Deprecated:
			converted.PreRelease = featuregate.Deprecated
		default:
			// The default case implies programmer error.  The same set of prerelease
			// constants must exist in both component-base and client-go, and each one
			// must have a case here.
			panic(fmt.Sprintf("unrecognized prerelease %q of feature %q", spec.PreRelease, name))
		}
		out[featuregate.Feature(name)] = converted
	}
	return a.mfg.Add(out)
}
