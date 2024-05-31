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
	"k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"

	clientfeatures "github.com/olive-io/olive/client-go/features"
)

const ()

func init() {
	runtime.Must(utilfeature.DefaultMutableFeatureGate.Add(defaultOliveFeatureGates))

	// Register all client features with kube's feature gate instance and make all client-go
	// feature checks use kube's instance. The effect is that for kube binaries, client-go
	// features are wired to the existing --feature-gates flag just as all other features
	// are. Further, client-go features automatically support the existing mechanisms for
	// feature enablement metrics and test overrides.
	ca := &clientAdapter{utilfeature.DefaultMutableFeatureGate}
	runtime.Must(clientfeatures.AddFeaturesToExistingFeatureGates(ca))
	clientfeatures.ReplaceFeatureGates(ca)
}

// defaultOliveFeatureGates consists of all known Olive-specific feature keys.
// To add a new feature, define a key for it above and add it here. The features will be
// available throughout Olive binaries.
//
// Entries are separated from each other with blank lines to avoid sweeping gofmt changes
// when adding or removing one entry.
var defaultOliveFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{}
